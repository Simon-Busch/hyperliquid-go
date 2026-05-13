package hyperliquid

// PlaceALO places an add-liquidity-only (post-only) limit order. Required
// args are positional; everything else is supplied via options.
func (t *Trader) PlaceALO(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error) {
	spec := ALO(coin, side, size, px, opts...)
	return t.place(&spec)
}

// PlaceIOC places an immediate-or-cancel limit order.
func (t *Trader) PlaceIOC(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error) {
	spec := IOC(coin, side, size, px, opts...)
	return t.place(&spec)
}

// PlaceGTC places a good-til-cancel limit order.
func (t *Trader) PlaceGTC(coin string, side Side, size, px float64, opts ...PlaceOpt) (Result, error) {
	spec := GTC(coin, side, size, px, opts...)
	return t.place(&spec)
}

// ALO returns an OrderSpec describing a post-only limit order. Pass it to
// Trader.PlaceMany to batch multiple legs into one signed action.
func ALO(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec {
	s := OrderSpec{Method: "alo", Coin: coin, Side: side, Size: size, Price: px, TIF: tifALO}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// IOC returns an OrderSpec describing an immediate-or-cancel limit order.
func IOC(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec {
	s := OrderSpec{Method: "ioc", Coin: coin, Side: side, Size: size, Price: px, TIF: tifIOC}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// GTC returns an OrderSpec describing a good-til-cancel limit order.
func GTC(coin string, side Side, size, px float64, opts ...PlaceOpt) OrderSpec {
	s := OrderSpec{Method: "gtc", Coin: coin, Side: side, Size: size, Price: px, TIF: tifGTC}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// PlaceMarket places a market order, implemented as an IOC limit at the
// current mid price adjusted by the requested slippage fraction (default
// 5% if WithSlippage is not supplied).
func (t *Trader) PlaceMarket(coin string, side Side, size float64, opts ...PlaceOpt) (Result, error) {
	spec := Market(coin, side, size, opts...)
	slippage := spec.Slippage
	if slippage == 0 {
		slippage = 0.05
	}
	px, err := t.SlippagePrice(coin, side.IsBuy(), slippage, nil)
	if err != nil {
		return Result{}, err
	}
	spec.Price = px
	return t.place(&spec)
}

// Market returns an OrderSpec describing a market order. The Price field is
// resolved later (against mid) when the spec is consumed by PlaceMany or
// PlaceMarket; callers do not need to supply px.
func Market(coin string, side Side, size float64, opts ...PlaceOpt) OrderSpec {
	s := OrderSpec{Method: "market", Coin: coin, Side: side, Size: size, TIF: tifIOC}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// PlaceTrigger places a trigger order (stop-market by default, or
// stop-limit when AsLimit(px) is supplied).
func (t *Trader) PlaceTrigger(coin string, side Side, size, triggerPx float64, opts ...PlaceOpt) (Result, error) {
	spec := Trigger(coin, side, size, triggerPx, opts...)
	return t.place(&spec)
}

// ClosePosition flattens the caller's open position on coin. The order
// side is inferred from the position: long positions are exited with a
// sell, short with a buy. By default this is a market close; supply
// WithLimit(px) to close at a specific price and WithSize(x) to close
// partially.
func (t *Trader) ClosePosition(coin string, opts ...PlaceOpt) (Result, error) {
	addr := t.accountAddr
	if addr == "" {
		addr = t.vault
	}
	state, err := t.info.UserState(addr)
	if err != nil {
		return Result{}, err
	}
	var (
		found bool
		szi   float64
	)
	for _, ap := range state.AssetPositions {
		if ap.Position.Coin == coin {
			szi = parseFloat(ap.Position.Szi)
			found = true
			break
		}
	}
	if !found || szi == 0 {
		return Result{}, &ValidationError{Field: "Coin", Code: "no_position", Message: "no open position to close"}
	}

	isBuy := szi < 0
	size := abs(szi)

	// Apply options to a placeholder spec to surface WithSize/WithLimit/etc.
	tmp := OrderSpec{Method: "close", Coin: coin, Size: size, Side: sideFromIsBuy(isBuy), TIF: tifIOC}
	for _, o := range opts {
		o(&tmp)
	}
	if tmp.OverrideSize > 0 {
		tmp.Size = tmp.OverrideSize
	}

	var price float64
	if tmp.LimitPrice > 0 {
		price = tmp.LimitPrice
	} else {
		slip := tmp.Slippage
		if slip == 0 {
			slip = 0.05
		}
		price, err = t.SlippagePrice(coin, isBuy, slip, nil)
		if err != nil {
			return Result{}, err
		}
	}
	tmp.Price = price
	tmp.ReduceOnly = true
	return t.place(&tmp)
}

// sideFromIsBuy is a tiny adapter used while we still convert between the
// boolean wire encoding and the typed Side enum.
func sideFromIsBuy(isBuy bool) Side {
	if isBuy {
		return Buy
	}
	return Sell
}

// Trigger returns an OrderSpec describing a trigger order. Default fills
// as a market; combine with AsLimit(px) to fill as a limit.
func Trigger(coin string, side Side, size, triggerPx float64, opts ...PlaceOpt) OrderSpec {
	s := OrderSpec{
		Method:    "trigger",
		Coin:      coin,
		Side:      side,
		Size:      size,
		TriggerPx: triggerPx,
		Price:     triggerPx,
		IsMarket:  true,
	}
	for _, o := range opts {
		o(&s)
	}
	return s
}
