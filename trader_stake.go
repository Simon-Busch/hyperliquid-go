package hyperliquid

import "time"

// StakeGroup exposes HYPE staking actions on Trader.
type StakeGroup struct {
	t *Trader
}

// TokenDelegate delegates (or undelegates) HYPE stake.
func (e *Trader) TokenDelegate(
	validator string,
	wei int,
	isUndelegate bool,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":         "tokenDelegate",
		"validator":    validator,
		"wei":          int64(wei),
		"isUndelegate": isUndelegate,
		"nonce":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, tokenDelegateSignTypes,
		"HyperliquidTransaction:TokenDelegate", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delegate stakes wei units of HYPE to validator.
func (g *StakeGroup) Delegate(validator string, wei int) (*TransferResponse, error) {
	return g.t.TokenDelegate(validator, wei, false)
}

// Undelegate unstakes wei units of HYPE from validator.
func (g *StakeGroup) Undelegate(validator string, wei int) (*TransferResponse, error) {
	return g.t.TokenDelegate(validator, wei, true)
}
