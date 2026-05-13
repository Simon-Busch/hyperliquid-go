# Integration testing

The default `go test ./...` runs unit tests only — no network. The integration suite lives under `tests/integration/` and is gated behind a build tag so accidental `go test` runs never hit the exchange.

## Running the suite

```bash
go test -tags=integration ./tests/integration/...
```

To run a single scenario:

```bash
go test -tags=integration -run TestPlaceALORoundTrip ./tests/integration/...
```

Add `-v` for verbose output and `-count=1` to disable the test cache.

## Environment variables

The suite loads a `.env` from the project root via [godotenv](https://github.com/joho/godotenv) before each test.

| Variable              | Required | Default                                           | Purpose                                              |
|-----------------------|----------|---------------------------------------------------|------------------------------------------------------|
| `HL_PRIVATE_KEY`      | yes      | —                                                 | Hex-encoded private key for the test wallet.         |
| `HL_ACCOUNT_ADDRESS`  | no       | derived from `HL_PRIVATE_KEY`                     | Owner address (set when using an agent flow).        |
| `HL_BASE_URL`         | no       | `https://api.hyperliquid-testnet.xyz`             | Endpoint to target. **Always default to testnet.**   |
| `HL_TEST_COIN`        | no       | `ETH`                                             | Coin used by placement scenarios.                    |
| `HL_TEST_SIZE`        | no       | `0.01`                                            | Size used by placement scenarios.                    |
| `HL_BUILDER_ADDR`     | no       | —                                                 | Builder address for `WithBuilder` scenarios.         |
| `HL_BUILDER_FEE_BPS`  | no       | —                                                 | Fee bps for `WithBuilder` scenarios.                 |

Recommended `.env`:

```bash
HL_BASE_URL=https://api.hyperliquid-testnet.xyz
HL_PRIVATE_KEY=0x<your-test-wallet-key>
HL_ACCOUNT_ADDRESS=0x<your-owner-address>
HL_TEST_COIN=ETH
HL_TEST_SIZE=0.01
```

Never commit `.env`. The repo `.gitignore` already excludes it.

## Scenarios

The integration files in `tests/integration/` cover, at minimum, these scenarios. Each is one test function:

1. **Place ALO round-trip** — place an ALO well off the book, list open orders, cancel.
2. **Place IOC market fill** — place an IOC that fills, query fills.
3. **Bracketed GTC** — place a GTC with `WithBracket(tp, sl)`, cancel the parent, verify TP/SL are also cancelled.
4. **Trigger stop-market** — place a stop-market via `PlaceTrigger`, cancel.
5. **SetLeverage round-trip** — switch leverage Cross ↔ Isolated, verify the new state.
6. **USD transfer to self** — call `Trade.Transfer.SendUSD` to the same wallet, verify the ledger entry.
7. **Approve agent + place from agent** — call `ApproveAgent`, build a new client with the agent key + `WithAccount(owner)`, place an order, cancel it.
8. **Stream trades, 5 s** — `Subscribe(hl.Trades(coin), ...)`, count messages over 5 s, assert > 0.
9. **Stream PostInfoRequest** — call `PostInfoRequest`, compare the payload with the REST `Info.Book` snapshot.
10. **Stream PostActionRequest** — place an order and cancel it entirely over the WS.
11. **ClosePosition end-to-end** — open a tiny position, call `ClosePosition`, verify no position remains.

## Troubleshooting

- **`429 Too Many Requests`** — the test suite does not throttle. Re-run with `-parallel 1` or insert your own delays in custom scenarios.
- **`insufficient balance`** — top the test wallet up on testnet via the [Hyperliquid testnet faucet](https://app.hyperliquid-testnet.xyz/).
- **`validation: refresh user state: ...`** — the SDK could not fetch `UserState` for your address. Confirm `HL_ACCOUNT_ADDRESS` matches the wallet that traded last, or pass `hl.SkipValidation()` if you're intentionally trading from a fresh address.
- **`signature mismatch`** — the chain you're targeting (`HL_BASE_URL`) does not match what `WithMainnet()` / `WithTestnet()` selected. If you set `WithBaseURL` directly, `isMainnet` is inferred from the URL — keep it consistent.
- **Test occasionally cancels itself with "context deadline exceeded"** — the Stream's initial dial uses the test's `context.Context`. Widen the timeout or split the test into smaller per-call contexts.

## Writing new scenarios

The shared helper `loadIntegrationEnv(t)` returns a typed config struct (`coin`, `size`, `base URL`, derived addresses). Build scenarios as ordinary `*testing.T` functions tagged with `//go:build integration`. Avoid hardcoding mainnet URLs.

A minimal scenario template:

```go
//go:build integration

package integration

import (
    "context"
    "testing"
    "time"

    hl "github.com/Simon-Busch/hyperliquid-go"
)

func TestMyScenario(t *testing.T) {
    env := loadIntegrationEnv(t)

    c, err := hl.New(
        hl.WithBaseURL(env.BaseURL),
        hl.WithPrivateKey(env.PrivateKey),
        hl.WithAccount(env.Account),
    )
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := c.Stream.Connect(ctx); err != nil { t.Fatal(err) }
    defer c.Stream.Close()

    // ... scenario body ...
}
```
