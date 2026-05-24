# Package Reorganization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the flat single-package Go SDK (`hyperliquid`, ~75 files at root) into domain-bounded subpackages (`types/`, `info/`, `trade/`, `stream/`, `signing/`) behind an unchanged `hyperliquid.Client` facade, on a dedicated branch where every commit compiles and passes `go test ./...`.

**Architecture:** Bottom-up dependency order — leaves (`types/`, `internal/msgpack/`) first, then `signing/`, then the three domain packages, then the facade rewrite, then cleanup. A single root `compat.go` holds temporary `type X = subpkg.X` aliases so intermediate commits stay green; it is deleted in the final phase along with the `v0.x-pre-rename` tag.

**Tech Stack:** Go 1.23, `gorilla/websocket`, `valyala/fastjson`, `vmihailenco/msgpack`, `ethereum/go-ethereum` (signing). Module path: `github.com/Simon-Busch/hyperliquid-go`.

---

## How to read this plan

This is a **refactor**, not a feature build. No new behaviour is introduced; the unit + integration test suite is the regression net. Every task therefore follows the same micro-pattern:

1. Move/create code.
2. Run the **verification gate** (see below).
3. Commit.

There is no TDD loop because we are not writing new behaviour — there is nothing to red-then-green. If a move regresses a test, the task is not "done" until the test is green again on the same commit.

### Verification gate

Run after **every** code change, before every commit:

```bash
go build ./...
go vet ./...
go test ./...
```

Expected output: zero compiler errors, zero vet warnings, all tests `PASS`. If any of these fail, the task is incomplete.

At phase boundaries (end of Phase 0, 1, 3, 4, 5, 6, 7, 9) also run:

```bash
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

(Integration tests respect their existing env-var gates; absent credentials they will skip rather than fail.)

### Commit message convention

Every commit on this branch uses one of two prefixes:

- `refactor(layout): ...` — file moves, package introductions, alias additions.
- `refactor(layout!): ...` — breaking-change commits (Phase 9 cleanup only).

End with the standard footer:

```
Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

### Working directory

All paths are relative to repo root `/Users/simonbusch/code/opensource-fork/go-hl-0xsi`. Do not `cd`; use absolute paths or run from repo root.

### `compat.go` conventions

The compat shim grows across many tasks. Every task that "appends to compat.go" means:

1. Add the new `import` paths into the **existing single `import (...)` block** at the top (don't create a second block — Go will reject multiple top-level `import` blocks; well, it accepts them but `gofmt` collapses them — keep the file `gofmt`-clean).
2. Append type/var/const declarations under a new `// --- <source-file>.go aliases ---` section header at the end of the file.
3. Run `gofmt -w compat.go` before committing.

Re-export style: `type X = pkg.X` for types, `var X = pkg.X` for functions and variables, `const X = pkg.X` for constants. If any caller needs `X` as a `func` value (rare), replace the `var` form with an explicit wrapper:

```go
func X(args...) ResultType { return pkg.X(args...) }
```

---

## Phase 0 — Branch & baseline

### Task 0.1: Create the refactor branch

**Files:** none changed.

- [ ] **Step 1: Confirm starting branch is clean and at the expected commit.**

Run:
```bash
git status
git rev-parse --abbrev-ref HEAD
git log -1 --oneline
```

Expected: working tree clean, branch `refactor/ux-api`, HEAD at `0228767 docs: spec for package reorganization (domain-first layout)` (or descendant).

- [ ] **Step 2: Cut the dedicated refactor branch.**

Run:
```bash
git checkout -b refactor/package-layout
```

Expected: `Switched to a new branch 'refactor/package-layout'`.

- [ ] **Step 3: Capture baseline test output.**

Run:
```bash
go build ./... && go vet ./... && go test ./... 2>&1 | tee /tmp/baseline-tests.txt
tail -20 /tmp/baseline-tests.txt
```

Expected: all packages `ok`. Save the last 5 lines for the commit body.

- [ ] **Step 4: Record baseline in an empty commit.**

Run (substitute the actual `ok` summary lines into the HEREDOC):
```bash
git commit --allow-empty -m "$(cat <<'EOF'
refactor(layout): baseline for package reorganization

Starting point for the refactor described in
docs/superpowers/specs/2026-05-24-package-reorganization-design.md.
`go build ./... && go vet ./... && go test ./...` green at this commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: empty commit created on `refactor/package-layout`.

---

## Phase 1 — Introduce `types/` package

The shared-types package holds only what is referenced from **more than one** of `info/`, `trade/`, `stream/`, or by typical caller code (`hl.Buy`, `hl.GTC(...)`, etc.). Domain-specific response types stay at the root for now and move into their owning subpackages in Phases 4–6.

### Task 1.1: Create the `types/` package skeleton

**Files:**
- Create: `types/doc.go`
- Create: `compat.go`

- [ ] **Step 1: Create the package doc file.**

Write `types/doc.go`:

```go
// Package types holds Hyperliquid SDK domain types that are shared
// across the info, trade, and stream subpackages or referenced directly
// by typical caller code. Subpackage-specific request/response shapes
// live in their respective packages.
package types
```

- [ ] **Step 2: Create the empty compat shim at root.**

Write `compat.go`:

```go
package hyperliquid

// Compat aliases re-export types that have moved into subpackages during
// the refactor described in docs/superpowers/specs/2026-05-24-package-reorganization-design.md.
// This file grows phase by phase and is deleted whole in the final
// cleanup phase. Do NOT add new symbols here outside that refactor.
```

- [ ] **Step 3: Verify build/test gate.**

Run:
```bash
go build ./...
go vet ./...
go test ./...
```

Expected: all green; `types` is a new but empty package.

- [ ] **Step 4: Commit.**

```bash
git add types/doc.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): introduce types/ package skeleton

Empty subpackage that subsequent Phase 1 commits will populate with
shared domain types (Side, Tif, Grouping, OrderSpec, Bracket, Result
family, AssetClass, Cloid, OutcomeQuestion). Adds an empty root
compat.go that will accumulate type aliases for backwards-compatibility
of intermediate commits.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.2: Move `Side` and related constants

**Files:**
- Create: `types/side.go`
- Delete: `side.go`
- Modify: `compat.go`

- [ ] **Step 1: Create `types/side.go` with the moved code.**

Copy the full contents of `side.go` into `types/side.go`, changing only the `package hyperliquid` line to `package types`. Do not modify any logic or doc comments. The exported symbols moving are: `Side`, `Buy`, `Sell`, `SideBid`, `SideAsk`, `Side.IsBuy`, `TIF`, `tifALO`, `tifIOC`, `tifGTC`, `MarginMode`, `Cross`, `Isolated`.

- [ ] **Step 2: Delete the original root file.**

Run:
```bash
rm side.go
```

- [ ] **Step 3: Add compat aliases.**

Append to `compat.go`:

```go
import "github.com/Simon-Busch/hyperliquid-go/types"

// --- side.go aliases ---

type Side = types.Side

const (
	Buy     = types.Buy
	Sell    = types.Sell
	SideBid = types.SideBid
	SideAsk = types.SideAsk
)

type TIF = types.TIF

type MarginMode = types.MarginMode

const (
	Cross    = types.Cross
	Isolated = types.Isolated
)
```

- [ ] **Step 4: Verify build/test gate.**

Run:
```bash
go build ./...
go vet ./...
go test ./...
```

Expected: all green. The `side_test.go` file is in the root package and exercises the aliases — it must still pass unchanged.

- [ ] **Step 5: Commit.**

```bash
git add types/side.go compat.go side.go
git commit -m "$(cat <<'EOF'
refactor(layout): move Side / TIF / MarginMode to types/

Pure relocation; behaviour unchanged. Root re-exports via compat.go so
existing call sites and the in-package side_test.go continue to compile.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.3: Move `Cloid` and the wire order-type structs

**Rationale:** These are needed by both `trade` (placing orders) and `info` (echoing back open orders / fills) so they belong in `types/`. `BuilderInfo` rides along — it's used both by placement (HIP-1 builder fee) and by `info.PerpDexs()`-derived metadata.

**Files:**
- Create: `types/order_type.go`
- Modify: `types.go` (remove the moved structs)
- Modify: `compat.go`

- [ ] **Step 1: Create `types/order_type.go`.**

Write the file containing exactly the following blocks (extracted verbatim from `types.go` lines 175–248):

```go
package types

// OrderType discriminates a limit order from a trigger order. Exactly
// one of Limit or Trigger should be populated.
type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

// LimitOrderType holds the time-in-force tag for a limit order.
type LimitOrderType struct {
	Tif string `json:"tif"` // TifAlo, TifIoc, TifGtc
}

// TriggerOrderType describes a trigger (stop) order.
type TriggerOrderType struct {
	TriggerPx float64 `json:"triggerPx"`
	IsMarket  bool    `json:"isMarket"`
	Tpsl      string  `json:"tpsl"` // "tp" or "sl"
}

// BuilderInfo carries the builder address and per-order fee (in basis
// points) used by HIP-3 builder-deployed perp markets.
type BuilderInfo struct {
	Builder string `json:"b" msgpack:"b"`
	Fee     int    `json:"f" msgpack:"f"`
}

// OrderTypeWire is the wire variant of OrderType.
type OrderTypeWire struct {
	Limit   *LimitOrderTypeWire   `json:"limit,omitempty" msgpack:"limit,omitempty"`
	Trigger *TriggerOrderTypeWire `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

// LimitOrderTypeWire is the wire variant of LimitOrderType.
type LimitOrderTypeWire struct {
	Tif string `json:"tif" msgpack:"tif"`
}

// TriggerOrderTypeWire is the wire variant of TriggerOrderType.
// TriggerPx is encoded as a string for stable msgpack ordering.
type TriggerOrderTypeWire struct {
	IsMarket  bool   `json:"isMarket" msgpack:"isMarket"`
	TriggerPx string `json:"triggerPx" msgpack:"triggerPx"`
	Tpsl      string `json:"tpsl" msgpack:"tpsl"`
}

// Cloid wraps a client order id string in a typed value.
type Cloid struct {
	Value string
}

// ToRaw returns the underlying client order id string.
func (c Cloid) ToRaw() string {
	return c.Value
}
```

- [ ] **Step 2: Remove the same blocks from `types.go`.**

Open `types.go` and delete the `OrderType`, `LimitOrderType`, `TriggerOrderType`, `BuilderInfo`, `OrderTypeWire`, `LimitOrderTypeWire`, `TriggerOrderTypeWire`, `CancelRequest` is NOT moved (it's trade-only — stays for Phase 5), `CancelByCloidRequest` NOT moved (same), and `Cloid` (and its `ToRaw` method) blocks.

- [ ] **Step 3: Add compat aliases.**

Append to `compat.go`:

```go
// --- types.go order-type aliases ---

type OrderType = types.OrderType
type LimitOrderType = types.LimitOrderType
type TriggerOrderType = types.TriggerOrderType
type BuilderInfo = types.BuilderInfo
type OrderTypeWire = types.OrderTypeWire
type LimitOrderTypeWire = types.LimitOrderTypeWire
type TriggerOrderTypeWire = types.TriggerOrderTypeWire
type Cloid = types.Cloid
```

- [ ] **Step 4: Verify.**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: green. `actions.go`, `validate.go`, `trader_place.go`, and the `*_test.go` files that reference these structs compile through the aliases.

- [ ] **Step 5: Commit.**

```bash
git add types/order_type.go types.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move OrderType / Cloid / BuilderInfo to types/

These are shared by every domain (trade places them, info echoes them,
stream emits them in fill events) and belong in the shared types
package. Root compat.go re-exports them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.4: Move `Grouping` and the Tif constants

**Files:**
- Create: `types/grouping.go`
- Modify: `types.go` (remove moved block)
- Modify: `compat.go`

- [ ] **Step 1: Create `types/grouping.go`.**

```go
package types

// Grouping is the order-grouping discriminator used by /exchange order
// actions.
type Grouping string

const (
	GroupingNA           Grouping = "na"
	GroupingNormalTpsl   Grouping = "normalTpsl"
	GroupingPositionTpls Grouping = "positionTpsl"
)

// DefaultSlippage is the default worst-case-fill slippage for PlaceMarket.
const DefaultSlippage = 0.05

// Order Time-in-Force constants.
const (
	TifAlo = "Alo"
	TifIoc = "Ioc"
	TifGtc = "Gtc"
)
```

- [ ] **Step 2: Delete the matching block (lines 5–29 of the current `types.go`) from `types.go`.**

- [ ] **Step 3: Append compat aliases.**

In `compat.go`:

```go
// --- types.go grouping/Tif aliases ---

type Grouping = types.Grouping

const (
	GroupingNA           = types.GroupingNA
	GroupingNormalTpsl   = types.GroupingNormalTpsl
	GroupingPositionTpls = types.GroupingPositionTpls

	DefaultSlippage = types.DefaultSlippage

	TifAlo = types.TifAlo
	TifIoc = types.TifIoc
	TifGtc = types.TifGtc
)
```

- [ ] **Step 4: Verify.**

```bash
go build ./...
go vet ./...
go test ./...
```

- [ ] **Step 5: Commit.**

```bash
git add types/grouping.go types.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move Grouping / TifAlo / DefaultSlippage to types/

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.5: Move `OrderSpec` and `PlaceOpt`-target fields

**Files:**
- Create: `types/orderspec.go`
- Delete: `orderspec.go`
- Modify: `compat.go`

- [ ] **Step 1: Copy `orderspec.go` to `types/orderspec.go`, changing `package hyperliquid` to `package types`.**

Important: the field types reference `Side` and `TIF` which already moved in Task 1.2 — they resolve in-package now, no import needed.

- [ ] **Step 2: Delete the original.**

```bash
rm orderspec.go
```

- [ ] **Step 3: Append compat alias.**

In `compat.go`:

```go
// --- orderspec.go alias ---

type OrderSpec = types.OrderSpec
```

- [ ] **Step 4: Verify.**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: `opts.go` (the `PlaceOpt` functions) and `trader_place.go` compile through the alias.

- [ ] **Step 5: Commit.**

```bash
git add types/orderspec.go orderspec.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move OrderSpec to types/

PlaceOpt callers (opts.go) and trader_place.go reach OrderSpec via the
root alias; behaviour unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.6: Move the `Result` family

**Files:**
- Create: `types/result.go`
- Delete: `result.go`
- Modify: `compat.go`

- [ ] **Step 1: Copy `result.go` to `types/result.go`, changing package to `types`.**

- [ ] **Step 2: Delete original.** `rm result.go`

- [ ] **Step 3: Add aliases to `compat.go`:**

```go
// --- result.go aliases ---

type Result            = types.Result
type BatchResult       = types.BatchResult
type CancelResult      = types.CancelResult
type BatchCancelResult = types.BatchCancelResult
```

- [ ] **Step 4: Verify gate.**

- [ ] **Step 5: Commit.**

```bash
git add types/result.go result.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move Result family to types/

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.7: Move `Bracket` helper

**Files:**
- Create: `types/bracket.go`
- Delete: `bracket.go`

**Note:** `bracketOrders` references `CreateOrderRequest`, which is currently defined in `actions.go` at the root and is signing-internal. To avoid pulling `CreateOrderRequest` into `types/`, we keep `bracketOrders` at the root for now and move it as part of Phase 3 (signing) or Phase 5 (trade) — whichever ends up owning `CreateOrderRequest`. **This task is therefore a NO-OP and is recorded only to keep phase count honest.**

- [ ] **Step 1: Verify no work needed.**

Run:
```bash
grep -n "CreateOrderRequest" bracket.go
```

Expected: matches confirm `bracketOrders` depends on a non-`types` symbol. Skip the move.

- [ ] **Step 2: Add a comment to `bracket.go` noting deferred move.**

Insert at the top of `bracket.go` (right under `package hyperliquid`):

```go
// NOTE: bracketOrders stays in the root package during Phase 1 because
// it depends on CreateOrderRequest, which lives in actions.go. It moves
// alongside CreateOrderRequest in Phase 3 (signing/) or Phase 5 (trade/).
```

- [ ] **Step 3: Verify + commit.**

```bash
go build ./...
git add bracket.go
git commit -m "$(cat <<'EOF'
refactor(layout): note Bracket move deferred to Phase 3/5

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.8: Move `AssetClass`

**Files:**
- Create: `types/asset_class.go`
- Delete: `asset_class.go`
- Modify: `compat.go`

**Note:** `ClassifyAsset` references the `outcomeAssetBase`, `builderPerpAssetBase`, `spotAssetIndexOffset` constants defined in `info.go`. These constants must move into `types/` alongside `AssetClass`, since they are referenced elsewhere too (e.g. `Asset` lookups). Strategy: copy the three constants into `types/asset_class.go` as unexported `assetSpotOffset`, `assetBuilderPerpBase`, `assetOutcomeBase`; the root copies remain unchanged.

- [ ] **Step 1: Create `types/asset_class.go`.**

```go
package types

// Asset-id range constants. Duplicated from internal info package
// constants during the refactor; the info package retains its own copy.
const (
	assetSpotOffset      = 10_000
	assetBuilderPerpBase = 100_000
	assetOutcomeBase     = 100_000_000
)

// AssetClass categorises a numeric asset ID by its origin and tick rules.
//
// Ranges (per Hyperliquid asset-IDs reference):
//   - default perp:    0..9_999
//   - spot:            10_000..99_999
//   - builder perp:    100_000..99_999_999       (HIP-3)
//   - outcome market:  100_000_000+              (HIP-4)
type AssetClass int

const (
	AssetClassPerp AssetClass = iota
	AssetClassSpot
	AssetClassBuilderPerp
	AssetClassOutcome
)

// ClassifyAsset maps a numeric asset ID to its AssetClass.
func ClassifyAsset(asset int) AssetClass {
	switch {
	case asset >= assetOutcomeBase:
		return AssetClassOutcome
	case asset >= assetBuilderPerpBase:
		return AssetClassBuilderPerp
	case asset >= assetSpotOffset:
		return AssetClassSpot
	default:
		return AssetClassPerp
	}
}

// MaxPriceDecimals returns MAX_DECIMALS used in the tick-size formula:
//
//	allowedPriceDecimals = MaxPriceDecimals() - szDecimals
//
// 8 for spot, 6 for everything else.
func (c AssetClass) MaxPriceDecimals() int {
	if c == AssetClassSpot {
		return 8
	}
	return 6
}

// IsSpotLike reports whether this asset class uses spot pricing rules.
func (c AssetClass) IsSpotLike() bool {
	return c == AssetClassSpot
}
```

- [ ] **Step 2: Delete `asset_class.go`.**

```bash
rm asset_class.go
```

- [ ] **Step 3: Append compat aliases:**

```go
// --- asset_class.go aliases ---

type AssetClass = types.AssetClass

const (
	AssetClassPerp        = types.AssetClassPerp
	AssetClassSpot        = types.AssetClassSpot
	AssetClassBuilderPerp = types.AssetClassBuilderPerp
	AssetClassOutcome     = types.AssetClassOutcome
)

var ClassifyAsset = types.ClassifyAsset
```

- [ ] **Step 4: Verify.**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: `asset_class_test.go` at the root continues to pass via aliases.

- [ ] **Step 5: Commit.**

```bash
git add types/asset_class.go asset_class.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move AssetClass to types/

Asset-id range constants are duplicated in types/ (unexported) so the
package is self-contained; the originals in info.go remain until Phase 4.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.9: Move `OutcomeQuestion` helpers

**Files:**
- Create: `types/outcome_question.go`
- Delete: `outcome_question.go`
- Modify: `compat.go`

**Dependency:** `outcome_question.go` references `OutcomeMeta`, `OutcomeInfo`, `OutcomeSideSpec`, `Question` from `types.go`. Those response types are **info-specific**, not shared. They will move into `info/` in Phase 4. To avoid coupling `types` to `info`, we keep the `OutcomeMeta.FindQuestion`, `OutcomeMeta.BucketLabel`, `OutcomeMeta.QuestionByName` methods at the root (they are methods on root types) and move only the pure-functional helpers `ParseOutcomeDescription`, `Question.BucketLabels`, `Question.Buckets` — but those also receive `Question`, an info-owned type. **This task is therefore also a NO-OP for Phase 1; the file moves with the rest of the outcome types in Phase 4.**

- [ ] **Step 1: Insert a deferred-move comment in `outcome_question.go`.**

At the top, under `package hyperliquid`:

```go
// NOTE: outcome_question.go stays in the root package during Phase 1
// because its helpers receive Question / OutcomeMeta, which are
// info-owned response shapes. It moves into info/ in Phase 4.
```

- [ ] **Step 2: Verify + commit.**

```bash
go build ./...
git add outcome_question.go
git commit -m "$(cat <<'EOF'
refactor(layout): note outcome_question.go move deferred to Phase 4

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 1.10: Phase 1 boundary check

- [ ] **Step 1: Run the full verification gate including integration tests.**

```bash
go build ./...
go vet ./...
go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

Expected: all green; integration tests either pass or skip (no credentials).

- [ ] **Step 2: Tag the phase boundary.**

```bash
git tag refactor/phase-1-done
```

---

## Phase 2 — Extract msgpack encoder to `internal/msgpack/`

**Goal:** Move the deterministic-msgpack encoding helpers out of `actions.go` into an internal package so `actions.go` becomes a pure declaration file.

### Task 2.1: Create `internal/msgpack/` skeleton and identify movable code

**Files:**
- Read: `actions.go`
- Create: `internal/msgpack/msgpack.go`

- [ ] **Step 1: Read `actions.go` end to end and identify all functions whose body uses `github.com/vmihailenco/msgpack/v5` or constructs wire-form encodings.**

Specifically look for: any function that returns `[]byte`, calls `msgpack.Marshal`, or constructs a `bytes.Buffer` for encoding. Do **not** move the action-struct type declarations themselves — only the encoding functions.

- [ ] **Step 2: Create `internal/msgpack/msgpack.go`.**

```go
// Package msgpack holds the deterministic msgpack encoders used by the
// signing pipeline. The encoding must be byte-stable because the hash
// of the encoded action is part of the EIP-712 payload signed by the
// trader's wallet — any non-determinism would invalidate signatures.
package msgpack
```

- [ ] **Step 3: Move each identified encoder function into `internal/msgpack/msgpack.go`, exporting it (capitalize first letter) and updating callers in `actions.go` (and any other root file that calls it) to use the new import path `github.com/Simon-Busch/hyperliquid-go/internal/msgpack`.**

For example, if `actions.go` had:
```go
func encodeAction(a any) ([]byte, error) { /* msgpack.Marshal ... */ }
```
move it as `msgpack.EncodeAction` in the internal package and replace the call site with `msgpack.EncodeAction(a)`.

- [ ] **Step 4: Verify gate.**

```bash
go build ./...
go vet ./...
go test ./...
```

Specifically watch `types_msgpack_test.go`, `signing_test.go`, `fixtures_signing_test.go` — these exercise the deterministic encoding and must remain green.

- [ ] **Step 5: Commit.**

```bash
git add internal/msgpack/msgpack.go actions.go
git commit -m "$(cat <<'EOF'
refactor(layout): extract deterministic msgpack encoder to internal/msgpack

actions.go retains only the action struct declarations; the encoding
helpers move to an internal package so they can be reused by signing/
without going through the root package.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 2.2: Sanity check signing fixtures

- [ ] **Step 1: Re-run the signature fixture suite explicitly.**

```bash
go test -run "TestFixtureSigning|TestSign" -v ./...
```

Expected: every test that compares against `testfixtures/` golden files PASSES. If any FAILS, revert the previous commit immediately — msgpack determinism was broken.

- [ ] **Step 2: No commit (verification-only task).** Move on to Phase 3.

---

## Phase 3 — Public `signing/` package

### Task 3.1: Create `signing/` skeleton and move signing.go

**Files:**
- Create: `signing/signing.go`
- Delete: `signing.go`
- Modify: `compat.go`

- [ ] **Step 1: Copy `signing.go` to `signing/signing.go`.**

Change `package hyperliquid` to `package signing`. Update the internal import path that previously read:
```go
"github.com/Simon-Busch/hyperliquid-go/internal/eip712"
```
That path stays the same — `signing/` is still inside the module and can reach `internal/eip712`.

Any reference to root-package types (e.g. action structs) must be qualified with `hl "github.com/Simon-Busch/hyperliquid-go"` for now. Better: defer this task until Task 3.2 moves the action structs alongside.

- [ ] **Step 2: Audit the file's symbols for root-package references.**

Run:
```bash
grep -nE "\\b([A-Z][A-Za-z0-9]*Action|CreateOrderRequest|OrderSpec|Trader)\\b" signing/signing.go
```

For each match that is a root-package symbol but **not** an action struct, plan to:
(a) qualify it with `hl.` and add the import, OR
(b) move it into `signing/` if it's signing-specific.

Action structs themselves will move in Task 3.2.

- [ ] **Step 3: Add the `hl` import if any reference remains.**

Top of `signing/signing.go`:
```go
hl "github.com/Simon-Busch/hyperliquid-go"
```

This creates a temporary dependency `signing -> hyperliquid (root)` which is allowed during the refactor (root never imports signing, so there is no cycle).

- [ ] **Step 4: Delete the root `signing.go`.**

```bash
rm signing.go
```

- [ ] **Step 5: Append compat alias in `compat.go`.**

```go
import "github.com/Simon-Busch/hyperliquid-go/signing"

// --- signing.go aliases ---

type SignatureResult = signing.SignatureResult
```

Plus any other previously-exported signing symbols (e.g. `SignL1Action`, `SignUserSigned`):

```go
var (
	SignL1Action   = signing.SignL1Action
	SignUserSigned = signing.SignUserSigned
)
```

(Adjust to actual exported surface; verify with `grep -E "^func [A-Z]" signing/signing.go`.)

- [ ] **Step 6: Verify gate.**

```bash
go build ./...
go vet ./...
go test ./...
```

- [ ] **Step 7: Commit.**

```bash
git add signing/signing.go compat.go signing.go
git commit -m "$(cat <<'EOF'
refactor(layout): move signing.go to signing/ subpackage

Public package for callers that need to sign Hyperliquid actions
externally (custody integrations, hardware wallets). Root re-exports
SignatureResult and the SignL1Action / SignUserSigned helpers via
compat.go.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3.2: Move action structs to `signing/actions.go`

**Files:**
- Create: `signing/actions.go`
- Delete: `actions.go`
- Modify: `compat.go`

- [ ] **Step 1: Move every action struct from `actions.go` into `signing/actions.go`.**

Change `package hyperliquid` to `package signing`. The structs (full list to verify with `grep "^type " actions.go`): `CancelOrderWire`, `CancelAction`, `CancelByCloidWire`, `CancelByCloidAction`, `CreateOrderAction`, `CreateOrderRequest`, `CreateOrderWire`, `ModifyOrderAction`, `ModifyOrderWire`, `BatchModifyAction`, plus the deterministic-tagged variants and any other `*Action` / `*Wire` types in the file.

If `actions.go` still contains helper functions left over from Task 2.1, move those too.

- [ ] **Step 2: Update root callers (`trader_place.go`, `trader_modify_cancel.go`, `exchange_orders.go`, `exchange_orders_cancel.go`, `bracket.go`) to reference these via `signing.CreateOrderRequest`, etc., OR via root aliases (preferred — keeps the diff narrower).**

- [ ] **Step 3: Append the full set of compat aliases for every moved type.**

For each `type Foo struct { ... }` moved, add:
```go
type Foo = signing.Foo
```

Group them under a comment:
```go
// --- actions.go aliases ---
```

- [ ] **Step 4: Delete `actions.go`.**

```bash
rm actions.go
```

- [ ] **Step 5: Verify gate, with explicit attention to signing fixtures.**

```bash
go build ./...
go vet ./...
go test ./...
go test -run "TestFixtureSigning|TestSign|TestMsgpack" -v ./...
```

- [ ] **Step 6: Commit.**

```bash
git add signing/actions.go compat.go actions.go
git commit -m "$(cat <<'EOF'
refactor(layout): move action structs to signing/actions.go

CreateOrderRequest, CancelAction, ModifyOrderAction and the rest of the
signature-payload structs live next to the signing helpers that consume
them. Root call sites resolve via compat aliases; field ordering is
preserved exactly (msgpack determinism guaranteed by signing fixture
tests).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3.3: Move `bracketOrders` next to `CreateOrderRequest`

**Files:**
- Modify: `signing/actions.go` (or new `signing/bracket.go`)
- Delete: `bracket.go`

- [ ] **Step 1: Move the `bracketOrders` function and the `Bracket` doc comments into `signing/bracket.go`.**

```go
package signing

import (
	"github.com/Simon-Busch/hyperliquid-go/types"
)

// bracketOrders builds the TP and SL CreateOrderRequests that bracket a parent OrderSpec.
func bracketOrders(spec *types.OrderSpec) []CreateOrderRequest {
	// ... copy verbatim from root bracket.go, replacing OrderSpec, OrderType,
	// TriggerOrderType references with types.OrderSpec / types.OrderType / types.TriggerOrderType ...
}
```

Note: `bracketOrders` was unexported. If root callers (`trader_place.go`) call it directly, export it as `BracketOrders` and update the call.

- [ ] **Step 2: Delete `bracket.go` at root.**

```bash
rm bracket.go
```

- [ ] **Step 3: Verify, then commit.**

```bash
go build ./...
go vet ./...
go test ./...
git add signing/bracket.go bracket.go
git commit -m "$(cat <<'EOF'
refactor(layout): move bracketOrders to signing/ next to CreateOrderRequest

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 3.4: Phase 3 boundary check

- [ ] **Step 1: Full gate + integration tests.**

```bash
go build ./...
go vet ./...
go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 2: Tag.**

```bash
git tag refactor/phase-3-done
```

---

## Phase 4 — `info/` package

Each task below moves exactly one file. The pattern is identical, so the first task is fully detailed; subsequent tasks list only the per-file substitutions.

### Per-task pattern (apply to every task in this phase)

1. Create `info/<basename>.go` containing the source of the corresponding `info_*.go` (or `info.go`) at root.
2. Change `package hyperliquid` to `package info`.
3. Receiver rename: `(i *Info)` → `(c *Client)`. Type rename: `type Info struct` → `type Client struct`.
4. Replace any reference to root-package shared types with their `types.` equivalent (e.g. `*Meta` → `*types.Meta` after Phase 4 moves those — but for now the root types still exist, so the simpler rule is: add `hl "github.com/Simon-Busch/hyperliquid-go"` and qualify as `hl.Meta`, OR import `types` and use `types.X` for the symbols already moved in Phase 1).
5. Delete the original root file.
6. Add a compat alias to `compat.go`. For the first moved file (`info.go`) this is `type Info = info.Client`.
7. Update any integration test files in `tests/integration/` that referenced methods on the moved file (the receiver type alias keeps existing `*hl.Info` calls valid; new method additions are not part of this refactor).
8. Run the verification gate.
9. Commit with message `refactor(layout): move info_<name>.go to info/`.

### Task 4.1: Bootstrap `info/` package — move `info.go`

**Files:**
- Create: `info/info.go`
- Delete: `info.go`
- Modify: `compat.go`

- [ ] **Step 1: Create `info/info.go`.**

Apply the per-task pattern to `info.go` (the 7388-byte file containing `type Info struct`, the asset-index constants, `NewInfo`, `OutcomeMetaCached`, `postTimeRangeRequest`, `PerpDexName`, helpers).

Important details:
- `type Info struct` → `type Client struct`.
- `NewInfo(...) *Info` → `New(...) *Client`.
- The asset-index constants `spotAssetIndexOffset`, `builderPerpAssetBase`, `outcomeAssetBase` stay (they are still used inside the file). The duplicate copies in `types/asset_class.go` are unexported — no conflict.
- `client *httpAPI` field — `httpAPI` lives at root in `http_api.go`. For now import it via `hl "github.com/Simon-Busch/hyperliquid-go"` and type the field as `*hl.HTTPAPI`. **If `httpAPI` is unexported at root, first export it (rename `httpAPI` → `HTTPAPI`, `newHTTPAPI` → `NewHTTPAPI`) in a tiny prep commit before this one** (see Step 0 below).
- `Stake *InfoStakeGroup` — `InfoStakeGroup` is defined in `info_staking.go`; it moves later. For now it's still at the root: use `hl.InfoStakeGroup`.
- The function references `info.Meta(...)`, `info.SpotMeta(...)`, `info.PerpDexs(...)`, `info.OutcomeMeta(...)` — these methods still live in root `info_*.go` files at this point. They will move in subsequent tasks. **To make this work**, the root `*Info` type and the new `info.Client` type must be the SAME thing during the migration. Achieve this by making the root `Info` an alias to `info.Client` in compat.go (`type Info = info.Client`) and **moving the methods one file at a time** — each method only exists in one place (the new file) and is reachable from both packages because the receiver type is the alias.

- [ ] **Step 0 (prep, separate commit):** Export `httpAPI` and `newHTTPAPI`.

```bash
git grep -l "httpAPI\\|newHTTPAPI" -- ':!docs/'
```
For each match, rename `httpAPI` → `HTTPAPI` and `newHTTPAPI` → `NewHTTPAPI`. Verify and commit:

```bash
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "$(cat <<'EOF'
refactor(layout): export httpAPI as HTTPAPI for cross-package access

Prepares for moving info/, trade/, stream/ into subpackages that need
to construct the HTTP client.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 1: Create `info/info.go` as described above.**

- [ ] **Step 2: Delete `info.go` at root.**

```bash
rm info.go
```

- [ ] **Step 3: Append compat alias.**

```go
import "github.com/Simon-Busch/hyperliquid-go/info"

// --- info package alias ---

type Info = info.Client

var NewInfo = info.New
```

- [ ] **Step 4: Verify.**

```bash
go build ./...
go vet ./...
go test ./...
```

If a method like `info.SpotMeta()` is called from inside `info/info.go` but defined in root `info_meta.go`, the build will fail because the methods are on different concrete types. Resolve by either:
(a) moving `info_meta.go` immediately as part of this same commit, OR
(b) temporarily duplicating the missing method stubs in `info/info.go` and removing them in the matching later task.

Option (a) is cleaner — adjust the task ordering: do `info_meta.go` and `info.go` in the same commit if the dependency requires it.

- [ ] **Step 5: Commit.**

```bash
git add info/info.go info.go compat.go
git commit -m "$(cat <<'EOF'
refactor(layout): move info.go to info/ subpackage

Renames *Info → *info.Client; root keeps type Info = info.Client alias
so existing callers in trader_*.go, stream_*.go and tests/integration/
continue to compile unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 4.2–4.8: Move remaining `info_*.go` files

Apply the per-task pattern to each file. Subsequent tasks have NO `New` rename or `Info struct` declaration — only methods.

| # | Source | Target | Receiver rename | Notes |
|---|--------|--------|-----------------|-------|
| 4.2 | `info_account.go` | `info/account.go` | `(i *Info)` → `(c *Client)` | Methods: `UserState`, `SpotBalances`, `Positions`, `Position`, `Fees`. Response types `UserState`, `SpotClearinghouseState`, `Position`, `MarginSummary`, `UserFees`, `FeeSchedule`, `Tiers`, `MMTier`, `VIPTier`, `UserVolume` move with them. Add corresponding compat aliases. |
| 4.3 | `info_market.go` | `info/market.go` | same | Methods: `Asset`, `AssetID`, `Mid`, `AllMids`, `AllMidsOn`, `Book`, `Candles`, `MetaAndAssetCtxs`, `SpotMetaAndAssetCtxs`. Types `L2Book`, `Level`, `Candle`, `AssetCtx`, `SpotAssetCtx`, `AssetMeta`, `MetaAndAssetCtxsResponse`, `MetaAndAssetCtxsRawResponse` move. Aliases. |
| 4.4 | `info_meta.go` | `info/meta.go` | same | Methods: `Meta`, `SpotMeta`, `OutcomeMeta`, `PerpDexs`. Types `Meta`, `AssetInfo`, `MarginTable`, `MarginTier`, `SpotMeta`, `SpotAssetInfo`, `SpotTokenInfo`, `EvmContract`, `OutcomeMeta`, `OutcomeInfo`, `OutcomeSideSpec`, `Question`, `PerpDex`, `PerpDexLimits`, `PerpDexStatus`, `PerpDeployAuctionStatus`, `PerpDexSchemaInput` move with their aliases. |
| 4.5 | `info_orders.go` | `info/orders.go` | same | Methods: `OpenOrders`, `FrontendOpenOrders`, `Fills`, `FillsBetween`, `Order`, `Fill`, `OrderByCloid`, `Referral`. Types `OpenOrder`, `FrontendOpenOrder`, `Fill`, `OrderStatus`, `OrderStatusResponse`, `ReferralState`, `ReferredBy`, `ReferrerState`, `ReferrerData`, `ReferralMember` move with their aliases. |
| 4.6 | `info_funding.go` | `info/funding.go` | same | Methods: `Funding`, `UserFunding`. Types `FundingHistory`, `UserFundingHistory` move. |
| 4.7 | `info_staking.go` | `info/staking.go` | same | Methods: `Validators`, `StakingSummary`, `Delegations`, `Rewards` (verify exact names). Types `StakingSummary`, `StakingDelegation`, `StakingReward` move. Also: rename `InfoStakeGroup` → `StakeGroup` and update the `Stake *InfoStakeGroup` field in `info/info.go`. Add `type InfoStakeGroup = info.StakeGroup` alias. |
| 4.8 | `outcome_question.go` | `info/outcome_question.go` | n/a — package-level helpers | `ParseOutcomeDescription`, `Question.BucketLabels`, `OutcomeMeta.FindQuestion`, `OutcomeMeta.BucketLabel`, `OutcomeMeta.QuestionByName`, `Question.Buckets`, `Bucket` struct. Aliases for `Bucket`, `var ParseOutcomeDescription = info.ParseOutcomeDescription`. |

For each task in this table:

- [ ] **Step 1: Create the target file** by copying source verbatim, changing `package hyperliquid` → `package info`, renaming receivers per the table, and replacing root-package shared-type references with `types.X` (for Phase-1-moved types) or `hl.X` (for not-yet-moved types).
- [ ] **Step 2: Delete the source root file.**
- [ ] **Step 3: Add type aliases** for every moved response struct to `compat.go` under a `// --- info_<name>.go aliases ---` header.
- [ ] **Step 4: Update integration tests in `tests/integration/`** that reference any renamed-but-not-aliased symbol (`grep -l "hl\\.OldName" tests/integration/`). Most will be no-ops because aliases keep them green.
- [ ] **Step 5: Verify gate.**
- [ ] **Step 6: Commit** with message `refactor(layout): move info_<name>.go to info/`.

### Task 4.9: Phase 4 boundary check

- [ ] **Step 1: Verify there are no `info_*.go` files left at root.**

```bash
ls info*.go 2>/dev/null && echo "FAIL: files still at root" || echo "OK"
```

- [ ] **Step 2: Full gate + integration tests.**

```bash
go build ./...
go vet ./...
go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 3: Tag.**

```bash
git tag refactor/phase-4-done
```

---

## Phase 5 — `trade/` package

Apply the same per-task pattern as Phase 4. Receiver rename: `(t *Trader)` → `(c *Client)`. Type rename: `type Trader struct` → `type Client struct`. Root alias: `type Trader = trade.Client`.

### Pre-task: dependency note

`trade/` will import:
- `github.com/Simon-Busch/hyperliquid-go/types` (OrderSpec, Side, Result, etc.)
- `github.com/Simon-Busch/hyperliquid-go/signing` (action structs, signers)
- `github.com/Simon-Busch/hyperliquid-go/info` (for the cached `*info.Client` reference used by `validate()`)
- `github.com/Simon-Busch/hyperliquid-go/internal/transport` (after Task 5.1 moves the generic HTTP envelope)

It must **not** be imported by `info/` or `stream/`.

### Task 5.1: Move generic HTTP envelope to `internal/transport/`

**Files:**
- Modify: `internal/transport/transport.go`
- Delete: `api.go`, `http_api.go`

- [ ] **Step 1: Read `api.go` and `http_api.go`.** Confirm they contain (a) the `APIResponse[T]` generic envelope and its parser, (b) the `HTTPAPI` struct with `post`/`get` methods. Both are generic — not trade-specific.

- [ ] **Step 2: Move the contents into `internal/transport/transport.go`**, changing `package hyperliquid` → `package transport`. Rename `HTTPAPI` → `Client` (within the transport package; the public API surface keeps using the type via existing aliases). Rename `NewHTTPAPI` → `New`.

- [ ] **Step 3: Add root compat aliases pointing into `internal/transport`.**

Wait — `internal/` packages cannot be imported by external code, but they CAN be imported by sibling packages in the same module. The root `hyperliquid` package can import `internal/transport`. Add to `compat.go`:

```go
import xtransport "github.com/Simon-Busch/hyperliquid-go/internal/transport"

type HTTPAPI = xtransport.Client

var NewHTTPAPI = xtransport.New

type APIResponse[T any] = xtransport.APIResponse[T]
```

(Verify the generic alias syntax compiles — Go 1.23 supports it.)

- [ ] **Step 4: Delete the originals.**

```bash
rm api.go http_api.go
```

- [ ] **Step 5: Verify gate.**

```bash
go build ./...
go vet ./...
go test ./...
```

`api_test.go` at root tests `APIResponse` parsing via aliases — must remain green.

- [ ] **Step 6: Commit.**

```bash
git add internal/transport/transport.go compat.go api.go http_api.go
git commit -m "$(cat <<'EOF'
refactor(layout): move generic HTTP envelope to internal/transport

api.go (APIResponse[T] parsing) and http_api.go (HTTPAPI client) hold
no trade-specific logic. Moving them into internal/transport lets
info/, trade/ and stream/ all depend on the same envelope without
either depending on the other.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 5.2: Bootstrap `trade/` — move `trader.go` and `exchange.go`

**Files:**
- Create: `trade/trade.go` (from `trader.go`)
- Create: `trade/exchange.go` (from `exchange.go`)
- Delete: `trader.go`, `exchange.go`
- Modify: `compat.go`

These two move together because `Trader` is declared in `exchange.go` (the larger file) and used in `trader.go` (the placement pipeline).

- [ ] **Step 1: Move both files** following the per-task pattern. `type Trader struct` → `type Client struct`, all `(t *Trader)` receivers → `(c *Client)`, package change to `trade`.

- [ ] **Step 2: The fields `client *HTTPAPI`, `info *Info`** in `Trader` become `client *transport.Client` and `info *info.Client` — adjust imports.

- [ ] **Step 3: Delete originals.**

```bash
rm trader.go exchange.go
```

- [ ] **Step 4: Append aliases.**

```go
import "github.com/Simon-Busch/hyperliquid-go/trade"

// --- trade package alias ---

type Trader = trade.Client
```

- [ ] **Step 5: Add a `New` constructor to `trade/trade.go`** matching this signature:

```go
package trade

import (
	"crypto/ecdsa"

	"github.com/Simon-Busch/hyperliquid-go/info"
	xtransport "github.com/Simon-Busch/hyperliquid-go/internal/transport"
)

// Config holds the wiring required to construct a trade.Client.
type Config struct {
	BaseURL      string
	PrivateKey   *ecdsa.PrivateKey
	Vault        string
	AccountAddr  string
	Dex          string
	Info         *info.Client
	ExpiresAfter *int64
}

// New constructs a Client from cfg. The HTTP transport is built from BaseURL.
func New(cfg Config) *Client {
	c := &Client{
		client:       xtransport.New(cfg.BaseURL, nil),
		privateKey:   cfg.PrivateKey,
		vault:        cfg.Vault,
		accountAddr:  cfg.AccountAddr,
		dex:          cfg.Dex,
		info:         cfg.Info,
		expiresAfter: cfg.ExpiresAfter,
	}
	c.attachSubgroups()
	return c
}
```

Then update `client.go`'s `New(...)` (root) to call `trade.New(trade.Config{...})` instead of constructing `&Trader{...}` directly.

- [ ] **Step 6: Verify + commit.**

```bash
go build ./... && go vet ./... && go test ./...
git add trade/trade.go trade/exchange.go trader.go exchange.go compat.go client.go
git commit -m "$(cat <<'EOF'
refactor(layout): bootstrap trade/ package with trader.go + exchange.go

Trader becomes trade.Client; root keeps type Trader = trade.Client.
client.go's New() now delegates to trade.New(...) for the trader
subhandle.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 5.3–5.13: Move remaining `trader_*.go` and `exchange_*.go` files

| # | Source | Target | Methods moved (sample) | Notes |
|---|--------|--------|------------------------|-------|
| 5.3 | `trader_place.go` | `trade/place.go` | `PlaceALO`, `PlaceIOC`, `PlaceGTC`, `PlaceMarket`, `PlaceTrigger`, `ClosePosition`, `PlaceMany`, `SlippagePrice` | |
| 5.4 | `trader_modify_cancel.go` | `trade/modify_cancel.go` | `Modify`, `ModifyByCloid`, `CancelAll`, `doModify` | `CancelRequest`, `CancelByCloidRequest` types move here. Aliases. |
| 5.5 | `exchange_orders.go` | `trade/exchange_orders.go` | order-execute helpers | All non-cancel exchange routing. |
| 5.6 | `exchange_orders_cancel.go` | `trade/exchange_orders_cancel.go` | cancel helpers | |
| 5.7 | `trader_account.go` | `trade/account.go` | `SetLeverage`, `AdjustMargin`, `SetExpiresAfter`, `ScheduleCancelAll`, `SetReferrer`, `UseBigBlocks`, `ApproveAgent`, `ApproveBuilderFee` | Types `ScheduleCancelResponse`, `SetReferrerResponse`, `ApprovalResponse`, `AgentApprovalResponse`, `Agent` move. Aliases. |
| 5.8 | `trader_transfer.go` | `trade/transfer.go` | usdSend, spotSend, vaultTransfer, … | `TransferResponse` moves. Alias. |
| 5.9 | `trader_withdraw.go` | `trade/withdraw.go` | `Withdraw` | |
| 5.10 | `trader_convert.go` | `trade/convert.go` | `Convert`, `convertSpot`, `convertSpotSize`, `findSpotPair` | |
| 5.11 | `trader_subaccount.go` | `trade/subaccount.go` | `CreateSubAccount`, `RenameSubAccount`, `SubAccountTransfer` | `CreateSubAccountResponse` moves. Alias. |
| 5.12 | `trader_stake.go` | `trade/stake.go` | `Delegate`, `Undelegate`, `Deposit`, `Withdraw` (stake variants) | |
| 5.13 | `trader_multisig.go` | `trade/multisig.go` | `MultiSig`, `ConvertToMultiSigUser` | `MultiSigResponse`, `MultiSigConversionResponse` move. Aliases. |
| 5.14 | `trader_validators.go` | `trade/validators.go` | `CSignerUnjailSelf`, `CSignerJailSelf`, `CSignerInner`, `CValidatorRegister`, `CValidatorChangeProfile`, `CValidatorUnregister` | `ValidatorResponse` moves. Alias. |
| 5.15 | `trader_deploy_perp.go` | `trade/deploy_perp.go` | `PerpDeployRegisterAsset`, `PerpDeploySetOracle` | `PerpDeployResponse`, `TxStatus` move. Aliases. |
| 5.16 | `trader_deploy_spot.go` | `trade/deploy_spot.go` | `SpotDeploy*` family | `SpotDeployResponse` moves. Alias. |
| 5.17 | `trader_outcome.go` | `trade/outcome.go` | (already exists at root per grep — confirm) | HIP-4 userOutcome actions. |

For each task:

- [ ] **Step 1: Create target file** per pattern.
- [ ] **Step 2: Delete source.**
- [ ] **Step 3: Add type aliases for every moved response type.**
- [ ] **Step 4: Update integration test imports for the matching feature** (`tests/integration/<feature>_test.go`).
- [ ] **Step 5: Verify gate.**
- [ ] **Step 6: Commit** with `refactor(layout): move <source>.go to trade/<target>.go`.

### Task 5.18: Phase 5 boundary check

- [ ] **Step 1: Confirm no `trader_*.go` or `exchange*.go` left at root.**

```bash
ls trader_*.go exchange*.go 2>/dev/null && echo "FAIL" || echo "OK"
```

- [ ] **Step 2: Full gate + integration tests.**

```bash
go build ./... && go vet ./... && go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 3: Tag.**

```bash
git tag refactor/phase-5-done
```

---

## Phase 6 — `stream/` package

Pattern: same as Phases 4–5. Receiver rename: `(s *Stream)` → `(c *Client)`. Type rename: `type Stream struct` → `type Client struct`. Root alias: `type Stream = stream.Client`.

### Task 6.1: Bootstrap `stream/` — move `stream.go`

**Files:**
- Create: `stream/stream.go`
- Delete: `stream.go`
- Modify: `compat.go`, `client.go`

- [ ] **Step 1: Apply per-task pattern.** `stream.go` defines the `Stream` struct, `NewStream` constructor, `Connect`, `Subscribe`, `Close`, the read/ping pumps, `dispatch`, `resubscribeAll`, `sendSubscribe`, `sendUnsubscribe`, `sendPing`, `writeJSON`.

- [ ] **Step 2: Delete root file.** `rm stream.go`

- [ ] **Step 3: Append alias.**

```go
import "github.com/Simon-Busch/hyperliquid-go/stream"

// --- stream package alias ---

type Stream = stream.Client

var NewStream = stream.New
```

- [ ] **Step 4: Update `client.go`'s `New(...)`** to call `stream.New(...)`.

- [ ] **Step 5: Verify + commit.**

### Tasks 6.2–6.5

| # | Source | Target | Methods |
|---|--------|--------|---------|
| 6.2 | `stream_post.go` | `stream/post.go` | `Post`, `PostInfo`, `PostAction` |
| 6.3 | `stream_reconnect.go` | `stream/reconnect.go` | `handleDisconnect`, `scheduleReconnect` |
| 6.4 | `stream_subscriptions.go` | `stream/subscriptions.go` | subscription filter constructors (`Trades`, `L2Book`, `UserEvents`, …) and `Subscription` type |
| 6.5 | `ws_types.go` | `stream/ws_types.go` | WS message types (`WSMessage`, etc.) Also move `WsMsg`, `Trade` from root `types.go` if still there. |

For each: per-task pattern → verify → commit.

### Task 6.6: Phase 6 boundary check

- [ ] **Step 1: Confirm cleanup.**

```bash
ls stream*.go ws_types.go 2>/dev/null && echo "FAIL" || echo "OK"
```

- [ ] **Step 2: Full gate + integration tests (include stream tests).**

```bash
go build ./... && go vet ./... && go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 3: Tag.**

```bash
git tag refactor/phase-6-done
```

---

## Phase 7 — Facade rewrite

### Task 7.1: Slim down `client.go`

**Files:**
- Modify: `client.go`

The current `client.go` has 99 lines and reaches into subpackage internals (`info.coinToAsset`, etc.) only via constructors. After Phases 4–6, all of those constructors live in their respective packages. Rewrite `client.go` to be a thin orchestrator.

- [ ] **Step 1: Replace `client.go` contents** with:

```go
// Package hyperliquid provides a Go client library for the Hyperliquid
// exchange API. It bundles the read-only info API, the signed trade
// API, and the websocket stream API behind a single Client; advanced
// callers can also import any subpackage directly.
package hyperliquid

import (
	"errors"

	"github.com/Simon-Busch/hyperliquid-go/info"
	"github.com/Simon-Busch/hyperliquid-go/stream"
	"github.com/Simon-Busch/hyperliquid-go/trade"
)

// Client is the top-level Hyperliquid client.
//
//	c.Info   — read-only queries
//	c.Trade  — signed actions (requires WithPrivateKey)
//	c.Stream — websocket subscriptions and POST requests
type Client struct {
	Info   *info.Client
	Trade  *trade.Client
	Stream *stream.Client
}

// New builds a Client configured by the supplied options. WithMainnet,
// WithTestnet, or WithBaseURL selects the endpoint. Signed actions on
// Trade require WithPrivateKey.
func New(opts ...Option) (*Client, error) {
	cfg := defaultClientConfig()
	for _, o := range opts {
		o(cfg)
	}

	i := info.New(cfg.baseURL, true, cfg.meta, cfg.spotMeta, cfg.perpDexs, cfg.builderDex)

	c := &Client{Info: i}

	if cfg.privateKey != nil {
		c.Trade = trade.New(trade.Config{
			BaseURL:      cfg.baseURL,
			PrivateKey:   cfg.privateKey,
			Vault:        cfg.vault,
			AccountAddr:  cfg.account,
			Dex:          cfg.builderDex,
			Info:         i,
			ExpiresAfter: cfg.expiresAfter,
		})
	}

	if !cfg.skipStream {
		s, err := stream.New(cfg.baseURL)
		if err != nil {
			return nil, err
		}
		s.SetLogger(cfg.logger)
		s.SetMaxReconnectAttempts(cfg.maxReconnectAttempts)
		if cfg.reconnectWait > 0 {
			s.SetReconnectWait(cfg.reconnectWait)
		}
		c.Stream = s
	}

	return c, nil
}

// ErrMissingPrivateKey is returned by Trader methods called on a Client
// constructed without WithPrivateKey.
var ErrMissingPrivateKey = errors.New("hyperliquid: trader requires WithPrivateKey")
```

(The `trade.Config` struct and `stream.SetMaxReconnectAttempts` / `SetReconnectWait` setters must exist — add them as part of this commit if they don't.)

- [ ] **Step 2: `clientConfig` and `defaultClientConfig`** live in this file in the old version. Keep them, but trim fields that are no longer used.

- [ ] **Step 3: Verify gate.**

```bash
go build ./... && go vet ./... && go test ./...
```

- [ ] **Step 4: Commit.**

```bash
git add client.go trade/trade.go stream/stream.go
git commit -m "$(cat <<'EOF'
refactor(layout): slim client.go down to a facade over subpackage constructors

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 8 — Documentation sweep

### Task 8.1: Update `doc.go` and `README.md`

**Files:**
- Modify: `doc.go`
- Modify: `README.md`
- Modify: `docs/*.md` (only where code samples appear)

- [ ] **Step 1: Update `doc.go`** to reflect the new subpackage layout. Mention that `info`, `trade`, `stream`, `signing`, and `types` are public.

- [ ] **Step 2: Update `README.md` code samples** so any direct subpackage imports (`import "github.com/.../hyperliquid-go/info"`) are shown alongside the facade. Quickstart should still use the facade.

- [ ] **Step 3: Update `docs/quickstart.md`, `docs/trading.md`, `docs/info.md`, `docs/stream.md`, `docs/signing.md`** to remove any references to `hl.Trader`, `hl.Info`, `hl.Stream` if they are out of date; replace with the new names if helpful but keep facade examples primary.

- [ ] **Step 4: Verify samples compile.** Extract any non-trivial sample into a scratch `_examples/` file and run `go build ./_examples/...`. (Optional — quick eye-pass is acceptable.)

- [ ] **Step 5: Commit.**

```bash
git add doc.go README.md docs/
git commit -m "$(cat <<'EOF'
refactor(layout): update doc.go and README for subpackage layout

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Phase 9 — Cleanup (breaking-change commit)

### Task 9.1: Tag the pre-rename state

- [ ] **Step 1: Tag.**

```bash
git tag v0.x-pre-rename
git push origin v0.x-pre-rename
```

(Push requires user confirmation per session policy. Ask first.)

### Task 9.2: Delete compat aliases and update all internal callers

**Files:**
- Delete: `compat.go`
- Modify: every root file that still references aliased symbols by their pre-refactor names.

- [ ] **Step 1: Identify all alias names.**

```bash
grep -E "^(type|var|const) " compat.go | awk '{print $2}' | sort -u > /tmp/alias-names.txt
wc -l /tmp/alias-names.txt
```

- [ ] **Step 2: For every name in the file, grep the entire tree** (excluding `compat.go` itself and the subpackages that own the symbol) and rewrite each reference to use the new canonical name.

```bash
while read name; do
  echo "--- $name ---"
  grep -rln "\\b${name}\\b" --include='*.go' . | grep -v compat.go
done < /tmp/alias-names.txt
```

Update each match. Examples:
- `hl.Trader` (in any callsite) → `*trade.Client` + import `trade`
- `hl.NewInfo(...)` → `info.New(...)` + import `info`
- `hl.Buy` → `types.Buy` + import `types`

The integration tests (`tests/integration/*.go`) get the bulk of these edits. Use `goimports -w` to fix imports automatically.

- [ ] **Step 3: Delete `compat.go`.**

```bash
rm compat.go
```

- [ ] **Step 4: Verify gate, including integration tests.**

```bash
go build ./...
go vet ./...
go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 5: Commit.**

```bash
git add -A
git commit -m "$(cat <<'EOF'
refactor(layout!): remove root compat aliases — BREAKING

Final commit of the package reorganization. Callers must update imports:

  hyperliquid.Info     -> info.Client
  hyperliquid.Trader   -> trade.Client
  hyperliquid.Stream   -> stream.Client
  hyperliquid.Side     -> types.Side
  hyperliquid.OrderSpec -> types.OrderSpec
  hyperliquid.NewInfo  -> info.New
  hyperliquid.NewStream-> stream.New
  ... (see types/, info/, trade/, stream/, signing/ for the full surface)

The hyperliquid.New() facade and hyperliquid.Client struct remain — the
common quickstart still works unchanged.

Pre-rename tag: v0.x-pre-rename.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Task 9.3: Final boundary check

- [ ] **Step 1: Confirm root contains only the expected files.**

```bash
ls *.go
```

Expected: `client.go`, `options.go`, `opts.go`, `errors.go`, `doc.go`, `logger.go`, `utils.go`, `validate.go` (until validate moves into trade/, optional follow-up), and the corresponding `_test.go` files. No `info_*.go`, `trader_*.go`, `stream_*.go`, `actions.go`, `signing.go`, `bracket.go`, `side.go`, `result.go`, `orderspec.go`, `asset_class.go`, `outcome_question.go`, `types.go`, `api.go`, `http_api.go`, `ws_types.go`, `exchange*.go`, or `compat.go`.

- [ ] **Step 2: Full gate + integration tests one final time.**

```bash
go build ./... && go vet ./... && go test ./...
HYPERLIQUID_RUN_INTEGRATION=1 go test ./tests/integration/... -timeout 10m
```

- [ ] **Step 3: Tag the completed refactor.**

```bash
git tag refactor/package-layout-done
```

- [ ] **Step 4: Push branch + tags (ask user first per session policy).**

```bash
git push -u origin refactor/package-layout
git push origin refactor/phase-1-done refactor/phase-3-done refactor/phase-4-done refactor/phase-5-done refactor/phase-6-done v0.x-pre-rename refactor/package-layout-done
```

- [ ] **Step 5: Open a PR (ask user first).**

```bash
gh pr create --title "refactor: reorganize package into domain subpackages" --body "$(cat <<'EOF'
## Summary
- Split flat root hyperliquid package into types/, info/, trade/, stream/, signing/ subpackages
- Generic HTTP envelope and deterministic msgpack encoder moved to internal/{transport,msgpack}
- Top-level Client facade preserved; advanced users can import subpackages directly
- BREAKING: callers must update non-facade imports (see commit message of refactor(layout!) commit)

Spec: docs/superpowers/specs/2026-05-24-package-reorganization-design.md
Plan: docs/superpowers/plans/2026-05-24-package-reorganization.md

## Test plan
- [x] go build ./... green on every commit
- [x] go test ./... green on every commit
- [x] go test ./tests/integration/... green at every phase boundary
- [x] Signature fixture tests confirm msgpack determinism preserved
- [ ] Manual smoke: README quickstart compiles and runs against testnet

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-review checklist

- **Spec coverage:** every target-layout entry, every phase, and every risk in the spec has a task in this plan.
- **Compat strategy:** single `compat.go`, grows phase by phase (Task 1.1 creates it; Tasks 1.2–6.5 append; Task 9.2 deletes).
- **Integration tests:** updated phase-by-phase (per spec) — each Phase-4/5/6 task includes Step 4 "Update integration tests in `tests/integration/`".
- **Sequential within a phase:** verified — no phase-internal task runs in parallel.
- **Subagent prompt shape:** each Task is self-contained (file paths, exact commands, exact compat content, exact commit message) — ready to dispatch to a fresh subagent per task.
- **Signature determinism:** Tasks 2.2 and 3.2 include explicit fixture-test runs as gates.
- **Known deferrals tracked:** Tasks 1.7 (Bracket) and 1.9 (outcome_question) are no-ops in Phase 1 with explicit handoff to Phase 3/4 respectively.
