package hyperliquid

import (
	"encoding/json"
	"time"
)

// SubAccountGroup exposes sub-account management on Trader.
type SubAccountGroup struct {
	t *Trader
}

// CreateSubAccount creates a new sub-account under the signing account.
func (e *Trader) CreateSubAccount(name string) (*CreateSubAccountResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := CreateSubAccountAction{
		Type: "createSubAccount",
		Name: name,
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

	var result CreateSubAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubAccountTransfer moves USDC to or from a sub-account.
func (e *Trader) SubAccountTransfer(
	subAccountUser string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Usd:            usd,
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

// SubAccountSpotTransfer moves spot tokens to or from a sub-account.
func (e *Trader) SubAccountSpotTransfer(
	subAccountUser string,
	isDeposit bool,
	token string,
	amount float64,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amount,
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

// Create allocates a new sub-account under the current signer.
func (g *SubAccountGroup) Create(name string) (*CreateSubAccountResponse, error) {
	return g.t.CreateSubAccount(name)
}

// DepositUSD funds a sub-account from the parent's USDC balance.
func (g *SubAccountGroup) DepositUSD(subAddr string, amount float64) (*TransferResponse, error) {
	return g.t.SubAccountTransfer(subAddr, true, FloatToUsdInt(amount))
}

// WithdrawUSD pulls USDC from a sub-account back to the parent.
func (g *SubAccountGroup) WithdrawUSD(subAddr string, amount float64) (*TransferResponse, error) {
	return g.t.SubAccountTransfer(subAddr, false, FloatToUsdInt(amount))
}

// DepositSpot funds a sub-account's spot balance with token.
func (g *SubAccountGroup) DepositSpot(subAddr, token string, amount float64) (*TransferResponse, error) {
	return g.t.SubAccountSpotTransfer(subAddr, true, token, amount)
}

// WithdrawSpot pulls a spot token back from a sub-account.
func (g *SubAccountGroup) WithdrawSpot(subAddr, token string, amount float64) (*TransferResponse, error) {
	return g.t.SubAccountSpotTransfer(subAddr, false, token, amount)
}
