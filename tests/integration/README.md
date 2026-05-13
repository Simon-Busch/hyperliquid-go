# Integration tests

Network-dependent scenarios for the Hyperliquid Go SDK. Every test in
this directory is gated behind the `integration` build tag so the
default `go test ./...` run never reaches the network.

## Running

```sh
go test -tags=integration -count=1 ./tests/integration/...
```

To list scenarios without running them:

```sh
go test -tags=integration -list "Test.*" ./tests/integration/...
```

The same suite runs against testnet (default) and mainnet — point
`HL_BASE_URL` at the environment you want:

```sh
# testnet (default)
HL_BASE_URL=https://api.hyperliquid-testnet.xyz

# mainnet
HL_BASE_URL=https://api.hyperliquid.xyz
```

Scenarios use the small order size from `HL_TEST_SIZE` (default `0.01`)
so mainnet runs stay cheap. Read-only scenarios cost nothing.

## Configuration

Configuration is read from a `.env` file. The loader resolves `.env`,
then `../.env`, then `../../.env` (so the file can live next to the
test directory, the repo root, or one level above).

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `HL_PRIVATE_KEY` | yes | — | 32-byte hex (with or without `0x`); signs every action. |
| `HL_ACCOUNT_ADDRESS` | no | derived from key | Account address Trader acts on behalf of (agent flow). |
| `HL_BASE_URL` | no | `https://api.hyperliquid-testnet.xyz` | REST endpoint. |
| `HL_TEST_COIN` | no | `ETH` | Coin used for placement scenarios. |
| `HL_TEST_NOTIONAL` | no | `10` | Target USD notional per test order. Size is derived from the current mid and snapped down to the coin's size step. Set to `0` to fall back to `HL_TEST_SIZE`. |
| `HL_TEST_SIZE` | no | `0.01` | Fixed coin-unit fallback. Used only when `HL_TEST_NOTIONAL=0`. |
| `HL_BUILDER_ADDR` | no | — | Builder fee referral target. |
| `HL_BUILDER_FEE_BPS` | no | `1` | Builder fee in basis points. |
| `HL_SKIP_TRANSFER` | no | — | Set `true` to skip transfer scenarios. |
| `HL_SKIP_WS` | no | — | Set `true` to skip websocket scenarios. |

Tests skip when their preconditions are unmet (account empty, coin
missing from metadata, WS disabled, etc.) — they do not fail on
configuration gaps.

## Scenarios

| Scenario | What it does |
|---|---|
| `TestPlaceALO_QueryAndCancel` | Place a resting ALO, find it in OpenOrders, cancel. |
| `TestCancelAll` | Place two resting ALOs, call CancelAll, assert empty. |
| `TestPlaceIOC_Market` | IOC aggressively above mid, query the fill. |
| `TestPlaceMarket` | Market buy, check Position, then close. |
| `TestPlaceGTC_WithBracket` | GTC entry plus TP/SL bracket, cancel parent. |
| `TestPlaceTrigger_Cancel` | Far-away stop-market, cancel by oid. |
| `TestModify_PriceAndSize` | Resting ALO → Modify price + size, verify open order. |
| `TestClosePosition_AutoDirection` | Open long via market, close via auto-direction. |
| `TestSetLeverage` | Cross/5x then Isolated/3x. |
| `TestTransfer_USDToSelf` | Send 1.0 USDC to self, balance unchanged. |
| `TestApproveAgent_AndPlace` | Approve a fresh agent, place + cancel through it. |
| `TestStream_TradesReceived` | Subscribe to trades feed, expect ≥1 message. |
| `TestStream_PostInfo_MatchesREST` | PostInfo Meta over WS, compare to REST. |
| `TestStream_PostAction` | Place an ALO via Stream.PostAction, REST cancel. |
| `TestValidation_LongShortHardErrors` | Open a long, expect Buy reduce-only to fail. |
