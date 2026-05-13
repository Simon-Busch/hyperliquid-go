//go:build integration

package integration

import (
	"context"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	hl "github.com/Simon-Busch/hyperliquid-go"
)

// requireEnv fails the test if name is unset in the environment.
func requireEnv(t *testing.T, name string) string {
	t.Helper()
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		t.Skipf("env %s not set; skipping", name)
	}
	return v
}

// newClient builds a fresh hl.Client wired to the integration config. The
// client is not connected to the websocket by default; tests that need
// streaming call Stream.Connect explicitly.
func newClient(t *testing.T, opts ...hl.Option) *hl.Client {
	t.Helper()
	c, err := loadConfig()
	if err != nil {
		t.Skipf("integration env not configured: %v", err)
	}
	full := append([]hl.Option{
		hl.WithBaseURL(c.BaseURL),
		hl.WithPrivateKey(c.privateKey),
		hl.WithAccount(c.AccountAddr),
		hl.WithSkipStream(true),
	}, opts...)
	client, err := hl.New(full...)
	if err != nil {
		t.Fatalf("hl.New: %v", err)
	}
	return client
}

// newStreamingClient constructs a Client with the Stream enabled and
// connected, returning both the client and a deferred-close hook for
// t.Cleanup.
func newStreamingClient(t *testing.T) *hl.Client {
	t.Helper()
	c, err := loadConfig()
	if err != nil {
		t.Skipf("integration env not configured: %v", err)
	}
	if c.SkipWS {
		t.Skip("HL_SKIP_WS=true; skipping WS scenario")
	}
	client, err := hl.New(
		hl.WithBaseURL(c.BaseURL),
		hl.WithPrivateKey(c.privateKey),
		hl.WithAccount(c.AccountAddr),
	)
	if err != nil {
		t.Fatalf("hl.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Stream.Connect(ctx); err != nil {
		t.Fatalf("Stream.Connect: %v", err)
	}
	t.Cleanup(func() { _ = client.Stream.Close() })
	return client
}

// testCoin returns the integration coin under test.
func testCoin(t *testing.T) string {
	t.Helper()
	c, err := loadConfig()
	if err != nil {
		t.Skipf("integration env not configured: %v", err)
	}
	return c.TestCoin
}

// testSize returns the integration order size in coin units. When
// HL_TEST_NOTIONAL is set (default $10), the size is derived from the
// current mid so mainnet and testnet stay within the same USD budget.
// HL_TEST_SIZE is a fixed-size fallback used only when HL_TEST_NOTIONAL=0.
// The result is always snapped to the asset's MinSize step and clamped
// to at least one step.
func testSize(t *testing.T, client *hl.Client, coin string) float64 {
	t.Helper()
	c, _ := loadConfig()

	target := c.TestSize
	if c.TestNotional > 0 {
		px := mid(t, client, coin)
		if px > 0 {
			target = c.TestNotional / px
		}
	}

	meta, err := client.Info.Asset(coin)
	if err != nil || meta.MinSize == 0 {
		return target
	}
	if target < meta.MinSize {
		return meta.MinSize
	}
	// Snap UP to the size step. Rounding down could put us under the
	// venue's $10 minimum order value when the target sits between two
	// steps; the small overshoot is acceptable for an integration budget.
	steps := math.Ceil(target / meta.MinSize)
	if steps < 1 {
		steps = 1
	}
	return steps * meta.MinSize
}

// testSizeForLimit returns a coin-unit size such that size * limitPx is
// at least HL_TEST_NOTIONAL — used by tests that place orders far from
// mid (resting ALO/GTC, brackets, triggers) where sizing at mid would
// produce a sub-$10 notional and get rejected by the venue's minimum
// order-value rule. Snaps UP to the size step to ensure the threshold
// is cleared.
func testSizeForLimit(t *testing.T, client *hl.Client, coin string, limitPx float64) float64 {
	t.Helper()
	cfg, _ := loadConfig()
	if cfg.TestNotional <= 0 || limitPx <= 0 {
		return testSize(t, client, coin)
	}
	target := cfg.TestNotional / limitPx
	meta, err := client.Info.Asset(coin)
	if err != nil || meta.MinSize == 0 {
		return target
	}
	if target < meta.MinSize {
		return meta.MinSize
	}
	steps := math.Ceil(target / meta.MinSize)
	return steps * meta.MinSize
}

// mid returns the current mid price for coin.
func mid(t *testing.T, client *hl.Client, coin string) float64 {
	t.Helper()
	m, err := client.Info.AllMids()
	if err != nil {
		t.Fatalf("AllMids: %v", err)
	}
	raw, ok := m[coin]
	if !ok {
		t.Skipf("no mid for %s — skipping", coin)
	}
	px, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		t.Fatalf("parse mid %q: %v", raw, err)
	}
	return px
}

// skipIfNoBalance skips the test when the account has no perp margin
// to trade with. Withdrawable is the wrong field here: it is zero
// whenever margin is locked by an open position, even if the account
// has plenty of equity. AccountValue is the right semantic check for
// "can this account place perp orders".
func skipIfNoBalance(t *testing.T, client *hl.Client) {
	t.Helper()
	c, _ := loadConfig()
	state, err := client.Info.UserState(c.AccountAddr)
	if err != nil {
		t.Skipf("UserState failed: %v", err)
	}
	v, err := strconv.ParseFloat(state.MarginSummary.AccountValue, 64)
	if err != nil || v <= 0 {
		t.Skipf("account %s has no perp account value (%q); fund the perp wallet to run trading scenarios",
			c.AccountAddr, state.MarginSummary.AccountValue)
	}
}

// skipIfNoSpotBalance skips when the account has no spot USDC, used by
// scenarios that transfer or convert between spot and perp classes.
func skipIfNoSpotBalance(t *testing.T, client *hl.Client) {
	t.Helper()
	c, _ := loadConfig()
	spot, err := client.Info.SpotBalances(c.AccountAddr)
	if err != nil {
		t.Skipf("SpotBalances failed: %v", err)
	}
	for _, b := range spot.Balances {
		if v, err := strconv.ParseFloat(b.Total, 64); err == nil && v > 0 {
			return
		}
	}
	t.Skipf("account %s has no spot balance", c.AccountAddr)
}

// awaitPosition polls Info.Position until coin has a non-zero size or
// the timeout elapses. Returns the position or nil. Used by scenarios
// that need to act on a freshly-opened position when testnet/mainnet
// state propagation lags behind a market order ack.
func awaitPosition(t *testing.T, client *hl.Client, coin string, timeout time.Duration) *hl.Position {
	t.Helper()
	cfg, _ := loadConfig()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pos, err := client.Info.Position(cfg.AccountAddr, coin)
		if err == nil && pos != nil {
			if szi, perr := strconv.ParseFloat(pos.Szi, 64); perr == nil && szi != 0 {
				return pos
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}

// awaitFill polls Info.Fill for up to timeout, returning the first fill
// matching oid or nil if none seen.
func awaitFill(t *testing.T, client *hl.Client, oid int64, timeout time.Duration) bool {
	t.Helper()
	c, _ := loadConfig()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		fill, err := client.Info.Fill(c.AccountAddr, oid)
		if err == nil && fill != nil {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// snapPrice rounds px to a wire-valid Hyperliquid price: at most five
// significant figures AND a multiple of the asset's tick size. The
// significant-figure cap is the protocol rule; tick alignment is the
// per-asset constraint. Order matters: round to 5 sig figs first, then
// snap to tick — otherwise a tick-aligned price like 1250.25 (6 sig
// figs) sails through the tick check but fails the wire validator.
func snapPrice(px float64, client *hl.Client, coin string) float64 {
	if px <= 0 {
		return px
	}
	// 1. Round to 5 significant figures.
	digits := math.Ceil(math.Log10(math.Abs(px)))
	scale := math.Pow(10, 5-digits)
	px = math.Round(px*scale) / scale

	// 2. Snap to the asset's tick size.
	meta, err := client.Info.Asset(coin)
	if err != nil || meta.TickSize == 0 {
		return px
	}
	return math.Round(px/meta.TickSize) * meta.TickSize
}

// cancelIfResting attempts to cancel oid on coin, ignoring "order not
// found" errors so test cleanup is idempotent.
func cancelIfResting(t *testing.T, client *hl.Client, coin string, oid int64) {
	t.Helper()
	if oid == 0 {
		return
	}
	if _, err := client.Trade.Cancel(coin, oid); err != nil {
		t.Logf("Cancel(%s, %d): %v (best-effort)", coin, oid, err)
	}
}
