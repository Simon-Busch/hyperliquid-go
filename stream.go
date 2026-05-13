package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// pingInterval matches Hyperliquid's expected interval (upstream uses 50s)
	pingInterval = 50 * time.Second

	// readDeadline is the maximum time to wait for a read before timing out.
	// Set slightly longer than pingInterval to allow for network latency.
	readDeadline = pingInterval + 10*time.Second

	// reconnectBaseWait is the initial wait time before reconnecting
	reconnectBaseWait = time.Second

	// maxReconnectWait caps the exponential backoff
	maxReconnectWait = time.Minute

	// connectTimeout for dial operations
	connectTimeout = 10 * time.Second
)

// Stream manages a WebSocket connection to the Hyperliquid API.
// It handles automatic reconnection, subscription management, and POST requests.
type Stream struct {
	url string

	// Connection state (atomic for lock-free reads)
	connected atomic.Bool

	// Connection and synchronization
	conn    *websocket.Conn
	connMu  sync.RWMutex  // Protects conn
	writeMu sync.Mutex    // Serializes writes
	cancel  func()        // Cancels current connection's goroutines
	wg      sync.WaitGroup

	// Subscription management
	subscriptions map[subKey]map[int]*subscriptionCallback
	subMu         sync.RWMutex
	nextSubID     atomic.Int32

	// Reconnection settings
	reconnectMu          sync.Mutex
	reconnectTimer       *time.Timer
	reconnectAttempts    int
	maxReconnectAttempts int           // 0 = unlimited (default)
	reconnectWait        time.Duration // initial backoff, reset on each successful Connect

	// POST request tracking
	nextPostID      atomic.Int32
	pendingRequests map[int]*pendingRequest
	pendingMu       sync.RWMutex

	// Lifecycle
	closed    atomic.Bool
	closedMu  sync.Mutex
	closeOnce sync.Once

	// Pluggable logger for non-fatal warnings; defaults to no-op.
	logger Logger
}

// SetLogger plugs in a Logger to receive warnings and reconnection
// diagnostics. A nil logger reverts to the no-op default.
func (s *Stream) SetLogger(l Logger) {
	if l == nil {
		s.logger = nopLogger{}
		return
	}
	s.logger = l
}

// logf is a tiny helper that dispatches to the configured Logger's Warnf,
// falling back to the no-op logger when none is set.
func (s *Stream) warnf(format string, args ...any) {
	if s.logger == nil {
		return
	}
	s.logger.Warnf(format, args...)
}

// NewStream creates a new WebSocket Stream targeting baseURL. The Stream
// is not connected until Connect is called. An error is returned if
// baseURL cannot be parsed as a URL.
func NewStream(baseURL string) (*Stream, error) {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: invalid stream URL %q: %w", baseURL, err)
	}
	parsedURL.Scheme = "wss"
	parsedURL.Path = "/ws"
	wsURL := parsedURL.String()

	return &Stream{
		url:             wsURL,
		subscriptions:   make(map[subKey]map[int]*subscriptionCallback),
		pendingRequests: make(map[int]*pendingRequest),
		reconnectWait:   reconnectBaseWait,
		logger:          nopLogger{},
	}, nil
}

// Connect establishes the WebSocket connection.
// It's safe to call multiple times - subsequent calls return immediately if already connected.
func (s *Stream) Connect(ctx context.Context) error {
	if s.closed.Load() {
		return fmt.Errorf("client is closed")
	}

	if s.connected.Load() {
		return nil
	}

	s.connMu.Lock()
	defer s.connMu.Unlock()

	// Double-check after acquiring lock
	if s.conn != nil && s.connected.Load() {
		return nil
	}

	// Clean up any existing connection
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}

	// Cancel previous goroutines if any
	if s.cancel != nil {
		s.cancel()
		s.wg.Wait()
	}

	// Dial new connection
	dialer := websocket.Dialer{
		ReadBufferSize:  16384,
		WriteBufferSize: 4096,
	}
	dialCtx, dialCancel := context.WithTimeout(ctx, connectTimeout)
	defer dialCancel()

	//nolint:bodyclose // WebSocket connections don't have response bodies to close
	conn, _, err := dialer.DialContext(dialCtx, s.url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	s.conn = conn
	s.connected.Store(true)

	// Reset reconnection state on successful connection.
	// reconnectWait is reset only if it was never customized via
	// WithReconnectWait; in that case it remains at its starting value.
	s.reconnectMu.Lock()
	s.reconnectAttempts = 0
	s.reconnectMu.Unlock()

	// Create context for this connection's goroutines
	connCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Start read and ping pumps (per-connection, like upstream)
	s.wg.Add(2)
	go s.readPump(connCtx)
	go s.pingPump(connCtx)

	// Resubscribe to all previous subscriptions
	if err := s.resubscribeAll(); err != nil {
		s.connected.Store(false)
		cancel()
		_ = conn.Close()
		s.conn = nil
		return fmt.Errorf("resubscribe failed: %w", err)
	}

	return nil
}

// Subscription is the handle returned by Stream.Subscribe. Call Close to
// deregister the callback and emit an unsubscribe frame when no listener
// remains.
type Subscription struct {
	filter subscriptionFilter
	id     int
	stream *Stream
	closed atomic.Bool
}

// Close deregisters the callback registered by Stream.Subscribe. It is
// safe to call multiple times: subsequent calls after the first return
// nil without sending another unsubscribe frame.
func (s *Subscription) Close() error {
	if s == nil {
		return nil
	}
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	return s.stream.unsubscribe(s)
}

// Subscribe registers a callback for the given subscription filter and
// returns a Subscription handle. Call sub.Close() to deregister.
func (s *Stream) Subscribe(filter subscriptionFilter, callback func(WSMessage)) (*Subscription, error) {
	if callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}

	s.subMu.Lock()
	key := filter.key()
	id := int(s.nextSubID.Add(1))

	if s.subscriptions[key] == nil {
		s.subscriptions[key] = make(map[int]*subscriptionCallback)
	}

	s.subscriptions[key][id] = &subscriptionCallback{
		id:       id,
		callback: callback,
	}
	s.subMu.Unlock()

	// Send subscribe message (outside lock to avoid deadlock)
	if err := s.sendSubscribe(filter); err != nil {
		s.subMu.Lock()
		delete(s.subscriptions[key], id)
		s.subMu.Unlock()
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	return &Subscription{filter: filter, id: id, stream: s}, nil
}

// unsubscribe drops the callback registration recorded by Subscribe and
// emits an unsubscribe frame once no listener remains for the filter.
func (s *Stream) unsubscribe(sub *Subscription) error {
	s.subMu.Lock()
	key := sub.filter.key()
	subs, ok := s.subscriptions[key]
	if !ok {
		s.subMu.Unlock()
		return fmt.Errorf("subscription not found")
	}

	if _, ok := subs[sub.id]; !ok {
		s.subMu.Unlock()
		return fmt.Errorf("subscription ID not found")
	}

	delete(subs, sub.id)

	shouldUnsubscribe := len(subs) == 0
	if shouldUnsubscribe {
		delete(s.subscriptions, key)
	}
	s.subMu.Unlock()

	if shouldUnsubscribe {
		if err := s.sendUnsubscribe(sub.filter); err != nil {
			return fmt.Errorf("unsubscribe: %w", err)
		}
	}

	return nil
}

// Close shuts down the WebSocket client and releases all resources.
func (s *Stream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		s.connected.Store(false)

		// Cancel reconnection timer
		s.reconnectMu.Lock()
		if s.reconnectTimer != nil {
			s.reconnectTimer.Stop()
			s.reconnectTimer = nil
		}
		s.reconnectMu.Unlock()

		// Clean up pending POST requests
		s.pendingMu.Lock()
		for id, pending := range s.pendingRequests {
			close(pending.responseChan)
			delete(s.pendingRequests, id)
		}
		s.pendingMu.Unlock()

		// Cancel goroutines and close connection
		s.connMu.Lock()
		if s.cancel != nil {
			s.cancel()
		}
		if s.conn != nil {
			err = s.conn.Close()
			s.conn = nil
		}
		s.connMu.Unlock()

		// Wait for goroutines to finish
		s.wg.Wait()
	})
	return err
}

// readPump reads messages from the WebSocket connection.
// Runs for the lifetime of a single connection (context-aware, like upstream).
func (s *Stream) readPump(ctx context.Context) {
	defer s.wg.Done()
	defer s.handleDisconnect()

	// Grab conn once — if it changes, context will be cancelled and we exit.
	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()
	if conn == nil {
		return
	}

	for {
		// Set read deadline to detect dead connections.
		_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				s.warnf("websocket read error: %v", err)
			}
			return // Exit pump, handleDisconnect will trigger reconnection
		}

		// Skip server hello
		if string(msg) == "Websocket connection established." {
			continue
		}

		// Parse and dispatch
		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			s.warnf("websocket message parse error: %v", err)
			continue
		}

		s.dispatch(wsMsg)
	}
}

// pingPump sends periodic ping messages to keep the connection alive.
// Runs for the lifetime of a single connection (context-aware, like upstream).
func (s *Stream) pingPump(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.sendPing(); err != nil {
				s.warnf("ping error: %v", err)
				// Don't return here - readPump will detect the error
				// and handleDisconnect will trigger reconnection
			}
		}
	}
}

// dispatch routes messages to appropriate callbacks.
func (s *Stream) dispatch(msg WSMessage) {
	// Handle pong responses (sent by Hyperliquid as JSON, not WebSocket protocol pongs)
	if msg.Channel == "pong" {
		// Pong received - connection is alive, nothing to do
		return
	}

	// Handle subscription confirmations
	if msg.Channel == "subscriptionResponse" {
		// Subscription confirmed by server, nothing to do
		return
	}

	// Handle POST responses
	if msg.Channel == "post" {
		var postResp WsPostResponseData
		if err := json.Unmarshal(msg.Data, &postResp); err != nil {
			s.warnf("failed to unmarshal post response: %v", err)
			return
		}

		s.pendingMu.RLock()
		pending, ok := s.pendingRequests[postResp.ID]
		s.pendingMu.RUnlock()

		if ok {
			select {
			case pending.responseChan <- postResp:
			default:
				// Channel closed (timeout) or full
			}
		}
		return
	}

	// Copy matching callbacks under lock
	s.subMu.RLock()
	var callbacks []func(WSMessage)
	for key, subs := range s.subscriptions {
		if matchSubscription(key, msg) {
			for _, sub := range subs {
				callbacks = append(callbacks, sub.callback)
			}
		}
	}
	s.subMu.RUnlock()

	// Execute callbacks without holding lock
	for _, cb := range callbacks {
		cb(msg)
	}
}

// resubscribeAll resends subscribe messages for all active subscriptions.
func (s *Stream) resubscribeAll() error {
	s.subMu.RLock()
	keys := make([]subKey, 0, len(s.subscriptions))
	for key, subs := range s.subscriptions {
		if len(subs) > 0 {
			keys = append(keys, key)
		}
	}
	s.subMu.RUnlock()

	for _, key := range keys {
		f := subscriptionFilter{
			Type:     key.typ,
			Coin:     key.coin,
			User:     key.user,
			Interval: key.interval,
			Dex:      key.dex,
		}
		if err := s.sendSubscribe(f); err != nil {
			return fmt.Errorf("resubscribe %s: %w", key.typ, err)
		}
	}
	return nil
}

func (s *Stream) sendSubscribe(f subscriptionFilter) error {
	return s.writeJSON(WsCommand{
		Method:       "subscribe",
		Subscription: &f,
	})
}

func (s *Stream) sendUnsubscribe(f subscriptionFilter) error {
	return s.writeJSON(WsCommand{
		Method:       "unsubscribe",
		Subscription: &f,
	})
}

func (s *Stream) sendPing() error {
	return s.writeJSON(WsCommand{Method: "ping"})
}

func (s *Stream) writeJSON(v any) error {
	// Marshal outside the lock so serialization doesn't block other writers
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if !s.connected.Load() {
		return fmt.Errorf("not connected")
	}

	s.connMu.RLock()
	conn := s.conn
	s.connMu.RUnlock()

	if conn == nil {
		return fmt.Errorf("connection closed")
	}

	// WriteMessage is a single frame write — no NextWriter/Close dance
	return conn.WriteMessage(websocket.TextMessage, data)
}

// matchSubscription checks if a message matches a subscription key.
func matchSubscription(key subKey, msg WSMessage) bool {
	// Channel matching
	// NOTE: Most subscription types have matching channel names, but there are exceptions:
	// - "userEvents" subscription sends messages on "user" channel (Hyperliquid API quirk)
	channelMatch := false
	switch key.typ {
	case "allMids":
		channelMatch = msg.Channel == "allMids"
	case "notification":
		channelMatch = msg.Channel == "notification"
	case "webData2":
		channelMatch = msg.Channel == "webData2"
	case "candle":
		channelMatch = msg.Channel == "candle"
	case "l2Book":
		channelMatch = msg.Channel == "l2Book"
	case "trades":
		channelMatch = msg.Channel == "trades"
	case "orderUpdates":
		channelMatch = msg.Channel == "orderUpdates"
	case "userEvents":
		// API quirk: "userEvents" subscription type receives messages on "user" channel
		channelMatch = msg.Channel == "user"
	case "userFills":
		channelMatch = msg.Channel == "userFills"
	case "userFundings":
		channelMatch = msg.Channel == "userFundings"
	case "userNonFundingLedgerUpdates":
		channelMatch = msg.Channel == "userNonFundingLedgerUpdates"
	case "activeAssetCtx":
		channelMatch = msg.Channel == "activeAssetCtx"
	case "activeAssetData":
		channelMatch = msg.Channel == "activeAssetData"
	case "userTwapSliceFills":
		channelMatch = msg.Channel == "userTwapSliceFills"
	case "userTwapHistory":
		channelMatch = msg.Channel == "userTwapHistory"
	case "bbo":
		channelMatch = msg.Channel == "bbo"
	default:
		return false
	}

	if !channelMatch {
		return false
	}

	// Early return if no additional filtering needed
	if key.coin == "" && key.user == "" {
		return true
	}

	// orderUpdates data is a JSON array, not an object — matching is purely by channel.
	// User is implicit from the subscription, not present in message data.
	if key.typ == "orderUpdates" {
		return true
	}

	// Single unmarshal for both coin and user matching
	var msgData struct {
		Coin string `json:"coin"`
		User string `json:"user"`
	}
	if err := json.Unmarshal(msg.Data, &msgData); err != nil {
		return false
	}

	// Coin matching
	if key.coin != "" && msgData.Coin != key.coin {
		return false
	}

	// User matching
	if key.user != "" {
		if !strings.EqualFold(msgData.User, key.user) {
			return false
		}
	}

	return true
}
