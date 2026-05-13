# Info reference (`c.Info`)

`c.Info` is the read-only surface. Every method here issues a single POST to `/info`, parses the response, and returns a typed value. No signing, no caching beyond what the HTTP client does, no rate-limit handling — wrap calls in your own backoff if you hammer the endpoint.

`c.Info` is always non-`nil` on a `Client` built by `hyperliquid.New`.

## Contents

- [Market data](#market-data)
- [Account state](#account-state)
- [Orders and fills](#orders-and-fills)
- [Funding](#funding)
- [Staking](#staking)
- [Metadata](#metadata)
- [Account directory](#account-directory)

---

## Market data

### `Mid` {#mid}

Return the current mid price for `coin`, parsed from the wire's string-encoded format.

```go
func (i *Info) Mid(coin string) (float64, error)
```

**Example**

```go
mid, err := c.Info.Mid("ETH")
```

### `AllMids`, `AllMidsOn` {#allmids}

Return mids for every coin in a single snapshot. `AllMids` accepts an optional dex argument; `AllMidsOn` is the explicit single-dex variant.

```go
func (i *Info) AllMids(dex ...string) (map[string]string, error)
func (i *Info) AllMidsOn(dex string) (map[string]string, error)
```

**Example**

```go
all, err   := c.Info.AllMids()         // default dex
flx, err   := c.Info.AllMidsOn("flx")  // HIP-3 dex
```

### `Book` {#book}

Return the current L2 order book for `coin`.

```go
func (i *Info) Book(coin string) (*L2Book, error)
```

`L2Book.Levels` is a `[2][]Level` — index 0 = bids, index 1 = asks.

### `Candles` {#candles}

Return historical candles for `coin` at `interval` between `start` and `end` (Unix millis).

```go
func (i *Info) Candles(coin, interval string, start, end int64) ([]Candle, error)
```

**Example**

```go
now  := time.Now().UnixMilli()
last := now - 24*3600*1000
candles, err := c.Info.Candles("ETH", "1h", last, now)
```

### `MetaAndAssetCtxs`, `SpotMetaAndAssetCtxs` {#metaandassetctxs}

Fetch perp (or spot) metadata together with the per-asset context snapshot in one call.

```go
func (i *Info) MetaAndAssetCtxs() (*MetaAndAssetCtxsResponse, error)
func (i *Info) SpotMetaAndAssetCtxs() (map[string]any, error)
```

`SpotMetaAndAssetCtxs` returns a raw map for now — the typed envelope is a follow-up.

---

## Account state

### `UserState` {#userstate}

Return the caller's perpetuals account summary (positions, margin, withdrawable).

```go
func (i *Info) UserState(address string, dex ...string) (*UserState, error)
```

### `SpotBalances` {#spotbalances}

Return the caller's spot clearinghouse state.

```go
func (i *Info) SpotBalances(addr string) (*SpotClearinghouseState, error)
```

### `Positions`, `Position` {#positions}

`Positions` returns every open position for `addr`. `Position` returns the single position on `coin`, or `(nil, nil)` if none.

```go
func (i *Info) Positions(addr string, dex ...string) ([]Position, error)
func (i *Info) Position(addr, coin string) (*Position, error)
```

**Example**

```go
pos, err := c.Info.Position("0xabc...", "ETH")
if err != nil  { log.Fatal(err) }
if pos == nil  { fmt.Println("no ETH position") }
```

### `Fees` {#fees}

Return the fee snapshot for `addr`.

```go
func (i *Info) Fees(addr string) (*UserFees, error)
```

### `Asset`, `AssetID` {#asset}

`Asset` returns the per-coin metadata snapshot used by `validate()`: asset id, size decimals, tick size, minimum size, and the asset class (perp / spot / HIP-3 / outcome). `AssetID` returns just the numeric id.

```go
func (i *Info) Asset(coin string) (AssetMeta, error)
func (i *Info) AssetID(coin string) int
```

**Example**

```go
meta, _ := c.Info.Asset("ETH")
fmt.Printf("tick=%v min=%v dec=%d\n", meta.TickSize, meta.MinSize, meta.SzDecimals)
```

---

## Orders and fills

### `OpenOrders`, `FrontendOpenOrders` {#openorders}

`OpenOrders` returns the wire-shape open orders. `FrontendOpenOrders` returns the richer UI-shape variant which includes TP/SL leg metadata.

```go
func (i *Info) OpenOrders(address string, dex ...string) ([]OpenOrder, error)
func (i *Info) FrontendOpenOrders(address string, dex ...string) ([]FrontendOpenOrder, error)
```

### `Fills`, `FillsBetween` {#fills}

`Fills` returns all fills for `addr`. `FillsBetween` filters by time range (Unix millis); `end == nil` means "until now".

```go
func (i *Info) Fills(addr string) ([]Fill, error)
func (i *Info) FillsBetween(addr string, start int64, end *int64) ([]Fill, error)
```

### `Order`, `OrderByCloid`, `Fill` {#order}

Look up a specific order or fill by id.

```go
func (i *Info) Order(addr string, oid int64) (*OrderStatusResponse, error)
func (i *Info) OrderByCloid(addr, cloid string) (*OpenOrder, error)
func (i *Info) Fill(addr string, oid int64) (*Fill, error)
```

---

## Funding

### `Funding`, `UserFunding` {#funding}

`Funding` returns the historical funding rates for `coin`. `UserFunding` returns the per-user funding payments for `addr`. Both accept a `*int64` end timestamp (`nil` = now).

```go
func (i *Info) Funding(coin string, start int64, end *int64) ([]FundingHistory, error)
func (i *Info) UserFunding(addr string, start int64, end *int64) ([]UserFundingHistory, error)
```

---

## Staking

Accessible via `c.Info.Stake`.

### `Stake.Summary`, `Stake.Delegations`, `Stake.Rewards` {#staking}

```go
func (g *InfoStakeGroup) Summary(addr string) (*StakingSummary, error)
func (g *InfoStakeGroup) Delegations(addr string) ([]StakingDelegation, error)
func (g *InfoStakeGroup) Rewards(addr string) ([]StakingReward, error)
```

---

## Metadata

### `Meta`, `SpotMeta`, `OutcomeMeta`, `PerpDexs` {#metadata}

Raw metadata endpoints. `OutcomeMeta` covers HIP-4 binary-prediction markets.

```go
func (i *Info) Meta(dex ...string) (*Meta, error)
func (i *Info) SpotMeta() (*SpotMeta, error)
func (i *Info) OutcomeMeta() (*OutcomeMeta, error)
func (i *Info) PerpDexs() (MixedArray, error)
```

`Info.Meta` accepts an optional dex name; pass it to fetch a HIP-3 dex's universe.

---

## Account directory

### `SubAccounts`, `Referral`, `MultiSigSigners` {#account-directory}

```go
func (i *Info) SubAccounts(addr string) ([]SubAccount, error)
func (i *Info) Referral(addr string) (*ReferralState, error)
func (i *Info) MultiSigSigners(multiSigAddr string) ([]MultiSigSigner, error)
```
