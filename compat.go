package hyperliquid

// Compat aliases re-export types that have moved into subpackages during
// the refactor described in docs/superpowers/specs/2026-05-24-package-reorganization-design.md.
// This file grows phase by phase and is deleted whole in the final
// cleanup phase. Do NOT add new symbols here outside that refactor.

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

// --- side.go unexported re-declarations (transitional) ---
//
// The lowercase TIF wire constants stay unexported inside types/, so the
// root package re-declares them here against the aliased TIF type. They
// disappear with this whole file in the final cleanup phase.

const (
	tifALO TIF = "Alo"
	tifIOC TIF = "Ioc"
	tifGTC TIF = "Gtc"
)

// --- types.go order-type aliases ---

type OrderType = types.OrderType
type LimitOrderType = types.LimitOrderType
type TriggerOrderType = types.TriggerOrderType
type BuilderInfo = types.BuilderInfo
type OrderTypeWire = types.OrderTypeWire
type LimitOrderTypeWire = types.LimitOrderTypeWire
type TriggerOrderTypeWire = types.TriggerOrderTypeWire
type Cloid = types.Cloid

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

// --- orderspec.go alias ---

type OrderSpec = types.OrderSpec

// --- result.go aliases ---

type Result            = types.Result
type BatchResult       = types.BatchResult
type CancelResult      = types.CancelResult
type BatchCancelResult = types.BatchCancelResult
