package hyperliquid

// SubAccountGroup exposes sub-account management on Trader.
type SubAccountGroup struct {
	t *Trader
}

// SubAccount returns the sub-account management handle.
func (t *Trader) SubAccount() *SubAccountGroup { return &SubAccountGroup{t: t} }

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
