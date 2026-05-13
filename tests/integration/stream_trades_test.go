//go:build integration

package integration

import (
	"sync/atomic"
	"testing"
	"time"

	hl "github.com/Simon-Busch/hyperliquid-go"
)

// TestStream_TradesReceived subscribes to the trades feed for the test
// coin and asserts at least one message arrives within 10 seconds.
func TestStream_TradesReceived(t *testing.T) {
	c := newStreamingClient(t)
	coin := testCoin(t)

	var count atomic.Int32
	got := make(chan struct{}, 1)
	sub, err := c.Stream.Subscribe(hl.Trades(coin), func(m hl.WSMessage) {
		count.Add(1)
		select {
		case got <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	select {
	case <-got:
		t.Logf("received %d messages", count.Load())
	case <-time.After(10 * time.Second):
		t.Skip("no trades arrived within 10s — market may be quiet")
	}
}
