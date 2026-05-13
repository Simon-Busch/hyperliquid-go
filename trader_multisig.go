package hyperliquid

// MultiSigGroup exposes multi-sig conversion and execution actions.
type MultiSigGroup struct {
	t *Trader
}

// Convert converts the signing account to a multi-sig user authorising
// the supplied signer addresses with a threshold of valid signatures.
func (g *MultiSigGroup) Convert(authorized []string, threshold int) (*MultiSigConversionResponse, error) {
	return g.t.ConvertToMultiSigUser(authorized, threshold)
}

// Execute submits a multi-sig action signed by signers with the supplied
// hex signatures.
func (g *MultiSigGroup) Execute(action map[string]any, signers []string, signatures []string) (*MultiSigResponse, error) {
	return g.t.MultiSig(action, signers, signatures)
}
