package stream

import (
	"encoding/json"
	"testing"
)

// webData2Fixture is a captured webData2 `data` payload, trimmed to the
// fields the SDK decodes. The account snapshot is under
// clearinghouseState (the correct key) and carries one fully-populated
// position so every position field is exercised on decode.
const webData2Fixture = `{
  "clearinghouseState": {
    "marginSummary": {
      "accountValue": "4609.76",
      "totalNtlPos": "1200.50",
      "totalRawUsd": "3409.26",
      "totalMarginUsed": "120.05"
    },
    "crossMarginSummary": {
      "accountValue": "4609.76",
      "totalNtlPos": "1200.50",
      "totalRawUsd": "3409.26",
      "totalMarginUsed": "120.05"
    },
    "crossMaintenanceMarginUsed": "48.02",
    "withdrawable": "4489.71",
    "assetPositions": [
      {
        "type": "oneWay",
        "position": {
          "coin": "BTC",
          "szi": "0.012",
          "entryPx": "100000.0",
          "positionValue": "1200.50",
          "unrealizedPnl": "0.50",
          "returnOnEquity": "0.004",
          "liquidationPx": "85000.0",
          "marginUsed": "120.05",
          "leverage": {"type": "cross", "value": 10}
        }
      }
    ],
    "time": 1781930462815
  },
  "leadingVaults": [],
  "totalVaultEquity": "0.0",
  "openOrders": [
    {"coin": "ETH", "limitPx": "2500.0", "oid": 99887766, "side": "B", "sz": "0.5", "timestamp": 1781930000000}
  ],
  "agentAddress": "0xabc0000000000000000000000000000000000001",
  "agentValidUntil": 1781930462815,
  "cumLedger": "4609.76",
  "meta": {
    "universe": [
      {"name": "BTC", "szDecimals": 5, "maxLeverage": 50, "marginTableId": 1},
      {"name": "ETH", "szDecimals": 4, "maxLeverage": 50, "marginTableId": 1, "isDelisted": false}
    ]
  },
  "user": "0xdef0000000000000000000000000000000000002"
}`

func TestDecodeWebData2(t *testing.T) {
	msg := WSMessage{Channel: "webData2", Data: json.RawMessage(webData2Fixture)}

	wd, err := DecodeWebData2(msg)
	if err != nil {
		t.Fatalf("DecodeWebData2: %v", err)
	}

	// Top-level scalar/optional fields.
	if wd.User != "0xdef0000000000000000000000000000000000002" {
		t.Errorf("User = %q", wd.User)
	}
	if wd.AgentAddress != "0xabc0000000000000000000000000000000000001" {
		t.Errorf("AgentAddress = %q", wd.AgentAddress)
	}
	if wd.AgentValidUntil != 1781930462815 {
		t.Errorf("AgentValidUntil = %d", wd.AgentValidUntil)
	}
	if wd.CumLedger != "4609.76" {
		t.Errorf("CumLedger = %q", wd.CumLedger)
	}
	if wd.TotalVaultEquity != "0.0" {
		t.Errorf("TotalVaultEquity = %q", wd.TotalVaultEquity)
	}

	// ClearinghouseState — reused info.UserState.
	cs := wd.ClearinghouseState
	if cs.MarginSummary.AccountValue != "4609.76" {
		t.Errorf("MarginSummary.AccountValue = %q", cs.MarginSummary.AccountValue)
	}
	if cs.Withdrawable != "4489.71" {
		t.Errorf("Withdrawable = %q", cs.Withdrawable)
	}
	if cs.CrossMarginSummary.TotalMarginUsed != "120.05" {
		t.Errorf("CrossMarginSummary.TotalMarginUsed = %q", cs.CrossMarginSummary.TotalMarginUsed)
	}

	// Non-empty position: every field must decode.
	if len(cs.AssetPositions) != 1 {
		t.Fatalf("AssetPositions len = %d, want 1", len(cs.AssetPositions))
	}
	p := cs.AssetPositions[0].Position
	if p.Coin != "BTC" {
		t.Errorf("position Coin = %q", p.Coin)
	}
	if p.Szi != "0.012" {
		t.Errorf("position Szi = %q", p.Szi)
	}
	if p.EntryPx == nil || *p.EntryPx != "100000.0" {
		t.Errorf("position EntryPx = %v", p.EntryPx)
	}
	if p.LiquidationPx == nil || *p.LiquidationPx != "85000.0" {
		t.Errorf("position LiquidationPx = %v", p.LiquidationPx)
	}
	if p.MarginUsed != "120.05" {
		t.Errorf("position MarginUsed = %q", p.MarginUsed)
	}
	if p.PositionValue != "1200.50" {
		t.Errorf("position PositionValue = %q", p.PositionValue)
	}
	if p.UnrealizedPnl != "0.50" {
		t.Errorf("position UnrealizedPnl = %q", p.UnrealizedPnl)
	}
	if p.Leverage.Value != 10 {
		t.Errorf("position Leverage.Value = %d", p.Leverage.Value)
	}

	// Open orders — reused info.OpenOrder.
	if len(wd.OpenOrders) != 1 {
		t.Fatalf("OpenOrders len = %d, want 1", len(wd.OpenOrders))
	}
	if wd.OpenOrders[0].Coin != "ETH" || wd.OpenOrders[0].Oid != 99887766 {
		t.Errorf("OpenOrders[0] = %+v", wd.OpenOrders[0])
	}

	// Meta universe — reused info.Meta.
	if len(wd.Meta.Universe) != 2 {
		t.Fatalf("Meta.Universe len = %d, want 2", len(wd.Meta.Universe))
	}
	if wd.Meta.Universe[0].Name != "BTC" || wd.Meta.Universe[0].SzDecimals != 5 {
		t.Errorf("Meta.Universe[0] = %+v", wd.Meta.Universe[0])
	}
}

// TestDecodeWebData2_LegacyUserStateTagIsWrong locks in the bug this
// decoder exists to prevent: the account snapshot lives under
// clearinghouseState, so a payload using the legacy "userState" key must
// leave ClearinghouseState at its zero value. If this ever starts
// populating, the wire key changed and the struct tag must follow.
func TestDecodeWebData2_LegacyUserStateTagIsWrong(t *testing.T) {
	const legacy = `{
      "userState": {
        "marginSummary": {"accountValue": "4609.76", "totalNtlPos": "0", "totalRawUsd": "0", "totalMarginUsed": "0"},
        "withdrawable": "4489.71",
        "assetPositions": [{"type": "oneWay", "position": {"coin": "BTC", "szi": "0.012", "marginUsed": "120.05", "positionValue": "1200.50", "unrealizedPnl": "0.50", "returnOnEquity": "0", "leverage": {"type": "cross", "value": 10}}}]
      }
    }`

	wd, err := DecodeWebData2(WSMessage{Channel: "webData2", Data: json.RawMessage(legacy)})
	if err != nil {
		t.Fatalf("DecodeWebData2: %v", err)
	}
	if wd.ClearinghouseState.MarginSummary.AccountValue != "" {
		t.Errorf("legacy userState key populated AccountValue = %q; clearinghouseState is the correct key",
			wd.ClearinghouseState.MarginSummary.AccountValue)
	}
	if len(wd.ClearinghouseState.AssetPositions) != 0 {
		t.Errorf("legacy userState key populated %d positions; expected 0", len(wd.ClearinghouseState.AssetPositions))
	}
}

// TestDecodeWebData2_BadJSON ensures malformed payloads surface an error
// rather than a half-populated value.
func TestDecodeWebData2_BadJSON(t *testing.T) {
	if _, err := DecodeWebData2(WSMessage{Channel: "webData2", Data: json.RawMessage(`{not json`)}); err == nil {
		t.Fatal("expected error on malformed JSON, got nil")
	}
}
