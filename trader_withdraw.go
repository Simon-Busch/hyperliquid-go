package hyperliquid

// Withdraw off-ramps USDC from L1 to destination.
func (t *Trader) Withdraw(amount float64, destination string) (*TransferResponse, error) {
	return t.WithdrawFromBridge(amount, destination)
}
