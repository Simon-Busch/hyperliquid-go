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

// UsdClassTransfer transfers between USD classes (perps <-> spot).
func (e *Trader) UsdClassTransfer(amount float64, toPerp bool) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	amountStr := formatUsdAmount(amount)
	if e.vault != "" {
		amountStr += " subaccount:" + e.vault
	}
	action := map[string]any{
		"type":   "usdClassTransfer",
		"amount": amountStr,
		"toPerp": toPerp,
		"nonce":  nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, usdClassTransferSignTypes,
		"HyperliquidTransaction:UsdClassTransfer", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// VaultUsdTransfer deposits or withdraws USDC to/from a vault.
func (e *Trader) VaultUsdTransfer(
	vaultAddress string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := VaultUsdTransferAction{
		Type:         "vaultTransfer",
		VaultAddress: vaultAddress,
		IsDeposit:    isDeposit,
		Usd:          usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"",
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UsdTransfer sends USDC to another address on Hyperliquid.
func (e *Trader) UsdTransfer(amount float64, destination string) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "usdSend",
		"destination": destination,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, usdSendSignTypes,
		"HyperliquidTransaction:UsdSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotTransfer sends spot tokens to another address.
func (e *Trader) SpotTransfer(
	amount float64,
	destination, token string,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "spotSend",
		"destination": destination,
		"token":       token,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, spotTransferSignTypes,
		"HyperliquidTransaction:SpotSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpDexClassTransfer moves tokens between perp dex classes.
func (e *Trader) PerpDexClassTransfer(
	dex, token string,
	amount float64,
	toPerp bool,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := PerpDexClassTransferAction{
		Type:   "perpDexClassTransfer",
		Dex:    dex,
		Token:  token,
		Amount: amount,
		ToPerp: toPerp,
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendUSD sends USDC on Hyperliquid to toAddr.
func (g *TransferGroup) SendUSD(toAddr string, amount float64) (*TransferResponse, error) {
	return g.t.UsdTransfer(amount, toAddr)
}

// SendSpot sends a spot token to toAddr.
func (g *TransferGroup) SendSpot(toAddr, token string, amount float64) (*TransferResponse, error) {
	return g.t.SpotTransfer(amount, toAddr, token)
}

// DepositToVault deposits USDC into the given vault.
func (g *TransferGroup) DepositToVault(vaultAddr string, amount float64) (*TransferResponse, error) {
	return g.t.VaultUsdTransfer(vaultAddr, true, FloatToUsdInt(amount))
}

// WithdrawFromVault withdraws USDC from the given vault.
func (g *TransferGroup) WithdrawFromVault(vaultAddr string, amount float64) (*TransferResponse, error) {
	return g.t.VaultUsdTransfer(vaultAddr, false, FloatToUsdInt(amount))
}

// PerpToSpot transfers USDC from perp to spot for the signing account.
func (g *TransferGroup) PerpToSpot(amount float64) (*TransferResponse, error) {
	return g.t.UsdClassTransfer(amount, false)
}

// SpotToPerp transfers USDC from spot to perp for the signing account.
func (g *TransferGroup) SpotToPerp(amount float64) (*TransferResponse, error) {
	return g.t.UsdClassTransfer(amount, true)
}

// MoveToDex moves tokens into a HIP-3 builder-deployed perp dex.
func (g *TransferGroup) MoveToDex(dex, token string, amount float64) (*TransferResponse, error) {
	return g.t.PerpDexClassTransfer(dex, token, amount, true)
}

// MoveFromDex moves tokens out of a HIP-3 builder-deployed perp dex.
func (g *TransferGroup) MoveFromDex(dex, token string, amount float64) (*TransferResponse, error) {
	return g.t.PerpDexClassTransfer(dex, token, amount, false)
}
