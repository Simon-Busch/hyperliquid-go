package hyperliquid

// TransferGroup exposes transfer-related Trader actions in a sub-grouped
// namespace. Reach it via t.Transfer (initialised lazily).
type TransferGroup struct {
	t *Trader
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
