package examples

import (
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Simon-Busch/hyperliquid-go"
	"github.com/joho/godotenv"
)

// Mainnet integration tests. Double-gated: HL_INTEGRATION=1 AND HL_PRIVATE_KEY
// must both be set so a wrong-foot CI run can never touch real funds.
//
// These exercise the pure-Go signing pipeline end-to-end against the real
// /exchange endpoint. The actions chosen are reversible and free of cost:
//   - UsdClassTransfer round-trip (perp -> spot -> perp), $1
//   - Limit order far from market price, then cancel
//
// Run with:
//   HL_INTEGRATION=1 HL_PRIVATE_KEY=0x... HL_ACCOUNT_ADDRESS=0x... \
//     go test ./examples -run TestIntegrationMainnet -v -count=1

func requireMainnetIntegration(t *testing.T) {
	t.Helper()
	godotenv.Overload("../.env")
	if os.Getenv("HL_INTEGRATION") != "1" {
		t.Skip("skipping: set HL_INTEGRATION=1 to run mainnet integration tests")
	}
	if os.Getenv("HL_PRIVATE_KEY") == "" {
		t.Skip("skipping: HL_PRIVATE_KEY not set")
	}
}

func perpWithdrawable(t *testing.T, ex *hyperliquid.Exchange, addr string) float64 {
	t.Helper()
	st, err := ex.GetInfo().UserState(addr)
	if err != nil {
		t.Fatalf("UserState: %v", err)
	}
	w, err := strconv.ParseFloat(st.Withdrawable, 64)
	if err != nil {
		t.Fatalf("parse withdrawable %q: %v", st.Withdrawable, err)
	}
	return w
}

// TestIntegrationMainnet_UsdClassTransferRoundTrip transfers $1 perp -> spot
// then $1 spot -> perp. Validates the user-signed EIP-712 signing path
// (HyperliquidTransaction:UsdClassTransfer) against the real exchange.
func TestIntegrationMainnet_UsdClassTransferRoundTrip(t *testing.T) {
	requireMainnetIntegration(t)
	ex := newTestExchange(t)
	addr := accountAddress(t)

	const amount = 1.0
	const tolerance = 0.001 // USDC

	before := perpWithdrawable(t, ex, addr)
	t.Logf("perp withdrawable before: %.6f USDC", before)
	if before < amount {
		t.Skipf("not enough USDC on perp to test: %.6f < %.6f", before, amount)
	}

	t.Logf("step 1/2: transfer $%.2f perp -> spot", amount)
	res, err := ex.UsdClassTransfer(amount, false /*toPerp*/)
	if err != nil {
		t.Fatalf("perp -> spot failed: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("perp -> spot status=%s err=%s", res.Status, res.Error)
	}

	// Brief settle pause; the API is consistent but balance reads may lag a tick.
	time.Sleep(500 * time.Millisecond)

	t.Logf("step 2/2: transfer $%.2f spot -> perp", amount)
	res, err = ex.UsdClassTransfer(amount, true /*toPerp*/)
	if err != nil {
		t.Fatalf("spot -> perp failed: %v", err)
	}
	if res.Status != "ok" {
		t.Fatalf("spot -> perp status=%s err=%s", res.Status, res.Error)
	}

	time.Sleep(500 * time.Millisecond)
	after := perpWithdrawable(t, ex, addr)
	t.Logf("perp withdrawable after: %.6f USDC (delta %+.6f)", after, after-before)

	if math.Abs(after-before) > tolerance {
		t.Fatalf("balance drift exceeds %.6f USDC: before=%.6f after=%.6f",
			tolerance, before, after)
	}
}

// btcPositionSize returns the user's signed BTC perp position size, or 0
// if no position exists. Positive = long, negative = short.
func btcPositionSize(t *testing.T, ex *hyperliquid.Exchange, addr string) float64 {
	t.Helper()
	st, err := ex.GetInfo().UserState(addr)
	if err != nil {
		t.Fatalf("UserState: %v", err)
	}
	for _, p := range st.AssetPositions {
		if p.Position.Coin == "BTC" {
			sz, err := strconv.ParseFloat(p.Position.Szi, 64)
			if err != nil {
				t.Fatalf("parse szi %q: %v", p.Position.Szi, err)
			}
			return sz
		}
	}
	return 0
}

// TestIntegrationMainnet_TakerRoundTrip opens then closes a tiny BTC long
// using IOC market orders. Validates the full taker L1 signing path: real
// fills and real (small) fees, net position back to zero.
//
// Safety: refuses to run if any BTC position already exists, so we never
// touch a real trading position.
func TestIntegrationMainnet_TakerRoundTrip(t *testing.T) {
	requireMainnetIntegration(t)
	ex := newTestExchange(t)
	addr := accountAddress(t)

	if sz := btcPositionSize(t, ex, addr); sz != 0 {
		t.Skipf("existing BTC position size %g — refusing to interfere", sz)
	}

	const size = 0.0002    // ~$16 notional at ~$80k BTC; clears the $10 min.
	const slippage = 0.005 // 0.5% — tight, taker fills should hit closer.

	t.Logf("step 1/2: market BUY %g BTC (IOC, slippage %.1f%%)", size, slippage*100)
	openStatus, err := ex.MarketOpen("BTC", true, size, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen: %v", err)
	}
	if openStatus.Error != nil {
		t.Fatalf("exchange rejected open: %s", *openStatus.Error)
	}
	if openStatus.Filled == nil {
		t.Fatalf("expected fill, got %+v", openStatus)
	}
	t.Logf("opened: oid=%d filled %s @ avg %s",
		openStatus.Filled.Oid, openStatus.Filled.TotalSz, openStatus.Filled.AvgPx)

	// Verify position exists before attempting to close.
	if sz := btcPositionSize(t, ex, addr); sz <= 0 {
		t.Fatalf("position not visible after fill: szi=%g", sz)
	}

	t.Logf("step 2/2: market close BTC")
	closeStatus, err := ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose: %v", err)
	}
	if closeStatus.Error != nil {
		t.Fatalf("exchange rejected close: %s", *closeStatus.Error)
	}
	if closeStatus.Filled == nil {
		t.Fatalf("expected close fill, got %+v", closeStatus)
	}
	t.Logf("closed: oid=%d filled %s @ avg %s",
		closeStatus.Filled.Oid, closeStatus.Filled.TotalSz, closeStatus.Filled.AvgPx)

	// Settle, then assert flat.
	time.Sleep(500 * time.Millisecond)
	if sz := btcPositionSize(t, ex, addr); sz != 0 {
		t.Fatalf("expected flat position after close, got szi=%g", sz)
	}
	t.Logf("position back to flat")
}

// TestIntegrationMainnet_TriggerMarketTP opens a tiny long with a
// take-profit-MARKET trigger attached in the same grouped action
// (MarketOpenWithSLTP -> bulkOrdersWithGrouping=normalTpsl). The TP is set
// 10% above mid so it cannot trigger. Then cancels the TP and closes.
//
// Validates two paths not covered by earlier tests:
//   - Grouped action signing (Grouping=normalTpsl)
//   - Trigger order wire (IsMarket=true, Tpsl="tp")
func TestIntegrationMainnet_TriggerMarketTP(t *testing.T) {
	requireMainnetIntegration(t)
	ex := newTestExchange(t)
	addr := accountAddress(t)

	if sz := btcPositionSize(t, ex, addr); sz != 0 {
		t.Skipf("existing BTC position size %g — refusing to interfere", sz)
	}

	const size = 0.0002
	const slippage = 0.005
	const tpDistance = 0.10 // 10% above entry — cannot trigger in a short test.

	t.Logf("step 1/3: market BUY %g BTC with TP-market +%.0f%%", size, tpDistance*100)
	resp, err := ex.MarketOpenWithSLTP(
		"BTC", true /*isBuy*/, size, nil, slippage,
		tpDistance, true /*isTP*/, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("MarketOpenWithSLTP: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("grouped action failed: %s", resp.Err)
	}
	if len(resp.Data.Statuses) < 2 {
		t.Fatalf("expected 2 statuses (open + tp), got %d", len(resp.Data.Statuses))
	}

	// Status[0] = the IOC open (filled). Status[1] = the TP trigger (resting).
	var openSt, tpSt hyperliquid.OrderStatus
	if err := resp.Data.Statuses[0].Parse(&openSt); err != nil {
		t.Fatalf("parse open status: %v", err)
	}
	if openSt.Error != nil {
		t.Fatalf("exchange rejected open: %s", *openSt.Error)
	}
	if openSt.Filled == nil {
		t.Fatalf("expected open to fill, got %+v", openSt)
	}
	t.Logf("opened: oid=%d filled %s @ avg %s",
		openSt.Filled.Oid, openSt.Filled.TotalSz, openSt.Filled.AvgPx)

	if err := resp.Data.Statuses[1].Parse(&tpSt); err != nil {
		t.Fatalf("parse tp status: %v", err)
	}
	if tpSt.Error != nil {
		// Cleanup before failing — leaving a position open is worse than the test.
		_, _ = ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
		t.Fatalf("exchange rejected tp trigger: %s", *tpSt.Error)
	}
	if tpSt.Resting == nil {
		_, _ = ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
		t.Fatalf("expected tp to rest, got %+v", tpSt)
	}
	tpOid := tpSt.Resting.Oid
	t.Logf("tp trigger resting: oid=%d", tpOid)

	t.Logf("step 2/3: cancel tp trigger oid=%d", tpOid)
	if cancelResp, err := ex.Cancel("BTC", tpOid); err != nil {
		_, _ = ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
		t.Fatalf("Cancel tp: %v", err)
	} else if !cancelResp.Ok {
		_, _ = ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
		t.Fatalf("cancel tp failed: %s", cancelResp.Err)
	}

	t.Logf("step 3/3: market close position")
	closeSt, err := ex.MarketClose("BTC", nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose: %v", err)
	}
	if closeSt.Error != nil {
		t.Fatalf("close rejected: %s", *closeSt.Error)
	}
	if closeSt.Filled == nil {
		t.Fatalf("expected close to fill, got %+v", closeSt)
	}
	t.Logf("closed: oid=%d filled %s @ avg %s",
		closeSt.Filled.Oid, closeSt.Filled.TotalSz, closeSt.Filled.AvgPx)

	time.Sleep(500 * time.Millisecond)
	if sz := btcPositionSize(t, ex, addr); sz != 0 {
		t.Fatalf("expected flat position after close, got szi=%g", sz)
	}
	t.Logf("position flat, all order paths exercised")
}

// TestIntegrationMainnet_PlaceAndCancelLimitOrder places a BTC perp buy
// far below market (cannot fill) then cancels it. Validates the L1 signing
// path for both Order and Cancel actions.
func TestIntegrationMainnet_PlaceAndCancelLimitOrder(t *testing.T) {
	requireMainnetIntegration(t)
	ex := newTestExchange(t)

	mids, err := ex.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("AllMids: %v", err)
	}
	mid, ok := mids["BTC"]
	if !ok {
		t.Fatal("no BTC mid")
	}
	midPx, err := strconv.ParseFloat(mid, 64)
	if err != nil {
		t.Fatalf("parse BTC mid %q: %v", mid, err)
	}
	// 1% below mid keeps us inside the price-band, ALO (post-only) prevents
	// taking liquidity even if the book moves while we're cancelling.
	limitPx := math.Round(midPx * 0.99)
	const size = 0.001 // ~$80 notional, clears the $10 min.
	t.Logf("BTC mid=%.2f, placing ALO buy at %.2f size=%g", midPx, limitPx, size)

	status, err := ex.Order(hyperliquid.CreateOrderRequest{
		Coin:       "BTC",
		IsBuy:      true,
		Price:      limitPx,
		Size:       size,
		ReduceOnly: false,
		OrderType:  hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifAlo}},
	}, nil)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	if status.Error != nil {
		t.Fatalf("exchange rejected order: %s", *status.Error)
	}
	if status.Resting == nil {
		t.Fatalf("expected resting order, got %+v", status)
	}
	oid := status.Resting.Oid
	t.Logf("placed: oid=%d", oid)

	cancelResp, err := ex.Cancel("BTC", oid)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !cancelResp.Ok {
		t.Fatalf("cancel failed: %s", cancelResp.Err)
	}
	t.Logf("cancelled oid=%d ok", oid)
}
