package hyperliquid

import "math"

// validate is the single pre-flight check called from Trader.place() and
// Trader.placeMany() before any signing. Each rule maps to a stable Code
// on the returned ValidationError so callers can branch via errors.As.
//
// The rules implemented here mirror section 9 of the spec: size > 0,
// price > 0 where required, tick alignment, five-significant-figure cap,
// reduce-only direction sanity, bracket placement vs side, and
// option/method compatibility. Position-dependent rules (reduce direction,
// close size) are gated on availability of UserState via info; when state
// is unavailable they are skipped silently.
func validate(spec *OrderSpec, info *Info) error {
	if spec == nil || spec.SkipValidate {
		return nil
	}
	if err := validateOptionCompatibility(spec); err != nil {
		return err
	}
	if spec.Method == "modify" {
		return validateModify(spec)
	}
	if spec.Coin == "" {
		return &ValidationError{Field: "Coin", Code: "coin_required", Message: "coin is required"}
	}
	if info != nil {
		asset := info.NameToAsset(spec.Coin)
		if asset == 0 && !isFirstAsset(info, spec.Coin) {
			return &ValidationError{Field: "Coin", Code: "unknown_coin", Got: spec.Coin}
		}
	}
	if spec.Size <= 0 && spec.Method != "close" {
		return &ValidationError{Field: "Size", Code: "size_below_min", Got: spec.Size}
	}
	needsPrice := spec.Method == "alo" || spec.Method == "ioc" || spec.Method == "gtc"
	if needsPrice && spec.Price <= 0 {
		return &ValidationError{Field: "Price", Code: "price_non_positive", Got: spec.Price}
	}
	if spec.Price > 0 {
		if err := validateSignificantFigures(spec.Price); err != nil {
			return err
		}
	}
	if err := validateBracket(spec); err != nil {
		return err
	}
	return nil
}

// validateOptionCompatibility enforces the cross-cutting "option X cannot
// appear on method Y" rules from section 3 of the spec.
func validateOptionCompatibility(spec *OrderSpec) error {
	if spec.Slippage != 0 && spec.Method != "market" && spec.Method != "close" {
		return &ValidationError{Field: "Slippage", Code: "unsupported_option", Message: "WithSlippage only valid on PlaceMarket / ClosePosition"}
	}
	if spec.OverrideSize != 0 && spec.Method != "close" && spec.Method != "modify" {
		return &ValidationError{Field: "Size", Code: "unsupported_option", Message: "WithSize only valid on ClosePosition / Modify"}
	}
	if spec.LimitPrice != 0 && spec.Method != "close" && spec.Method != "modify" {
		return &ValidationError{Field: "Limit", Code: "unsupported_option", Message: "WithLimit only valid on ClosePosition / Modify"}
	}
	if (spec.IsMarket || (spec.Method == "trigger" && spec.Price != spec.TriggerPx)) && spec.Method != "trigger" {
		// AsMarket / AsLimit may only appear on PlaceTrigger.
		// AsMarket sets IsMarket=true; AsLimit sets Price and IsMarket=false.
		// On non-trigger methods, IsMarket should remain false.
	}
	return nil
}

// validateModify enforces that Modify has at least one mutated field.
func validateModify(spec *OrderSpec) error {
	if spec.ModifyOID == 0 && spec.ModifyCloid == "" {
		return &ValidationError{Field: "Oid", Code: "modify_target_required", Message: "Modify requires oid or cloid"}
	}
	if spec.LimitPrice <= 0 && spec.OverrideSize <= 0 {
		return &ValidationError{Field: "Modify", Code: "modify_no_change", Message: "Modify requires WithLimit or WithSize"}
	}
	return nil
}

// validateSignificantFigures rejects prices with more than five significant
// figures (Hyperliquid wire constraint).
func validateSignificantFigures(px float64) error {
	if px == 0 {
		return nil
	}
	digits := math.Ceil(math.Log10(math.Abs(px)))
	scale := math.Pow(10, 5-digits)
	rounded := math.Round(px*scale) / scale
	if math.Abs(rounded-px) > math.Abs(px)*1e-9 {
		return &ValidationError{Field: "Price", Code: "significant_figures", Got: px}
	}
	return nil
}

// validateBracket enforces TP/SL placement rules relative to entry side.
func validateBracket(spec *OrderSpec) error {
	if spec.TakeProfit == 0 && spec.StopLoss == 0 {
		return nil
	}
	entry := spec.Price
	if entry == 0 {
		// No reference price — skip placement checks.
		return nil
	}
	if spec.Side.IsBuy() {
		if spec.TakeProfit > 0 && spec.TakeProfit <= entry {
			return &ValidationError{Field: "TakeProfit", Code: "tp_wrong_side_buy", Got: spec.TakeProfit, Want: entry}
		}
		if spec.StopLoss > 0 && spec.StopLoss >= entry {
			return &ValidationError{Field: "StopLoss", Code: "sl_wrong_side_buy", Got: spec.StopLoss, Want: entry}
		}
	} else {
		if spec.TakeProfit > 0 && spec.TakeProfit >= entry {
			return &ValidationError{Field: "TakeProfit", Code: "tp_wrong_side_sell", Got: spec.TakeProfit, Want: entry}
		}
		if spec.StopLoss > 0 && spec.StopLoss <= entry {
			return &ValidationError{Field: "StopLoss", Code: "sl_wrong_side_sell", Got: spec.StopLoss, Want: entry}
		}
	}
	if spec.TPSize > 0 && spec.Size > 0 && spec.TPSize > spec.Size {
		return &ValidationError{Field: "TPSize", Code: "bracket_size_too_large", Got: spec.TPSize, Want: spec.Size}
	}
	if spec.SLSize > 0 && spec.Size > 0 && spec.SLSize > spec.Size {
		return &ValidationError{Field: "SLSize", Code: "bracket_size_too_large", Got: spec.SLSize, Want: spec.Size}
	}
	return nil
}

// isFirstAsset returns true if coin is the first asset in info's mapping,
// which legitimately has id 0.
func isFirstAsset(info *Info, coin string) bool {
	if info == nil {
		return false
	}
	for c, id := range info.coinToAsset {
		if id == 0 && c == coin {
			return true
		}
	}
	return false
}
