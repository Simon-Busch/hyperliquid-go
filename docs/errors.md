# Errors reference

The SDK distinguishes three kinds of failure:

1. **Pre-flight validation failures** — `*ValidationError`, returned by `validate()` before any signing happens. Every placement, modify, cancel and close call funnels through one shared validator; this is your first line of defence against malformed orders.
2. **Server-side API failures** — `APIError`, returned by the HTTP layer when `/info` or `/exchange` rejects a request.
3. **Sentinel errors** — exported `error` values for static comparison via `errors.Is`.

## `ValidationError` {#validationerror}

```go
type ValidationError struct {
    Field   string // "Size", "Price", "Coin", "ReduceOnly", "Bracket", ...
    Code    string // stable machine code; see table below
    Message string // human-readable
    Got     any    // optional: the value that was rejected
    Want    any    // optional: what would have been accepted
}
func (e *ValidationError) Error() string
```

Returned as a `*ValidationError`; use `errors.As`:

```go
var ve *hl.ValidationError
if _, err := c.Trade.PlaceALO("ETH", hl.Buy, 0.001, 1500); err != nil {
    if errors.As(err, &ve) {
        switch ve.Code {
        case "size_below_min":
            fmt.Println("bump size to at least", ve.Want)
        case "tp_wrong_side_buy":
            fmt.Println("TP must be above entry on a buy")
        default:
            fmt.Println("validation:", ve.Message)
        }
    }
}
```

### Code table

| `Code`                          | Meaning                                                                         | Returned by                                                       |
|---------------------------------|---------------------------------------------------------------------------------|-------------------------------------------------------------------|
| `coin_required`                 | `OrderSpec.Coin` was empty.                                                     | every placement method, `Modify*`.                                |
| `unknown_coin`                  | The coin is not in the cached metadata.                                         | every placement method, `Modify*`.                                |
| `size_below_min`                | `size <= 0` or `size < AssetMeta.MinSize`.                                      | every placement method, `Modify*` (when `WithSize` is used).      |
| `size_step_violation`           | `size` is not an integer multiple of `MinSize` (sz-decimals quantum).           | every placement method, `Modify*` (when `WithSize` is used).      |
| `price_non_positive`            | `px <= 0` on a method that takes an explicit price.                             | `PlaceALO/IOC/GTC`, `PlaceTrigger`, `Modify*`.                    |
| `significant_figures`           | `px` has more than 5 significant figures (post-rounding).                       | `PlaceALO/IOC/GTC`, `PlaceTrigger`, `Modify*`.                    |
| `wrong_side_for_reduce`         | Reduce-only `Buy` on a long, or reduce-only `Sell` on a short — would *grow* exposure. | every reduce-only placement.                              |
| `tp_wrong_side_buy`             | Bracket TP is at or below entry on a buy.                                       | placements with `WithBracket` / `WithTakeProfit` on a buy.        |
| `tp_wrong_side_sell`            | Bracket TP is at or above entry on a sell.                                      | placements with `WithBracket` / `WithTakeProfit` on a sell.       |
| `sl_wrong_side_buy`             | Bracket SL is at or above entry on a buy.                                       | placements with `WithBracket` / `WithStopLoss` on a buy.          |
| `sl_wrong_side_sell`            | Bracket SL is at or below entry on a sell.                                      | placements with `WithBracket` / `WithStopLoss` on a sell.         |
| `bracket_size_exceeds_entry`    | `WithTPSize` or `WithSLSize` is larger than the parent size.                    | placements with bracket size overrides.                           |
| `no_position`                   | `ClosePosition` called for a coin with no open position in the cached state.    | `ClosePosition`.                                                  |
| `close_size_exceeds_position`   | `WithSize(x)` on `ClosePosition` where `x > abs(position.size)`.                | `ClosePosition`.                                                  |
| `unsupported_option`            | An option was passed to a method that doesn't honour it (e.g. `WithSlippage` on `PlaceALO`, `AsMarket` outside `PlaceTrigger`, `WithSize` outside `Close`/`Modify`). | any placement or modify verb. |
| `modify_target_required`        | `Modify` called without an OID and `ModifyByCloid` called without a cloid.      | `Modify*`.                                                        |
| `modify_no_change`              | `Modify` called without `WithLimit(px)` or `WithSize(sz)`.                      | `Modify*`.                                                        |

### State refresh failures

Validation also refreshes the cached `UserState` before the position-aware rules run. If `RefreshState` fails the placement returns a wrapped error of the form

```
validate: refresh user state: <underlying error> (use hl.SkipValidation() to bypass)
```

This is not a `ValidationError.Code` — it's a `fmt.Errorf` wrap. Use `errors.As` to detect `*ValidationError`; fall through to message-string inspection for refresh failures, or pass `hl.SkipValidation()` per-call if you accept the trade-off.

## `APIError` {#apierror}

Returned when the server rejects a request.

```go
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"msg"`
    Data    any    `json:"data,omitempty"`
}
func (e APIError) Error() string
```

Use `errors.As` to match it:

```go
var apiErr hl.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("api error %d: %s\n", apiErr.Code, apiErr.Message)
}
```

Some endpoints surface their errors inside the `Result.Error` string instead of the returned `error` — always inspect both when placing orders.

## `ErrMissingPrivateKey` {#errmissingprivatekey}

Sentinel returned when a `Trader` method is invoked on a `Client` constructed without `WithPrivateKey`.

```go
var ErrMissingPrivateKey = errors.New("hyperliquid: trader requires WithPrivateKey")
```

Match via `errors.Is`:

```go
if errors.Is(err, hl.ErrMissingPrivateKey) {
    // rebuild the client with WithPrivateKey
}
```

In practice `c.Trade` is `nil` in this scenario, so most callers see a nil-pointer panic before this sentinel surfaces — guard with `if c.Trade == nil` if you build clients with optional signing keys.
