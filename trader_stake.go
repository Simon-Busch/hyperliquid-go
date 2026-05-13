package hyperliquid

// StakeGroup exposes HYPE staking actions on Trader.
type StakeGroup struct {
	t *Trader
}

// Stake returns the staking sub-group.
func (t *Trader) Stake() *StakeGroup { return &StakeGroup{t: t} }

// Delegate stakes wei units of HYPE to validator.
func (g *StakeGroup) Delegate(validator string, wei int) (*TransferResponse, error) {
	return g.t.TokenDelegate(validator, wei, false)
}

// Undelegate unstakes wei units of HYPE from validator.
func (g *StakeGroup) Undelegate(validator string, wei int) (*TransferResponse, error) {
	return g.t.TokenDelegate(validator, wei, true)
}
