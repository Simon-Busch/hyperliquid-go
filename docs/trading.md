# Trading reference (`c.Trade`)

`c.Trade` is the signed-action surface. Every method on `*Trader` (and on its sub-groups `Transfer`, `SubAccount`, `Stake`, `MultiSig`) builds an action map, EIP-712-signs it with the private key supplied to `hyperliquid.New`, and POSTs it to `/exchange`.

Placement methods run a single shared `validate()` pipeline before signing. Failures surface as `*ValidationError`; server-side rejects surface as `APIError` wrapped in the returned `error`. See [errors.md](./errors.md).

`c.Trade` is `nil` if the `Client` was built without `WithPrivateKey`. The sentinel [`ErrMissingPrivateKey`](./errors.md#errmissingprivatekey) describes that case.

## Contents

- [Construction](#new) and [client options](#client-options)
- [Placement](#placement) and [OrderSpec constructors](#orderspec-constructors)
- [Placement options](#placement-options)
- [Modify and cancel](#modify-and-cancel)
- [Position management](#position-management)
- [Transfers](#transfers) and [`Withdraw`](#withdraw)
- [Sub-accounts](#sub-accounts), [Staking](#staking), [Multi-sig](#multi-sig)
- [Account control](#account-control)
- [HIP-2 / HIP-3 deploy](#deploy)
- [Validator operations](#validators)

---

## New

### `hyperliquid.New` {#new}

Builds a `*Client` with configured `Info`, `Trade`, and `Stream` handles.

```go
func New(opts ...Option) (*Client, error)
```

**Example**

```go
c, err := hl.New(
    hl.WithTestnet(),
    hl.WithPrivateKey(pk),
    hl.WithAccount("0xabc..."),
)
```

If no `WithMainnet` / `WithTestnet` / `WithBaseURL` is supplied, mainnet is used. `c.Trade` is populated only if `WithPrivateKey` was set; `c.Stream` is populated unless `WithSkipStream(true)` was supplied.

## Client options {#client-options}

All options have the type `hyperliquid.Option = func(*clientConfig)`.

| Option                      | Effect                                                                 |
|-----------------------------|------------------------------------------------------------------------|
| `WithMainnet()`             | Pin the client to `https://api.hyperliquid.xyz` (default).             |
| `WithTestnet()`             | Pin the client to `https://api.hyperliquid-testnet.xyz`.               |
| `WithBaseURL(url)`          | Override the base URL (for local proxies or alternate endpoints).      |
| `WithPrivateKey(pk)`        | Required for `c.Trade`. Accepts an `*ecdsa.PrivateKey`.                |
| `WithAccount(addr)`         | Owner-address override for agent flows. Defaults to the pk's address.  |
| `WithVault(addr)`           | Sign actions on behalf of a vault address.                             |
| `WithBuilderDex(dex)`       | Pin the client to a HIP-3 builder-deployed dex (e.g. `"flx"`).         |
| `WithHTTPClient(c)`         | Inject a custom `*http.Client` (timeouts, proxies, trace transport).   |
| `WithMeta(meta, spotMeta, perpDexs)` | Use pre-fetched metadata; skips the warm-up calls.            |
| `WithSkipStream(true)`      | Do not construct `c.Stream`. Use for REST-only programs.               |
| `WithLogger(l)`             | Plug a `Logger` into `c.Stream` for reconnect diagnostics.             |

---

## Placement

Required parameters are positional; optional parameters flow through the `PlaceOpt` variadic. All placement methods share a single validate-then-sign pipeline and return a flat `Result`.

### `PlaceALO` {#placealo}

Place an Add-Liquidity-Only limit order. ALO orders that would cross are rejected by the exchange.

```go
func (t *Trader) PlaceALO(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error)
```

**Validation**: `coin_required`, `unknown_coin`, `size_below_min`, `size_step_violation`, `price_non_positive`, `significant_figures`, `wrong_side_for_reduce`, plus bracket rules when `WithBracket`/`WithTakeProfit`/`WithStopLoss` are used.

**Example**

```go
res, err := c.Trade.PlaceALO("ETH", hl.Buy, 0.01, 1500, hl.WithCloid("0x..."))
```

**Related**: [`hl.ALO`](#orderspec-constructors), [`PlaceGTC`](#placegtc), [placement options](#placement-options).

### `PlaceIOC` {#placeioc}

Place an Immediate-Or-Cancel limit order. Anything that does not fill at submission is cancelled.

```go
func (t *Trader) PlaceIOC(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error)
```

**Validation**: same as `PlaceALO`.

**Example**

```go
res, err := c.Trade.PlaceIOC("BTC", hl.Sell, 0.001, 70000)
```

### `PlaceGTC` {#placegtc}

Place a Good-Til-Cancelled limit order.

```go
func (t *Trader) PlaceGTC(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error)
```

**Validation**: same as `PlaceALO`. Additional bracket rules when `WithBracket`/`WithTakeProfit`/`WithStopLoss` are used: `tp_wrong_side_buy`, `tp_wrong_side_sell`, `sl_wrong_side_buy`, `sl_wrong_side_sell`, `bracket_size_exceeds_entry`.

**Example — bracketed entry**

```go
res, err := c.Trade.PlaceGTC(
    "ETH", hl.Buy, 0.01, 1500,
    hl.WithBracket(1600, 1450),
)
```

### `PlaceMarket` {#placemarket}

Submit a market-style order. Internally an IOC at the current mid plus or minus `slippage` (default 5%).

```go
func (t *Trader) PlaceMarket(coin string, side Side, size float64, opts ...PlaceOpt) (Result, error)
```

**Validation**: `coin_required`, `unknown_coin`, `size_below_min`, `size_step_violation`, plus `unsupported_option` if `WithSlippage` is passed to any other method.

**Example**

```go
res, err := c.Trade.PlaceMarket("ETH", hl.Buy, 0.01, hl.WithSlippage(0.02))
```

### `PlaceTrigger` {#placetrigger}

Place a stop-market (default) or stop-limit trigger order. Use [`AsMarket`](#placement-options) / [`AsLimit`](#placement-options) to switch.

```go
func (t *Trader) PlaceTrigger(coin string, side Side, size, triggerPx float64, opts ...PlaceOpt) (Result, error)
```

The TP/SL discriminator is inferred from `side`: `Buy → "sl"`, `Sell → "tp"`. The trigger price is stored both as `TriggerPx` and as the parent's `Price` so the wire serialization stays consistent.

**Example — stop-loss at 1450**

```go
res, err := c.Trade.PlaceTrigger("ETH", hl.Sell, 0.01, 1450, hl.WithReduceOnly())
```

**Example — stop-limit (rest at limit after trigger)**

```go
res, err := c.Trade.PlaceTrigger("ETH", hl.Sell, 0.01, 1450, hl.AsLimit(1448), hl.WithReduceOnly())
```

### `PlaceMany` {#placemany}

Place multiple legs with one signature.

```go
func (t *Trader) PlaceMany(orders ...OrderSpec) (BatchResult, error)
```

Each spec is validated individually before any signing happens. The result contains one `Result` per leg, in the same order as the inputs.

**Example**

```go
res, err := c.Trade.PlaceMany(
    hl.GTC("ETH", hl.Buy,  0.01, 1500),
    hl.IOC("BTC", hl.Sell, 0.001, 70_000),
)
```

## OrderSpec constructors {#orderspec-constructors}

Top-level (package-level) helpers that return an `OrderSpec` for `PlaceMany`. Same option set as the corresponding `Trader.Place*` methods.

```go
func ALO(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec
func IOC(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec
func GTC(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec
func Market(coin string, side Side, size float64, opts ...PlaceOpt) OrderSpec
func Trigger(coin string, side Side, size, triggerPx float64, opts ...PlaceOpt) OrderSpec
```

---

## Placement options {#placement-options}

All options have the type `PlaceOpt = func(*OrderSpec)`. They never report errors directly; misuse surfaces as a `*ValidationError` with `Code == "unsupported_option"` at `place()` time.

| Option                | Effect                                                      | Valid on                        |
|-----------------------|-------------------------------------------------------------|---------------------------------|
| `WithTakeProfit(px)`  | Attach reduce-only TP trigger leg.                          | `PlaceALO/IOC/GTC/Market`, `PlaceMany`. |
| `WithStopLoss(px)`    | Attach reduce-only SL trigger leg.                          | same as above.                  |
| `WithBracket(tp, sl)` | Shortcut for `WithTakeProfit + WithStopLoss`.               | same as above.                  |
| `WithTPSize(sz)`      | Partial TP leg size.                                        | same as above.                  |
| `WithSLSize(sz)`      | Partial SL leg size.                                        | same as above.                  |
| `WithTPCloid(s)`      | Pin a cloid on the TP leg.                                  | same as above.                  |
| `WithSLCloid(s)`      | Pin a cloid on the SL leg.                                  | same as above.                  |
| `WithReduceOnly()`    | Mark the order reduce-only.                                 | all placement verbs.            |
| `WithCloid(s)`        | Pin a client-supplied 32-byte hex order id.                 | all placement verbs.            |
| `WithBuilder(addr, bps)` | Attach a HIP-1 builder-fee referral.                     | all placement verbs.            |
| `WithSlippage(frac)`  | Worst-case fill fraction off mid for market orders.         | `PlaceMarket`, `ClosePosition`. |
| `WithSize(sz)`        | Override the order size (partial-close, modify resize).     | `ClosePosition`, `Modify`.      |
| `WithLimit(px)`       | Limit-style close or new modify price.                      | `ClosePosition`, `Modify`.      |
| `AsMarket()`          | Force trigger to fill as a market.                          | `PlaceTrigger`.                 |
| `AsLimit(px)`         | Force trigger to rest as a limit at px.                     | `PlaceTrigger`.                 |
| `SkipValidation()`    | Bypass the pre-flight `validate()`.                         | any placement verb.             |

`SkipValidation()` is an escape hatch — use only when calling against a network where the SDK cannot fetch metadata, or where the caller has its own validation.

---

## Modify and cancel

### `Modify` {#modify}

Change the price (or size, or both) of a resting order identified by `oid`.

```go
func (t *Trader) Modify(oid int64, opts ...PlaceOpt) (Result, error)
```

Either `WithLimit(newPx)` or `WithSize(newSz)` (or both) must be supplied. Otherwise the validator returns `Code == "modify_no_change"`.

**Validation**: `modify_target_required` (if neither oid nor cloid resolves), `modify_no_change`.

**Example**

```go
res, err := c.Trade.Modify(oid, hl.WithLimit(1502.5))
```

### `ModifyByCloid` {#modifybycloid}

Identical to `Modify` but addresses the order by its `Cloid`.

```go
func (t *Trader) ModifyByCloid(cloid string, opts ...PlaceOpt) (Result, error)
```

### `Cancel` {#cancel}

Cancel a single open order by `oid`.

```go
func (t *Trader) Cancel(coin string, oid int64) (CancelResult, error)
```

### `CancelByCloid` {#cancelbycloid}

Cancel a single open order by its client order id.

```go
func (t *Trader) CancelByCloid(coin, cloid string) (CancelResult, error)
```

### `CancelAll` {#cancelall}

Cancel every open order across the supplied coins. With no coins supplied it cancels everything across every asset.

```go
func (t *Trader) CancelAll(coins ...string) (BatchCancelResult, error)
```

**Example**

```go
_, err := c.Trade.CancelAll()        // cancel everything
_, err  = c.Trade.CancelAll("ETH")   // ETH only
```

---

## Position management

### `ClosePosition` {#closeposition}

Flatten the caller's open position on `coin`. Direction is inferred from the cached `UserState`; long positions exit with a sell, shorts with a buy. By default the close is an IOC at mid plus or minus `slippage` (default 5%). Pass `WithLimit(px)` for a limit close or `WithSize(x)` for a partial close.

```go
func (t *Trader) ClosePosition(coin string, opts ...PlaceOpt) (Result, error)
```

**Validation**: `no_position` (no open position on `coin`), `close_size_exceeds_position` (when `WithSize` exceeds the absolute position size).

**Example**

```go
res, err := c.Trade.ClosePosition("ETH")
```

### `SetLeverage` {#setleverage}

Update the leverage on `coin`. `mode` picks `Cross` (shared collateral) or `Isolated` (per-position).

```go
func (t *Trader) SetLeverage(coin string, leverage int, mode MarginMode) (*UserState, error)
```

**Example**

```go
state, err := c.Trade.SetLeverage("ETH", 5, hl.Cross)
```

### `AdjustMargin` {#adjustmargin}

Add or remove isolated-margin collateral on the position in `coin`. Positive amount adds; negative withdraws. `amount` is in decimal USDC.

```go
func (t *Trader) AdjustMargin(coin string, amount float64) (*APIResponse[DefaultResponse], error)
```

### `ScheduleCancelAll` {#schedulecancelall}

Schedule cancellation of all open orders at a deadline. `nil` clears any scheduled cancel.

```go
func (t *Trader) ScheduleCancelAll(deadline *time.Time) (*ScheduleCancelResponse, error)
```

### `RefreshState` {#refreshstate}

Refresh the cached `UserState` snapshot used by position-aware validation. The placement pipeline refreshes implicitly on every call unless `SkipValidation()` is in effect; calling `RefreshState` directly is useful if you want a recent state for your own logic.

```go
func (t *Trader) RefreshState(ctx context.Context) error
```

**Related**: [`ValidationError` codes](./errors.md#validationerror).

---

## Transfers {#transfers}

All transfer actions are reachable via `c.Trade.Transfer`. They return `*TransferResponse`.

### `Transfer.SendUSD` {#transfer-sendusd}

Send USDC to another address.

```go
func (g *TransferGroup) SendUSD(toAddr string, amount float64) (*TransferResponse, error)
```

### `Transfer.SendSpot` {#transfer-sendspot}

Send a spot token to another address.

```go
func (g *TransferGroup) SendSpot(toAddr, token string, amount float64) (*TransferResponse, error)
```

### `Transfer.DepositToVault` {#transfer-deposittovault}

Deposit USDC into a vault.

```go
func (g *TransferGroup) DepositToVault(vaultAddr string, amount float64) (*TransferResponse, error)
```

### `Transfer.WithdrawFromVault` {#transfer-withdrawfromvault}

Withdraw USDC from a vault.

```go
func (g *TransferGroup) WithdrawFromVault(vaultAddr string, amount float64) (*TransferResponse, error)
```

### `Transfer.PerpToSpot` {#transfer-perptospot}

Move USDC from the perps wallet to the spot wallet.

```go
func (g *TransferGroup) PerpToSpot(amount float64) (*TransferResponse, error)
```

### `Transfer.SpotToPerp` {#transfer-spottoperp}

Move USDC from the spot wallet to the perps wallet.

```go
func (g *TransferGroup) SpotToPerp(amount float64) (*TransferResponse, error)
```

### `Transfer.MoveToDex` {#transfer-movetodex}

Move balance from the default perp dex into a HIP-3 builder-deployed dex.

```go
func (g *TransferGroup) MoveToDex(dex, token string, amount float64) (*TransferResponse, error)
```

### `Transfer.MoveFromDex` {#transfer-movefromdex}

Move balance back from a HIP-3 builder-deployed dex to the default perp dex.

```go
func (g *TransferGroup) MoveFromDex(dex, token string, amount float64) (*TransferResponse, error)
```

---

## Withdraw {#withdraw}

### `Withdraw`

Withdraw USDC off the Hyperliquid L1 bridge to an external destination.

```go
func (t *Trader) Withdraw(amount float64, destination string) (*TransferResponse, error)
```

**Example**

```go
_, err := c.Trade.Withdraw(100.0, "0xabc...")
```

---

## Sub-accounts {#sub-accounts}

Accessible via `c.Trade.SubAccount`.

### `SubAccount.Create` {#subaccount-create}

```go
func (g *SubAccountGroup) Create(name string) (*CreateSubAccountResponse, error)
```

### `SubAccount.DepositUSD` {#subaccount-depositusd}

```go
func (g *SubAccountGroup) DepositUSD(subAddr string, amount float64) (*TransferResponse, error)
```

### `SubAccount.WithdrawUSD` {#subaccount-withdrawusd}

```go
func (g *SubAccountGroup) WithdrawUSD(subAddr string, amount float64) (*TransferResponse, error)
```

### `SubAccount.DepositSpot` {#subaccount-depositspot}

```go
func (g *SubAccountGroup) DepositSpot(subAddr, token string, amount float64) (*TransferResponse, error)
```

### `SubAccount.WithdrawSpot` {#subaccount-withdrawspot}

```go
func (g *SubAccountGroup) WithdrawSpot(subAddr, token string, amount float64) (*TransferResponse, error)
```

---

## Staking {#staking}

Accessible via `c.Trade.Stake`. The `wei` argument is the staked amount in HYPE wei.

### `Stake.Delegate` {#stake-delegate}

```go
func (g *StakeGroup) Delegate(validator string, wei int) (*TransferResponse, error)
```

### `Stake.Undelegate` {#stake-undelegate}

```go
func (g *StakeGroup) Undelegate(validator string, wei int) (*TransferResponse, error)
```

---

## Multi-sig {#multi-sig}

Accessible via `c.Trade.MultiSig`.

### `MultiSig.Convert` {#multisig-convert}

Convert the caller's account to a multi-sig wallet with the given authorized signers and approval threshold.

```go
func (g *MultiSigGroup) Convert(authorized []string, threshold int) (*MultiSigConversionResponse, error)
```

### `MultiSig.Execute` {#multisig-execute}

Execute a previously assembled action with a set of signatures.

```go
func (g *MultiSigGroup) Execute(action map[string]any, signers []string, signatures []string) (*MultiSigResponse, error)
```

---

## Account control

### `ApproveAgent` {#approveagent}

Provision a fresh agent key authorized for trading on behalf of the caller's account. The returned `Agent` carries the agent address and freshly generated private key — keep the key secret.

```go
func (t *Trader) ApproveAgent(name string) (Agent, error)
```

**Example**

```go
agent, err := c.Trade.ApproveAgent("my-bot")
// Use agent.PrivateKey + WithAccount(myMainAddress) on subsequent New(...) calls.
```

### `ApproveBuilderFee` {#approvebuilderfee}

Approve a HIP-1 builder to charge a max fee rate (string-encoded, e.g. `"0.001%"`) on the caller's orders.

```go
func (t *Trader) ApproveBuilderFee(builder string, maxFeeRate string) (*ApprovalResponse, error)
```

### `SetReferrer` {#setreferrer}

Set the caller's referrer code (once per account, irreversibly).

```go
func (t *Trader) SetReferrer(code string) (*SetReferrerResponse, error)
```

### `UseBigBlocks` {#usebigblocks}

Opt the caller's address into "big block" inclusion.

```go
func (t *Trader) UseBigBlocks(enable bool) (*ApprovalResponse, error)
```

---

## HIP-2 / HIP-3 deploy {#deploy}

Rare expert operations. All methods return `*SpotDeployResponse` (spot) or `*PerpDeployResponse` (perp).

### Spot deploy

```go
func (t *Trader) SpotDeployRegisterToken(tokenName string, szDecimals, weiDecimals, maxGas int, fullName string) (*SpotDeployResponse, error)
func (t *Trader) SpotDeployUserGenesis(balances map[string]float64) (*SpotDeployResponse, error)
func (t *Trader) SpotDeployEnableFreezePrivilege() (*SpotDeployResponse, error)
func (t *Trader) SpotDeployFreezeUser(userAddress string) (*SpotDeployResponse, error)
func (t *Trader) SpotDeployRevokeFreezePrivilege() (*SpotDeployResponse, error)
func (t *Trader) SpotDeployGenesis(deployer string, dexName string) (*SpotDeployResponse, error)
func (t *Trader) SpotDeployRegisterSpot(baseToken, quoteToken string) (*SpotDeployResponse, error)
func (t *Trader) SpotDeployRegisterHyperliquidity(name string, tokens []string) (*SpotDeployResponse, error)
func (t *Trader) SpotDeploySetDeployerTradingFeeShare(feeShare float64) (*SpotDeployResponse, error)
```

### Perp deploy

```go
func (t *Trader) PerpDeployRegisterAsset(asset string, perpDexInput PerpDexSchemaInput) (*PerpDeployResponse, error)
func (t *Trader) PerpDeploySetOracle(asset string, oracleAddress string) (*SpotDeployResponse, error)
```

---

## Validator operations {#validators}

Validator-only actions; pass-through wrappers around the signed-action endpoint.

```go
func (t *Trader) CSignerJailSelf() (*ValidatorResponse, error)
func (t *Trader) CSignerUnjailSelf() (*ValidatorResponse, error)
func (t *Trader) CSignerInner(innerAction map[string]any) (*ValidatorResponse, error)
func (t *Trader) CValidatorRegister(validatorProfile map[string]any) (*ValidatorResponse, error)
func (t *Trader) CValidatorChangeProfile(newProfile map[string]any) (*ValidatorResponse, error)
func (t *Trader) CValidatorUnregister() (*ValidatorResponse, error)
```
