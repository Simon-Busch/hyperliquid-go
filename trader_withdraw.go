package hyperliquid

import "time"

// Withdraw off-ramps USDC from L1 to destination.
func (t *Trader) Withdraw(amount float64, destination string) (*TransferResponse, error) {
	return t.WithdrawFromBridge(amount, destination)
}

// WithdrawFromBridge withdraws USDC to destination on L1.
func (e *Trader) WithdrawFromBridge(
	amount float64,
	destination string,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "withdraw3",
		"destination": destination,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, withdrawSignTypes,
		"HyperliquidTransaction:Withdraw", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}
