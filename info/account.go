package info

import (
	"encoding/json"
	"fmt"

	"github.com/Simon-Busch/hyperliquid-go/types"
)

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

// SpotBalance represents a single spot token balance entry returned by
// the spotClearinghouseState endpoint.
type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Hold     string `json:"hold"`
	Total    string `json:"total"`
	EntryNtl string `json:"entryNtl"`
}

// SpotClearinghouseState is the response model for the spot balances
// endpoint.
type SpotClearinghouseState struct {
	Balances []SpotBalance `json:"balances"`
}

// AssetMeta is the per-asset metadata snapshot exposed by Asset. MinSize
// is the smallest legal size step (10^-SzDecimals); sizes that are not an
// integer multiple of MinSize are rejected by validate.
type AssetMeta struct {
	ID          int
	SzDecimals  int
	TickSize    float64
	MinSize     float64
	MaxLeverage int
	Class       types.AssetClass
}

// UserState retrieves the caller's perpetuals account summary. If dex is
// provided and non-empty, the snapshot is pinned to that HIP-3 dex.
func (c *Client) UserState(address string, dex ...string) (*UserState, error) {
	payload := map[string]any{
		"type": "clearinghouseState",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := c.client.Post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state: %w", err)
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}
	return &result, nil
}

// SpotBalances returns the spot clearinghouse state for addr.
func (c *Client) SpotBalances(addr string) (*SpotClearinghouseState, error) {
	resp, err := c.client.Post("/info", map[string]any{
		"type": "spotClearinghouseState",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot user state: %w", err)
	}

	var result SpotClearinghouseState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal spot user state: %w", err)
	}
	return &result, nil
}

// Positions returns the open positions for addr, optionally pinned to dex.
func (c *Client) Positions(addr string, dex ...string) ([]Position, error) {
	state, err := c.UserState(addr, dex...)
	if err != nil {
		return nil, err
	}
	out := make([]Position, 0, len(state.AssetPositions))
	for _, ap := range state.AssetPositions {
		out = append(out, ap.Position)
	}
	return out, nil
}

// Position returns the open position for coin held by addr, or nil if
// none.
func (c *Client) Position(addr, coin string) (*Position, error) {
	state, err := c.UserState(addr)
	if err != nil {
		return nil, err
	}
	for _, ap := range state.AssetPositions {
		if ap.Position.Coin == coin {
			p := ap.Position
			return &p, nil
		}
	}
	return nil, nil
}

// Fees returns the fee snapshot for addr.
func (c *Client) Fees(addr string) (*UserFees, error) {
	resp, err := c.client.Post("/info", map[string]any{
		"type": "userFees",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fees: %w", err)
	}

	var result UserFees
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fees: %w", err)
	}
	return &result, nil
}

// Asset returns the metadata snapshot for coin.
func (c *Client) Asset(coin string) (AssetMeta, error) {
	id := c.AssetID(coin)
	class := types.ClassifyAsset(id)
	szDecimals := c.assetToDecimal[id]
	maxPriceDecimals := class.MaxPriceDecimals() - szDecimals
	if maxPriceDecimals < 0 {
		maxPriceDecimals = 0
	}
	tick := 1.0
	for k := 0; k < maxPriceDecimals; k++ {
		tick /= 10
	}
	minSize := 1.0
	for k := 0; k < szDecimals; k++ {
		minSize /= 10
	}
	return AssetMeta{
		ID:         id,
		SzDecimals: szDecimals,
		TickSize:   tick,
		MinSize:    minSize,
		Class:      class,
	}, nil
}

// AssetID returns the numeric asset id for coin.
func (c *Client) AssetID(coin string) int {
	cc := c.nameToCoin[coin]
	return c.coinToAsset[cc]
}
