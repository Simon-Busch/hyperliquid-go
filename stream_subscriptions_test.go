package hyperliquid

import "testing"

// TestSubscriptionConstructors covers every typed Stream subscription
// constructor. Each builds a Subscription value matching the wire shape
// expected by the Hyperliquid WS API.
func TestSubscriptionConstructors(t *testing.T) {
	cases := []struct {
		name string
		got  subscriptionFilter
		want subscriptionFilter
	}{
		{"Trades", Trades("BTC"), subscriptionFilter{Type: "trades", Coin: "BTC"}},
		{"Book", Book("ETH"), subscriptionFilter{Type: "l2Book", Coin: "ETH"}},
		{"BBO", BBO("SOL"), subscriptionFilter{Type: "bbo", Coin: "SOL"}},
		{"ActiveAssetCtx", ActiveAssetCtx("BTC"), subscriptionFilter{Type: "activeAssetCtx", Coin: "BTC"}},
		{"Candles", Candles("BTC", "1m"), subscriptionFilter{Type: "candle", Coin: "BTC", Interval: "1m"}},
		{"AllMids", AllMids(), subscriptionFilter{Type: "allMids"}},
		{"AllMidsOn", AllMidsOn("flx"), subscriptionFilter{Type: "allMids", Dex: "flx"}},
		{"UserEvents", UserEvents("0xabc"), subscriptionFilter{Type: "userEvents", User: "0xabc"}},
		{"UserFills", UserFills("0xabc"), subscriptionFilter{Type: "userFills", User: "0xabc"}},
		{"OrderUpdates", OrderUpdates("0xabc"), subscriptionFilter{Type: "orderUpdates", User: "0xabc"}},
		{"UserFundings", UserFundings("0xabc"), subscriptionFilter{Type: "userFundings", User: "0xabc"}},
		{"UserLedger", UserLedger("0xabc"), subscriptionFilter{Type: "userNonFundingLedgerUpdates", User: "0xabc"}},
		{"WebData", WebData("0xabc"), subscriptionFilter{Type: "webData2", User: "0xabc"}},
		{"Notifications", Notifications("0xabc"), subscriptionFilter{Type: "notification", User: "0xabc"}},
		{"ActiveAssetData", ActiveAssetData("0xabc", "BTC"), subscriptionFilter{Type: "activeAssetData", User: "0xabc", Coin: "BTC"}},
		{"UserTwapFills", UserTwapFills("0xabc"), subscriptionFilter{Type: "userTwapSliceFills", User: "0xabc"}},
		{"UserTwapHistory", UserTwapHistory("0xabc"), subscriptionFilter{Type: "userTwapHistory", User: "0xabc"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %+v, want %+v", tc.got, tc.want)
			}
		})
	}
}

func TestSubscriptionKey(t *testing.T) {
	cases := []struct {
		name string
		sub  subscriptionFilter
		want subKey
	}{
		{"allMids", AllMids(), subKey{typ: "allMids"}},
		{"trades BTC", Trades("BTC"), subKey{typ: "trades", coin: "BTC"}},
		{"userEvents 0xabc", UserEvents("0xabc"), subKey{typ: "userEvents", user: "0xabc"}},
		{"candle BTC 1h", Candles("BTC", "1h"), subKey{typ: "candle", coin: "BTC", interval: "1h"}},
		{"allMids flx", AllMidsOn("flx"), subKey{typ: "allMids", dex: "flx"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.sub.key(); got != tc.want {
				t.Errorf("key() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
