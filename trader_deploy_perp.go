package hyperliquid

import (
	"encoding/json"
	"time"
)

// PerpDeployRegisterAsset registers a new perpetual asset
func (t *Trader) PerpDeployRegisterAsset(
	asset string,
	perpDexInput PerpDexSchemaInput,
) (*PerpDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":         "perpDeployRegisterAsset",
		"asset":        asset,
		"perpDexInput": perpDexInput,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := t.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result PerpDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpDeploySetOracle sets oracle for perpetual asset
func (t *Trader) PerpDeploySetOracle(
	asset string,
	oracleAddress string,
) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":          "perpDeploySetOracle",
		"asset":         asset,
		"oracleAddress": oracleAddress,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := t.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
