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
