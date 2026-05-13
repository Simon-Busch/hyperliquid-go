# Stream reference (`c.Stream`)

`c.Stream` is the WebSocket surface. It maintains one connection, multiplexes subscriptions, services in-flight POST requests, and reconnects with exponential backoff. The Stream is not connected by `hyperliquid.New` — call `Connect(ctx)` once before subscribing or posting. On reconnect every active subscription is re-issued.

`c.Stream` is `nil` if the `Client` was built with `WithSkipStream(true)`.

## Contents

- [Lifecycle](#lifecycle)
- [Subscribe and Close](#subscribe)
- [Market subscription constructors](#market-subscriptions)
- [User subscription constructors](#user-subscriptions)
- [POST over WS](#post-over-ws)
- [Reconnect behaviour](#reconnect)
- [Logging](#logging)

---

## Lifecycle

### `Connect` {#lifecycle}

Open the WebSocket. Safe to call multiple times; subsequent calls return immediately if already connected. Returns an error if the URL cannot be dialled or if resubscription of any previously-registered subscription fails.

```go
func (w *Stream) Connect(ctx context.Context) error
```

`ctx` bounds the initial dial. After the connection is established it is *not* used to cancel the read/ping pumps — those are owned by the Stream and cancelled by `Close`.

### `Close`

Tear down the connection, stop reconnect timers, and drain pending POST requests.

```go
func (w *Stream) Close() error
```

`Close` is idempotent (a `sync.Once` guards the shutdown).

**Example**

```go
if err := c.Stream.Connect(ctx); err != nil { log.Fatal(err) }
defer c.Stream.Close()
```

---

## Subscribe and Close {#subscribe}

### `Subscribe`

Register a callback for the supplied subscription filter. Returns a `*Subscription` handle whose `Close()` method deregisters the callback and emits an unsubscribe frame once the last listener for that filter is gone.

```go
func (w *Stream) Subscribe(filter subscriptionFilter, callback func(WSMessage)) (*Subscription, error)
```

**Example**

```go
sub, err := c.Stream.Subscribe(hl.Trades("ETH"), func(m hl.WSMessage) {
    fmt.Printf("%s -> %s\n", m.Channel, string(m.Data))
})
defer sub.Close()
```

### `Subscription.Close`

Tear down the callback. `Close` is idempotent: a second call returns `nil` without sending another unsubscribe frame.

```go
func (s *Subscription) Close() error
```

### Subscription constructors

The `hl.Trades`, `hl.Book`, etc. functions return a `subscriptionFilter` value that is passed straight into `Subscribe`. The field shape (`type`, `coin`, `user`, `interval`, `dex`) matches the wire envelope expected by the Hyperliquid websocket API.

### `WSMessage`

Every subscription callback receives a `WSMessage`. The payload is a raw `json.RawMessage` so callers can decode lazily into the appropriate type from `types.go`.

```go
type WSMessage struct {
    Channel string          `json:"channel"`
    Data    json.RawMessage `json:"data"`
}
```

---

## Market subscription constructors {#market-subscriptions}

All constructors are top-level functions returning a `subscriptionFilter` value.

| Constructor             | WS channel            | Payload type (`types.go`)                 |
|-------------------------|-----------------------|-------------------------------------------|
| `Trades(coin)`          | `trades`              | array of trade records                    |
| `Book(coin)`            | `l2Book`              | `L2Book`                                  |
| `BBO(coin)`             | `bbo`                 | best bid/offer snapshot                   |
| `Candles(coin, "1m")`   | `candle`              | `Candle`                                  |
| `AllMids()`             | `allMids`             | `map[string]string`                       |
| `AllMidsOn(dex)`        | `allMids` (with dex)  | `map[string]string` for that dex          |
| `ActiveAssetCtx(coin)`  | `activeAssetCtx`      | `AssetCtx`                                |
| `ActiveAssetData(addr, coin)` | `activeAssetData` | per-user asset context                   |

**Example — book stream**

```go
sub, err := c.Stream.Subscribe(hl.Book("ETH"), func(m hl.WSMessage) {
    var b hl.L2Book
    _ = json.Unmarshal(m.Data, &b)
})
defer sub.Close()
```

---

## User subscription constructors {#user-subscriptions}

Each takes an account address.

| Constructor             | WS channel                       | Notes                                      |
|-------------------------|----------------------------------|--------------------------------------------|
| `UserEvents(addr)`      | `userEvents`                     | Quirk: upstream channel literally `userEvents`. |
| `UserFills(addr)`       | `userFills`                      | All fills for the account.                 |
| `OrderUpdates(addr)`    | `orderUpdates`                   | Lifecycle events for resting orders.       |
| `UserFundings(addr)`    | `userFundings`                   | Funding payments stream.                   |
| `UserLedger(addr)`      | `userNonFundingLedgerUpdates`    | Ledger entries excluding funding.          |
| `WebData(addr)`         | `webData2`                       | UI-shaped snapshot (formerly WebData2).    |
| `Notifications(addr)`   | `notification`                   | Per-user notifications.                    |
| `UserTwapFills(addr)`   | `userTwapSliceFills`             | TWAP slice fills.                          |
| `UserTwapHistory(addr)` | `userTwapHistory`                | TWAP order history.                        |

**Example — order updates**

```go
sub, err := c.Stream.Subscribe(hl.OrderUpdates(addr), func(m hl.WSMessage) {
    fmt.Println("order update:", string(m.Data))
})
defer sub.Close()
```

---

## POST over WS {#post-over-ws}

The Stream also services REST-style requests. This is useful when the caller wants tighter latency or to keep all traffic on one socket.

### `PostInfoRequest`

Send an info request and wait up to `timeout` (`0` means 30s). Returns the raw payload bytes.

```go
func (w *Stream) PostInfoRequest(payload map[string]any, timeout time.Duration) (json.RawMessage, error)
```

**Example**

```go
data, err := c.Stream.PostInfoRequest(map[string]any{
    "type": "l2Book",
    "coin": "ETH",
}, 5*time.Second)
```

### `PostActionRequest`

Send a pre-signed action. `vaultAddress` is forwarded verbatim — pass `""` to set the wire field to `null`. When `timeout == 0` the call waits up to 30s.

```go
func (w *Stream) PostActionRequest(action any, signature SignatureResult, nonce int64, vaultAddress string, timeout time.Duration) (json.RawMessage, error)
```

The action must already be signed via `SignL1Action` or `SignUserSignedAction` — see [signing.md](./signing.md).

### `PostRequest`

Lower-level than `PostInfoRequest` / `PostActionRequest`. Prefer those.

```go
func (w *Stream) PostRequest(requestType string, payload any, timeout time.Duration) (*WsPostResponseData, error)
```

---

## Reconnect behaviour {#reconnect}

The Stream owns its reconnect state machine:

- On disconnect (read error, server close, ping timeout) the read pump exits and `handleDisconnect` schedules a reconnect.
- The initial wait is `reconnectBaseWait` (1 s); each failed attempt doubles up to `maxReconnectWait` (1 min).
- `Stream.MaxReconnectAttempts` is an exported field (default 0 = unlimited).
- `Stream.ReconnectWait` is the current backoff interval and is reset to 1 s on every successful connect.
- On successful reconnect, every callback registered via `Subscribe` is re-issued before the call returns.

Tune the field directly if you need a non-default backoff:

```go
c.Stream.MaxReconnectAttempts = 5
```

## Logging {#logging}

`Stream` accepts a `Logger` via `Stream.SetLogger(l)`. It is also wired by `hyperliquid.New` from `WithLogger(...)`. The default is a no-op. The Stream uses `Warnf` for transient errors (read/write/ping failures) and otherwise stays silent.

```go
type Logger interface {
    Debugf(format string, args ...any)
    Infof(format string, args ...any)
    Warnf(format string, args ...any)
    Errorf(format string, args ...any)
}
```

**Related**: [signing.md](./signing.md) for signing actions you intend to post over WS.
