package hyperliquid

import (
	"encoding/json"
	"time"
)

// PerpDeployRegisterAsset registers a new perpetual asset
func (e *Trader) PerpDeployRegisterAsset(
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
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
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
func (e *Trader) PerpDeploySetOracle(
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
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result SpotDeployResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
