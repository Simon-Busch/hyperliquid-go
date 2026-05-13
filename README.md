# go-hyperliquid-0xsi

Idiomatic Go SDK for the [Hyperliquid](https://hyperliquid.xyz) exchange.

> **Status.** The public API documented here landed on `refactor/ux-api` and will graduate to `main` once the network integration suite is green. The interface is intentionally narrow: one constructor, three handles (`Info`, `Trade`, `Stream`), and a single shared validation pipeline behind every signed action.
>
> Forked from [`sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid). The fork rewrites the public surface, isolates the EIP-712 signing and msgpack/wire code under `internal/`, and removes the residual Python-bridge shape.

## What's covered

- Perps trading (placement, modify, cancel, brackets, triggers, close).
- Spot trading and balances.
- HIP-3 builder-deployed perp dexes (via `WithBuilderDex`).
- HIP-4 outcome (binary prediction) markets.
- WebSocket subscriptions (market and per-user feeds) with automatic resubscription on reconnect.
- POST-over-WebSocket for low-latency reads and signed actions.
- Native Go EIP-712 signing — no Python bridge.
- Pre-flight validation against the cached `UserState` and per-asset metadata.

## Install

```bash
go get github.com/Simon-Busch/hyperliquid-go@latest
```

Conventionally aliased to `hl`:

```go
import hl "github.com/Simon-Busch/hyperliquid-go"
```

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ethereum/go-ethereum/crypto"
    hl "github.com/Simon-Busch/hyperliquid-go"
)

func main() {
    pk, err := crypto.HexToECDSA("<hex private key>")
    if err != nil { log.Fatal(err) }

    c, err := hl.New(hl.WithTestnet(), hl.WithPrivateKey(pk))
    if err != nil { log.Fatal(err) }

    mid, err := c.Info.Mid("ETH")
    if err != nil { log.Fatal(err) }
    fmt.Printf("ETH mid: %.2f\n", mid)

    res, err := c.Trade.PlaceALO("ETH", hl.Buy, 0.01, mid*0.99)
    if err != nil { log.Fatal(err) }
    fmt.Printf("placed oid=%d\n", res.OID)

    _, _ = c.Trade.CancelAll("ETH")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := c.Stream.Connect(ctx); err != nil { log.Fatal(err) }
    defer c.Stream.Close()

    sub, err := c.Stream.Subscribe(hl.Trades("ETH"), func(m hl.WSMessage) {
        fmt.Println("trade:", string(m.Data))
    })
    if err != nil { log.Fatal(err) }
    defer sub.Close()
    time.Sleep(5 * time.Second)
}
```

A full walkthrough — `.env` setup, bracketed entries, `ClosePosition`, multi-leg batches — lives in [docs/quickstart.md](./docs/quickstart.md).

## The three services

| Handle     | Responsibility                                                  | Reference                            |
|------------|-----------------------------------------------------------------|--------------------------------------|
| `c.Info`   | Read-only queries — mids, books, orders, fills, metadata.       | [docs/info.md](./docs/info.md)       |
| `c.Trade`  | Signed actions — placement, transfers, leverage, governance.    | [docs/trading.md](./docs/trading.md) |
| `c.Stream` | WebSocket streaming + POST-over-WS.                             | [docs/stream.md](./docs/stream.md)   |

## Common patterns

**Bracketed entry.** `WithBracket(tp, sl)` attaches reduce-only trigger legs that fire when the parent fills; cancelling the parent cancels the children.

```go
res, err := c.Trade.PlaceGTC(
    "ETH", hl.Buy, 0.01, 1500,
    hl.WithBracket(1600, 1450),
)
```

**Close a position with auto-direction.** `ClosePosition` reads the cached state, infers direction, and submits a reduce-only IOC. `WithLimit(px)` switches to a limit close; `WithSize(x)` makes it partial.

```go
res, err := c.Trade.ClosePosition("ETH")
```

**Multi-leg single-signature batch.** Build specs with the top-level constructors and submit them all under one signed action.

```go
res, err := c.Trade.PlaceMany(
    hl.GTC("ETH", hl.Buy,  0.01, 1500),
    hl.IOC("BTC", hl.Sell, 0.001, 70_000),
)
```

## Validation

Every placement, modify, cancel, and close call runs `validate()` before signing. The validator refreshes the cached `UserState` and checks the spec against the asset metadata: size step, tick alignment, significant figures, reduce-only direction, bracket TP/SL placement, close direction, option/method compatibility. Failures surface as `*hyperliquid.ValidationError` with a stable machine-readable `Code` — see [docs/errors.md](./docs/errors.md) for the full table.

Opt out per call with `hl.SkipValidation()` if you have your own checks or you're trading from a state the SDK cannot see (e.g. an unprovisioned agent address).

## Configuration

`hl.New` accepts the following functional options:

| Option                            | Effect                                                                 |
|-----------------------------------|------------------------------------------------------------------------|
| `WithMainnet()`                   | Pin the client to `https://api.hyperliquid.xyz` (default).             |
| `WithTestnet()`                   | Pin the client to `https://api.hyperliquid-testnet.xyz`.               |
| `WithBaseURL(url)`                | Override the base URL.                                                 |
| `WithPrivateKey(pk)`              | Required for `c.Trade`. Accepts `*ecdsa.PrivateKey`.                   |
| `WithAccount(addr)`               | Owner-address override for agent flows.                                |
| `WithVault(addr)`                 | Sign actions on behalf of a vault address.                             |
| `WithBuilderDex(dex)`             | Pin the client to a HIP-3 builder-deployed dex.                        |
| `WithHTTPClient(c)`               | Inject a custom `*http.Client`.                                        |
| `WithMeta(m, sm, pd)`             | Use pre-fetched metadata; skips the warm-up calls.                     |
| `WithSkipStream(true)`            | Do not construct `c.Stream`.                                           |
| `WithLogger(l)`                   | Plug a `Logger` into `c.Stream` for reconnect diagnostics.             |

## Running the integration suite

The default `go test ./...` is unit-only — no network. To run the live suite:

```bash
go test -tags=integration ./tests/integration/...
```

Required env: `HL_PRIVATE_KEY`. Optional: `HL_ACCOUNT_ADDRESS`, `HL_BASE_URL` (defaults to testnet), `HL_TEST_COIN`, `HL_TEST_SIZE`. Full env table, scenario list, and troubleshooting tips: [docs/integration-testing.md](./docs/integration-testing.md).

## Documentation

- [`docs/quickstart.md`](./docs/quickstart.md) — end-to-end first-trade walkthrough.
- [`docs/README.md`](./docs/README.md) — reference TOC with anchors to every exported function.
- [`docs/trading.md`](./docs/trading.md) — every method on `c.Trade`.
- [`docs/info.md`](./docs/info.md) — every method on `c.Info`.
- [`docs/stream.md`](./docs/stream.md) — `c.Stream` and the subscription constructors.
- [`docs/signing.md`](./docs/signing.md) — `SignL1Action`, `SignUserSignedAction`, and friends.
- [`docs/errors.md`](./docs/errors.md) — `ValidationError` codes, `APIError`, sentinel errors.
- [`docs/integration-testing.md`](./docs/integration-testing.md) — env, scenarios, troubleshooting.
- godoc on [pkg.go.dev](https://pkg.go.dev/github.com/Simon-Busch/hyperliquid-go).

## Project layout

```
hyperliquid/                # public package
├── client.go               # New(), Client, top-level fields
├── options.go              # WithMainnet/Testnet/PrivateKey/...
├── trader*.go              # Trader, placement, modify/cancel, transfers, subgroups, deploy
├── info*.go                # Info, market/account/orders/funding/staking/meta queries
├── stream*.go              # Stream, subscription constructors, POST over WS, reconnect
├── opts.go                 # PlaceOpt + WithTakeProfit/StopLoss/.../SkipValidation
├── orderspec.go            # OrderSpec value type + hl.ALO/IOC/GTC/Market/Trigger
├── validate.go             # single validate() pipeline
├── bracket.go              # bracket-leg builder
├── result.go               # Result, BatchResult, CancelResult, BatchCancelResult
├── errors.go               # APIError, ValidationError, ErrMissingPrivateKey
├── side.go                 # Side, TIF, MarginMode enums
├── logger.go               # Logger interface + nopLogger
├── signing.go              # SignL1Action, SignUserSignedAction, FloatToUsdInt, GetTimestampMs
├── types.go                # domain types: Order, Position, Fill, Candle, Meta, ...
├── actions.go              # wire action structs
└── internal/
    ├── eip712/             # EIP-712 internals (hash, sign, phantom agent)
    ├── wire/               # msgpack + price/size rounding
    └── transport/          # http warm-up
```

## License

This repository does not currently ship a `LICENSE` file. Upstream [`sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid) is the reference for licensing terms while one is added.
