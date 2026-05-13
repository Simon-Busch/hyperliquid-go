package hyperliquid

import (
	"encoding/json"
	"fmt"
)

// OrderStatusResponse represents the actual response from the orderStatus endpoint
type OrderStatusResponse struct {
	Status string `json:"status"`
	Order  struct {
		Order struct {
			Coin             string  `json:"coin"`
			Side             string  `json:"side"`
			LimitPx          string  `json:"limitPx"`
			Sz               string  `json:"sz"`
			Oid              int64   `json:"oid"`
			Timestamp        int64   `json:"timestamp"`
			TriggerCondition string  `json:"triggerCondition"`
			IsTrigger        bool    `json:"isTrigger"`
			TriggerPx        string  `json:"triggerPx"`
			Children         []any   `json:"children"`
			IsPositionTpsl   bool    `json:"isPositionTpsl"`
			ReduceOnly       bool    `json:"reduceOnly"`
			OrderType        string  `json:"orderType"`
			OrigSz           string  `json:"origSz"`
			Tif              *string `json:"tif"`
			Cloid            *string `json:"cloid"`
		} `json:"order"`
		Status          string `json:"status"`
		StatusTimestamp int64  `json:"statusTimestamp"`
	} `json:"order"`
}

// OpenOrders retrieves the user's open orders. If dex is provided and
// non-empty, the query is pinned to that HIP-3 dex. Spot open orders are
// only returned with the first perp dex.
func (i *Info) OpenOrders(address string, dex ...string) ([]OpenOrder, error) {
	payload := map[string]any{
		"type": "openOrders",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	var result []OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal open orders: %w", err)
	}
	return result, nil
}

// FrontendOpenOrders retrieves the user's open orders with frontend info.
// If dex is provided and non-empty, the query is pinned to that HIP-3
// dex. Spot open orders are only returned with the first perp dex.
func (i *Info) FrontendOpenOrders(address string, dex ...string) ([]FrontendOpenOrder, error) {
	payload := map[string]any{
		"type": "frontendOpenOrders",
		"user": address,
	}
	if len(dex) > 0 && dex[0] != "" {
		payload["dex"] = dex[0]
	}

	resp, err := i.client.post("/info", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch frontend open orders: %w", err)
	}

	var result []FrontendOpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frontend open orders: %w", err)
	}
	return result, nil
}

// Fills retrieves the trailing fill history for addr.
func (i *Info) Fills(addr string) ([]Fill, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userFills",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills: %w", err)
	}
	return result, nil
}

// FillsBetween retrieves the fill history for addr in [start, end].
func (i *Info) FillsBetween(addr string, start int64, end *int64) ([]Fill, error) {
	resp, err := i.postTimeRangeRequest("userFillsByTime", addr, start, end, nil)
	if err != nil {
		return nil, err
	}

	var result []Fill
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user fills by time: %w", err)
	}
	return result, nil
}

// Order returns the order status for the supplied (addr, oid) pair.
func (i *Info) Order(addr string, oid int64) (*OrderStatusResponse, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": addr,
		"oid":  oid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status: %w", err)
	}

	var result OrderStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}

	return &result, nil
}

// Fill finds the fill matching (addr, oid) by scanning the user's fill
// history; there is no direct endpoint for this query.
func (i *Info) Fill(addr string, oid int64) (*Fill, error) {
	fills, err := i.Fills(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	for _, fill := range fills {
		if fill.Oid == oid {
			return &fill, nil
		}
	}

	return nil, fmt.Errorf("fill with OID %d not found for user %s", oid, addr)
}

// OrderByCloid returns the order status for the supplied (addr, cloid)
// pair.
func (i *Info) OrderByCloid(addr, cloid string) (*OpenOrder, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": addr,
		"oid":  cloid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status by cloid: %w", err)
	}

	var result OpenOrder
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order status: %w", err)
	}
	return &result, nil
}

// Referral returns the referral state for addr.
func (i *Info) Referral(addr string) (*ReferralState, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "referral",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch referral state: %w", err)
	}

	var result ReferralState
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal referral state: %w", err)
	}
	return &result, nil
}

// SubAccounts returns the sub-account list for addr.
func (i *Info) SubAccounts(addr string) ([]SubAccount, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "subAccounts",
		"user": addr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sub accounts: %w", err)
	}

	var result []SubAccount
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sub accounts: %w", err)
	}
	return result, nil
}

// MultiSigSigners returns the signer list for multiSigAddr.
func (i *Info) MultiSigSigners(multiSigAddr string) ([]MultiSigSigner, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userToMultiSigSigners",
		"user": multiSigAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch multi-sig signers: %w", err)
	}

	var result []MultiSigSigner
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multi-sig signers: %w", err)
	}
	return result, nil
}
