package hyperliquid

import (
	"encoding/json"
	"fmt"
)

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

// AssetMeta is the per-asset metadata snapshot exposed by Asset.
type AssetMeta struct {
	ID          int
	SzDecimals  int
	TickSize    float64
	MinSize     float64
	MaxLeverage int
	Class       AssetClass
}

// UserState retrieves the caller's perpetuals account summary. If dex is
// provided and non-empty, the snapshot is pinned to that HIP-3 dex.
func (i *Info) UserState(address string, dex ...string) (*UserState, error) {
	payload := map[string]any{
		"type": "clearinghouseState",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state: %w", err)
	}

	var result UserState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user state: %w", err)
	}
	return &result, nil
}

// SpotUserState retrieves the caller's spot clearinghouse snapshot.
func (i *Info) SpotUserState(address string) (*SpotClearinghouseState, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "spotClearinghouseState",
		"user": address,
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

// SpotBalances returns the spot clearinghouse state for addr.
func (i *Info) SpotBalances(addr string) (*SpotClearinghouseState, error) {
	return i.SpotUserState(addr)
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

// Position returns the open position for coin held by addr, or nil if
// none.
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

// UserFees fetches the fee snapshot for address.
func (i *Info) UserFees(address string) (*UserFees, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userFees",
		"user": address,
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

// Fees returns the fee snapshot for addr.
func (i *Info) Fees(addr string) (*UserFees, error) {
	return i.UserFees(addr)
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

// AssetID returns the numeric asset id for coin.
func (i *Info) AssetID(coin string) int {
	return i.NameToAsset(coin)
}

// NameToAsset returns the asset id for the canonical coin name.
func (i *Info) NameToAsset(name string) int {
	coin := i.nameToCoin[name]
	return i.coinToAsset[coin]
}
