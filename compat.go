package hyperliquid

// Compat aliases re-export types that have moved into subpackages during
// the refactor described in docs/superpowers/specs/2026-05-24-package-reorganization-design.md.
// This file grows phase by phase and is deleted whole in the final
// cleanup phase. Do NOT add new symbols here outside that refactor.

import (
	"github.com/Simon-Busch/hyperliquid-go/info"
	"github.com/Simon-Busch/hyperliquid-go/internal/transport"
	"github.com/Simon-Busch/hyperliquid-go/signing"
	"github.com/Simon-Busch/hyperliquid-go/stream"
	"github.com/Simon-Busch/hyperliquid-go/trade"
	"github.com/Simon-Busch/hyperliquid-go/types"
)

// --- stream package aliases ---
//
// The websocket surface — Stream (renamed Client inside stream/),
// SubscriptionFilter, WSMessage, the WS POST envelope types, plus the
// WsMsg / Trade message types — moved into stream/ in the Phase-6
// extraction. Root callers and integration tests keep working through
// these aliases. The Logger interface lives in stream/ now too because
// stream/ is the only consumer.

type Stream = stream.Client
type Subscription = stream.Subscription
type SubscriptionFilter = stream.SubscriptionFilter
type WSMessage = stream.WSMessage
type WsCommand = stream.WsCommand
type WsRequest = stream.WsRequest
type WsResponse = stream.WsResponse
type WsPostRequest = stream.WsPostRequest
type WsPostResponseData = stream.WsPostResponseData
type WsMsg = stream.WsMsg
type Trade = stream.Trade
type Logger = stream.Logger

var NewStream = stream.New

// Stream subscription-filter constructors re-exported.
var (
	Trades          = stream.Trades
	Book            = stream.Book
	BBO             = stream.BBO
	ActiveAssetCtx  = stream.ActiveAssetCtx
	Candles         = stream.Candles
	AllMids         = stream.AllMids
	AllMidsOn       = stream.AllMidsOn
	UserEvents      = stream.UserEvents
	UserFills       = stream.UserFills
	OrderUpdates    = stream.OrderUpdates
	UserFundings    = stream.UserFundings
	UserLedger      = stream.UserLedger
	WebData         = stream.WebData
	Notifications   = stream.Notifications
	ActiveAssetData = stream.ActiveAssetData
	UserTwapFills   = stream.UserTwapFills
	UserTwapHistory = stream.UserTwapHistory
)

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

type Result = types.Result
type BatchResult = types.BatchResult
type CancelResult = types.CancelResult
type BatchCancelResult = types.BatchCancelResult

// --- asset_class.go aliases ---

type AssetClass = types.AssetClass

const (
	AssetClassPerp        = types.AssetClassPerp
	AssetClassSpot        = types.AssetClassSpot
	AssetClassBuilderPerp = types.AssetClassBuilderPerp
	AssetClassOutcome     = types.AssetClassOutcome
)

var ClassifyAsset = types.ClassifyAsset

// --- api.go mixed-array aliases ---

type MixedValue = types.MixedValue
type MixedArray = types.MixedArray

// --- errors.go alias (moved to types/) ---

type ValidationError = types.ValidationError

// --- signing.go aliases ---

type SignatureResult = signing.SignatureResult

var (
	SignL1Action         = signing.SignL1Action
	SignUserSignedAction = signing.SignUserSignedAction
	FloatToUsdInt        = signing.FloatToUsdInt
	GetTimestampMs       = signing.GetTimestampMs
)

// --- actions.go aliases ---

type (
	CancelOrderWire              = signing.CancelOrderWire
	CancelAction                 = signing.CancelAction
	CancelByCloidWire            = signing.CancelByCloidWire
	CancelByCloidAction          = signing.CancelByCloidAction
	UsdClassTransferAction       = signing.UsdClassTransferAction
	SpotTransferAction           = signing.SpotTransferAction
	UsdTransferAction            = signing.UsdTransferAction
	SubAccountTransferAction     = signing.SubAccountTransferAction
	VaultUsdTransferAction       = signing.VaultUsdTransferAction
	UpdateLeverageAction         = signing.UpdateLeverageAction
	UpdateIsolatedMarginAction   = signing.UpdateIsolatedMarginAction
	OrderWire                    = signing.OrderWire
	OrderAction                  = signing.OrderAction
	ModifyAction                 = signing.ModifyAction
	BatchModifyAction            = signing.BatchModifyAction
	PerpDexClassTransferAction   = signing.PerpDexClassTransferAction
	SubAccountSpotTransferAction = signing.SubAccountSpotTransferAction
	ScheduleCancelAction         = signing.ScheduleCancelAction
	SetReferrerAction            = signing.SetReferrerAction
	CreateSubAccountAction       = signing.CreateSubAccountAction
	UseBigBlocksAction           = signing.UseBigBlocksAction
	TokenDelegateAction          = signing.TokenDelegateAction
	WithdrawFromBridgeAction     = signing.WithdrawFromBridgeAction
	ApproveAgentAction           = signing.ApproveAgentAction
	ApproveBuilderFeeAction      = signing.ApproveBuilderFeeAction
	ConvertToMultiSigUserAction  = signing.ConvertToMultiSigUserAction
	MultiSigAction               = signing.MultiSigAction
	TWAPOrderAction              = signing.TWAPOrderAction
	TWAPOrderWire                = signing.TWAPOrderWire
	TWAPCancelAction             = signing.TWAPCancelAction
	ReserveRequestWeightAction   = signing.ReserveRequestWeightAction
	SplitOutcomeWire             = signing.SplitOutcomeWire
	MergeOutcomeWire             = signing.MergeOutcomeWire
	MergeQuestionWire            = signing.MergeQuestionWire
	NegateOutcomeWire            = signing.NegateOutcomeWire
	SplitOutcomeAction           = signing.SplitOutcomeAction
	MergeOutcomeAction           = signing.MergeOutcomeAction
	MergeQuestionAction          = signing.MergeQuestionAction
	NegateOutcomeAction          = signing.NegateOutcomeAction
)

// --- transport / URL aliases (moved from http_api.go to internal/transport) ---

const (
	MainnetAPIURL = transport.MainnetAPIURL
	TestnetAPIURL = transport.TestnetAPIURL
	LocalAPIURL   = transport.LocalAPIURL
)

// APIError is the wire-error envelope returned by /info and /exchange.
// Aliased through transport so callers (including external code reaching
// in via hl.APIError) keep working after the move.
type APIError = transport.APIError

// HTTPAPI is the low-level HTTP client wrapper. It used to live at the
// root as the unexported httpAPI; the rename + move to internal/transport
// happened atomically with the info/ extraction.
type HTTPAPI = transport.Client

// NewHTTPAPI mirrors the legacy newHTTPAPI constructor.
var NewHTTPAPI = transport.New

// --- info package aliases ---

type Info = info.Client
type InfoStakeGroup = info.StakeGroup

var NewInfo = info.New

// Response-type aliases — these move with their methods into info/ in the
// Phase-4 commit but root callers (compat-shim coverage and external
// downstream) still reach them through these aliases.
type (
	Meta                        = info.Meta
	AssetInfo                   = info.AssetInfo
	MarginTable                 = info.MarginTable
	MarginTier                  = info.MarginTier
	SpotMeta                    = info.SpotMeta
	SpotAssetInfo               = info.SpotAssetInfo
	SpotTokenInfo               = info.SpotTokenInfo
	EvmContract                 = info.EvmContract
	OutcomeMeta                 = info.OutcomeMeta
	OutcomeInfo                 = info.OutcomeInfo
	OutcomeSideSpec             = info.OutcomeSideSpec
	Question                    = info.Question
	SpotAssetCtx                = info.SpotAssetCtx
	AssetCtx                    = info.AssetCtx
	MetaAndAssetCtxsResponse    = info.MetaAndAssetCtxsResponse
	MetaAndAssetCtxsRawResponse = info.MetaAndAssetCtxsRawResponse
	PerpDex                     = info.PerpDex
	PerpDexLimits               = info.PerpDexLimits
	PerpDexStatus               = info.PerpDexStatus
	PerpDexSchemaInput          = info.PerpDexSchemaInput
	PerpDeployAuctionStatus     = info.PerpDeployAuctionStatus
	L2Book                      = info.L2Book
	Level                       = info.Level
	Candle                      = info.Candle
	AssetPosition               = info.AssetPosition
	Position                    = info.Position
	Leverage                    = info.Leverage
	UserState                   = info.UserState
	MarginSummary               = info.MarginSummary
	UserFees                    = info.UserFees
	UserVolume                  = info.UserVolume
	FeeSchedule                 = info.FeeSchedule
	Tiers                       = info.Tiers
	MMTier                      = info.MMTier
	VIPTier                     = info.VIPTier
	OpenOrder                   = info.OpenOrder
	FrontendOpenOrder           = info.FrontendOpenOrder
	Fill                        = info.Fill
	ReferralState               = info.ReferralState
	ReferredBy                  = info.ReferredBy
	ReferrerState               = info.ReferrerState
	ReferrerData                = info.ReferrerData
	ReferralMember              = info.ReferralMember
	FundingHistory              = info.FundingHistory
	UserFundingHistory          = info.UserFundingHistory
	StakingSummary              = info.StakingSummary
	StakingDelegation           = info.StakingDelegation
	StakingReward               = info.StakingReward
	SubAccount                  = info.SubAccount
	MultiSigSigner              = info.MultiSigSigner
	SpotBalance                 = info.SpotBalance
	SpotClearinghouseState      = info.SpotClearinghouseState
	AssetMeta                   = info.AssetMeta
	OrderStatusResponse         = info.OrderStatusResponse
	Bucket                      = info.Bucket
)

var ParseOutcomeDescription = info.ParseOutcomeDescription

// --- trade package aliases ---

// Trader is the historical name for the signed-action client; it now lives
// in the trade subpackage as trade.Client.
type Trader = trade.Client

// NewTrader mirrors the trade.New constructor. Retained for parity with
// the pre-refactor public surface; root code uses trade.New directly.
var NewTrader = trade.New

// PlaceOpt is the option-function type consumed by placement verbs.
type PlaceOpt = trade.PlaceOpt

// Placement helper constructors (re-exported from trade/).
var (
	ALO     = trade.ALO
	IOC     = trade.IOC
	GTC     = trade.GTC
	Market  = trade.Market
	Trigger = trade.Trigger
)

// Option functions for placement (re-exported from trade/).
var (
	WithTakeProfit = trade.WithTakeProfit
	WithStopLoss   = trade.WithStopLoss
	WithBracket    = trade.WithBracket
	WithReduceOnly = trade.WithReduceOnly
	WithCloid      = trade.WithCloid
	WithBuilder    = trade.WithBuilder
	WithSlippage   = trade.WithSlippage
	WithSize       = trade.WithSize
	WithLimit      = trade.WithLimit
	AsMarket       = trade.AsMarket
	AsLimit        = trade.AsLimit
	WithTPSize     = trade.WithTPSize
	WithSLSize     = trade.WithSLSize
	WithTPCloid    = trade.WithTPCloid
	WithSLCloid    = trade.WithSLCloid
	SkipValidation = trade.SkipValidation
)

// PriceToWire re-exports trade.PriceToWire so legacy tests keep compiling.
var PriceToWire = trade.PriceToWire

// formatPriceToTickSize re-exports trade.FormatPriceToTickSize for the
// legacy asset_class_test at root.
var formatPriceToTickSize = trade.FormatPriceToTickSize

// Trade-package response/request type aliases.
type (
	CancelRequest              = trade.CancelRequest
	CancelByCloidRequest       = trade.CancelByCloidRequest
	BulkOrderResponse          = trade.BulkOrderResponse
	CancelResponse             = trade.CancelResponse
	BulkCancelResponse         = trade.BulkCancelResponse
	ModifyResponse             = trade.ModifyResponse
	TransferResponse           = trade.TransferResponse
	ApprovalResponse           = trade.ApprovalResponse
	AgentApprovalResponse      = trade.AgentApprovalResponse
	CreateSubAccountResponse   = trade.CreateSubAccountResponse
	SetReferrerResponse        = trade.SetReferrerResponse
	ScheduleCancelResponse     = trade.ScheduleCancelResponse
	MultiSigConversionResponse = trade.MultiSigConversionResponse
	MultiSigResponse           = trade.MultiSigResponse
	SpotDeployResponse         = trade.SpotDeployResponse
	ValidatorResponse          = trade.ValidatorResponse
	PerpDeployResponse         = trade.PerpDeployResponse
	TxStatus                   = trade.TxStatus
	Agent                      = trade.Agent
	DefaultResponse            = trade.DefaultResponse
	OrderResponse              = trade.OrderResponse
	OrderStatus                = trade.OrderStatus
	CreateOrderRequest         = trade.CreateOrderRequest
	CancelOrderResponse        = trade.CancelOrderResponse
)
