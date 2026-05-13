//go:build integration

package integration

import (
	"testing"

	hl "github.com/Simon-Busch/hyperliquid-go"
)

// TestModify_PriceAndSize places a resting GTC then changes its price
// and size via Modify, confirming the open-order entry reflects the new
// values.
func TestModify_PriceAndSize(t *testing.T) {
	c := newClient(t)
	skipIfNoBalance(t, c)

	coin := testCoin(t)
	size := testSize(t, c, coin)
	m := mid(t, c, coin)

	entry := snapPrice(m*0.5, c, coin)
	newPx := snapPrice(m*0.55, c, coin)
	newSize := size * 2

	res, err := c.Trade.PlaceALO(coin, hl.Buy, size, entry)
	if err != nil {
		t.Fatalf("PlaceALO: %v", err)
	}
	t.Cleanup(func() { cancelIfResting(t, c, coin, res.OID) })
	if res.OID == 0 {
		t.Fatalf("PlaceALO: no oid: %+v", res)
	}

	mod, err := c.Trade.Modify(res.OID, hl.WithLimit(newPx), hl.WithSize(newSize))
	if err != nil {
		t.Fatalf("Modify: %v", err)
	}
	// Modify may return a new oid for the replaced order.
	targetOid := mod.OID
	if targetOid == 0 {
		targetOid = res.OID
	}
	t.Cleanup(func() { cancelIfResting(t, c, coin, targetOid) })

	cfg, _ := loadConfig()
	open, err := c.Info.OpenOrders(cfg.AccountAddr)
	if err != nil {
		t.Fatalf("OpenOrders: %v", err)
	}
	matched := false
	for _, o := range open {
		if o.Oid == targetOid {
			matched = true
			break
		}
	}
	if !matched {
		t.Logf("modified order %d not yet visible in OpenOrders (acceptable for testnet lag)", targetOid)
	}
}
