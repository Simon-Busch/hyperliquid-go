package hyperliquid

import "encoding/json"

// Grouping is the order-grouping discriminator used by /exchange order
// actions. Distinct groupings allow TP/SL trigger legs to attach to a
// parent or to an existing position.
type Grouping string

const (
	// GroupingNA is the default (no grouping).
	GroupingNA Grouping = "na"
	// GroupingNormalTpsl groups TP/SL legs with their parent order.
	GroupingNormalTpsl Grouping = "normalTpsl"
	// GroupingPositionTpls binds TP/SL legs to an existing position.
	GroupingPositionTpls Grouping = "positionTpsl"
)

// Constants for default values
const (
	DefaultSlippage = 0.05 // 5%
)

// Order Time-in-Force constants
const (
	TifAlo = "Alo" // Add Liquidity Only
	TifIoc = "Ioc" // Immediate or Cancel
	TifGtc = "Gtc" // Good Till Cancel
)

// AssetInfo is the per-asset row inside a Meta universe.
type AssetInfo struct {
	Name          string `json:"name"`
	SzDecimals    int    `json:"szDecimals"`
	MaxLeverage   int    `json:"maxLeverage"`
	MarginTableId int    `json:"marginTableId"`
	OnlyIsolated  bool   `json:"onlyIsolated"`
	IsDelisted    bool   `json:"isDelisted"`
}

// Meta is the perp universe metadata returned by /info {"type":"meta"}.
type Meta struct {
	Universe        []AssetInfo   `json:"universe"`
	MarginTables    []MarginTable `json:"marginTables"`
	CollateralToken int           `json:"collateralToken"`
}

// SpotAssetInfo is one entry in SpotMeta.Universe.
type SpotAssetInfo struct {
	Name        string `json:"name"`
	Tokens      []int  `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// EvmContract describes the EVM-side companion contract for a spot token.
type EvmContract struct {
	Address             string `json:"address"`
	EvmExtraWeiDecimals int    `json:"evm_extra_wei_decimals"`
}

// SpotTokenInfo describes a single spot token in the spot universe.
type SpotTokenInfo struct {
	Name        string       `json:"name"`
	SzDecimals  int          `json:"szDecimals"`
	WeiDecimals int          `json:"weiDecimals"`
	Index       int          `json:"index"`
	TokenID     string       `json:"tokenId"`
	IsCanonical bool         `json:"isCanonical"`
	EvmContract *EvmContract `json:"evmContract"`
	FullName    *string      `json:"fullName"`
}

// SpotMeta is the spot universe metadata returned by /info
// {"type":"spotMeta"}.
type SpotMeta struct {
	Universe []SpotAssetInfo `json:"universe"`
	Tokens   []SpotTokenInfo `json:"tokens"`
}

// OutcomeSideSpec describes one side (YES or NO) of a binary HIP-4 outcome.
type OutcomeSideSpec struct {
	Name string `json:"name"` // "Yes" or "No"
}

// OutcomeInfo describes a single binary prediction market.
//
// The Description field is a pipe-delimited string of key:value pairs,
// e.g. "class:priceBinary|underlying:BTC|expiry:20260507-0600|targetPrice:81287|period:1d".
type OutcomeInfo struct {
	Outcome     int               `json:"outcome"`     // numeric outcome ID
	Name        string            `json:"name"`        // e.g. "Recurring"
	Description string            `json:"description"` // structured metadata (see above)
	SideSpecs   []OutcomeSideSpec `json:"sideSpecs"`   // [YES, NO] in that order
}

// Question groups several binary outcomes into a multi-bucket market.
// A price-bucket question over thresholds [T1, T2, ..., Tn] is split
// into n+1 child outcomes referenced by NamedOutcomes, each tradable
// on its own YES/NO sides. FallbackOutcome catches edge cases the
// named buckets do not cover (e.g. an oracle outage). The Description
// is the same pipe-delimited "k:v|k:v" format as OutcomeInfo and
// usually carries class, underlying, expiry, priceThresholds, period.
type Question struct {
	Question             int    `json:"question"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	FallbackOutcome      int    `json:"fallbackOutcome"`
	NamedOutcomes        []int  `json:"namedOutcomes"`
	SettledNamedOutcomes []int  `json:"settledNamedOutcomes"`
}

// OutcomeMeta is the response to POST /info {"type":"outcomeMeta"}.
// Outcomes lists every tradable binary YES/NO market; Questions groups
// them — multi-bucket markets (e.g. BTC price ranges) appear here.
type OutcomeMeta struct {
	Outcomes  []OutcomeInfo `json:"outcomes"`
	Questions []Question    `json:"questions"`
}

// SpotAssetCtx is the spot asset context payload returned alongside
// SpotMeta in spotMetaAndAssetCtxs.
type SpotAssetCtx struct {
	DayNtlVlm         string  `json:"dayNtlVlm"`
	MarkPx            string  `json:"markPx"`
	MidPx             *string `json:"midPx"`
	PrevDayPx         string  `json:"prevDayPx"`
	CirculatingSupply string  `json:"circulatingSupply"`
	Coin              string  `json:"coin"`
}

// AssetCtx represents perpetual asset context data including mark price, funding, open interest, etc.
type AssetCtx struct {
	DayNtlVlm    string   `json:"dayNtlVlm"`
	Funding      string   `json:"funding"`
	ImpactPxs    []string `json:"impactPxs"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx"`
	OpenInterest string   `json:"openInterest"`
	OraclePx     string   `json:"oraclePx"`
	Premium      string   `json:"premium"`
	PrevDayPx    string   `json:"prevDayPx"`
}

// MarginTier represents a single margin tier
type MarginTier struct {
	LowerBound  string `json:"lowerBound"`
	MaxLeverage int    `json:"maxLeverage"`
}

// MarginTable represents a margin table with description and tiers
type MarginTable struct {
	ID          int
	Description string       `json:"description"`
	MarginTiers []MarginTier `json:"marginTiers"`
}

// MetaAndAssetCtxsResponse represents the response from the metaAndAssetCtxs endpoint
// The API returns an array with two elements: [meta, assetCtxs]
type MetaAndAssetCtxsResponse struct {
	Meta      Meta       `json:"universe"`
	AssetCtxs []AssetCtx `json:"assetCtxs"`
}

// MetaAndAssetCtxsRawResponse represents the raw array response from the API
type MetaAndAssetCtxsRawResponse [2]interface{}

// WsMsg represents a WebSocket message with a channel and data payload.
type WsMsg struct {
	Channel string         `json:"channel"`
	Data    map[string]any `json:"data"`
}

// CancelRequest names a single order to cancel by exchange oid.
type CancelRequest struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

// CancelByCloidRequest names a single order to cancel by client order id.
type CancelByCloidRequest struct {
	Coin  string `json:"coin"`
	Cloid string `json:"cloid"`
}

// PerpDexSchemaInput is the per-dex registration payload for HIP-3 perp
// deploys.
type PerpDexSchemaInput struct {
	FullName        string  `json:"fullName"`
	CollateralToken int     `json:"collateralToken"`
	OracleUpdater   *string `json:"oracleUpdater"`
}

// L2Book is the L2 order book snapshot returned by /info {"type":"l2Book"}.
type L2Book struct {
	Coin   string    `json:"coin"`
	Levels [][]Level `json:"levels"`
	Time   int64     `json:"time"`
}

// Level is one price level inside an L2Book.
type Level struct {
	N  int     `json:"n"`
	Px float64 `json:"px,string"`
	Sz float64 `json:"sz,string"`
}

// AssetPosition is one entry of UserState.AssetPositions.
type AssetPosition struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
}

// Position is the per-asset position snapshot inside a UserState.
type Position struct {
	Coin           string   `json:"coin"`
	EntryPx        *string  `json:"entryPx"`
	Leverage       Leverage `json:"leverage"`
	LiquidationPx  *string  `json:"liquidationPx"`
	MarginUsed     string   `json:"marginUsed"`
	PositionValue  string   `json:"positionValue"`
	ReturnOnEquity string   `json:"returnOnEquity"`
	Szi            string   `json:"szi"`
	UnrealizedPnl  string   `json:"unrealizedPnl"`
}

// Leverage describes the leverage configuration on a position
// (Cross/Isolated, integer multiplier, raw USD where applicable).
type Leverage struct {
	Type   string  `json:"type"`
	Value  int     `json:"value"`
	RawUsd *string `json:"rawUsd,omitempty"`
}

// UserState is the perpetuals account summary returned by
// /info {"type":"clearinghouseState"}.
type UserState struct {
	AssetPositions     []AssetPosition `json:"assetPositions"`
	CrossMarginSummary MarginSummary   `json:"crossMarginSummary"`
	MarginSummary      MarginSummary   `json:"marginSummary"`
	Withdrawable       string          `json:"withdrawable"`
}

// MarginSummary summarises an account's margin usage.
type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUsd     string `json:"totalRawUsd"`
}

// OpenOrder is the slim open-orders row returned by /info
// {"type":"openOrders"}.
type OpenOrder struct {
	Coin      string  `json:"coin"`
	LimitPx   float64 `json:"limitPx,string"`
	Oid       int64   `json:"oid"`
	Side      string  `json:"side"`
	Size      float64 `json:"sz,string"`
	Timestamp int64   `json:"timestamp"`
}

// FrontendOpenOrder represents the detailed order information returned by frontendOpenOrders
type FrontendOpenOrder struct {
	Coin             string  `json:"coin"`
	IsPositionTpsl   bool    `json:"isPositionTpsl"`
	IsTrigger        bool    `json:"isTrigger"`
	LimitPx          float64 `json:"limitPx,string"`
	Oid              int64   `json:"oid"`
	OrderType        string  `json:"orderType"`
	OrigSz           float64 `json:"origSz,string"`
	ReduceOnly       bool    `json:"reduceOnly"`
	Side             string  `json:"side"`
	Size             float64 `json:"sz,string"`
	Timestamp        int64   `json:"timestamp"`
	TriggerCondition string  `json:"triggerCondition"`
	TriggerPx        float64 `json:"triggerPx,string"`
}

// Fill is a single trade execution row in the userFills feed.
type Fill struct {
	ClosedPnl     string `json:"closedPnl"`
	Coin          string `json:"coin"`
	Crossed       bool   `json:"crossed"`
	Dir           string `json:"dir"`
	Hash          string `json:"hash"`
	Oid           int64  `json:"oid"`
	Price         string `json:"px"`
	Side          string `json:"side"`
	StartPosition string `json:"startPosition"`
	Size          string `json:"sz"`
	Time          int64  `json:"time"`
	Fee           string `json:"fee"`
	FeeToken      string `json:"feeToken"`
}

// FundingHistory is one row in the per-coin funding rate history.
type FundingHistory struct {
	Coin        string `json:"coin"`
	FundingRate string `json:"fundingRate"`
	Premium     string `json:"premium"`
	Time        int64  `json:"time"`
}

// UserFundingHistory is one row in the per-user funding payment history.
type UserFundingHistory struct {
	User      string `json:"user"`
	Type      string `json:"type"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// Candle is a single OHLC bar.
type Candle struct {
	Timestamp int64  `json:"T"`
	Close     string `json:"c"`
	High      string `json:"h"`
	Interval  string `json:"i"`
	Low       string `json:"l"`
	Number    int    `json:"n"`
	Open      string `json:"o"`
	Symbol    string `json:"s"`
	Time      int64  `json:"t"`
	Volume    string `json:"v"`
}

// UserFees is the per-user fee snapshot returned by /info
// {"type":"userFees"}.
type UserFees struct {
	ActiveReferralDiscount string       `json:"activeReferralDiscount"`
	DailyUserVolume        []UserVolume `json:"dailyUserVlm"`
	FeeSchedule            FeeSchedule  `json:"feeSchedule"`
	UserAddRate            string       `json:"userAddRate"`
	UserCrossRate          string       `json:"userCrossRate"`
}

// UserVolume is one daily-volume row inside UserFees.
type UserVolume struct {
	Date      string `json:"date"`
	Exchange  string `json:"exchange"`
	UserAdd   string `json:"userAdd"`
	UserCross string `json:"userCross"`
}

// FeeSchedule is the maker/taker fee schedule attached to a UserFees
// snapshot.
type FeeSchedule struct {
	Add              string `json:"add"`
	Cross            string `json:"cross"`
	ReferralDiscount string `json:"referralDiscount"`
	Tiers            Tiers  `json:"tiers"`
}

// Tiers groups the market-maker and VIP fee tiers exposed by FeeSchedule.
type Tiers struct {
	MM  []MMTier  `json:"mm"`
	VIP []VIPTier `json:"vip"`
}

// MMTier is one market-maker fee tier row inside Tiers.
type MMTier struct {
	Add                 string `json:"add"`
	MakerFractionCutoff string `json:"makerFractionCutoff"`
}

// VIPTier is one VIP fee tier row inside Tiers.
type VIPTier struct {
	Add       string `json:"add"`
	Cross     string `json:"cross"`
	NtlCutoff string `json:"ntlCutoff"`
}

// StakingSummary is the per-user staking snapshot returned by
// /info {"type":"delegatorSummary"}.
type StakingSummary struct {
	Delegated              string `json:"delegated"`
	Undelegated            string `json:"undelegated"`
	TotalPendingWithdrawal string `json:"totalPendingWithdrawal"`
	NPendingWithdrawals    int    `json:"nPendingWithdrawals"`
}

// StakingDelegation is one active-delegation row in /info
// {"type":"delegations"}.
type StakingDelegation struct {
	Validator            string `json:"validator"`
	Amount               string `json:"amount"`
	LockedUntilTimestamp int64  `json:"lockedUntilTimestamp"`
}

// StakingReward is one staking-reward row in /info
// {"type":"delegatorRewards"}.
type StakingReward struct {
	Time        int64  `json:"time"`
	Source      string `json:"source"`
	TotalAmount string `json:"totalAmount"`
}

// ReferralState is the per-user referral snapshot returned by /info
// {"type":"referral"}.
type ReferralState struct {
	ReferredBy       *ReferredBy    `json:"referredBy,omitempty"`
	CumVlm           string         `json:"cumVlm"`
	UnclaimedRewards string         `json:"unclaimedRewards"`
	ClaimedRewards   string         `json:"claimedRewards"`
	BuilderRewards   string         `json:"builderRewards"`
	ReferrerState    *ReferrerState `json:"referrerState,omitempty"`
	RewardHistory    []interface{}  `json:"rewardHistory"`
}

// ReferredBy describes the referrer of an account.
type ReferredBy struct {
	Referrer string `json:"referrer"`
	Code     string `json:"code"`
}

// ReferrerState is the per-referrer portion of ReferralState.
type ReferrerState struct {
	Stage string        `json:"stage"`
	Data  *ReferrerData `json:"data,omitempty"`
}

// ReferrerData groups the referrer code with the list of referred
// accounts.
type ReferrerData struct {
	Code           string           `json:"code"`
	ReferralStates []ReferralMember `json:"referralStates"`
}

// ReferralMember is one referred-account row inside ReferrerData.
type ReferralMember struct {
	CumVlm                       string `json:"cumVlm"`
	CumRewardedFeesSinceReferred string `json:"cumRewardedFeesSinceReferred"`
	CumFeesRewardedToReferrer    string `json:"cumFeesRewardedToReferrer"`
	TimeJoined                   int64  `json:"timeJoined"`
	User                         string `json:"user"`
}

// SubAccount is the per-sub-account directory entry returned by /info
// {"type":"subAccounts"}.
type SubAccount struct {
	Name        string   `json:"name"`
	User        string   `json:"user"`
	Permissions []string `json:"permissions"`
}

// MultiSigSigner is one signer row returned by /info
// {"type":"userToMultiSigSigners"}.
type MultiSigSigner struct {
	User      string `json:"user"`
	Threshold int    `json:"threshold"`
}

// Trade is a single trade record (used in the trades subscription feed).
type Trade struct {
	Coin  string   `json:"coin"`
	Side  string   `json:"side"`
	Px    string   `json:"px"`
	Sz    string   `json:"sz"`
	Time  int64    `json:"time"`
	Hash  string   `json:"hash"`
	Tid   int64    `json:"tid"`
	Users []string `json:"users"`
}

// BulkOrderResponse is the legacy response shape for a bulk order action.
type BulkOrderResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// CancelResponse is the wire response for a single-cancel action.
type CancelResponse struct {
	Status string     `json:"status"`
	Data   *OpenOrder `json:"data,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// BulkCancelResponse is the wire response for a bulk-cancel action.
type BulkCancelResponse struct {
	Status string      `json:"status"`
	Data   []OpenOrder `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// ModifyResponse is the legacy response shape for an order modify action.
type ModifyResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// TransferResponse is the response shape returned by transfer-style
// signed actions (usdSend, spotSend, vaultTransfer, etc.) and the
// HIP-4 userOutcome action family. Hyperliquid encodes failure as
// {"status":"err","response":"<message>"}; Response captures the raw
// payload so callers can extract the reason without re-parsing the
// wire bytes.
type TransferResponse struct {
	Status   string          `json:"status"`
	TxHash   string          `json:"txHash,omitempty"`
	Error    string          `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

// ApprovalResponse is the response shape returned by approval-style
// actions (approveBuilderFee, evmUserModify, ...).
type ApprovalResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CreateSubAccountResponse is returned by the createSubAccount action.
type CreateSubAccountResponse struct {
	Status string      `json:"status"`
	Data   *SubAccount `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// SetReferrerResponse is returned by the setReferrer action.
type SetReferrerResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ScheduleCancelResponse is returned by the scheduleCancel action.
type ScheduleCancelResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// AgentApprovalResponse is returned by the approveAgent action.
// Hyperliquid encodes failure as {"status":"err","response":"<message>"};
// success as {"status":"ok","response":{...}}. The Response field
// captures whichever form was returned so callers can surface the
// rejection reason verbatim.
type AgentApprovalResponse struct {
	Status   string          `json:"status"`
	TxHash   string          `json:"txHash,omitempty"`
	Error    string          `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

// MultiSigConversionResponse is returned by the
// convertToMultiSigUser action.
type MultiSigConversionResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SpotDeployResponse is returned by HIP-2 / HIP-3 spot deploy actions.
type SpotDeployResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ValidatorResponse is returned by CValidator and CSigner actions.
type ValidatorResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// MultiSigResponse is returned by the multiSig action envelope.
type MultiSigResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PerpDeployResponse is returned by HIP-3 perp deploy actions; the inner
// statuses array reports per-asset outcomes.
type PerpDeployResponse struct {
	Status string `json:"status"`
	Data   struct {
		Statuses []TxStatus `json:"statuses"`
	} `json:"data"`
}

// TxStatus is one per-asset outcome inside PerpDeployResponse.
type TxStatus struct {
	Coin   string `json:"coin"`
	Status string `json:"status"`
}

// PerpDex represents a perpetual DEX
type PerpDex struct {
	Name                     string     `json:"name"`
	FullName                 string     `json:"fullName"`
	Deployer                 string     `json:"deployer"`
	OracleUpdater            *string    `json:"oracleUpdater"`
	FeeRecipient             *string    `json:"feeRecipient"`
	AssetToStreamingOiCap    [][]string `json:"assetToStreamingOiCap"`    // Array of [coin, cap] tuples
	AssetToFundingMultiplier [][]string `json:"assetToFundingMultiplier"` // Array of [coin, multiplier] tuples
}

// PerpDexLimits represents limits for a builder-deployed perp DEX
type PerpDexLimits struct {
	TotalOiCap     string     `json:"totalOiCap"`
	OiSzCapPerPerp string     `json:"oiSzCapPerPerp"`
	MaxTransferNtl string     `json:"maxTransferNtl"`
	CoinToOiCap    [][]string `json:"coinToOiCap"` // Array of [coin, cap] tuples
}

// PerpDexStatus represents status for a builder-deployed perp DEX
type PerpDexStatus struct {
	TotalNetDeposit string `json:"totalNetDeposit"`
}

// PerpDeployAuctionStatus represents the status of a perp deploy auction
type PerpDeployAuctionStatus struct {
	StartTimeSeconds int64   `json:"startTimeSeconds"`
	DurationSeconds  int64   `json:"durationSeconds"`
	StartGas         string  `json:"startGas"`
	CurrentGas       string  `json:"currentGas"`
	EndGas           *string `json:"endGas"`
}
