package hyperliquid

import (
	"encoding/json"
	"time"
)

// SubAccountGroup exposes sub-account management on Trader.
type SubAccountGroup struct {
	t *Trader
}

// Create allocates a new sub-account under the current signer.
func (g *SubAccountGroup) Create(name string) (*CreateSubAccountResponse, error) {
	t := g.t
	timestamp := time.Now().UnixMilli()

	action := CreateSubAccountAction{
		Type: "createSubAccount",
		Name: name,
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

	var result CreateSubAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DepositUSD funds a sub-account from the parent's USDC balance.
func (g *SubAccountGroup) DepositUSD(subAddr string, amount float64) (*TransferResponse, error) {
	return g.transfer(subAddr, true, FloatToUsdInt(amount))
}

// WithdrawUSD pulls USDC from a sub-account back to the parent.
func (g *SubAccountGroup) WithdrawUSD(subAddr string, amount float64) (*TransferResponse, error) {
	return g.transfer(subAddr, false, FloatToUsdInt(amount))
}

// transfer signs and submits a subAccountTransfer action.
func (g *SubAccountGroup) transfer(subAccountUser string, isDeposit bool, usd int) (*TransferResponse, error) {
	t := g.t
	timestamp := time.Now().UnixMilli()

	action := SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Usd:            usd,
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DepositSpot funds a sub-account's spot balance with token.
func (g *SubAccountGroup) DepositSpot(subAddr, token string, amount float64) (*TransferResponse, error) {
	return g.spotTransfer(subAddr, true, token, amount)
}

// WithdrawSpot pulls a spot token back from a sub-account.
func (g *SubAccountGroup) WithdrawSpot(subAddr, token string, amount float64) (*TransferResponse, error) {
	return g.spotTransfer(subAddr, false, token, amount)
}

// spotTransfer signs and submits a subAccountSpotTransfer action.
func (g *SubAccountGroup) spotTransfer(subAccountUser string, isDeposit bool, token string, amount float64) (*TransferResponse, error) {
	t := g.t
	timestamp := time.Now().UnixMilli()

	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amount,
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

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
