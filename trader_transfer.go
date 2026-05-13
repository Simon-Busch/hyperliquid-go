package hyperliquid

import (
	"encoding/json"
	"time"
)

// TransferGroup exposes transfer-related Trader actions in a sub-grouped
// namespace. Reach it via t.Transfer (initialised lazily).
type TransferGroup struct {
	t *Trader
}

// SendUSD sends USDC on Hyperliquid to toAddr.
func (g *TransferGroup) SendUSD(toAddr string, amount float64) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "usdSend",
		"destination": toAddr,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := g.t.executeUserSignedAction(
		action, usdSendSignTypes,
		"HyperliquidTransaction:UsdSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendSpot sends a spot token to toAddr.
func (g *TransferGroup) SendSpot(toAddr, token string, amount float64) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "spotSend",
		"destination": toAddr,
		"token":       token,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := g.t.executeUserSignedAction(
		action, spotTransferSignTypes,
		"HyperliquidTransaction:SpotSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// DepositToVault deposits USDC into the given vault.
func (g *TransferGroup) DepositToVault(vaultAddr string, amount float64) (*TransferResponse, error) {
	return g.vaultTransfer(vaultAddr, true, FloatToUsdInt(amount))
}

// WithdrawFromVault withdraws USDC from the given vault.
func (g *TransferGroup) WithdrawFromVault(vaultAddr string, amount float64) (*TransferResponse, error) {
	return g.vaultTransfer(vaultAddr, false, FloatToUsdInt(amount))
}

// vaultTransfer signs and submits a vaultTransfer action for either a
// deposit or a withdrawal.
func (g *TransferGroup) vaultTransfer(vaultAddress string, isDeposit bool, usd int) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()
	t := g.t

	action := VaultUsdTransferAction{
		Type:         "vaultTransfer",
		VaultAddress: vaultAddress,
		IsDeposit:    isDeposit,
		Usd:          usd,
	}

	sig, err := SignL1Action(
		t.privateKey,
		action,
		"",
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpToSpot transfers USDC from perp to spot for the signing account.
func (g *TransferGroup) PerpToSpot(amount float64) (*TransferResponse, error) {
	return g.usdClassTransfer(amount, false)
}

// SpotToPerp transfers USDC from spot to perp for the signing account.
func (g *TransferGroup) SpotToPerp(amount float64) (*TransferResponse, error) {
	return g.usdClassTransfer(amount, true)
}

// usdClassTransfer signs and submits a usdClassTransfer action.
func (g *TransferGroup) usdClassTransfer(amount float64, toPerp bool) (*TransferResponse, error) {
	t := g.t
	nonce := time.Now().UnixMilli()
	amountStr := formatUsdAmount(amount)
	if t.vault != "" {
		amountStr += " subaccount:" + t.vault
	}
	action := map[string]any{
		"type":   "usdClassTransfer",
		"amount": amountStr,
		"toPerp": toPerp,
		"nonce":  nonce,
	}
	var result TransferResponse
	if err := t.executeUserSignedAction(
		action, usdClassTransferSignTypes,
		"HyperliquidTransaction:UsdClassTransfer", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// MoveToDex moves tokens into a HIP-3 builder-deployed perp dex.
func (g *TransferGroup) MoveToDex(dex, token string, amount float64) (*TransferResponse, error) {
	return g.perpDexClassTransfer(dex, token, amount, true)
}

// MoveFromDex moves tokens out of a HIP-3 builder-deployed perp dex.
func (g *TransferGroup) MoveFromDex(dex, token string, amount float64) (*TransferResponse, error) {
	return g.perpDexClassTransfer(dex, token, amount, false)
}

// perpDexClassTransfer signs and submits a perpDexClassTransfer action.
func (g *TransferGroup) perpDexClassTransfer(dex, token string, amount float64, toPerp bool) (*TransferResponse, error) {
	t := g.t
	timestamp := time.Now().UnixMilli()

	action := PerpDexClassTransferAction{
		Type:   "perpDexClassTransfer",
		Dex:    dex,
		Token:  token,
		Amount: amount,
		ToPerp: toPerp,
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
