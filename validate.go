package hyperliquid

// validate is the single pre-flight check called from Trader.place() and
// Trader.placeMany() before any signing. The full rule set is implemented
// in a later commit; this stub returns nil so the placement verbs can be
// wired up first.
func validate(spec *OrderSpec, info *Info) error {
	if spec == nil || spec.SkipValidate {
		return nil
	}
	return nil
}
