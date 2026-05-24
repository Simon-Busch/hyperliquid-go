package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"strconv"
)

// validate is the single pre-flight check called from Trader.place() and
// Trader.placeMany() before any signing. Each rule maps to a stable Code
// on the returned ValidationError so callers can branch via errors.As.
//
// On every invocation the cached UserState is refreshed via the Info HTTP
// path (one round-trip). Callers in latency-sensitive paths can disable
// the entire check by attaching SkipValidation() to the spec.
//
// The rules implemented here mirror section 9 of the spec: size > 0,
// price > 0 where required, five-significant-figure cap, reduce-only
// direction sanity, bracket placement vs side, close direction / size, and
// option/method compatibility.
func (t *Trader) validate(spec *OrderSpec) error {
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
	if t != nil && t.info != nil {
		asset := t.info.AssetID(spec.Coin)
		if asset == 0 && !isFirstAsset(t.info, spec.Coin) {
			return &ValidationError{Field: "Coin", Code: "unknown_coin", Got: spec.Coin}
		}
	}
	if spec.Size <= 0 && spec.Method != "close" {
		return &ValidationError{Field: "Size", Code: "size_below_min", Got: spec.Size}
	}
	if spec.Size > 0 && t != nil && t.info != nil {
		if meta, err := t.info.Asset(spec.Coin); err == nil && meta.MinSize > 0 {
			if spec.Size < meta.MinSize {
				return &ValidationError{Field: "Size", Code: "size_below_min", Got: spec.Size, Want: meta.MinSize}
			}
			if !isMultipleOf(spec.Size, meta.MinSize) {
				return &ValidationError{Field: "Size", Code: "size_step_violation", Got: spec.Size, Want: meta.MinSize}
			}
		}
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

	// Position-state rules. RefreshState must succeed; if the SDK cannot
	// fetch the caller's state we fail closed rather than silently skip
	// the long/short safety checks. Callers that cannot reach the API or
	// want to bypass these rules must pass hl.SkipValidation() per call.
	if t != nil {
		if err := t.RefreshState(context.Background()); err != nil {
			return fmt.Errorf("validate: refresh user state: %w (use hl.SkipValidation() to bypass)", err)
		}
		if err := t.validatePositionState(spec); err != nil {
			return err
		}
	}
	return nil
}

// validatePositionState enforces the reduce-only direction, close
// direction, and close size rules using the cached UserState. If the
// cache is nil (refresh never ran or the account has never traded) the
// rules are skipped; the cache is guaranteed non-nil by the caller in
// validate, which propagates any RefreshState failure.
func (t *Trader) validatePositionState(spec *OrderSpec) error {
	state := t.cachedUserState()
	if state == nil {
		return nil
	}
	pos, szi := positionFor(state, spec.Coin)

	if spec.ReduceOnly && pos != nil {
		if spec.Side.IsBuy() && szi > 0 {
			return &ValidationError{Field: "ReduceOnly", Code: "wrong_side_for_reduce", Message: "Buy reduce-only on a long position would increase exposure"}
		}
		if !spec.Side.IsBuy() && szi < 0 {
			return &ValidationError{Field: "ReduceOnly", Code: "wrong_side_for_reduce", Message: "Sell reduce-only on a short position would increase exposure"}
		}
	}

	if spec.Method == "close" {
		if pos == nil || szi == 0 {
			return &ValidationError{Field: "Coin", Code: "no_position", Message: "no open position to close"}
		}
		if spec.OverrideSize > 0 && spec.OverrideSize > math.Abs(szi) {
			return &ValidationError{Field: "Size", Code: "close_size_exceeds_position", Got: spec.OverrideSize, Want: math.Abs(szi)}
		}
	}
	return nil
}

// positionFor returns the position struct and signed size for coin within
// state, or (nil, 0) if no such position exists.
func positionFor(state *UserState, coin string) (*Position, float64) {
	for i := range state.AssetPositions {
		p := &state.AssetPositions[i].Position
		if p.Coin == coin {
			szi, _ := strconv.ParseFloat(p.Szi, 64)
			return p, szi
		}
	}
	return nil, 0
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

// validateBracket enforces TP/SL placement rules relative to entry side
// and ensures bracket leg sizes do not exceed the entry order size.
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
		return &ValidationError{Field: "TPSize", Code: "bracket_size_exceeds_entry", Got: spec.TPSize, Want: spec.Size}
	}
	if spec.SLSize > 0 && spec.Size > 0 && spec.SLSize > spec.Size {
		return &ValidationError{Field: "SLSize", Code: "bracket_size_exceeds_entry", Got: spec.SLSize, Want: spec.Size}
	}
	return nil
}

// isMultipleOf reports whether x is a positive integer multiple of step,
// within a small floating-point tolerance.
func isMultipleOf(x, step float64) bool {
	if step <= 0 {
		return true
	}
	q := x / step
	return math.Abs(q-math.Round(q)) < 1e-9
}

// isFirstAsset returns true if coin is the first asset in info's mapping,
// which legitimately has id 0.
func isFirstAsset(info *Info, coin string) bool {
	if info == nil {
		return false
	}
	for c, id := range info.CoinToAssetMap() {
		if id == 0 && c == coin {
			return true
		}
	}
	return false
}
