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

// testSize returns the integration order size, capped so it is at least
// the asset's MinSize.
func testSize(t *testing.T, client *hl.Client, coin string) float64 {
	t.Helper()
	c, _ := loadConfig()
	meta, err := client.Info.Asset(coin)
	if err != nil || meta.MinSize == 0 {
		return c.TestSize
	}
	if c.TestSize < meta.MinSize {
		return meta.MinSize
	}
	// Snap to the size step.
	steps := math.Round(c.TestSize / meta.MinSize)
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

// skipIfNoBalance skips the test when the account has no withdrawable
// USDC, since trading scenarios cannot meaningfully execute.
func skipIfNoBalance(t *testing.T, client *hl.Client) {
	t.Helper()
	c, _ := loadConfig()
	state, err := client.Info.UserState(c.AccountAddr)
	if err != nil {
		t.Skipf("UserState failed: %v", err)
	}
	w, err := strconv.ParseFloat(state.Withdrawable, 64)
	if err != nil || w <= 0 {
		t.Skipf("account %s has no withdrawable balance (%s)", c.AccountAddr, state.Withdrawable)
	}
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

// snapPrice rounds px to the asset's tick size so the order does not
// fail the 5-significant-figure / tick-alignment validators.
func snapPrice(px float64, client *hl.Client, coin string) float64 {
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
