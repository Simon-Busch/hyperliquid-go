# Package Reorganization — Design

**Status:** Approved, ready for implementation planning
**Branch:** `refactor/package-layout` (off `refactor/ux-api`)
**Scope:** Reshape the public Go package layout of `github.com/Simon-Busch/hyperliquid-go` from a single flat root package (~75 files) into domain-bounded subpackages.

## Problem

The repo currently holds ~75 `.go` files in the root `hyperliquid` package, grouped only by filename prefix (`info_*`, `trader_*`, `stream_*`, `exchange_*`). Symptoms:

- `types.go` is 22 KB and mixes truly shared types (`Side`, `Tif`) with domain-specific request/response shapes.
- New contributors cannot tell at a glance what is public surface vs internal plumbing.
- Test files sit next to source with no separation between read, write, and stream concerns.
- Cross-cutting concerns (signing, msgpack encoding, HTTP envelope) are interleaved with verb implementations.

## Goals

1. Each public subpackage answers one user question: *query* (`info`), *trade* (`trade`), *stream* (`stream`), *sign* (`signing`).
2. Files are small and focused — one verb cluster per file, no 20 KB grab-bags.
3. The top-level `hyperliquid.New()` facade continues to work — power users can also import any subpackage directly.
4. Migration is gradual: every commit on the refactor branch compiles and `go test ./...` passes.
5. Public API breakage is acceptable (consumers update imports once at the end).

## Non-Goals

- Behavioural changes. No new features, no bug fixes, no logic refactors. Move and rename only.
- Performance work.
- Documentation rewrites beyond updating import paths in `README.md` and `doc.go`.
- Renaming the module path itself.

## Target Layout

```
hyperliquid-go/
├── client.go              # New(), Client facade
├── options.go             # WithMainnet / WithPrivateKey / WithLogger / WithBaseURL …
├── compat.go              # Temporary type aliases during migration; deleted in phase 9
├── doc.go
├── errors.go              # APIError, ValidationError, sentinel errors
│
├── types/                 # Shared public domain types
│   # Side, Tif, Grouping, AssetClass, Cloid, OrderSpec, Bracket,
│   # Result family, OutcomeQuestion
│
├── info/                  # Read-only surface
│   ├── info.go            # info.Client + New(...)
│   ├── account.go
│   ├── market.go
│   ├── meta.go
│   ├── orders.go
│   ├── funding.go
│   └── staking.go
│
├── trade/                 # Signed-action surface
│   ├── trade.go           # trade.Client + New(...)
│   ├── place.go
│   ├── cancel.go          # cancel + modify
│   ├── account.go         # leverage, margin, approve-agent
│   ├── transfer.go        # transfer + withdraw
│   ├── convert.go
│   ├── subaccount.go
│   ├── stake.go
│   ├── multisig.go
│   ├── validators.go
│   ├── deploy_perp.go     # HIP-3
│   └── deploy_spot.go
│
├── stream/                # Websocket surface
│   ├── stream.go          # stream.Client + New(...), Subscribe()
│   ├── subscriptions.go
│   ├── post.go
│   ├── reconnect.go
│   └── ws_types.go
│
├── signing/               # Public signing helpers
│   ├── signing.go         # SignL1Action, SignUserSigned, SignatureResult
│   └── actions.go         # Action structs whose field order matters for signatures
│
├── internal/
│   ├── transport/         # HTTP plumbing
│   ├── wire/              # JSON envelope, fastjson pool
│   ├── eip712/            # Typed-data helpers
│   └── msgpack/           # Deterministic action encoding (new)
│
└── tests/integration/     # Unchanged location; imports updated in lockstep
```

### Dependency direction

```
internal/{eip712, msgpack, wire, transport}
        ↑
      types/
        ↑
      signing/
        ↑
   ┌────┼────┐
 info/ trade/ stream/
        ↑
   client.go (facade)
```

No cycles. Every package depends only on packages strictly below it.

### Type placement rules

A type goes in `types/` if and only if it is referenced from more than one of `info/`, `trade/`, `stream/`, or from a consumer's typical call site (e.g. `Side`, `OrderSpec`). Otherwise it lives next to its consumer:

- `info.UserState`, `info.L2Book` → `info/`
- `trade.PlaceRequest`, internal order envelope → `trade/`
- `stream.Subscription`, WS message shapes → `stream/`
- Action structs whose field order is signature-critical → `signing/`

### Facade contract

`hyperliquid.Client` is unchanged in shape:
```go
type Client struct {
    Info   *info.Client
    Trade  *trade.Client
    Stream *stream.Client
}
```
`hyperliquid.New(opts...)` parses options and forwards relevant subsets to `info.New`, `trade.New`, `stream.New`. Each subpackage also exposes its own `New` for advanced callers.

## Migration Plan

### Branching & invariants

- Branch: `refactor/package-layout` cut from `refactor/ux-api`.
- Every commit on the branch must satisfy `go build ./... && go vet ./... && go test ./...`.
- Integration tests under `tests/integration/` run on phase boundaries (slow; gated by env var).
- Compat layer: a single root `compat.go` file grows phase by phase with `type X = subpkg.X` aliases. Deleted whole in phase 9.

### Phases

| # | Phase | Approx commits | Output |
|---|-------|----------------|--------|
| 0 | Baseline | 1 | Branch created, baseline `go test` output recorded in commit message. |
| 1 | `types/` package | ~7 | One commit per type cluster: Side; Tif; Grouping + OrderSpec; Result family; Bracket; AssetClass + Cloid; OutcomeQuestion. Each adds matching aliases to `compat.go`. |
| 2 | `internal/msgpack/` | 2 | Extract deterministic msgpack encoder from `actions.go`. Public action structs remain at root for now. |
| 3 | `signing/` | 3 | Move `signing.go` → `signing/signing.go`. Move signature-bearing action structs → `signing/actions.go`. Root re-exports `SignatureResult`. |
| 4 | `info/` | ~8 | One commit per `info_*.go` → `info/<name>.go`. Rename receiver `*Info` → `*info.Client`. `compat.go` adds `type Info = info.Client`. Integration tests touching moved methods are updated in the same commit. |
| 5 | `trade/` | ~12 | One commit per `trader_*.go` → `trade/<name>.go`. Generic HTTP envelope (`api.go`, `http_api.go`) moves into `internal/transport/`. Trade-specific exchange dispatch (`exchange.go`, `exchange_orders*.go`) moves into `trade/`. Integration tests updated alongside. |
| 6 | `stream/` | ~5 | `stream_*.go` + `ws_types.go` → `stream/`. Receiver rename to `*stream.Client`. Integration tests updated alongside. |
| 7 | Facade rewrite | 1 | Root `client.go` becomes thin facade; `New()` wires subpackage constructors. |
| 8 | Docs | 1 | Update `README.md`, `doc.go`, `docs/*.md` examples to new import paths. |
| 9 | Cleanup | 1 | Delete `compat.go`. Tag `v0.x-pre-rename` on the commit immediately before. This is the breaking-change commit. |

### Subagent execution

- **Phases are sequential.** Each depends on packages introduced by prior phases.
- **Within a phase, work is sequential** (per user preference for safety). The orchestrator dispatches one subagent per atomic step (one file move + commit), runs the verification gate (`go build`, `go vet`, `go test`), and proceeds only on green.
- Subagents receive a self-contained prompt: which file to move, target path, receiver rename if any, compat alias to add, integration test files to update in the same commit, exact commit message format.

### Verification gate (run after every commit)

```bash
go build ./...
go vet ./...
go test ./...
```

Phase boundaries (0, 1, 3, 4, 5, 6, 7, 9) additionally run:

```bash
go test ./tests/integration/...
```

(Integration tests respect their existing env-var gates; absent credentials, they skip rather than fail.)

### Rollback

Each phase is a contiguous range of commits on a dedicated branch. If a phase produces a regression that escapes the gate, `git reset --hard <phase-start>` restores the prior phase. Because compat aliases insulate consumers up to phase 9, no external rollback coordination is needed mid-refactor.

## Risks

- **Hidden cross-file dependencies in `types.go`.** Mitigation: phase 1 commits are tiny and individually buildable; the compiler catches any miss.
- **Action struct field ordering is load-bearing for signatures.** Mitigation: phase 2/3 includes a signature-fixture regression test run (`signing_test.go`, `fixtures_signing_test.go`) before and after the move.
- **Integration tests reference methods on `*Info`/`*Trader` that now belong to subpackage types.** Mitigation: compat aliases (`type Info = info.Client`) keep them compiling; per-phase integration-test updates catch any alias gap immediately.
- **`exchange.go` and `http_api.go` straddle trade and info concerns.** Mitigation: during phase 5 the generic HTTP envelope (request/response decoder, retry, error mapping) moves to `internal/transport/`. Trade-specific exchange dispatch stays in `trade/`.

## Out of scope (explicit)

- Renaming methods or changing signatures beyond receiver type.
- Splitting `trade.Client` further (e.g. separate `trade.Account`, `trade.Stake`). Files are organized; the type stays unified.
- Reworking the WebSocket reconnection logic.
- Touching test fixtures under `testfixtures/`.

## Success criteria

1. `go build ./... && go vet ./... && go test ./...` green on every commit of the refactor branch.
2. Integration tests pass at every phase boundary.
3. Final commit deletes `compat.go`; root package contains only `client.go`, `options.go`, `errors.go`, `doc.go`.
4. No file in the new layout exceeds ~400 lines without a clear reason.
5. `README.md` examples compile against the new layout.
