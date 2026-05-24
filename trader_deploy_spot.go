package hyperliquid

import (
	"encoding/json"
	"time"
)

// SpotDeployRegisterToken registers a new spot token
func (t *Trader) SpotDeployRegisterToken(
	tokenName string,
	szDecimals int,
	weiDecimals int,
	maxGas int,
	fullName string,
) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "spotDeploy",
		"registerToken2": map[string]any{
			"spec": map[string]any{
				"name":        tokenName,
				"szDecimals":  szDecimals,
				"weiDecimals": weiDecimals,
			},
			"maxGas":   maxGas,
			"fullName": fullName,
		},
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		"", // No vault address for spot deploy
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployUserGenesis initializes user genesis for spot trading
func (t *Trader) SpotDeployUserGenesis(balances map[string]float64) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":     "spotDeployUserGenesis",
		"balances": balances,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployEnableFreezePrivilege enables freeze privilege for spot deployer
func (t *Trader) SpotDeployEnableFreezePrivilege() (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "spotDeployEnableFreezePrivilege",
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployFreezeUser freezes a user in spot trading
func (t *Trader) SpotDeployFreezeUser(userAddress string) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":        "spotDeployFreezeUser",
		"userAddress": userAddress,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployRevokeFreezePrivilege revokes freeze privilege for spot deployer
func (t *Trader) SpotDeployRevokeFreezePrivilege() (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "spotDeployRevokeFreezePrivilege",
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployGenesis initializes spot genesis
func (t *Trader) SpotDeployGenesis(deployer string, dexName string) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":     "spotDeployGenesis",
		"deployer": deployer,
		"dexName":  dexName,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployRegisterSpot registers spot market
func (t *Trader) SpotDeployRegisterSpot(
	baseToken string,
	quoteToken string,
) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":       "spotDeployRegisterSpot",
		"baseToken":  baseToken,
		"quoteToken": quoteToken,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeployRegisterHyperliquidity registers hyperliquidity spot
func (t *Trader) SpotDeployRegisterHyperliquidity(
	name string,
	tokens []string,
) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":   "spotDeployRegisterHyperliquidity",
		"name":   name,
		"tokens": tokens,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// SpotDeploySetDeployerTradingFeeShare sets deployer trading fee share
func (t *Trader) SpotDeploySetDeployerTradingFeeShare(
	feeShare float64,
) (*SpotDeployResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":     "spotDeploySetDeployerTradingFeeShare",
		"feeShare": feeShare,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		t.vault,
		timestamp,
		t.expiresAfter,
		t.client.BaseURL == MainnetAPIURL,
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

// Perp Deploy Methods
