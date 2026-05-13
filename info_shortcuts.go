package hyperliquid

import (
	"fmt"
	"strconv"
)

// Mid returns the current mid price for coin as a float64.
func (i *Info) Mid(coin string) (float64, error) {
	mids, err := i.AllMids()
	if err != nil {
		return 0, err
	}
	s, ok := mids[coin]
	if !ok {
		return 0, fmt.Errorf("no mid for %s", coin)
	}
	return strconv.ParseFloat(s, 64)
}

// AllMidsOn returns the AllMids snapshot pinned to a specific HIP-3 dex.
func (i *Info) AllMidsOn(dex string) (map[string]string, error) {
	return i.AllMids(dex)
}

// Book returns the current L2 order book for coin.
func (i *Info) Book(coin string) (*L2Book, error) {
	return i.L2Snapshot(coin)
}

// Candles returns historical candles for coin at interval between start
// and end (Unix millis).
func (i *Info) Candles(coin, interval string, start, end int64) ([]Candle, error) {
	return i.CandlesSnapshot(coin, interval, start, end)
}

// Fills returns all fills for addr.
func (i *Info) Fills(addr string) ([]Fill, error) {
	return i.UserFills(addr)
}

// FillsBetween returns fills for addr in the range [start, end].
func (i *Info) FillsBetween(addr string, start int64, end *int64) ([]Fill, error) {
	return i.UserFillsByTime(addr, start, end)
}

// Positions returns the open positions for addr, optionally pinned to dex.
func (i *Info) Positions(addr string, dex ...string) ([]Position, error) {
	state, err := i.UserState(addr, dex...)
	if err != nil {
		return nil, err
	}
	out := make([]Position, 0, len(state.AssetPositions))
	for _, ap := range state.AssetPositions {
		out = append(out, ap.Position)
	}
	return out, nil
}

// Position returns the open position for coin held by addr, or nil if none.
func (i *Info) Position(addr, coin string) (*Position, error) {
	state, err := i.UserState(addr)
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

// SpotBalances returns the spot clearinghouse state for addr.
func (i *Info) SpotBalances(addr string) (*SpotClearinghouseState, error) {
	return i.SpotUserState(addr)
}

// Funding returns historical funding rates for coin in [start, end].
func (i *Info) Funding(coin string, start int64, end *int64) ([]FundingHistory, error) {
	return i.FundingHistory(coin, start, end)
}

// UserFunding returns the funding history for addr in [start, end].
func (i *Info) UserFunding(addr string, start int64, end *int64) ([]UserFundingHistory, error) {
	return i.UserFundingHistory(addr, start, end)
}

// Fees returns the fee snapshot for addr.
func (i *Info) Fees(addr string) (*UserFees, error) {
	return i.UserFees(addr)
}

// Order returns the order with the supplied exchange oid.
func (i *Info) Order(addr string, oid int64) (*OrderStatusResponse, error) {
	return i.QueryOrderByOid(addr, oid)
}

// OrderByCloid returns the order with the supplied client order id.
func (i *Info) OrderByCloid(addr, cloid string) (*OpenOrder, error) {
	return i.QueryOrderByCloid(addr, cloid)
}

// Fill returns the fill matching addr and oid.
func (i *Info) Fill(addr string, oid int64) (*Fill, error) {
	return i.QueryFillByOid(addr, oid)
}

// AssetMeta is the per-asset metadata snapshot exposed by Asset.
type AssetMeta struct {
	ID          int
	SzDecimals  int
	TickSize    float64
	MinSize     float64
	MaxLeverage int
	Class       AssetClass
}

// Asset returns the metadata snapshot for coin.
func (i *Info) Asset(coin string) (AssetMeta, error) {
	id := i.NameToAsset(coin)
	class := ClassifyAsset(id)
	szDecimals := i.assetToDecimal[id]
	maxPriceDecimals := class.MaxPriceDecimals() - szDecimals
	if maxPriceDecimals < 0 {
		maxPriceDecimals = 0
	}
	tick := 1.0
	for k := 0; k < maxPriceDecimals; k++ {
		tick /= 10
	}
	return AssetMeta{
		ID:         id,
		SzDecimals: szDecimals,
		TickSize:   tick,
		Class:      class,
	}, nil
}

// AssetID returns the numeric asset id for coin (was NameToAsset).
func (i *Info) AssetID(coin string) int {
	return i.NameToAsset(coin)
}

// SubAccounts returns the sub-account list for addr (was QuerySubAccounts).
func (i *Info) SubAccounts(addr string) ([]SubAccount, error) {
	return i.QuerySubAccounts(addr)
}

// Referral returns the referral state for addr (was QueryReferralState).
func (i *Info) Referral(addr string) (*ReferralState, error) {
	return i.QueryReferralState(addr)
}

// MultiSigSigners returns the signer list for a multi-sig user
// (was QueryUserToMultiSigSigners).
func (i *Info) MultiSigSigners(multiSigAddr string) ([]MultiSigSigner, error) {
	return i.QueryUserToMultiSigSigners(multiSigAddr)
}

// InfoStakeGroup exposes the staking-info shortcuts.
type InfoStakeGroup struct{ i *Info }

// Stake returns the staking-info sub-group on Info.
func (i *Info) Stake() *InfoStakeGroup { return &InfoStakeGroup{i: i} }

// Summary returns the staking summary for addr.
func (g *InfoStakeGroup) Summary(addr string) (*StakingSummary, error) {
	return g.i.UserStakingSummary(addr)
}

// Delegations returns the active delegations for addr.
func (g *InfoStakeGroup) Delegations(addr string) ([]StakingDelegation, error) {
	return g.i.UserStakingDelegations(addr)
}

// Rewards returns the staking reward history for addr.
func (g *InfoStakeGroup) Rewards(addr string) ([]StakingReward, error) {
	return g.i.UserStakingRewards(addr)
}
