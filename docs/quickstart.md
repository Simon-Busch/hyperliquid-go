# Quickstart

This guide walks through the first end-to-end use of the SDK on testnet: build a client, fetch a price, place a resting order, see it appear in the open-order list, cancel it, place a bracketed entry, close the position, and subscribe to a live feed. Every snippet compiles against the current public API.

## 1. Install

```bash
go get github.com/Simon-Busch/hyperliquid-go@latest
```

The package is imported as `hyperliquid` and conventionally aliased to `hl`:

```go
import hl "github.com/Simon-Busch/hyperliquid-go"
```

## 2. Configure credentials

Create a `.env` file in your project root (the SDK does not load it for you, but the integration suite uses [godotenv](https://github.com/joho/godotenv)):

```bash
HL_BASE_URL=https://api.hyperliquid-testnet.xyz
HL_PRIVATE_KEY=0x<your-test-wallet-key>
HL_ACCOUNT_ADDRESS=0x<your-account-or-agent-owner>
HL_TEST_COIN=ETH
HL_TEST_SIZE=0.01
```

Load it explicitly in your program:

```go
_ = godotenv.Load()
pk, err := crypto.HexToECDSA(strings.TrimPrefix(os.Getenv("HL_PRIVATE_KEY"), "0x"))
if err != nil { log.Fatal(err) }
```

## 3. Build the client

```go
c, err := hl.New(
    hl.WithTestnet(),
    hl.WithPrivateKey(pk),
    hl.WithAccount(os.Getenv("HL_ACCOUNT_ADDRESS")),
)
if err != nil { log.Fatal(err) }
```

`hl.New` returns a `*Client` with three handles:

- `c.Info` — read-only queries.
- `c.Trade` — signed actions; requires `WithPrivateKey`.
- `c.Stream` — WebSocket subscriptions. Pass `WithSkipStream(true)` if you only need REST.

## 4. Read a price

```go
mid, err := c.Info.Mid("ETH")
if err != nil { log.Fatal(err) }
fmt.Printf("ETH mid: %.2f\n", mid)
```

`Info.Mid` returns a `float64` parsed from the wire's string-encoded price. For the full L2 snapshot use `c.Info.Book("ETH")`.

## 5. Place a resting limit order (ALO)

Place an Add-Liquidity-Only limit buy 1% below mid so it rests on the book:

```go
px := math.Round(mid*0.99*100) / 100 // round to tick
res, err := c.Trade.PlaceALO("ETH", hl.Buy, 0.01, px)
if err != nil { log.Fatal(err) }
fmt.Printf("placed oid=%d status=%s\n", res.OID, res.Status)
```

`PlaceALO` returns a `Result` with the resting order id, a stable status string, and an `Error` field populated when the server rejected the leg. Before signing, the call runs `validate()` against the cached `UserState` and the asset metadata. Failures surface as `*ValidationError` (see [errors.md](./errors.md)).

## 6. List open orders

```go
orders, err := c.Info.OpenOrders(os.Getenv("HL_ACCOUNT_ADDRESS"))
if err != nil { log.Fatal(err) }
for _, o := range orders {
    fmt.Printf("oid=%d %s %s @ %s sz=%s\n", o.Oid, o.Side, o.Coin, o.LimitPx, o.Sz)
}
```

For UI-shape responses including TP/SL leg metadata, use `c.Info.FrontendOpenOrders`.

## 7. Cancel the order

```go
cancelRes, err := c.Trade.Cancel("ETH", res.OID)
if err != nil { log.Fatal(err) }
fmt.Println("cancel status:", cancelRes.Status)
```

Or cancel everything across a coin (or every coin if called with no arguments):

```go
batch, err := c.Trade.CancelAll("ETH")
if err != nil { log.Fatal(err) }
fmt.Printf("cancelled %d orders\n", len(batch.Results))
```

## 8. Place a GTC with a bracket

`WithBracket(tp, sl)` attaches reduce-only trigger legs that fire when the parent fills. They are submitted as one signed action with `grouping = "normalTpsl"`:

```go
entry := math.Round(mid*0.995*100) / 100
tp    := math.Round(mid*1.02*100)  / 100
sl    := math.Round(mid*0.98*100)  / 100

res, err := c.Trade.PlaceGTC(
    "ETH", hl.Buy, 0.01, entry,
    hl.WithBracket(tp, sl),
)
if err != nil { log.Fatal(err) }
```

Cancelling the parent cancels the TP/SL legs as well.

## 9. Close a position

`ClosePosition` reads the cached `UserState`, infers direction (long → sell, short → buy), and submits a reduce-only IOC. Pass `WithLimit(px)` to close at a specific price; pass `WithSize(x)` for a partial close.

```go
res, err := c.Trade.ClosePosition("ETH")
if err != nil { log.Fatal(err) }
fmt.Println("close avg px:", res.AvgPx)
```

If the cached state shows no position in `coin`, the call returns a `*ValidationError` with `Code == "no_position"`.

## 10. Subscribe to live trades

`c.Stream` is not connected by default. Call `Connect` once, then `Subscribe`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := c.Stream.Connect(ctx); err != nil { log.Fatal(err) }
defer c.Stream.Close()

sub := hl.Trades("ETH")
id, err := c.Stream.Subscribe(sub, func(m hl.WSMessage) {
    fmt.Printf("trade: %s\n", string(m.Data))
})
if err != nil { log.Fatal(err) }

time.Sleep(5 * time.Second)
_ = c.Stream.Unsubscribe(sub, id)
```

`Subscribe` returns an integer id paired to the subscription; pass both to `Unsubscribe`. The Stream maintains its own reconnect loop; on disconnect it resubscribes everything you had registered.

## 11. Multi-leg batch (one signature)

```go
res, err := c.Trade.PlaceMany(
    hl.GTC("ETH", hl.Buy,  0.01, entry),
    hl.GTC("BTC", hl.Sell, 0.0005, 70_000),
)
if err != nil { log.Fatal(err) }
for i, r := range res.Results {
    fmt.Printf("leg %d oid=%d status=%s\n", i, r.OID, r.Status)
}
```

The constructors (`hl.ALO`, `hl.IOC`, `hl.GTC`, `hl.Market`, `hl.Trigger`) accept exactly the same `PlaceOpt` set as the corresponding `Trader.Place*` methods. `PlaceMany` validates each spec individually before sending a single batched action.

## Next steps

- Reference all trader operations: [trading.md](./trading.md).
- Reference all info queries: [info.md](./info.md).
- Stream subscription constructors and POST-over-WS: [stream.md](./stream.md).
- Validation codes and `errors.As` patterns: [errors.md](./errors.md).
- Run the network test suite: [integration-testing.md](./integration-testing.md).
