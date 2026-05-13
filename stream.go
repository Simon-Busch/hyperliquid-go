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
	MaxReconnectAttempts int           // 0 = unlimited (default)
	ReconnectWait        time.Duration // Can be customized

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
func (w *Stream) SetLogger(l Logger) {
	if l == nil {
		w.logger = nopLogger{}
		return
	}
	w.logger = l
}

// logf is a tiny helper that dispatches to the configured Logger's Warnf,
// falling back to the no-op logger when none is set.
func (w *Stream) warnf(format string, args ...any) {
	if w.logger == nil {
		return
	}
	w.logger.Warnf(format, args...)
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
		ReconnectWait:   reconnectBaseWait,
		logger:          nopLogger{},
	}, nil
}

// Connect establishes the WebSocket connection.
// It's safe to call multiple times - subsequent calls return immediately if already connected.
func (w *Stream) Connect(ctx context.Context) error {
	if w.closed.Load() {
		return fmt.Errorf("client is closed")
	}

	if w.connected.Load() {
		return nil
	}

	w.connMu.Lock()
	defer w.connMu.Unlock()

	// Double-check after acquiring lock
	if w.conn != nil && w.connected.Load() {
		return nil
	}

	// Clean up any existing connection
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}

	// Cancel previous goroutines if any
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait()
	}

	// Dial new connection
	dialer := websocket.Dialer{
		ReadBufferSize:  16384,
		WriteBufferSize: 4096,
	}
	dialCtx, dialCancel := context.WithTimeout(ctx, connectTimeout)
	defer dialCancel()

	//nolint:bodyclose // WebSocket connections don't have response bodies to close
	conn, _, err := dialer.DialContext(dialCtx, w.url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	w.conn = conn
	w.connected.Store(true)

	// Reset reconnection state on successful connection
	w.reconnectMu.Lock()
	w.reconnectAttempts = 0
	w.ReconnectWait = reconnectBaseWait
	w.reconnectMu.Unlock()

	// Create context for this connection's goroutines
	connCtx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	// Start read and ping pumps (per-connection, like upstream)
	w.wg.Add(2)
	go w.readPump(connCtx)
	go w.pingPump(connCtx)

	// Resubscribe to all previous subscriptions
	if err := w.resubscribeAll(); err != nil {
		w.connected.Store(false)
		cancel()
		_ = conn.Close()
		w.conn = nil
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
func (w *Stream) Subscribe(filter subscriptionFilter, callback func(WSMessage)) (*Subscription, error) {
	if callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}

	w.subMu.Lock()
	key := filter.key()
	id := int(w.nextSubID.Add(1))

	if w.subscriptions[key] == nil {
		w.subscriptions[key] = make(map[int]*subscriptionCallback)
	}

	w.subscriptions[key][id] = &subscriptionCallback{
		id:       id,
		callback: callback,
	}
	w.subMu.Unlock()

	// Send subscribe message (outside lock to avoid deadlock)
	if err := w.sendSubscribe(filter); err != nil {
		w.subMu.Lock()
		delete(w.subscriptions[key], id)
		w.subMu.Unlock()
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	return &Subscription{filter: filter, id: id, stream: w}, nil
}

// unsubscribe drops the callback registration recorded by Subscribe and
// emits an unsubscribe frame once no listener remains for the filter.
func (w *Stream) unsubscribe(sub *Subscription) error {
	w.subMu.Lock()
	key := sub.filter.key()
	subs, ok := w.subscriptions[key]
	if !ok {
		w.subMu.Unlock()
		return fmt.Errorf("subscription not found")
	}

	if _, ok := subs[sub.id]; !ok {
		w.subMu.Unlock()
		return fmt.Errorf("subscription ID not found")
	}

	delete(subs, sub.id)

	shouldUnsubscribe := len(subs) == 0
	if shouldUnsubscribe {
		delete(w.subscriptions, key)
	}
	w.subMu.Unlock()

	if shouldUnsubscribe {
		if err := w.sendUnsubscribe(sub.filter); err != nil {
			return fmt.Errorf("unsubscribe: %w", err)
		}
	}

	return nil
}

// Close shuts down the WebSocket client and releases all resources.
func (w *Stream) Close() error {
	var err error
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		w.connected.Store(false)

		// Cancel reconnection timer
		w.reconnectMu.Lock()
		if w.reconnectTimer != nil {
			w.reconnectTimer.Stop()
			w.reconnectTimer = nil
		}
		w.reconnectMu.Unlock()

		// Clean up pending POST requests
		w.pendingMu.Lock()
		for id, pending := range w.pendingRequests {
			close(pending.responseChan)
			delete(w.pendingRequests, id)
		}
		w.pendingMu.Unlock()

		// Cancel goroutines and close connection
		w.connMu.Lock()
		if w.cancel != nil {
			w.cancel()
		}
		if w.conn != nil {
			err = w.conn.Close()
			w.conn = nil
		}
		w.connMu.Unlock()

		// Wait for goroutines to finish
		w.wg.Wait()
	})
	return err
}

// readPump reads messages from the WebSocket connection.
// Runs for the lifetime of a single connection (context-aware, like upstream).
func (w *Stream) readPump(ctx context.Context) {
	defer w.wg.Done()
	defer w.handleDisconnect()

	// Grab conn once — if it changes, context will be cancelled and we exit.
	w.connMu.RLock()
	conn := w.conn
	w.connMu.RUnlock()
	if conn == nil {
		return
	}

	for {
		// Set read deadline to detect dead connections.
		_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				w.warnf("websocket read error: %v", err)
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
			w.warnf("websocket message parse error: %v", err)
			continue
		}

		w.dispatch(wsMsg)
	}
}

// pingPump sends periodic ping messages to keep the connection alive.
// Runs for the lifetime of a single connection (context-aware, like upstream).
func (w *Stream) pingPump(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.sendPing(); err != nil {
				w.warnf("ping error: %v", err)
				// Don't return here - readPump will detect the error
				// and handleDisconnect will trigger reconnection
			}
		}
	}
}

// dispatch routes messages to appropriate callbacks.
func (w *Stream) dispatch(msg WSMessage) {
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
			w.warnf("failed to unmarshal post response: %v", err)
			return
		}

		w.pendingMu.RLock()
		pending, ok := w.pendingRequests[postResp.ID]
		w.pendingMu.RUnlock()

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
	w.subMu.RLock()
	var callbacks []func(WSMessage)
	for key, subs := range w.subscriptions {
		if matchSubscription(key, msg) {
			for _, sub := range subs {
				callbacks = append(callbacks, sub.callback)
			}
		}
	}
	w.subMu.RUnlock()

	// Execute callbacks without holding lock
	for _, cb := range callbacks {
		cb(msg)
	}
}

// resubscribeAll resends subscribe messages for all active subscriptions.
func (w *Stream) resubscribeAll() error {
	w.subMu.RLock()
	keys := make([]subKey, 0, len(w.subscriptions))
	for key, subs := range w.subscriptions {
		if len(subs) > 0 {
			keys = append(keys, key)
		}
	}
	w.subMu.RUnlock()

	for _, key := range keys {
		f := subscriptionFilter{
			Type:     key.typ,
			Coin:     key.coin,
			User:     key.user,
			Interval: key.interval,
			Dex:      key.dex,
		}
		if err := w.sendSubscribe(f); err != nil {
			return fmt.Errorf("resubscribe %s: %w", key.typ, err)
		}
	}
	return nil
}

func (w *Stream) sendSubscribe(f subscriptionFilter) error {
	return w.writeJSON(WsCommand{
		Method:       "subscribe",
		Subscription: &f,
	})
}

func (w *Stream) sendUnsubscribe(f subscriptionFilter) error {
	return w.writeJSON(WsCommand{
		Method:       "unsubscribe",
		Subscription: &f,
	})
}

func (w *Stream) sendPing() error {
	return w.writeJSON(WsCommand{Method: "ping"})
}

func (w *Stream) writeJSON(v any) error {
	// Marshal outside the lock so serialization doesn't block other writers
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	if !w.connected.Load() {
		return fmt.Errorf("not connected")
	}

	w.connMu.RLock()
	conn := w.conn
	w.connMu.RUnlock()

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
