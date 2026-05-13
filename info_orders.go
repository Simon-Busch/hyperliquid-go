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

func (i *Info) QueryOrderByOid(user string, oid int64) (*OrderStatusResponse, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": user,
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

// QueryFillByOid finds a specific fill by OID from user fills
// Since there's no direct fill query endpoint, we filter userFills by OID
func (i *Info) QueryFillByOid(user string, oid int64) (*Fill, error) {
	fills, err := i.UserFills(user)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user fills: %w", err)
	}

	for _, fill := range fills {
		if fill.Oid == oid {
			return &fill, nil
		}
	}

	return nil, fmt.Errorf("fill with OID %d not found for user %s", oid, user)
}

func (i *Info) QueryOrderByCloid(user, cloid string) (*OpenOrder, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "orderStatus",
		"user": user,
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

func (i *Info) QueryReferralState(user string) (*ReferralState, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "referral",
		"user": user,
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

func (i *Info) QuerySubAccounts(user string) ([]SubAccount, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "subAccounts",
		"user": user,
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

func (i *Info) QueryUserToMultiSigSigners(multiSigUser string) ([]MultiSigSigner, error) {
	resp, err := i.client.post("/info", map[string]any{
		"type": "userToMultiSigSigners",
		"user": multiSigUser,
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
