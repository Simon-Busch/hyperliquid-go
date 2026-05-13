//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	hl "github.com/Simon-Busch/hyperliquid-go"
)

// TestApproveAgent_AndPlace generates a fresh agent via the owner's key,
// builds a second Client signing with the agent's private key but acting
// on behalf of the owner's account, places a far-from-mid ALO and
// cancels it.
func TestApproveAgent_AndPlace(t *testing.T) {
	owner := newClient(t)
	skipIfNoBalance(t, owner)

	name := fmt.Sprintf("itest-%d", time.Now().UnixMilli())
	agent, err := owner.Trade.ApproveAgent(name)
	if err != nil {
		t.Fatalf("ApproveAgent: %v", err)
	}
	if agent.Address == "" || agent.PrivateKey == nil {
		t.Fatalf("ApproveAgent returned blank agent: %+v", agent)
	}

	cfg, _ := loadConfig()
	agentClient, err := hl.New(
		hl.WithBaseURL(cfg.BaseURL),
		hl.WithPrivateKey(agent.PrivateKey),
		hl.WithAccount(cfg.AccountAddr),
		hl.WithSkipStream(true),
	)
	if err != nil {
		t.Fatalf("hl.New(agent): %v", err)
	}

	coin := testCoin(t)
	size := testSize(t, agentClient, coin)
	m := mid(t, agentClient, coin)
	px := snapPrice(m*0.5, agentClient, coin)

	res, err := agentClient.Trade.PlaceALO(coin, hl.Buy, size, px)
	if err != nil {
		t.Fatalf("PlaceALO via agent: %v", err)
	}
	t.Cleanup(func() { cancelIfResting(t, agentClient, coin, res.OID) })
	if res.OID == 0 {
		t.Fatalf("PlaceALO returned no oid: %+v", res)
	}
	if _, err := agentClient.Trade.Cancel(coin, res.OID); err != nil {
		t.Fatalf("Cancel via agent: %v", err)
	}
}
