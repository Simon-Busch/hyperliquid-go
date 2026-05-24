package hyperliquid

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// DefaultResponse is the wire shape of a simple {"type":"default"}
// envelope returned by some Hyperliquid endpoints.
type DefaultResponse struct {
	Type string `json:"type"`
}

// Agent is the typed handle returned by ApproveAgent. Address is the
// 0x-prefixed agent EOA; PrivateKey is the freshly generated ECDSA key
// associated with that address — keep it secret.
type Agent struct {
	// Address is the lower-case 0x-prefixed hex of the agent EOA.
	Address string
	// PrivateKey is the ECDSA private key controlling Address.
	PrivateKey *ecdsa.PrivateKey
}

// SetLeverage updates the leverage on coin. mode picks Cross or Isolated
// margin: Cross maps to isCross=true (shared collateral across
// positions), Isolated to isCross=false (per-position collateral).
// leverage is an integer multiple in the range allowed by the asset's
// margin table.
func (t *Trader) SetLeverage(coin string, leverage int, mode MarginMode) (*UserState, error) {
	action := UpdateLeverageAction{
		Type:     "updateLeverage",
		Asset:    t.info.AssetID(coin),
		IsCross:  mode == Cross,
		Leverage: leverage,
	}

	var result UserState
	if err := t.executeAction(action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AdjustMargin adds or removes isolated-margin collateral on the position
// in coin. A positive amount increases collateral; a negative amount
// withdraws it. amount is in USDC (decimal).
func (t *Trader) AdjustMargin(coin string, amount float64) (*APIResponse[DefaultResponse], error) {
	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: t.info.AssetID(coin),
		IsBuy: true,
		Ntli:  int64(math.Round(amount * 1e6)),
	}
	var result APIResponse[DefaultResponse]
	if err := t.executeAction(action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetExpiresAfter updates the expiration deadline stamped on every
// signed action the Trader subsequently dispatches. A zero deadline
// (the zero value of time.Time) clears the field.
func (t *Trader) SetExpiresAfter(deadline time.Time) {
	if deadline.IsZero() {
		t.expiresAfter = nil
		return
	}
	ms := deadline.UnixMilli()
	t.expiresAfter = &ms
}

// ScheduleCancelAll schedules cancellation of all open orders at
// deadline. A nil deadline clears any scheduled cancel and lets existing
// orders rest indefinitely. A non-nil deadline is converted to a
// Unix-millisecond timestamp before signing.
func (t *Trader) ScheduleCancelAll(deadline *time.Time) (*ScheduleCancelResponse, error) {
	timestamp := time.Now().UnixMilli()

	var scheduleTime *int64
	if deadline != nil {
		ms := deadline.UnixMilli()
		scheduleTime = &ms
	}

	action := ScheduleCancelAction{
		Type: "scheduleCancel",
		Time: scheduleTime,
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

	var result ScheduleCancelResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetReferrer sets a referral code for the signing account.
func (t *Trader) SetReferrer(code string) (*SetReferrerResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SetReferrerAction{
		Type: "setReferrer",
		Code: code,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		"",
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

	var result SetReferrerResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UseBigBlocks enables or disables big-block evm-user mode.
func (t *Trader) UseBigBlocks(enable bool) (*ApprovalResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := UseBigBlocksAction{
		Type:           "evmUserModify",
		UsingBigBlocks: enable,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		"",
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

	var result ApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveAgent generates a fresh agent key, registers it with
// Hyperliquid under the optional name, and returns the resulting Agent.
// The empty string disables the agent-name field on the wire.
func (t *Trader) ApproveAgent(name string) (Agent, error) {
	agentBytes := make([]byte, 32)
	if _, err := rand.Read(agentBytes); err != nil {
		return Agent{}, fmt.Errorf("generate agent key: %w", err)
	}
	agentKeyHex := hex.EncodeToString(agentBytes)
	pk, err := crypto.HexToECDSA(agentKeyHex)
	if err != nil {
		return Agent{}, fmt.Errorf("parse agent key: %w", err)
	}
	agentAddress := strings.ToLower(crypto.PubkeyToAddress(pk.PublicKey).Hex())

	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":         "approveAgent",
		"agentAddress": agentAddress,
		"agentName":    name,
		"nonce":        nonce,
	}
	var result AgentApprovalResponse
	if err := t.executeUserSignedAction(
		action, approveAgentSignTypes,
		"HyperliquidTransaction:ApproveAgent", nonce, &result,
	); err != nil {
		return Agent{}, err
	}
	if result.Status != "" && result.Status != "ok" {
		msg := result.Error
		if msg == "" {
			// On err the Response field is a JSON string with the reason.
			var reason string
			if err := json.Unmarshal(result.Response, &reason); err == nil && reason != "" {
				msg = reason
			}
		}
		if msg == "" {
			msg = result.Status
		}
		return Agent{}, fmt.Errorf("approveAgent rejected: %s", msg)
	}
	return Agent{Address: agentAddress, PrivateKey: pk}, nil
}

// ApproveBuilderFee approves a builder address to charge up to
// maxFeeRate. maxFeeRate must be a percent string like "0.1%".
func (t *Trader) ApproveBuilderFee(builder string, maxFeeRate string) (*ApprovalResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":       "approveBuilderFee",
		"maxFeeRate": maxFeeRate,
		"builder":    strings.ToLower(builder),
		"nonce":      nonce,
	}
	var result ApprovalResponse
	if err := t.executeUserSignedAction(
		action, approveBuilderFeeSignTypes,
		"HyperliquidTransaction:ApproveBuilderFee", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}
