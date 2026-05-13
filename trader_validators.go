package hyperliquid

import (
	"encoding/json"
	"time"
)

// CSigner Methods

// CSignerUnjailSelf unjails self as consensus signer
func (t *Trader) CSignerUnjailSelf() (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "cSignerUnjailSelf",
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CSignerJailSelf jails self as consensus signer
func (t *Trader) CSignerJailSelf() (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "cSignerJailSelf",
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CSignerInner executes inner consensus signer action
func (t *Trader) CSignerInner(innerAction map[string]any) (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":        "cSignerInner",
		"innerAction": innerAction,
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidator Methods

// CValidatorRegister registers as consensus validator
func (t *Trader) CValidatorRegister(validatorProfile map[string]any) (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":             "cValidatorRegister",
		"validatorProfile": validatorProfile,
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidatorChangeProfile changes validator profile
func (t *Trader) CValidatorChangeProfile(newProfile map[string]any) (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type":       "cValidatorChangeProfile",
		"newProfile": newProfile,
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CValidatorUnregister unregisters as consensus validator
func (t *Trader) CValidatorUnregister() (*ValidatorResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := map[string]any{
		"type": "cValidatorUnregister",
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

	var result ValidatorResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
