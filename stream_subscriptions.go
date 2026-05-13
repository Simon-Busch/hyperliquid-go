package hyperliquid

// Trades returns a Subscription for the trades feed of coin.
func Trades(coin string) Subscription {
	return Subscription{Type: "trades", Coin: coin}
}

// BookSub returns a Subscription for the L2 order book of coin. Named
// BookSub (rather than Book) because the package already exports an L2Book
// type used in REST responses; the suffix prevents type/identifier
// collision while keeping intent clear.
func BookSub(coin string) Subscription {
	return Subscription{Type: "l2Book", Coin: coin}
}

// BBO returns a Subscription for the best bid/offer of coin.
func BBO(coin string) Subscription {
	return Subscription{Type: "bbo", Coin: coin}
}

// AssetCtxSub returns a Subscription for the active-asset-context feed of
// coin. Named with a Sub suffix to avoid collision with the AssetCtx type.
func AssetCtxSub(coin string) Subscription {
	return Subscription{Type: "activeAssetCtx", Coin: coin}
}

// CandlesSub returns a Subscription for coin candles at the supplied
// interval (e.g. "1m", "5m", "1h"). Named CandlesSub to avoid clashing
// with the existing Candles type.
func CandlesSub(coin, interval string) Subscription {
	return Subscription{Type: "candle", Coin: coin, Interval: interval}
}

// AllMidsSub returns a Subscription for the global all-mids feed.
func AllMidsSub() Subscription {
	return Subscription{Type: "allMids"}
}

// AllMidsOn returns a Subscription for the all-mids feed pinned to a
// HIP-3 dex.
func AllMidsOn(dex string) Subscription {
	return Subscription{Type: "allMids", Dex: dex}
}

// UserEvents returns a Subscription for the per-user events stream.
func UserEvents(addr string) Subscription {
	return Subscription{Type: "userEvents", User: addr}
}

// UserFills returns a Subscription for the per-user fills stream.
func UserFills(addr string) Subscription {
	return Subscription{Type: "userFills", User: addr}
}

// OrderUpdates returns a Subscription for the per-user order updates stream.
func OrderUpdates(addr string) Subscription {
	return Subscription{Type: "orderUpdates", User: addr}
}

// UserFundings returns a Subscription for the per-user funding stream.
func UserFundings(addr string) Subscription {
	return Subscription{Type: "userFundings", User: addr}
}

// UserLedger returns a Subscription for the per-user non-funding ledger.
func UserLedger(addr string) Subscription {
	return Subscription{Type: "userNonFundingLedgerUpdates", User: addr}
}

// WebData returns a Subscription for the per-user web-data feed
// (formerly WebData2).
func WebData(addr string) Subscription {
	return Subscription{Type: "webData2", User: addr}
}

// Notifications returns a Subscription for the per-user notifications stream.
func Notifications(addr string) Subscription {
	return Subscription{Type: "notification", User: addr}
}

// ActiveAsset returns a Subscription for active-asset data for addr / coin.
func ActiveAsset(addr, coin string) Subscription {
	return Subscription{Type: "activeAssetData", User: addr, Coin: coin}
}

// UserTwapFills returns a Subscription for the per-user TWAP fills stream.
func UserTwapFills(addr string) Subscription {
	return Subscription{Type: "userTwapSliceFills", User: addr}
}

// UserTwapHistory returns a Subscription for the per-user TWAP history stream.
func UserTwapHistory(addr string) Subscription {
	return Subscription{Type: "userTwapHistory", User: addr}
}
