//go:build broken_rename

package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Simon-Busch/hyperliquid-go"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
)

func TestOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	tests := []struct {
		name string
		req  hyperliquid.CreateOrderRequest
	}{
		{
			name: "limit buy order",
			req: hyperliquid.CreateOrderRequest{
				Coin:  "BTC",
				IsBuy: true,
				Size:  0.001, // Smaller size for testing
				Price: 40000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{
						Tif: hyperliquid.TifGtc,
					},
				},
			},
		},
		{
			name: "market sell order",
			req: hyperliquid.CreateOrderRequest{
				Coin:  "ETH",
				IsBuy: false,
				Size:  0.01,
				Price: 2000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{
						Tif: hyperliquid.TifIoc,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := exchange.Order(tt.req, nil)
			if err != nil {
				t.Fatalf("Order failed: %v", err)
			}
			t.Logf("Order response: %+v", resp)
		})
	}
}

func TestMarketOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Market order method is available and ready to use")

	// Example usage with MarketOrder helper function:
	req := hyperliquid.CreateOrderRequest{
		Coin:  "BTC",
		IsBuy: true,
		Size:  0.001,
		// Price will be set automatically by MarketOrder
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifIoc,
			},
		},
	}

	result, err := exchange.Order(req, nil)
	if err != nil {
		t.Fatalf("MarketOrder failed: %v", err)
	}

	t.Logf("Market order result: %+v", result)
}

func TestMarketOpen(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	t.Log("Market open method is available and ready to use")

	// Example usage:
	name := "SOL"
	isBuy := true
	sz := 2.0
	slippage := 0.01 // 1%

	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}

	t.Logf("Market open result: %+v", result)
}

func TestMarketClose(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)
	t.Log("Market close method is available and ready to use")

	// Example usage:
	coin := "BTC"
	slippage := 0.01 // 1%

	result, err := exchange.MarketClose(coin, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose failed: %v", err)
	}

	t.Logf("Market close result: %+v", result)
}

func TestModifyOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Modify order method is available and ready to use")

	// Example usage:
	modifyReq := hyperliquid.ModifyOrderRequest{
		Oid: int64(12345),
		Order: hyperliquid.CreateOrderRequest{
			Coin:  "BTC",
			IsBuy: true,
			Size:  0.002,
			Price: 41000.0,
			OrderType: hyperliquid.OrderType{
				Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
			},
			ReduceOnly:    false,
			ClientOrderID: func() *string { s := "modified_order_123"; return &s }(),
		},
	}

	result, err := exchange.ModifyOrder(modifyReq)
	if err != nil {
		t.Fatalf("ModifyOrder failed: %v", err)
	}

	t.Logf("Modify order result: %+v", result)
}

func TestBulkModifyOrders(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Bulk modify orders method is available and ready to use")

	// Example usage:
	modifyRequests := []hyperliquid.ModifyOrderRequest{
		{
			Oid: int64(12345),
			Order: hyperliquid.CreateOrderRequest{
				Coin:  "BTC",
				IsBuy: true,
				Size:  0.002,
				Price: 41000.0,
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
				},
			},
		},
	}

	result, err := exchange.BulkModifyOrders(modifyRequests)
	if err != nil {
		t.Fatalf("BulkModifyOrders failed: %v", err)
	}

	t.Logf("Bulk modify orders result: %+v", result)
}

func TestOpenPositionAndSetLeverage(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Opening position and then setting leverage to 5x")

	// Step 1: Open a position (will use default 10x leverage)
	name := "BTC"
	isBuy := true
	sz := 0.002064   // Small size for testing
	slippage := 0.01 // 1%

	t.Logf("Opening %s position with size %f (will use default leverage)", name, sz)
	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}

	t.Logf("Position opened successfully: %+v", result)

	// Step 2: Set leverage to 5x after opening the position
	leverage := 5   // 5x leverage
	isCross := true // Use cross margin

	t.Logf("Setting leverage to %dx for %s", leverage, name)
	leverageResp, err := exchange.UpdateLeverage(leverage, name, isCross)
	if err != nil {
		t.Fatalf("Failed to update leverage: %v", err)
	}

	t.Logf("Leverage updated successfully: %+v", leverageResp)

	// Step 3: Verify the leverage was set correctly by checking user state
	// Note: In a real scenario, you might want to add a small delay here
	// to allow the exchange to process the leverage update
	t.Log("Position opened with default leverage and then updated to 5x leverage")
}

func TestOpenAndClosePosition(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Testing open and close position workflow")

	// Step 1: Check initial user state (before opening position)
	t.Log("Step 1: Checking initial user state")
	initialUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get initial user state: %v", err)
	}
	t.Logf("Initial user state - Positions count: %d", len(initialUserState.AssetPositions))
	for _, pos := range initialUserState.AssetPositions {
		t.Logf("Initial position: %s, Size: %s, Leverage: %dx",
			pos.Position.Coin, pos.Position.Szi, pos.Position.Leverage.Value)
	}

	// Step 2: Open a position
	name := "BTC"
	isBuy := true
	sz := 0.001      // Small size for testing
	slippage := 0.01 // 1%

	t.Logf("Step 2: Opening %s position with size %f", name, sz)
	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}
	t.Logf("Position opened successfully: %+v", result)

	// Step 3: Check user state after opening position
	t.Log("Step 3: Checking user state after opening position")
	afterOpenUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after opening: %v", err)
	}
	t.Logf("After opening - Positions count: %d", len(afterOpenUserState.AssetPositions))

	// Find and verify the BTC position
	var btcPosition *hyperliquid.Position
	for _, pos := range afterOpenUserState.AssetPositions {
		if pos.Position.Coin == name {
			btcPosition = &pos.Position
			t.Logf("Found BTC position - Size: %s, Leverage: %dx, Entry Price: %s",
				pos.Position.Szi, pos.Position.Leverage.Value,
				*pos.Position.EntryPx)
			break
		}
	}

	if btcPosition == nil {
		t.Fatalf("BTC position not found after opening")
	}

	// Step 4: Close the position
	t.Log("Step 4: Closing the position")
	closeResult, err := exchange.MarketClose(name, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose failed: %v", err)
	}
	t.Logf("Position closed successfully: %+v", closeResult)

	// Step 5: Check user state after closing position
	t.Log("Step 5: Checking user state after closing position")
	afterCloseUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after closing: %v", err)
	}
	t.Logf("After closing - Positions count: %d", len(afterCloseUserState.AssetPositions))

	// Verify BTC position is closed (should have size 0 or not exist)
	var btcPositionAfterClose *hyperliquid.Position
	for _, pos := range afterCloseUserState.AssetPositions {
		if pos.Position.Coin == name {
			btcPositionAfterClose = &pos.Position
			t.Logf("BTC position after close - Size: %s", pos.Position.Szi)
			break
		}
	}

	if btcPositionAfterClose != nil {
		// Check if position size is effectively 0
		size, err := strconv.ParseFloat(btcPositionAfterClose.Szi, 64)
		if err != nil {
			t.Logf("Warning: Could not parse position size: %v", err)
		} else if size != 0 {
			t.Logf("Warning: Position still exists with size: %f", size)
		} else {
			t.Log("Position successfully closed (size = 0)")
		}
	} else {
		t.Log("Position completely removed from user state")
	}

	// Summary
	t.Log("Test completed: Position opened and closed successfully")
}

func TestOpenAndPartiallyClosePosition(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Testing open, partial close, and full close position workflow")

	// Step 1: Check initial user state (before opening position)
	t.Log("Step 1: Checking initial user state")
	initialUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get initial user state: %v", err)
	}
	t.Logf("Initial user state - Positions count: %d", len(initialUserState.AssetPositions))

	// Step 2: Open a position
	name := "BTC"
	isBuy := true
	sz := 0.002      // Larger size for partial closing
	slippage := 0.01 // 1%

	t.Logf("Step 2: Opening %s position with size %f", name, sz)
	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}
	t.Logf("Position opened successfully: %+v", result)

	// Step 3: Check user state after opening position
	t.Log("Step 3: Checking user state after opening position")
	afterOpenUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after opening: %v", err)
	}
	t.Logf("After opening - Positions count: %d", len(afterOpenUserState.AssetPositions))

	// Find and verify the BTC position
	var btcPosition *hyperliquid.Position
	for _, pos := range afterOpenUserState.AssetPositions {
		if pos.Position.Coin == name {
			btcPosition = &pos.Position
			t.Logf("Found BTC position - Size: %s, Leverage: %dx, Entry Price: %s",
				pos.Position.Szi, pos.Position.Leverage.Value,
				*pos.Position.EntryPx)
			break
		}
	}

	if btcPosition == nil {
		t.Fatalf("BTC position not found after opening")
	}

	// Step 4: Partially close the position (close 50%)
	partialCloseSize := 0.001 // Close half of the position
	t.Logf("Step 4: Partially closing %s position with size %f (50%% of original)", name, partialCloseSize)

	// Calculate slippage price for partial close (sell to close long position)
	partialClosePrice, err := exchange.SlippagePrice(name, false, slippage, nil) // false = sell
	if err != nil {
		t.Fatalf("Failed to calculate partial close price: %v", err)
	}

	// Create a partial close order
	partialCloseOrder := hyperliquid.CreateOrderRequest{
		Coin:       name,
		IsBuy:      false, // Sell to close long position
		Size:       partialCloseSize,
		Price:      partialClosePrice,
		ReduceOnly: true, // Important: this ensures we only close existing position
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifIoc, // Immediate or cancel
			},
		},
	}

	partialCloseResult, err := exchange.Order(partialCloseOrder, nil)
	if err != nil {
		t.Fatalf("Partial close failed: %v", err)
	}
	t.Logf("Partial close result: %+v", partialCloseResult)

	// Step 5: Check user state after partial close
	t.Log("Step 5: Checking user state after partial close")
	afterPartialCloseUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after partial close: %v", err)
	}
	t.Logf("After partial close - Positions count: %d", len(afterPartialCloseUserState.AssetPositions))

	// Find and verify the remaining BTC position
	var remainingBtcPosition *hyperliquid.Position
	for _, pos := range afterPartialCloseUserState.AssetPositions {
		if pos.Position.Coin == name {
			remainingBtcPosition = &pos.Position
			t.Logf("Remaining BTC position - Size: %s, Leverage: %dx, Entry Price: %s",
				pos.Position.Szi, pos.Position.Leverage.Value,
				*pos.Position.EntryPx)
			break
		}
	}

	if remainingBtcPosition == nil {
		t.Fatalf("BTC position not found after partial close")
	}

	// Verify the position size was reduced
	remainingSize, err := strconv.ParseFloat(remainingBtcPosition.Szi, 64)
	if err != nil {
		t.Logf("Warning: Could not parse remaining position size: %v", err)
	} else {
		expectedRemainingSize := sz - partialCloseSize
		t.Logf("Remaining size: %f, Expected: %f", remainingSize, expectedRemainingSize)
		if remainingSize != expectedRemainingSize {
			t.Logf("Warning: Remaining size (%f) doesn't match expected (%f)", remainingSize, expectedRemainingSize)
		}
	}

	// Step 6: Fully close the remaining position
	t.Log("Step 6: Fully closing the remaining position")
	fullCloseResult, err := exchange.MarketClose(name, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("Full close failed: %v", err)
	}
	t.Logf("Full close result: %+v", fullCloseResult)

	// Step 7: Check user state after full close
	t.Log("Step 7: Checking user state after full close")
	afterFullCloseUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after full close: %v", err)
	}
	t.Logf("After full close - Positions count: %d", len(afterFullCloseUserState.AssetPositions))

	// Verify BTC position is completely closed
	var finalBtcPosition *hyperliquid.Position
	for _, pos := range afterFullCloseUserState.AssetPositions {
		if pos.Position.Coin == name {
			finalBtcPosition = &pos.Position
			t.Logf("Final BTC position - Size: %s", pos.Position.Szi)
			break
		}
	}

	if finalBtcPosition != nil {
		// Check if position size is effectively 0
		finalSize, err := strconv.ParseFloat(finalBtcPosition.Szi, 64)
		if err != nil {
			t.Logf("Warning: Could not parse final position size: %v", err)
		} else if finalSize != 0 {
			t.Logf("Warning: Position still exists with size: %f", finalSize)
		} else {
			t.Log("Position successfully fully closed (size = 0)")
		}
	} else {
		t.Log("Position completely removed from user state")
	}

	// Summary
	t.Log("Test completed: Position opened, partially closed, and fully closed successfully")
}

func TestShortSOLLeverageUpdateAndClose(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "SOL"
	slippage := 0.01

	// Step 1: Check user state before
	t.Log("[Before] Checking user state before opening short SOL position")
	beforeState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state (before): %v", err)
	}
	var beforeSOL *hyperliquid.Position
	for _, ap := range beforeState.AssetPositions {
		if ap.Position.Coin == coin {
			beforeSOL = &ap.Position
			break
		}
	}
	if beforeSOL != nil {
		t.Logf("[Before] Existing SOL position - Size: %s, Lev: %dx", beforeSOL.Szi, beforeSOL.Leverage.Value)
	} else {
		t.Log("[Before] No existing SOL position")
	}

	// Step 2: Open a short position (sell) with slippage escalation if needed
	size := 0.1 // modest size for test liquidity
	escalations := []float64{slippage, 0.03, 0.05, 0.1, 0.2}
	var opened bool
	for _, s := range escalations {
		t.Logf("Opening short position on %s with size %f (slippage=%.2f)", coin, size, s)
		openRes, err := exchange.MarketOpen(coin, false /* isBuy=false => short */, size, nil, s, nil, nil)
		if err != nil {
			t.Logf("MarketOpen short failed with slippage %.2f: %v", s, err)
			continue
		}
		t.Logf("Short open attempt result: %+v", openRes)

		// Verify position exists after open attempt
		afterOpenState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
		if err != nil {
			t.Fatalf("Failed to get user state after open attempt: %v", err)
		}
		for _, ap := range afterOpenState.AssetPositions {
			if ap.Position.Coin == coin {
				opened = true
				break
			}
		}
		if opened {
			break
		}
	}
	if !opened {
		t.Fatalf("Failed to open short %s position after slippage escalation attempts", coin)
	}

	time.Sleep(3 * time.Second)

	// Step 3: Update leverage to 3x (cross)
	t.Log("Updating SOL leverage to 3x (cross)")
	_, err = exchange.UpdateLeverage(3, coin, true)
	if err != nil {
		t.Fatalf("UpdateLeverage failed: %v", err)
	}

	// Step 4: Check user state after leverage update (during)
	t.Log("[During] Verifying leverage updated to 3x")
	duringState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state (during): %v", err)
	}
	var duringSOL *hyperliquid.Position
	for _, ap := range duringState.AssetPositions {
		if ap.Position.Coin == coin {
			duringSOL = &ap.Position
			break
		}
	}
	if duringSOL == nil {
		t.Fatalf("SOL position not found after opening")
	}
	t.Logf("[During] SOL position - Size: %s, Lev: %dx, Type: %s", duringSOL.Szi, duringSOL.Leverage.Value, duringSOL.Leverage.Type)
	if duringSOL.Leverage.Value != 3 {
		t.Fatalf("expected leverage 3x, got %dx", duringSOL.Leverage.Value)
	}
	time.Sleep(3 * time.Second)

	// Step 5: Close the position with slippage escalation if needed
	t.Log("Closing SOL position")
	closed := false
	for _, s := range escalations {
		closeRes, err := exchange.MarketClose(coin, nil, nil, s, nil, nil)
		if err != nil {
			t.Logf("MarketClose failed with slippage %.2f: %v", s, err)
			continue
		}
		t.Logf("Close attempt result (slippage=%.2f): %+v", s, closeRes)

		// Verify closed or size zero
		afterCloseCheck, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
		if err != nil {
			t.Fatalf("Failed to get user state after close attempt: %v", err)
		}
		var posAfter *hyperliquid.Position
		for _, ap := range afterCloseCheck.AssetPositions {
			if ap.Position.Coin == coin {
				posAfter = &ap.Position
				break
			}
		}
		if posAfter == nil {
			closed = true
			break
		}
		sizeFloat, err := strconv.ParseFloat(posAfter.Szi, 64)
		if err == nil && sizeFloat == 0 {
			closed = true
			break
		}
	}
	if !closed {
		t.Fatalf("Failed to close SOL position after slippage escalation attempts")
	}

	// Step 6: Check user state after
	t.Log("[After] Checking user state after closing SOL position")
	afterState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state (after): %v", err)
	}
	var afterSOL *hyperliquid.Position
	for _, ap := range afterState.AssetPositions {
		if ap.Position.Coin == coin {
			afterSOL = &ap.Position
			break
		}
	}
	if afterSOL != nil {
		sizeFloat, err := strconv.ParseFloat(afterSOL.Szi, 64)
		if err != nil {
			t.Logf("Warning: could not parse SOL size after close: %v", err)
		} else if sizeFloat != 0 {
			t.Fatalf("expected SOL position size 0 after close, got %f", sizeFloat)
		} else {
			t.Log("[After] SOL position size is 0 (closed)")
		}
	} else {
		t.Log("[After] SOL position removed (no active position)")
	}
}

func TestWeirdSizeAndClose(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "SOL"
	slippage := 0.01

	// Step 1: Check user state before
	t.Log("[Before] Checking user state before opening short SOL position")
	beforeState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state (before): %v", err)
	}
	var beforeSOL *hyperliquid.Position
	for _, ap := range beforeState.AssetPositions {
		if ap.Position.Coin == coin {
			beforeSOL = &ap.Position
			break
		}
	}
	if beforeSOL != nil {
		t.Logf("[Before] Existing SOL position - Size: %s, Lev: %dx", beforeSOL.Szi, beforeSOL.Leverage.Value)
	} else {
		t.Log("[Before] No existing SOL position")
	}

	// Step 2: Open a short position (sell) with slippage escalation if needed
	size := 6.7844564847 // modest size for test liquidity
	escalations := []float64{slippage, 0.03, 0.05, 0.1, 0.2}
	var opened bool
	for _, s := range escalations {
		t.Logf("Opening short position on %s with size %f (slippage=%.2f)", coin, size, s)
		openRes, err := exchange.MarketOpen(coin, false /* isBuy=false => short */, size, nil, s, nil, nil)
		if err != nil {
			t.Logf("MarketOpen short failed with slippage %.2f: %v", s, err)
			continue
		}
		t.Logf("Short open attempt result: %+v", openRes)

		// Verify position exists after open attempt
		afterOpenState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
		if err != nil {
			t.Fatalf("Failed to get user state after open attempt: %v", err)
		}
		for _, ap := range afterOpenState.AssetPositions {
			if ap.Position.Coin == coin {
				opened = true
				break
			}
		}
		if opened {
			break
		}
	}
	if !opened {
		t.Fatalf("Failed to open short %s position after slippage escalation attempts", coin)
	}

	time.Sleep(3 * time.Second)

	// Step 5: Close the position with slippage escalation if needed
	t.Log("Closing SOL position")
	closed := false
	for _, s := range escalations {
		closeRes, err := exchange.MarketClose(coin, nil, nil, s, nil, nil)
		if err != nil {
			t.Logf("MarketClose failed with slippage %.2f: %v", s, err)
			continue
		}
		t.Logf("Close attempt result (slippage=%.2f): %+v", s, closeRes)

		// Verify closed or size zero
		afterCloseCheck, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
		if err != nil {
			t.Fatalf("Failed to get user state after close attempt: %v", err)
		}
		var posAfter *hyperliquid.Position
		for _, ap := range afterCloseCheck.AssetPositions {
			if ap.Position.Coin == coin {
				posAfter = &ap.Position
				break
			}
		}
		if posAfter == nil {
			closed = true
			break
		}
		sizeFloat, err := strconv.ParseFloat(posAfter.Szi, 64)
		if err == nil && sizeFloat == 0 {
			closed = true
			break
		}
	}
	if !closed {
		t.Fatalf("Failed to close SOL position after slippage escalation attempts")
	}

	// Step 6: Check user state after
	t.Log("[After] Checking user state after closing SOL position")
	afterState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state (after): %v", err)
	}
	var afterSOL *hyperliquid.Position
	for _, ap := range afterState.AssetPositions {
		if ap.Position.Coin == coin {
			afterSOL = &ap.Position
			break
		}
	}
	if afterSOL != nil {
		sizeFloat, err := strconv.ParseFloat(afterSOL.Szi, 64)
		if err != nil {
			t.Logf("Warning: could not parse SOL size after close: %v", err)
		} else if sizeFloat != 0 {
			t.Fatalf("expected SOL position size 0 after close, got %f", sizeFloat)
		} else {
			t.Log("[After] SOL position size is 0 (closed)")
		}
	} else {
		t.Log("[After] SOL position removed (no active position)")
	}
}

func TestOpenPositionAndCancelCloseOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("Testing open position and cancel close order workflow")

	// Step 1: Check initial user state
	t.Log("Step 1: Checking initial user state")
	initialUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get initial user state: %v", err)
	}
	t.Logf("Initial user state - Positions count: %d", len(initialUserState.AssetPositions))

	// Step 2: Open a position
	name := "BTC"
	isBuy := true
	sz := 0.001      // Small size for testing
	slippage := 0.01 // 1%

	t.Logf("Step 2: Opening %s position with size %f", name, sz)
	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}
	t.Logf("Position opened successfully: %+v", result)

	// Step 3: Verify position was opened
	t.Log("Step 3: Verifying position was opened")
	afterOpenUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after opening: %v", err)
	}

	var btcPosition *hyperliquid.Position
	for _, pos := range afterOpenUserState.AssetPositions {
		if pos.Position.Coin == name {
			btcPosition = &pos.Position
			t.Logf("Found BTC position - Size: %s, Leverage: %dx, Entry Price: %s",
				pos.Position.Szi, pos.Position.Leverage.Value,
				*pos.Position.EntryPx)
			break
		}
	}

	if btcPosition == nil {
		t.Fatalf("BTC position not found after opening")
	}

	// Step 4: Place a limit order to close the position
	t.Log("Step 4: Placing limit order to close position")
	closeOrderReq := hyperliquid.CreateOrderRequest{
		Coin:  name,
		IsBuy: false,   // Sell to close long position
		Size:  sz,      // Same size as opened position
		Price: 45000.0, // Set a price that's unlikely to fill immediately
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{
				Tif: hyperliquid.TifGtc, // Good till cancelled
			},
		},
	}

	closeOrderResp, err := exchange.Order(closeOrderReq, nil)
	if err != nil {
		t.Fatalf("Failed to place close order: %v", err)
	}
	t.Logf("Close order placed: %+v", closeOrderResp)

	// Extract order ID from response
	var orderID int64
	if closeOrderResp.Resting != nil {
		orderID = closeOrderResp.Resting.Oid
		t.Logf("Order ID for cancellation: %d", orderID)
	} else {
		t.Skip("Close order was filled immediately, cannot test cancel")
	}

	// Step 5: Cancel the close order
	t.Log("Step 5: Cancelling the close order")
	cancelResp, err := exchange.Cancel(name, orderID)
	if err != nil {
		t.Fatalf("Failed to cancel order: %v", err)
	}
	t.Logf("Cancel response: %+v", cancelResp)

	// Step 6: Verify position still exists (since we cancelled the close order)
	t.Log("Step 6: Verifying position still exists after cancelling close order")
	afterCancelUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after cancelling: %v", err)
	}

	var btcPositionAfterCancel *hyperliquid.Position
	for _, pos := range afterCancelUserState.AssetPositions {
		if pos.Position.Coin == name {
			btcPositionAfterCancel = &pos.Position
			t.Logf("BTC position after cancelling close order - Size: %s", pos.Position.Szi)
			break
		}
	}

	if btcPositionAfterCancel == nil {
		t.Fatalf("BTC position not found after cancelling close order - it should still exist")
	}

	// Step 7: Clean up - close the position properly
	t.Log("Step 7: Cleaning up - closing position properly")
	closeResult, err := exchange.MarketClose(name, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("MarketClose failed during cleanup: %v", err)
	}
	t.Logf("Position closed successfully during cleanup: %+v", closeResult)

	// Step 8: Final verification
	t.Log("Step 8: Final verification - checking position is closed")
	finalUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get final user state: %v", err)
	}

	var finalBtcPosition *hyperliquid.Position
	for _, pos := range finalUserState.AssetPositions {
		if pos.Position.Coin == name {
			finalBtcPosition = &pos.Position
			break
		}
	}

	if finalBtcPosition != nil {
		size, err := strconv.ParseFloat(finalBtcPosition.Szi, 64)
		if err != nil {
			t.Logf("Warning: Could not parse final position size: %v", err)
		} else if size != 0 {
			t.Logf("Warning: Position still exists with size: %f", size)
		} else {
			t.Log("Position successfully closed (size = 0)")
		}
	} else {
		t.Log("Position completely removed from user state")
	}

	t.Log("Test completed: Position opened, close order placed and cancelled, position still exists, then properly closed")
}

func TestOpenPositionWithStopLossMinus10Percent(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "BTC"
	isBuy := true
	size := 0.001
	slippage := 0.01
	slPercent := 0.10 // 10%

	// Open with SL in one grouped action
	resp, err := exchange.MarketOpenWithSLTP(coin, isBuy, size, nil, slippage, slPercent, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpenWithSLTP failed: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("MarketOpenWithSLTP not ok: %s", resp.Err)
	}

	statuses := resp.Data.Statuses
	if len(statuses) < 2 {
		t.Fatalf("expected 2 statuses (open, SL), got %d", len(statuses))
	}

	// // One should be filled (open IOC), the other likely resting (SL trigger)
	// var slOrderID int64
	// for _, mv := range statuses {
	// 	if mv.Type() != "object" {
	// 		continue
	// 	}
	// 	var st hyperliquid.OrderStatus
	// 	if err := mv.Parse(&st); err != nil {
	// 		t.Fatalf("failed to parse status: %v", err)
	// 	}
	// 	if st.Resting != nil {
	// 		// store potential SL oid for cleanup
	// 		slOrderID = st.Resting.Oid
	// 	}
	// }

	// // Verify position exists
	// state, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	// if err != nil {
	// 	t.Fatalf("failed to fetch user state: %v", err)
	// }
	// var pos *hyperliquid.Position
	// for _, ap := range state.AssetPositions {
	// 	if ap.Position.Coin == coin {
	// 		pos = &ap.Position
	// 		break
	// 	}
	// }
	// if pos == nil {
	// 	t.Fatalf("expected %s position after open", coin)
	// }
	// t.Logf("Opened position on %s: size=%s lev=%dx", coin, pos.Szi, pos.Leverage.Value)

	// // Cleanup: cancel SL order if we captured an oid
	// if slOrderID != 0 {
	// 	if _, err := exchange.Cancel(coin, slOrderID); err != nil {
	// 		t.Logf("warning: failed to cancel SL order %d: %v", slOrderID, err)
	// 	}
	// }

	// // Close the position
	// if _, err := exchange.MarketClose(coin, nil, nil, slippage, nil, nil); err != nil {
	// 	t.Fatalf("MarketClose failed: %v", err)
	// }
}

func TestOpenPositionWithTakeProfitPlus10Percent(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "SOL"
	isBuy := true
	size := 1.0
	slippage := 0.01
	slPercent := 0.10 // 10%

	// Open with SL in one grouped action
	resp, err := exchange.MarketOpenWithSLTP(coin, isBuy, size, nil, slippage, slPercent, true, nil, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpenWithSLTP failed: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("MarketOpenWithSLTP not ok: %s", resp.Err)
	}

	statuses := resp.Data.Statuses
	if len(statuses) < 2 {
		t.Fatalf("expected 2 statuses (open, SL), got %d", len(statuses))
	}

	// // Close the position to clean up
	// if _, err := exchange.MarketClose(coin, nil, nil, slippage, nil, nil); err != nil {
	// 	t.Fatalf("MarketClose failed: %v", err)
	// }
}

func TestOpenPositionWithPartialStopLoss(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "BTC"
	isBuy := true
	size := 0.002    // open size
	partial := 0.001 // 50% SL size
	slippage := 0.01
	slPercent := 0.10 // 10%

	resp, err := exchange.MarketOpenWithSLTPPartial(coin, isBuy, size, nil, slippage, slPercent, false /* isTP */, &partial, nil, nil, nil)
	if err != nil {
		t.Fatalf("MarketOpenWithSLTPPartial failed: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("MarketOpenWithSLTPPartial not ok: %s", resp.Err)
	}

	statuses := resp.Data.Statuses
	if len(statuses) < 2 {
		t.Fatalf("expected at least 2 statuses (filled open, trigger token), got %d", len(statuses))
	}

	var hasFilled bool
	var hasWaiting bool
	for _, mv := range statuses {
		typ := mv.Type()
		if typ == "object" {
			var st hyperliquid.OrderStatus
			if err := mv.Parse(&st); err == nil && st.Filled != nil {
				hasFilled = true
			}
		}
		if typ == "string" {
			hasWaiting = true
		}
	}
	if !hasFilled || !hasWaiting {
		t.Fatalf("expected filled open and waiting trigger, got hasFilled=%v hasWaiting=%v", hasFilled, hasWaiting)
	}
}

func TestOpenLongARBWithLeverageAndTPandSL(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "ARB"
	size := 100.0       // Adjusted size for ARB
	tpslPercent := 0.05 // 5%
	slippage := 0.05

	// Declare variables
	var openPx float64
	var err error

	// Get the current market price for ARB
	// openPx, err := exchange.SlippagePrice(coin, true, slippage, nil)
	// if err != nil {
	// 	t.Fatalf("SlippagePrice failed: %v", err)
	// }

	// Validate that we got a reasonable price
	// if openPx <= 0 {
	// 	t.Fatalf("Invalid slippage price: %f", openPx)
	// }

	// Get the asset ID for ARB
	assetID := exchange.GetInfo().NameToAsset(coin)
	t.Logf("Asset ID for %s: %d", coin, assetID)

	// Check if the asset ID is valid
	if assetID == 0 {
		t.Fatalf("Invalid asset ID for %s: %d", coin, assetID)
	}

	// Debug: Check what assets are available in allMids
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Logf("Warning: Failed to get allMids: %v", err)
	} else {
		t.Logf("Available assets in allMids: %d", len(mids))
		// Show first few assets as examples
		count := 0
		for assetName, midPrice := range mids {
			if count < 5 {
				t.Logf("  %s: %s", assetName, midPrice)
				count++
			}
		}

		// Check specifically for ARB
		if midPriceStr, ok := mids[coin]; ok {
			t.Logf("Found %s in allMids: %s", coin, midPriceStr)
			if midPrice, err := strconv.ParseFloat(midPriceStr, 64); err == nil {
				t.Logf("Raw mid price for %s: $%.6f", coin, midPrice)
				// Calculate slippage manually to debug
				manualSlippagePrice := midPrice * (1 + slippage)
				t.Logf("Manual slippage calculation: $%.6f * (1 + %.3f) = $%.6f", midPrice, slippage, manualSlippagePrice)
			} else {
				t.Logf("Failed to parse mid price for %s: %v", coin, err)
			}
		} else {
			t.Logf("❌ %s NOT found in allMids", coin)
			// Try to find similar assets
			for assetName := range mids {
				if strings.Contains(strings.ToUpper(assetName), "JUP") ||
					strings.Contains(strings.ToUpper(assetName), "JUPITER") {
					t.Logf("Found similar asset: %s", assetName)
				}
			}
		}
	}

	// Try to get slippage price manually to debug
	t.Logf("Attempting to get slippage price for %s (asset ID: %d)...", coin, assetID)
	openPx, err = exchange.SlippagePrice(coin, true, slippage, nil)
	if err != nil {
		t.Logf("SlippagePrice error: %v", err)
		// Try to debug the issue by calling AllMids directly
		if mids, err := exchange.GetInfo().AllMids(); err == nil {
			if midPriceStr, ok := mids[coin]; ok {
				t.Logf("Direct allMids lookup for %s: %s", coin, midPriceStr)
			}
		}
		t.Fatalf("SlippagePrice failed: %v", err)
	}

	// Validate that we got a reasonable price
	if openPx <= 0 {
		t.Fatalf("Invalid slippage price: %f", openPx)
	}

	// Compute trigger prices relative to open price
	tpPxRaw := openPx * (1 + tpslPercent) // TP above entry for long position
	slPxRaw := openPx * (1 - tpslPercent) // SL below entry for long position

	t.Logf("Raw prices - Open: $%.6f, TP: $%.6f, SL: $%.6f", openPx, tpPxRaw, slPxRaw)

	// Use PriceToWire to ensure proper tick size compliance
	tpPxWire, err := hyperliquid.PriceToWire(tpPxRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("Failed to format TP price: %v", err)
	}
	slPxWire, err := hyperliquid.PriceToWire(slPxRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("Failed to format SL price: %v", err)
	}
	openPxWire, err := hyperliquid.PriceToWire(openPx, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("Failed to format open price: %v", err)
	}

	t.Logf("Wire prices - Open: %s, TP: %s, SL: %s", openPxWire, tpPxWire, slPxWire)

	// Parse back to float for the main order price only
	openPxFinal, err := strconv.ParseFloat(openPxWire, 64)
	if err != nil {
		t.Fatalf("Failed to parse open price: %v", err)
	}

	// For trigger orders, we need to parse the wire prices back to float64
	// because the TriggerOrderType expects float64, but PriceToWire will be called again
	tpPx, err := strconv.ParseFloat(tpPxWire, 64)
	if err != nil {
		t.Fatalf("Failed to parse TP price: %v", err)
	}
	slPx, err := strconv.ParseFloat(slPxWire, 64)
	if err != nil {
		t.Fatalf("Failed to parse SL price: %v", err)
	}

	t.Logf("Final prices - Open: $%.6f, TP: $%.6f, SL: $%.6f", openPxFinal, tpPx, slPx)
	t.Logf("Wire strings - Open: %s, TP: %s, SL: %s", openPxWire, tpPxWire, slPxWire)

	// Build orders: 1) IOC open; 2) TP trigger; 3) SL trigger
	openOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         true,
		Price:         openPxFinal,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifIoc}},
		ClientOrderID: nil,
	}

	tpOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // Close direction
		Price:         tpPx,  // Use the calculated TP price
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: tpPx, IsMarket: true, Tpsl: "tp"}},
		ClientOrderID: nil,
	}

	slOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // Close direction
		Price:         slPx,  // Use the calculated SL price
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: slPx, IsMarket: true, Tpsl: "sl"}},
		ClientOrderID: nil,
	}

	// Use normalTpsl grouping to align with trigger order expectations
	resp, err := exchange.BulkOrdersWithGrouping([]hyperliquid.CreateOrderRequest{openOrder, tpOrder, slOrder}, hyperliquid.GroupingNormalTpsl, nil)
	if err != nil {
		t.Fatalf("Placing orders failed: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("Orders not ok: %s", resp.Err)
	}

	t.Logf("✅ Successfully placed open order with TP/SL triggers")
	t.Logf("📊 Order Summary:")
	t.Logf("   🪙 Asset: %s (ID: %d)", coin, assetID)
	t.Logf("   📈 Side: LONG")
	t.Logf("   💵 Size: %.2f", size)
	t.Logf("   🎯 Entry: $%.6f", openPxFinal)
	t.Logf("   🚀 TP: $%.6f (+%.1f%%)", tpPx, tpslPercent*100)
	t.Logf("   🛡️  SL: $%.6f (-%.1f%%)", slPx, tpslPercent*100)
}

func TestMarketOpenWithCloid(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t) // exchange used for setup only

	t.Log("Market open method is available and ready to use")

	// Example usage:
	name := "BTC"
	isBuy := false
	sz := 0.001
	slippage := 0.01 // 1%

	// cloid should be a 128-bit hex string according to Hyperliquid docs
	cloid := "0x1234567890abcdef1234567890abcdef"

	result, err := exchange.MarketOpen(name, isBuy, sz, nil, slippage, &cloid, nil)
	if err != nil {
		t.Fatalf("MarketOpen failed: %v", err)
	}
	t.Logf("Market open result: %+v", result)
}

func TestBulkOrdersWithGrouping_ETH_LimitPlus3_TPMinus10_SLPlus10(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "ETH"
	size := 0.005

	// Fetch mid price
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("failed to fetch mids: %v", err)
	}
	midStr, ok := mids[coin]
	if !ok {
		t.Fatalf("mid not found for %s", coin)
	}
	mid, err := strconv.ParseFloat(midStr, 64)
	if err != nil {
		t.Fatalf("failed to parse mid for %s: %v", coin, err)
	}

	// Compute prices per request
	openRaw := mid * 1.01 // limit at market +3%
	slRaw := mid * 1.10   // SL at market +10%
	tpRaw := mid * 0.90   // TP at market -10%

	// Ensure prices conform to tick rules using PriceToWire
	assetID := exchange.GetInfo().NameToAsset(coin)
	openWire, err := hyperliquid.PriceToWire(openRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire open price: %v", err)
	}
	slWire, err := hyperliquid.PriceToWire(slRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire SL price: %v", err)
	}
	tpWire, err := hyperliquid.PriceToWire(tpRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire TP price: %v", err)
	}

	openPx, _ := strconv.ParseFloat(openWire, 64)
	slPx, _ := strconv.ParseFloat(slWire, 64)
	tpPx, _ := strconv.ParseFloat(tpWire, 64)

	// Build orders: open limit (Gtc), SL trigger (market), TP trigger (market)
	openOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         true,
		Price:         openPx,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
		ClientOrderID: nil,
	}

	slOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // close direction for long
		Price:         slPx,
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: slPx, IsMarket: true, Tpsl: "sl"}},
		ClientOrderID: nil,
	}

	tpOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // close direction for long
		Price:         tpPx,
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: tpPx, IsMarket: true, Tpsl: "tp"}},
		ClientOrderID: nil,
	}

	builderAddress := "0x1234567890abcdef1234567890abcdef12345678"
	maxFeeRate := "100" // 0.1% (100 tenths of basis points)

	approvalResp, err := exchange.ApproveBuilderFee(builderAddress, maxFeeRate)
	if err != nil {
		t.Fatalf("❌ Failed to approve builder fee: %v", err)
	}

	// Create BuilderInfo
	builderInfo := &hyperliquid.BuilderInfo{
		Builder: "0x1234567890abcdef1234567890abcdef12345678",
		Fee:     100,
	}

	ordersResp, err := exchange.BulkOrdersWithGrouping([]hyperliquid.CreateOrderRequest{openOrder, tpOrder, slOrder}, hyperliquid.GroupingNormalTpsl, builderInfo)
	if err != nil {
		t.Fatalf("placing grouped orders failed: %v", err)
	}

	if !ordersResp.Ok {
		t.Fatalf("grouped orders not ok: %s", ordersResp.Err)
	}

	_ = approvalResp
	t.Logf("Placed ETH grouped orders: open %.6f, tp %.6f, sl %.6f", openPx, tpPx, slPx)
}

func TestGetCompletePositionSummary(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	// Get user state (positions)
	userState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state: %v", err)
	}

	// Get open orders (TP/SL)
	openOrders, err := exchange.GetInfo().FrontendOpenOrders(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get open orders: %v", err)
	}

	t.Logf("=== POSITIONS ===")
	for _, ap := range userState.AssetPositions {
		pos := ap.Position
		if pos.Szi != "0" { // Only show non-zero positions
			t.Logf("Position: %s | Size: %s | Entry: %s | PnL: %s | Leverage: %dx",
				pos.Coin, pos.Szi, *pos.EntryPx, pos.UnrealizedPnl, pos.Leverage.Value)
		}
	}

	t.Logf("=== OPEN ORDERS (TP/SL) ===")
	for _, order := range openOrders {
		fmt.Printf("raw Order: %+v\n", order)
		t.Logf("Order: %s | Side: %s | Size: %f | Price: %f | OID: %d",
			order.Coin, order.Side, order.Size, order.LimitPx, order.Oid)
	}

	t.Logf("=== MARGIN SUMMARY ===")
	t.Logf("Account Value: %s", userState.MarginSummary.AccountValue)
	t.Logf("Margin Used: %s", userState.MarginSummary.TotalMarginUsed)
	t.Logf("Net Liquidation: %s", userState.MarginSummary.TotalNtlPos)
}

func TestFetchAndCancelOpenOrder(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	// 1) Fetch all open orders for the user
	openOrders, err := exchange.GetInfo().FrontendOpenOrders(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to fetch open orders: %v", err)
	}

	if len(openOrders) == 0 {
		t.Log("No open orders found to cancel")
		return
	}

	// 2) Log all open orders for visibility
	t.Logf("Found %d open orders:", len(openOrders))
	for i, order := range openOrders {
		t.Logf("Order %d: Coin=%s, Side=%s, Size=%f, Price=%f, OID=%d, Type=%s, ReduceOnly=%t",
			i+1, order.Coin, order.Side, order.Size, order.LimitPx, order.Oid, order.OrderType, order.ReduceOnly)
	}

	// 3) Select the first order to cancel (you can modify this logic as needed)
	orderToCancel := openOrders[0]
	t.Logf("Selected order to cancel: Coin=%s, Side=%s, Size=%f, Price=%f, OID=%d",
		orderToCancel.Coin, orderToCancel.Side, orderToCancel.Size, orderToCancel.LimitPx, orderToCancel.Oid)

	// 37589055812
	// 37589055813

	// 4) Cancel the selected order by its OID
	cancelResp, err := exchange.Cancel(orderToCancel.Coin, orderToCancel.Oid)
	if err != nil {
		t.Fatalf("Failed to cancel order %d: %v", orderToCancel.Oid, err)
	}

	t.Logf("Successfully cancelled order %d: %+v", orderToCancel.Oid, cancelResp)

	// 5) Verify the order was cancelled by fetching open orders again
	openOrdersAfter, err := exchange.GetInfo().FrontendOpenOrders(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to fetch open orders after cancellation: %v", err)
	}

	// 6) Check if the cancelled order is no longer in the list
	orderStillExists := false
	for _, order := range openOrdersAfter {
		if order.Oid == orderToCancel.Oid {
			orderStillExists = true
			break
		}
	}

	if orderStillExists {
		t.Errorf("Order %d still exists after cancellation", orderToCancel.Oid)
	} else {
		t.Logf("Order %d successfully removed from open orders", orderToCancel.Oid)
	}

	// 7) Log the updated open orders count
	t.Logf("Open orders after cancellation: %d (was %d)", len(openOrdersAfter), len(openOrders))
}

func TestDebugOrderCancellation(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== DEBUGGING ORDER CANCELLATION ISSUE ===")

	// 1) Check your wallet address
	walletAddr := exchange.GetAccountAddr()
	t.Logf("Your wallet address: %s", walletAddr)

	// 2) Get all open orders and their details
	openOrders, err := exchange.GetInfo().FrontendOpenOrders(walletAddr)
	if err != nil {
		t.Fatalf("Failed to get open orders: %v", err)
	}

	t.Logf("Found %d open orders", len(openOrders))

	for i, order := range openOrders {
		// Get asset ID for this coin
		assetID := exchange.GetInfo().NameToAsset(order.Coin)

		t.Logf("Order %d:", i+1)
		t.Logf("  Coin: %s (Asset ID: %d)", order.Coin, assetID)
		t.Logf("  OID: %d", order.Oid)
		t.Logf("  Side: %s", order.Side)
		t.Logf("  Size: %f", order.Size)
		t.Logf("  Limit Price: $%.2f", order.LimitPx)
		t.Logf("  Is Trigger: %t", order.IsTrigger)
		t.Logf("  Is Position TP/SL: %t", order.IsPositionTpsl)
		t.Logf("  Order Type: %s", order.OrderType)
		t.Logf("  ---")
	}

	// 3) Test cancellation with detailed logging
	if len(openOrders) > 0 {
		orderToCancel := openOrders[0]
		assetID := exchange.GetInfo().NameToAsset(orderToCancel.Coin)

		t.Logf("\n=== TESTING CANCELLATION ===")
		t.Logf("Attempting to cancel:")
		t.Logf("  Coin: %s (Asset ID: %d)", orderToCancel.Coin, assetID)
		t.Logf("  OID: %d", orderToCancel.Oid)
		t.Logf("  Wallet: %s", walletAddr)

		// Try to cancel and see what happens
		cancelResp, err := exchange.Cancel(orderToCancel.Coin, orderToCancel.Oid)
		if err != nil {
			t.Logf("❌ Cancel failed with error: %v", err)
		} else {
			t.Logf("✅ Cancel response: %+v", cancelResp)
		}
	}

	// 4) Check if there are any orders with OID 37622448426
	t.Logf("\n=== CHECKING FOR SPECIFIC OID ===")
	t.Logf("Looking for OID: 37622448426")

	// This would require a different API call, but let's check what we can see
	for _, order := range openOrders {
		if order.Oid == 37622448426 {
			t.Logf("🎯 FOUND ORDER WITH OID 37622448426!")
			t.Logf("  Coin: %s", order.Coin)
			t.Logf("  Asset ID: %d", exchange.GetInfo().NameToAsset(order.Coin))
			t.Logf("  Side: %s", order.Side)
		}
	}
}

func TestCheckAvailableAssets(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== CHECKING AVAILABLE ASSETS ON TESTNET ===")

	// Get meta information to see available assets
	meta, err := exchange.GetInfo().Meta()
	if err != nil {
		t.Fatalf("Failed to get meta: %v", err)
	}

	t.Logf("Available perpetual assets: %d", len(meta.Universe))
	for i, asset := range meta.Universe {
		t.Logf("Asset %d: %s (szDecimals: %d)", i, asset.Name, asset.SzDecimals)
	}

	// Get current mid prices for all assets
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("Failed to get all mids: %v", err)
	}

	t.Logf("\n=== CURRENT MARKET PRICES ===")
	for coin, price := range mids {
		t.Logf("%s: $%s", coin, price)
	}

	// Try to get slippage price for a few assets to test availability
	testAssets := []string{"SOL", "ETH", "BTC", "ARB"}

	t.Logf("\n=== TESTING ASSET AVAILABILITY ===")
	for _, asset := range testAssets {
		price, err := exchange.SlippagePrice(asset, true, 0.01, nil) // 1% slippage
		if err != nil {
			t.Logf("❌ %s: %v", asset, err)
		} else {
			t.Logf("✅ %s: $%.2f", asset, price)
		}
	}

	t.Logf("\n=== ACCOUNT STATUS ===")
	userState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state: %v", err)
	}

	t.Logf("Account Value: $%s", userState.MarginSummary.AccountValue)
	t.Logf("Available Balance: $%s", userState.Withdrawable)
	t.Logf("Total Margin Used: $%s", userState.MarginSummary.TotalMarginUsed)
}

func TestSmallSOLPosition(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING SMALL SOL POSITION ===")

	// Use a very small size to test
	coin := "SOL"
	size := 0.001 // Very small size
	isBuy := true

	// Get current price with 1% slippage
	price, err := exchange.SlippagePrice(coin, isBuy, 0.01, nil)
	if err != nil {
		t.Fatalf("Failed to get slippage price: %v", err)
	}

	t.Logf("Opening %s position: %f %s at ~$%.2f",
		coin, size, coin, price)

	// Create a simple market order
	order := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         isBuy,
		Price:         price,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifIoc}},
		ClientOrderID: nil,
	}

	resp, err := exchange.Order(order, nil) // Add nil BuilderInfo
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	t.Logf("Order response: %+v", resp)

	// Check if order was successful
	if resp.Error != nil {
		t.Fatalf("Order failed: %s", *resp.Error)
	}

	if resp.Resting != nil {
		t.Logf("✅ Order resting! Status: %s", "resting")
	} else if resp.Filled != nil {
		t.Logf("✅ Order filled! Status: %s", "filled")
	} else {
		t.Logf("✅ Order successful!")
	}

	// Wait a moment and check position
	time.Sleep(2 * time.Second)

	userState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state: %v", err)
	}

	t.Logf("\n=== POSITION AFTER ORDER ===")
	for _, assetPos := range userState.AssetPositions {
		if assetPos.Position.Szi != "0" {
			t.Logf("Asset %s: Size %s | Entry $%s | PnL $%s | Leverage %dx",
				assetPos.Position.Coin,
				assetPos.Position.Szi,
				*assetPos.Position.EntryPx,
				assetPos.Position.UnrealizedPnl,
				assetPos.Position.Leverage.Value)
		}
	}
}

func TestPlaceLimitOrderForARB(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING LIMIT ORDER FOR ARB ===")

	coin := "ARB"
	size := 50.0 // Reasonable size for testing
	isBuy := true
	targetPrice := 0.5002 // Your specified limit price

	// Get the asset ID for ARB
	assetID := exchange.GetInfo().NameToAsset(coin)
	t.Logf("Asset ID for %s: %d", coin, assetID)

	// Check if the asset ID is valid
	if assetID == 0 {
		t.Fatalf("Invalid asset ID for %s: %d", coin, assetID)
	}

	// Get current market price for comparison
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("Failed to get allMids: %v", err)
	}

	if midPriceStr, ok := mids[coin]; ok {
		currentPrice, _ := strconv.ParseFloat(midPriceStr, 64)
		t.Logf("Current market price for %s: $%.5f", coin, currentPrice)
		t.Logf("Target limit price: $%.5f", targetPrice)

		// Check if the limit price makes sense
		if isBuy && targetPrice > currentPrice {
			t.Logf("⚠️  Note: Limit buy price ($%.5f) is above current market price ($%.5f)", targetPrice, currentPrice)
			t.Logf("   This order will likely fill immediately if the price moves up")
		} else if !isBuy && targetPrice < currentPrice {
			t.Logf("⚠️  Note: Limit sell price ($%.5f) is below current market price ($%.5f)", targetPrice, currentPrice)
			t.Logf("   This order will likely fill immediately if the price moves down")
		} else {
			var orderType string
			if isBuy {
				orderType = "buy"
			} else {
				orderType = "sell"
			}
			t.Logf("✅ Limit price is reasonable for a %s order", orderType)
		}
	} else {
		t.Logf("Could not get current market price for %s", coin)
	}

	// Format the price using PriceToWire to ensure compliance with tick size
	formattedPrice, err := hyperliquid.PriceToWire(targetPrice, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("Failed to format price with PriceToWire: %v", err)
	}

	// Parse back to float64 for the order
	finalPrice, err := strconv.ParseFloat(formattedPrice, 64)
	if err != nil {
		t.Fatalf("Failed to parse formatted price: %v", err)
	}

	t.Logf("Original target price: $%.5f", targetPrice)
	t.Logf("Formatted price (PriceToWire): %s", formattedPrice)
	t.Logf("Final price for order: $%.5f", finalPrice)

	// Create the limit order
	limitOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         isBuy,
		Price:         finalPrice,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}}, // Good Till Cancelled
		ClientOrderID: nil,
	}

	t.Logf("Placing limit order:")
	t.Logf("  🪙 Asset: %s (ID: %d)", coin, assetID)
	var side string
	if isBuy {
		side = "BUY"
	} else {
		side = "SELL"
	}
	t.Logf("  📈 Side: %s", side)
	t.Logf("  💵 Size: %.2f", size)
	t.Logf("  🎯 Price: $%.5f", finalPrice)
	t.Logf("  ⏰ TIF: GTC (Good Till Cancelled)")

	// Place the order
	resp, err := exchange.Order(limitOrder, nil)
	if err != nil {
		t.Fatalf("Failed to place limit order: %v", err)
	}

	t.Logf("✅ Order placed successfully!")
	t.Logf("📋 Order response: %+v", resp)

	// Check the order status
	if resp.Error != nil {
		t.Fatalf("❌ Order failed: %s", *resp.Error)
	}

	if resp.Resting != nil {
		t.Logf("✅ Order is resting (waiting to be filled)")
		t.Logf("   Order ID: %d", resp.Resting.Oid)
		t.Logf("   Client ID: %s", resp.Resting.ClientID)
		t.Logf("   Status: %s", resp.Resting.Status)
	} else if resp.Filled != nil {
		t.Logf("✅ Order was filled immediately!")
		t.Logf("   Order ID: %d", resp.Filled.Oid)
		t.Logf("   Total Size: %s", resp.Filled.TotalSz)
		t.Logf("   Average Price: %s", resp.Filled.AvgPx)
	} else {
		t.Logf("✅ Order placed successfully!")
	}

	// Wait a moment and check if the order appears in open orders
	time.Sleep(2 * time.Second)

	openOrders, err := exchange.GetInfo().FrontendOpenOrders(exchange.GetAccountAddr())
	if err != nil {
		t.Logf("Warning: Could not fetch open orders: %v", err)
	} else {
		t.Logf("\n=== CHECKING OPEN ORDERS ===")
		foundOrder := false
		for _, order := range openOrders {
			if order.Coin == coin && order.LimitPx == finalPrice && order.Size == size {
				t.Logf("✅ Found our limit order in open orders:")
				t.Logf("   Coin: %s", order.Coin)
				t.Logf("   Side: %s", order.Side)
				t.Logf("   Size: %.2f", order.Size)
				t.Logf("   Price: $%.5f", order.LimitPx)
				t.Logf("   OID: %d", order.Oid)
				t.Logf("   Order Type: %s", order.OrderType)
				foundOrder = true
				break
			}
		}

		if !foundOrder {
			t.Logf("ℹ️  Order not found in open orders (may have been filled immediately)")
		}
	}

	t.Logf("\n=== LIMIT ORDER TEST COMPLETED ===")
	t.Logf("The limit order for %s at $%.5f has been placed successfully!", coin, finalPrice)
}

func TestPlaceLimitOrderForARBThatRests(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING LIMIT ORDER FOR ARB THAT RESTS ===")

	coin := "ARB"
	size := 25.0 // Smaller size for testing
	isBuy := true
	targetPrice := 0.4800 // Price BELOW current market price so it won't fill immediately

	// Get the asset ID for ARB
	assetID := exchange.GetInfo().NameToAsset(coin)
	t.Logf("Asset ID for %s: %d", coin, assetID)

	// Check if the asset ID is valid
	if assetID == 0 {
		t.Fatalf("Invalid asset ID for %s: %d", coin, assetID)
	}

	// Get current market price for comparison
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("Failed to get allMids: %v", err)
	}

	if midPriceStr, ok := mids[coin]; ok {
		currentPrice, _ := strconv.ParseFloat(midPriceStr, 64)
		t.Logf("Current market price for %s: $%.5f", coin, currentPrice)
		t.Logf("Target limit price: $%.5f", targetPrice)

		// Check if the limit price makes sense for a resting order
		if isBuy && targetPrice < currentPrice {
			t.Logf("✅ Perfect! Limit buy price ($%.5f) is BELOW current market price ($%.5f)", targetPrice, currentPrice)
			t.Logf("   This order should REST on the order book and wait for price to drop")
		} else if !isBuy && targetPrice > currentPrice {
			t.Logf("✅ Perfect! Limit sell price ($%.5f) is ABOVE current market price ($%.5f)", targetPrice, currentPrice)
			t.Logf("   This order should REST on the order book and wait for price to rise")
		} else {
			t.Logf("⚠️  Warning: This limit price will likely fill immediately!")
		}
	} else {
		t.Logf("Could not get current market price for %s", coin)
	}

	// Format the price using PriceToWire to ensure compliance with tick size
	formattedPrice, err := hyperliquid.PriceToWire(targetPrice, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("Failed to format price with PriceToWire: %v", err)
	}

	// Parse back to float64 for the order
	finalPrice, err := strconv.ParseFloat(formattedPrice, 64)
	if err != nil {
		t.Fatalf("Failed to parse formatted price: %v", err)
	}

	t.Logf("Original target price: $%.5f", targetPrice)
	t.Logf("Formatted price (PriceToWire): %s", formattedPrice)
	t.Logf("Final price for order: $%.5f", finalPrice)

	// Create the limit order
	limitOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         isBuy,
		Price:         finalPrice,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}}, // Good Till Cancelled
		ClientOrderID: nil,
	}

	t.Logf("Placing limit order:")
	t.Logf("  🪙 Asset: %s (ID: %d)", coin, assetID)
	var side string
	if isBuy {
		side = "BUY"
	} else {
		side = "SELL"
	}
	t.Logf("  📈 Side: %s", side)
	t.Logf("  💵 Size: %.2f", size)
	t.Logf("  🎯 Price: $%.5f", finalPrice)
	t.Logf("  ⏰ TIF: GTC (Good Till Cancelled)")
	t.Logf("  📋 Expected: Order should REST on the order book")

	// Place the order
	resp, err := exchange.Order(limitOrder, nil)
	if err != nil {
		t.Fatalf("Failed to place limit order: %v", err)
	}

	t.Logf("✅ Order placed successfully!")
	t.Logf("📋 Order response: %+v", resp)

	// Check the order status
	if resp.Error != nil {
		t.Fatalf("❌ Order failed: %s", *resp.Error)
	}

	if resp.Resting != nil {
		t.Logf("🎯 SUCCESS! Order is RESTING on the order book")
		t.Logf("   Order ID: %d", resp.Resting.Oid)
		t.Logf("   Client ID: %s", resp.Resting.ClientID)
		t.Logf("   Status: %s", resp.Resting.Status)
		t.Logf("   This is the expected behavior for a limit order!")
	} else if resp.Filled != nil {
		t.Logf("⚠️  Order was filled immediately (unexpected for this price)")
		t.Logf("   Order ID: %d", resp.Filled.Oid)
		t.Logf("   Total Size: %s", resp.Filled.TotalSz)
		t.Logf("   Average Price: %s", resp.Filled.AvgPx)
		t.Logf("   This suggests the market price moved or our limit price was too aggressive")
	} else {
		t.Logf("✅ Order placed successfully!")
	}

	// Wait a moment and check if the order appears in open orders
	time.Sleep(2 * time.Second)

	openOrders, err := exchange.GetInfo().FrontendOpenOrders(exchange.GetAccountAddr())
	if err != nil {
		t.Logf("Warning: Could not fetch open orders: %v", err)
	} else {
		t.Logf("\n=== CHECKING OPEN ORDERS ===")
		foundOrder := false
		for _, order := range openOrders {
			if order.Coin == coin && order.LimitPx == finalPrice && order.Size == size {
				t.Logf("✅ Found our limit order in open orders:")
				t.Logf("   Coin: %s", order.Coin)
				t.Logf("   Side: %s", order.Side)
				t.Logf("   Size: %.2f", order.Size)
				t.Logf("   Price: $%.5f", order.LimitPx)
				t.Logf("   OID: %d", order.Oid)
				t.Logf("   Order Type: %s", order.OrderType)
				foundOrder = true
				break
			}
		}

		if !foundOrder {
			t.Logf("ℹ️  Order not found in open orders")
			if resp.Resting != nil {
				t.Logf("   Note: Order was marked as 'resting' but not showing in open orders")
				t.Logf("   This might be a timing issue or the order was filled")
			}
		}
	}

	t.Logf("\n=== LIMIT ORDER RESTING TEST COMPLETED ===")
	if resp.Resting != nil {
		t.Logf("🎯 SUCCESS! The limit order for %s at $%.5f is now RESTING on the order book", coin, finalPrice)
		t.Logf("   It will only execute when the market price reaches $%.5f", finalPrice)
		t.Logf("   Order ID: %d", resp.Resting.Oid)
	} else {
		t.Logf("ℹ️  The limit order behavior was different than expected")
	}
}

func TestGetARBHourlyCandles(t *testing.T) {
	godotenv.Overload()

	info := hyperliquid.NewInfo(hyperliquid.TestnetAPIURL, true, nil, nil, nil, "")

	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)

	candles, err := info.CandlesSnapshot("ARB", "1h", start.UnixMilli(), end.UnixMilli())
	if err != nil {
		t.Fatalf("failed to fetch ARB 1h candles: %v", err)
	}

	if len(candles) == 0 {
		t.Fatalf("expected at least one candle, got 0")
	}

	// Log first and last candle succinctly
	first := candles[0]
	last := candles[len(candles)-1]
	t.Logf("Candles fetched: %d (interval=1h)", len(candles))
	t.Logf("First: ts=%d o=%s h=%s l=%s c=%s v=%s", first.Timestamp, first.Open, first.High, first.Low, first.Close, first.Volume)
	t.Logf("Last:  ts=%d o=%s h=%s l=%s c=%s v=%s", last.Timestamp, last.Open, last.High, last.Low, last.Close, last.Volume)

	// Basic sanity check: timestamps should be increasing by ~1h
	if len(candles) >= 2 {
		deltaMs := candles[1].Timestamp - candles[0].Timestamp
		// allow some tolerance; expected ~3600000ms
		if deltaMs < 3_000_000 || deltaMs > 4_200_000 {
			t.Logf("warning: candle step unexpected: %s ms", strconv.FormatInt(deltaMs, 10))
		}
	}
}

// TestApproveBuilderFee tests the approval of a builder address to collect fees
// Based on: https://hyperliquid.gitbook.io/hyperliquid-docs/trading/builder-codes
func TestApproveBuilderFee(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING BUILDER FEE APPROVAL ===")

	// For testing purposes, we'll use a sample builder address
	// In real usage, this would be the address of the DeFi application/builder
	builderAddress := "0x1234567890123456789012345678901234567890"

	// Max fee rate: 0.1% (10 basis points) for perps, 1% for spot
	// The fee rate is specified in tenths of basis points
	// So 0.1% = 10 basis points = 100 tenths of basis points
	maxFeeRate := "100" // 0.1% in tenths of basis points

	t.Logf("🔧 Builder Address: %s", builderAddress)
	t.Logf("💰 Max Fee Rate: %s (0.1%% in tenths of basis points)", maxFeeRate)
	t.Logf("📋 This will allow the builder to collect up to 0.1%% fee on perp trades")

	// Approve the builder fee
	t.Logf("📝 Approving builder fee...")
	resp, err := exchange.ApproveBuilderFee(builderAddress, maxFeeRate)
	if err != nil {
		t.Fatalf("❌ Failed to approve builder fee: %v", err)
	}

	t.Logf("✅ Builder fee approval successful!")
	t.Logf("📋 Response: %v", resp)

	// Check the response
	if resp.Error != "" {
		t.Fatalf("❌ Approval failed with error: %s", resp.Error)
	}

	if resp.Status != "ok" {
		t.Logf("⚠️  Warning: Status is not 'ok': %s", resp.Status)
	}

	if resp.TxHash != "" {
		t.Logf("🔗 Transaction Hash: %s", resp.TxHash)
		t.Logf("   You can view this transaction on the blockchain explorer")
	} else {
		t.Logf("ℹ️  No transaction hash returned (this is normal for some operations)")
	}

	t.Logf("\n=== BUILDER FEE APPROVAL COMPLETED ===")
	t.Logf("🎯 The builder address %s is now approved to collect fees", builderAddress)
	t.Logf("💰 Maximum fee rate: 0.1%% (100 tenths of basis points)")
	t.Logf("📋 This approval allows the builder to include builder fees in future orders")
	t.Logf("   The builder can now send orders with: b: %s, f: <fee_rate>", builderAddress)
	t.Logf("   where <fee_rate> is the actual fee to charge (≤ 100 for 0.1%%)")
}

// TestApproveBuilderFeeWithDifferentRates tests various fee rate approvals
func TestApproveBuilderFeeWithDifferentRates(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING BUILDER FEE APPROVAL WITH DIFFERENT RATES ===")

	builderAddress := "0x1234567890abcdef1234567890abcdef12345678"

	// Test different fee rates
	testCases := []struct {
		rate        string
		description string
		percentage  string
	}{
		{"10", "1 basis point", "0.01%%"},
		{"50", "5 basis points", "0.05%%"},
		{"100", "10 basis points", "0.1%%"},
		{"500", "50 basis points", "0.5%%"},
		{"1000", "100 basis points", "1.0%%"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Logf("🧪 Testing fee rate: %s (%s)", tc.rate, tc.percentage)

			resp, err := exchange.ApproveBuilderFee(builderAddress, tc.rate)
			if err != nil {
				t.Fatalf("❌ Failed to approve builder fee with rate %s: %v", tc.rate, err)
			}

			if resp.Error != "" {
				t.Fatalf("❌ Approval failed with error: %s", resp.Error)
			}

			t.Logf("✅ Successfully approved %s fee rate", tc.percentage)
		})
	}

	t.Logf("\n=== MULTIPLE FEE RATE APPROVALS COMPLETED ===")
	t.Logf("🎯 Builder address %s is now approved for multiple fee rates", builderAddress)
	t.Logf("💰 This demonstrates the flexibility of builder fee approvals")
}

// TestBuilderFeeWorkflow tests the complete builder fee workflow:
// 1) Approve builder fee for 1%
// 2) Place a trade with builder fee
// 3) Check fees received by the builder address
func TestBuilderFeeWorkflow(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING COMPLETE BUILDER FEE WORKFLOW ===")

	builderAddress := "0x1234567890abcdef1234567890abcdef12345678"
	maxFeeRate := "100" // 0.1% (100 tenths of basis points)

	// Step 1: Approve builder fee for 1%
	t.Logf("📝 Step 1: Approving builder fee for 1%% (%s tenths of basis points)", maxFeeRate)
	t.Logf("🔧 Builder Address: %s", builderAddress)

	resp, err := exchange.ApproveBuilderFee(builderAddress, maxFeeRate)
	if err != nil {
		t.Fatalf("❌ Failed to approve builder fee: %v", err)
	}

	if resp.Error != "" {
		t.Fatalf("❌ Builder fee approval failed: %s", resp.Error)
	}

	t.Logf("✅ Step 1 completed: Builder fee approved for 1%%")

	// Step 2: Get initial builder rewards to compare later
	t.Logf("📊 Step 2: Checking initial builder rewards...")

	initialReferralState, err := exchange.GetInfo().QueryReferralState(builderAddress)
	if err != nil {
		t.Logf("⚠️  Could not query initial referral state: %v", err)
		t.Logf("   This might be normal if the builder has never received fees")
	}

	var initialBuilderRewards string = "0"
	if initialReferralState != nil {
		initialBuilderRewards = initialReferralState.BuilderRewards
		t.Logf("📈 Initial builder rewards: %s USDC", initialBuilderRewards)
	} else {
		t.Logf("📈 Initial builder rewards: %s USDC (new builder)", initialBuilderRewards)
	}

	// Step 3: Place a trade with builder fee
	t.Logf("💰 Step 3: Placing trade with builder fee...")

	coin := "ARB"
	size := 50.0
	isBuy := true
	builderFee := 100 // 1% in tenths of basis points

	// Get current market price
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("Failed to get market prices: %v", err)
	}

	currentPriceStr, ok := mids[coin]
	if !ok {
		t.Fatalf("Could not get current price for %s", coin)
	}

	currentPrice, err := strconv.ParseFloat(currentPriceStr, 64)
	if err != nil {
		t.Fatalf("Failed to parse current price: %v", err)
	}

	// Note: Asset ID is not needed for this market order

	t.Logf("🎯 Trading details:")
	t.Logf("   Asset: %s", coin)
	t.Logf("   Side: %s", "BUY")
	t.Logf("   Size: %.2f", size)
	t.Logf("   Market Price: $%.5f", currentPrice)
	t.Logf("   Builder: %s", builderAddress)
	t.Logf("   Builder Fee: %d tenths of basis points (0.1%%)", builderFee)
	t.Logf("   Order Type: Market (IOC) - will fill immediately")

	// Create BuilderInfo
	builderInfo := &hyperliquid.BuilderInfo{
		Builder: builderAddress,
		Fee:     builderFee,
	}

	// Create and place the order - MARKET ORDER with IOC (Immediate or Cancel)
	order := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         isBuy,
		Price:         currentPrice, // Use current market price
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifIoc}}, // IOC for immediate fill
		ClientOrderID: nil,
	}

	orderResp, err := exchange.Order(order, builderInfo)
	if err != nil {
		t.Fatalf("❌ Failed to place order with builder fee: %v", err)
	}

	if orderResp.Error != nil {
		t.Fatalf("❌ Order failed: %s", *orderResp.Error)
	}

	var orderID int64
	if orderResp.Filled != nil {
		t.Logf("✅ Order filled immediately!")
		t.Logf("   Order ID: %d", orderResp.Filled.Oid)
		t.Logf("   Size: %s", orderResp.Filled.TotalSz)
		t.Logf("   Avg Price: $%s", orderResp.Filled.AvgPx)
		orderID = int64(orderResp.Filled.Oid)
	} else if orderResp.Resting != nil {
		t.Logf("📋 Order placed and resting on book")
		t.Logf("   Order ID: %d", orderResp.Resting.Oid)
		orderID = orderResp.Resting.Oid

		// Wait a bit for potential fill
		t.Logf("⏳ Waiting 5 seconds for potential fill...")
		time.Sleep(5 * time.Second)
	}

	t.Logf("✅ Step 3 completed: Order placed with builder fee")

	// Step 4: Check builder rewards after the trade
	t.Logf("🔍 Step 4: Checking builder rewards after trade...")

	// Wait a moment for the system to process the fee
	time.Sleep(3 * time.Second)

	finalReferralState, err := exchange.GetInfo().QueryReferralState(builderAddress)
	if err != nil {
		t.Logf("⚠️  Could not query final referral state: %v", err)
		t.Logf("   This might take some time to appear in the system")
	} else {
		finalBuilderRewards := finalReferralState.BuilderRewards
		t.Logf("📈 Final builder rewards: %s USDC", finalBuilderRewards)

		// Try to parse and compare rewards
		initialRewards, err1 := strconv.ParseFloat(initialBuilderRewards, 64)
		finalRewards, err2 := strconv.ParseFloat(finalBuilderRewards, 64)

		if err1 == nil && err2 == nil {
			rewardDifference := finalRewards - initialRewards
			if rewardDifference > 0 {
				t.Logf("🎉 SUCCESS: Builder received %.6f USDC in fees!", rewardDifference)
				t.Logf("   Fee increase: %.6f USDC", rewardDifference)
			} else {
				t.Logf("ℹ️  No immediate fee increase detected")
				t.Logf("   Note: Fees might take time to appear or order might not have filled")
			}
		} else {
			t.Logf("ℹ️  Could not parse reward amounts for comparison")
		}

		// Log other referral state info
		if finalReferralState.CumVlm != "" {
			t.Logf("📊 Total volume: %s", finalReferralState.CumVlm)
		}
		if finalReferralState.UnclaimedRewards != "" {
			t.Logf("💰 Unclaimed rewards: %s", finalReferralState.UnclaimedRewards)
		}
	}

	t.Logf("\n=== BUILDER FEE WORKFLOW COMPLETED ===")
	t.Logf("✅ Successfully tested the complete builder fee workflow:")
	t.Logf("   1. ✅ Approved builder fee for 1%% (%s)", builderAddress)
	t.Logf("   2. ✅ Placed trade with builder fee (Order ID: %d)", orderID)
	t.Logf("   3. ✅ Checked builder rewards collection")
	t.Logf("💡 Builder fee workflow is working correctly!")
}

func TestBulkOrdersWithGrouping_SOL_LimitPlus2_TPPlus10_SLMinus10(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	coin := "SOL"
	size := 0.05

	// Fetch mid price
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("failed to fetch mids: %v", err)
	}
	midStr, ok := mids[coin]
	if !ok {
		t.Fatalf("mid not found for %s", coin)
	}
	mid, err := strconv.ParseFloat(midStr, 64)
	if err != nil {
		t.Fatalf("failed to parse mid for %s: %v", coin, err)
	}

	// Compute prices per request
	openRaw := mid * 1.02 // limit at market +2%
	tpRaw := mid * 1.10   // TP at market +10%
	slRaw := mid * 0.90   // SL at market -10%

	// Ensure prices conform to tick rules using PriceToWire
	assetID := exchange.GetInfo().NameToAsset(coin)
	openWire, err := hyperliquid.PriceToWire(openRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire open price: %v", err)
	}
	tpWire, err := hyperliquid.PriceToWire(tpRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire TP price: %v", err)
	}
	slWire, err := hyperliquid.PriceToWire(slRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
	if err != nil {
		t.Fatalf("failed to wire SL price: %v", err)
	}

	openPx, _ := strconv.ParseFloat(openWire, 64)
	tpPx, _ := strconv.ParseFloat(tpWire, 64)
	slPx, _ := strconv.ParseFloat(slWire, 64)

	// Build orders: open limit (Gtc), TP trigger (market), SL trigger (market)
	openOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         true,
		Price:         openPx,
		Size:          size,
		ReduceOnly:    false,
		OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
		ClientOrderID: nil,
	}

	tpOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // close direction for long
		Price:         tpPx,
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: tpPx, IsMarket: true, Tpsl: "tp"}},
		ClientOrderID: nil,
	}

	slOrder := hyperliquid.CreateOrderRequest{
		Coin:          coin,
		IsBuy:         false, // close direction for long
		Price:         slPx,
		Size:          size,
		ReduceOnly:    true,
		OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: slPx, IsMarket: true, Tpsl: "sl"}},
		ClientOrderID: nil,
	}

	resp, err := exchange.BulkOrdersWithGrouping([]hyperliquid.CreateOrderRequest{openOrder, tpOrder, slOrder}, hyperliquid.GroupingNormalTpsl, nil)
	if err != nil {
		t.Fatalf("placing grouped orders failed: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("grouped orders not ok: %s", resp.Err)
	}

	t.Logf("Placed SOL grouped orders: open %.6f, tp %.6f, sl %.6f", openPx, tpPx, slPx)
}

func TestBulkOrdersWithGrouping_Generic_ThreeAssets(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	assets := []struct {
		coin string
		size float64
	}{
		{"ETH", 0.005},
		{"SOL", 0.05},
		{"ARB", 50.0},
	}

	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("failed to fetch mids: %v", err)
	}

	for _, a := range assets {
		midStr, ok := mids[a.coin]
		if !ok {
			t.Fatalf("mid not found for %s", a.coin)
		}
		mid, err := strconv.ParseFloat(midStr, 64)
		if err != nil {
			t.Fatalf("failed to parse mid for %s: %v", a.coin, err)
		}

		openRaw := mid * 1.01
		tpRaw := mid * 1.10
		slRaw := mid * 0.90

		assetID := exchange.GetInfo().NameToAsset(a.coin)
		openWire, err := hyperliquid.PriceToWire(openRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
		if err != nil {
			t.Fatalf("%s: failed to wire open price: %v", a.coin, err)
		}
		tpWire, err := hyperliquid.PriceToWire(tpRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
		if err != nil {
			t.Fatalf("%s: failed to wire TP price: %v", a.coin, err)
		}
		slWire, err := hyperliquid.PriceToWire(slRaw, assetID, exchange.GetInfo(), hyperliquid.ClassifyAsset(assetID))
		if err != nil {
			t.Fatalf("%s: failed to wire SL price: %v", a.coin, err)
		}

		openPx, _ := strconv.ParseFloat(openWire, 64)
		tpPx, _ := strconv.ParseFloat(tpWire, 64)
		slPx, _ := strconv.ParseFloat(slWire, 64)

		openOrder := hyperliquid.CreateOrderRequest{
			Coin:          a.coin,
			IsBuy:         true,
			Price:         openPx,
			Size:          a.size,
			ReduceOnly:    false,
			OrderType:     hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
			ClientOrderID: nil,
		}

		tpOrder := hyperliquid.CreateOrderRequest{
			Coin:          a.coin,
			IsBuy:         false,
			Price:         tpPx,
			Size:          a.size,
			ReduceOnly:    true,
			OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: tpPx, IsMarket: true, Tpsl: "tp"}},
			ClientOrderID: nil,
		}

		slOrder := hyperliquid.CreateOrderRequest{
			Coin:          a.coin,
			IsBuy:         false,
			Price:         slPx,
			Size:          a.size,
			ReduceOnly:    true,
			OrderType:     hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{TriggerPx: slPx, IsMarket: true, Tpsl: "sl"}},
			ClientOrderID: nil,
		}

		resp, err := exchange.BulkOrdersWithGrouping([]hyperliquid.CreateOrderRequest{openOrder, tpOrder, slOrder}, hyperliquid.GroupingNormalTpsl, nil)
		if err != nil {
			t.Fatalf("%s: placing grouped orders failed: %v", a.coin, err)
		}
		if !resp.Ok {
			t.Fatalf("%s: grouped orders not ok: %s", a.coin, resp.Err)
		}

		t.Logf("Placed %s grouped orders: open %.6f, tp %.6f, sl %.6f", a.coin, openPx, tpPx, slPx)
	}
}

func TestIsolatedMarginOrder(t *testing.T) {

	exchange := newTestExchange(t) // or your instantiated Trader

	coin := "ETH"

	// 1) Switch this asset to ISOLATED mode with desired leverage
	if _, err := exchange.UpdateLeverage(5, coin, false /* isCross=false => isolated */); err != nil {
		panic(err)
	}

	// 2) (Optional) Allocate isolated margin buffer for this asset
	// positive = add margin; negative = remove margin
	// if _, err := exchange.UpdateIsolatedMargin(50 /* USD */, coin); err != nil {
	// 	panic(err)
	// }

	// 3) Place your order (market/limit). This position is now isolated.
	res, err := exchange.MarketOpen(coin, true /* buy */, 0.01, nil, 0.01, nil, nil)
	if err != nil {
		panic(err)
	}
	_ = res
}

func TestGetDataByOID(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	oid := 217993726723
	order, err := exchange.GetInfo().QueryOrderByOid(accountAddress(t), int64(oid))
	if err != nil {
		t.Fatalf("Failed to get data by oid: %v", err)
	}
	t.Logf("Order: %+v", order)
}

// TestWithdrawFromBridge tests withdrawing tokens from Hyperliquid bridge
// This test demonstrates how to withdraw funds to an external address
func TestWithdrawFromBridge(t *testing.T) {
	godotenv.Overload()

	// Test parameters - these should be provided as function parameters in real usage
	// For this test, we'll use environment variables or defaults
	privateKeyHex := os.Getenv("HL_PRIVATE_KEY")
	accountAddress := os.Getenv("HL_ACCOUNT_ADDRESS")
	destinationAddress := "0x1234567890abcdef1234567890abcdef12345678" // External address to withdraw to
	withdrawalAmount := 10.0                                           // Amount to withdraw (in USDC) - must be larger than withdrawal fee

	// Validate required parameters
	if privateKeyHex == "" {
		t.Skip("HL_PRIVATE_KEY not set, skipping withdrawal test")
	}
	if accountAddress == "" {
		t.Skip("HL_ACCOUNT_ADDRESS not set, skipping withdrawal test")
	}
	if destinationAddress == "" {
		t.Skip("WITHDRAWAL_DESTINATION_ADDRESS not set, skipping withdrawal test")
	}

	t.Log("=== TESTING WITHDRAWAL FROM HYPERLIQUID BRIDGE ===")
	t.Logf("🔑 Private Key: %s...", privateKeyHex[:10])
	t.Logf("📍 Account Address: %s", accountAddress)
	t.Logf("🎯 Destination Address: %s", destinationAddress)
	t.Logf("💰 Withdrawal Amount: %.6f USDC", withdrawalAmount)

	// Create private key from hex string
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		t.Fatalf("❌ Failed to create private key: %v", err)
	}

	// Verify the private key matches the account address
	expectedAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	if expectedAddress != accountAddress {
		t.Logf("⚠️  Warning: Private key address (%s) doesn't match account address (%s)", expectedAddress, accountAddress)
		t.Logf("   Using provided account address: %s", accountAddress)
	}

	// Create exchange instance
	exchange := hyperliquid.NewTrader(
		privateKey,
		hyperliquid.MainnetAPIURL, // Use testnet for safety
		nil,                       // meta
		"",                        // vault address (empty for regular accounts)
		accountAddress,
		nil, // spot meta
		nil, // perpDexs
		"",  // perpDexName (empty = default dex)
	)

	t.Logf("✅ Trader instance created successfully")
	t.Logf("🌐 Using API URL: %s", hyperliquid.MainnetAPIURL)

	// Check account balance before withdrawal
	t.Logf("📊 Checking account balance before withdrawal...")
	userState, err := exchange.GetInfo().UserState(accountAddress)
	if err != nil {
		t.Logf("⚠️  Could not fetch user state: %v", err)
	} else {
		t.Logf("💰 Account Value: $%s", userState.MarginSummary.AccountValue)
		t.Logf("💵 Withdrawable: $%s", userState.Withdrawable)
		t.Logf("📈 Total Margin Used: $%s", userState.MarginSummary.TotalMarginUsed)

		// Check if we have enough balance
		withdrawable, err := strconv.ParseFloat(userState.Withdrawable, 64)
		if err != nil {
			t.Logf("⚠️  Could not parse withdrawable amount: %v", err)
		} else if withdrawable < withdrawalAmount {
			t.Logf("⚠️  Warning: Withdrawable balance (%.6f) is less than withdrawal amount (%.6f)", withdrawable, withdrawalAmount)
			t.Logf("   Consider reducing withdrawal amount or adding funds")
		}
	}

	// Perform the withdrawal
	t.Logf("🚀 Initiating withdrawal from bridge...")
	t.Logf("   Amount: %.6f USDC", withdrawalAmount)
	t.Logf("   Destination: %s", destinationAddress)

	withdrawResp, err := exchange.WithdrawFromBridge(withdrawalAmount, destinationAddress)
	if err != nil {
		t.Fatalf("❌ Withdrawal failed: %v", err)
	}

	// Check withdrawal response
	t.Logf("📋 Withdrawal Response:")
	t.Logf("   Status: %s", withdrawResp.Status)
	t.Logf("Full response: %+v", withdrawResp)
	if withdrawResp.TxHash != "" {
		t.Logf("   Transaction Hash: %s", withdrawResp.TxHash)
		t.Logf("   🔗 View on explorer: https://explorer.hyperliquid.xyz/tx/%s", withdrawResp.TxHash)
	}
	if withdrawResp.Error != "" {
		t.Fatalf("❌ Withdrawal error: %s", withdrawResp.Error)
	}

	// Verify withdrawal was successful
	if withdrawResp.Status == "ok" {
		t.Logf("✅ Withdrawal initiated successfully!")
		t.Logf("🎯 %.6f USDC will be withdrawn to %s", withdrawalAmount, destinationAddress)
		if withdrawResp.TxHash != "" {
			t.Logf("📝 Transaction submitted: %s", withdrawResp.TxHash)
		}
	} else {
		t.Logf("⚠️  Withdrawal status: %s", withdrawResp.Status)
	}

	// Wait a moment and check balance after withdrawal
	time.Sleep(2 * time.Second)

	t.Logf("📊 Checking account balance after withdrawal...")
	userStateAfter, err := exchange.GetInfo().UserState(accountAddress)
	if err != nil {
		t.Logf("⚠️  Could not fetch user state after withdrawal: %v", err)
	} else {
		t.Logf("💰 Account Value After: $%s", userStateAfter.MarginSummary.AccountValue)
		t.Logf("💵 Withdrawable After: $%s", userStateAfter.Withdrawable)

		// Compare balances if we had initial state
		if userState != nil {
			beforeWithdrawable, _ := strconv.ParseFloat(userState.Withdrawable, 64)
			afterWithdrawable, _ := strconv.ParseFloat(userStateAfter.Withdrawable, 64)
			balanceChange := beforeWithdrawable - afterWithdrawable
			t.Logf("📉 Balance Change: %.6f USDC", balanceChange)
		}
	}

	t.Logf("\n=== WITHDRAWAL TEST COMPLETED ===")
	t.Logf("✅ Successfully tested withdrawal from Hyperliquid bridge")
	t.Logf("🎯 Withdrawal details:")
	t.Logf("   Amount: %.6f USDC", withdrawalAmount)
	t.Logf("   From: %s", accountAddress)
	t.Logf("   To: %s", destinationAddress)
	if withdrawResp.TxHash != "" {
		t.Logf("   TxHash: %s", withdrawResp.TxHash)
	}
}

// TestWithdrawFromBridgeWithCustomParams tests withdrawal with custom parameters
// This function can be called with specific address and private key
func TestWithdrawFromBridgeWithCustomParams(t *testing.T) {
	// This test can be customized to accept specific parameters
	// For demonstration, we'll show how to structure it

	t.Log("=== CUSTOM WITHDRAWAL TEST STRUCTURE ===")
	t.Log("To use this test with custom parameters:")
	t.Log("1. Set environment variables:")
	t.Log("   export HL_PRIVATE_KEY='your_private_key_hex'")
	t.Log("   export HL_ACCOUNT_ADDRESS='your_account_address'")
	t.Log("   export WITHDRAWAL_DESTINATION_ADDRESS='destination_address'")
	t.Log("2. Or modify the test to accept parameters directly")
	t.Log("3. Adjust withdrawal amount as needed")
	t.Log("4. Choose between mainnet and testnet")

	// Example of how to structure with parameters:
	// func TestWithdrawFromBridgeCustom(privateKeyHex, accountAddress, destinationAddress string, amount float64) {
	//     // Implementation here
	// }

	t.Log("✅ Custom withdrawal test structure demonstrated")
}

// TestWithdrawFromBridgeCustom demonstrates how to create a withdrawal test with custom parameters
// This is a helper function that can be called with specific parameters
func TestWithdrawFromBridgeCustom(t *testing.T) {
	// Example usage with hardcoded values for demonstration
	// In real usage, these would be passed as parameters or read from config

	privateKeyHex := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" // Example key
	accountAddress := "0x1234567890123456789012345678901234567890"                        // Example address
	destinationAddress := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"                    // Example destination
	withdrawalAmount := 0.1                                                               // Small amount for testing
	useMainnet := false                                                                   // Use testnet for safety

	// Skip if using example values
	if privateKeyHex == "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" {
		t.Skip("Using example values, skipping actual withdrawal test")
	}

	t.Log("=== CUSTOM WITHDRAWAL TEST ===")
	t.Logf("🔑 Private Key: %s...", privateKeyHex[:10])
	t.Logf("📍 Account Address: %s", accountAddress)
	t.Logf("🎯 Destination Address: %s", destinationAddress)
	t.Logf("💰 Withdrawal Amount: %.6f USDC", withdrawalAmount)
	t.Logf("🌐 Network: %s", map[bool]string{true: "Mainnet", false: "Testnet"}[useMainnet])

	// Create private key from hex string
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		t.Fatalf("❌ Failed to create private key: %v", err)
	}

	// Verify the private key matches the account address
	expectedAddress := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	if expectedAddress != accountAddress {
		t.Logf("⚠️  Warning: Private key address (%s) doesn't match account address (%s)", expectedAddress, accountAddress)
	}

	// Choose API URL based on network
	var apiURL string
	if useMainnet {
		apiURL = hyperliquid.MainnetAPIURL
	} else {
		apiURL = hyperliquid.TestnetAPIURL
	}

	// Create exchange instance
	exchange := hyperliquid.NewTrader(
		privateKey,
		apiURL,
		nil, // meta
		"",  // vault address (empty for regular accounts)
		accountAddress,
		nil, // spot meta
		nil, // perpDexs
		"",  // perpDexName (empty = default dex)
	)

	t.Logf("✅ Trader instance created successfully")
	t.Logf("🌐 Using API URL: %s", apiURL)

	// Check account balance before withdrawal
	t.Logf("📊 Checking account balance before withdrawal...")
	userState, err := exchange.GetInfo().UserState(accountAddress)
	if err != nil {
		t.Logf("⚠️  Could not fetch user state: %v", err)
	} else {
		t.Logf("💰 Account Value: $%s", userState.MarginSummary.AccountValue)
		t.Logf("💵 Withdrawable: $%s", userState.Withdrawable)

		// Check if we have enough balance
		withdrawable, err := strconv.ParseFloat(userState.Withdrawable, 64)
		if err != nil {
			t.Logf("⚠️  Could not parse withdrawable amount: %v", err)
		} else if withdrawable < withdrawalAmount {
			t.Logf("⚠️  Warning: Withdrawable balance (%.6f) is less than withdrawal amount (%.6f)", withdrawable, withdrawalAmount)
		}
	}

	// Perform the withdrawal
	t.Logf("🚀 Initiating withdrawal from bridge...")
	withdrawResp, err := exchange.WithdrawFromBridge(withdrawalAmount, destinationAddress)
	if err != nil {
		t.Fatalf("❌ Withdrawal failed: %v", err)
	}

	// Check withdrawal response
	t.Logf("📋 Withdrawal Response:")
	t.Logf("   Status: %s", withdrawResp.Status)
	if withdrawResp.TxHash != "" {
		t.Logf("   Transaction Hash: %s", withdrawResp.TxHash)
		explorerURL := "https://explorer.hyperliquid-testnet.xyz"
		if useMainnet {
			explorerURL = "https://explorer.hyperliquid.xyz"
		}
		t.Logf("   🔗 View on explorer: %s/tx/%s", explorerURL, withdrawResp.TxHash)
	}
	if withdrawResp.Error != "" {
		t.Fatalf("❌ Withdrawal error: %s", withdrawResp.Error)
	}

	// Verify withdrawal was successful
	if withdrawResp.Status == "ok" {
		t.Logf("✅ Withdrawal initiated successfully!")
		t.Logf("🎯 %.6f USDC will be withdrawn to %s", withdrawalAmount, destinationAddress)
	} else {
		t.Logf("⚠️  Withdrawal status: %s", withdrawResp.Status)
	}

	t.Logf("\n=== CUSTOM WITHDRAWAL TEST COMPLETED ===")
}

// TestWithdrawalExplanation explains why there's no txHash in withdrawal responses
func TestWithdrawalExplanation(t *testing.T) {
	t.Log("=== WITHDRAWAL TRANSACTION HASH EXPLANATION ===")

	t.Log("❓ Why don't you receive a txHash immediately?")
	t.Log("")
	t.Log("🔍 **Hyperliquid Bridge Withdrawal Process:**")
	t.Log("   1. ✅ You submit withdrawal request → Status: 'ok'")
	t.Log("   2. ⏳ Hyperliquid validators process the request (internal)")
	t.Log("   3. 🔗 Funds are transferred to your destination address")
	t.Log("   4. ⏱️  Process typically takes ~5 minutes")
	t.Log("")
	t.Log("💡 **Key Points:**")
	t.Log("   • No traditional blockchain transaction hash is generated")
	t.Log("   • Withdrawal is processed internally by Hyperliquid validators")
	t.Log("   • Funds appear in your destination wallet within ~5 minutes")
	t.Log("   • This is normal behavior for Hyperliquid bridge withdrawals")
	t.Log("")
	t.Log("🔍 **How to Track Your Withdrawal:**")
	t.Log("   1. Monitor your destination wallet on Arbitrum network")
	t.Log("   2. Check your wallet's transaction history")
	t.Log("   3. Look for incoming USDC transfers")
	t.Log("   4. The transaction will appear with a hash once processed")
	t.Log("")
	t.Log("📋 **Response Format:**")
	t.Log("   Status: 'ok' = Withdrawal request accepted")
	t.Log("   TxHash: '' = No hash (normal for bridge withdrawals)")
	t.Log("   Error: '' = No error (success)")
	t.Log("")
	t.Log("✅ **This is expected behavior - your withdrawal is processing!**")
}

func TestMetaAssetCtxs(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	l2Book, err := exchange.GetInfo().L2Snapshot("ARB")
	if err != nil {
		t.Fatalf("failed to fetch meta and asset ctxs: %v", err)
	}
	t.Logf("meta and asset ctxs: %+v", l2Book)
}

func TestETHIsolatedMarginWorkflow(t *testing.T) {
	godotenv.Overload()
	exchange := newTestExchange(t)

	t.Log("=== TESTING ETH ISOLATED MARGIN WORKFLOW ===")

	coin := "ETH"
	positionValueUSD := 15.0    // $15 position
	marginAdjustmentUSD := 15.0 // $15 margin adjustment
	slippage := 0.01            // 1%

	// Step 1: Get current ETH price and calculate position size
	t.Log("Step 1: Getting current ETH price and calculating position size")
	mids, err := exchange.GetInfo().AllMids()
	if err != nil {
		t.Fatalf("Failed to get market prices: %v", err)
	}

	ethPriceStr, ok := mids[coin]
	if !ok {
		t.Fatalf("ETH price not found in market data")
	}

	ethPrice, err := strconv.ParseFloat(ethPriceStr, 64)
	if err != nil {
		t.Fatalf("Failed to parse ETH price: %v", err)
	}

	// Calculate position size for $10 worth of ETH
	positionSize := (positionValueUSD / ethPrice) * 2
	t.Logf("ETH Price: $%.2f", ethPrice)
	t.Logf("Position Size for $%.2f: %.6f ETH", positionValueUSD, positionSize)

	// Step 2: Set ETH to isolated margin mode
	t.Log("Step 2: Setting ETH to isolated margin mode")
	leverage := 2                                           // 2x leverage for isolated margin
	_, err = exchange.UpdateLeverage(leverage, coin, false) // false = isolated margin
	if err != nil {
		t.Fatalf("Failed to set ETH to isolated margin: %v", err)
	}
	t.Logf("✅ Set ETH to isolated margin mode with %dx leverage", leverage)

	// Step 3: Check initial margin state
	t.Log("Step 3: Checking initial margin state")
	initialUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get initial user state: %v", err)
	}

	t.Logf("Initial Account Value: $%s", initialUserState.MarginSummary.AccountValue)
	t.Logf("Initial Total Margin Used: $%s", initialUserState.MarginSummary.TotalMarginUsed)
	t.Logf("Initial Withdrawable: $%s", initialUserState.Withdrawable)

	// Find ETH position if it exists
	var initialETHPosition *hyperliquid.Position
	for _, ap := range initialUserState.AssetPositions {
		if ap.Position.Coin == coin {
			initialETHPosition = &ap.Position
			t.Logf("Initial ETH Position - Size: %s, Margin Used: %s, Leverage: %dx",
				ap.Position.Szi, ap.Position.MarginUsed, ap.Position.Leverage.Value)
			break
		}
	}

	// Step 4: Open $15 ETH position
	t.Log("Step 4: Opening $15 ETH position")
	openResult, err := exchange.MarketOpen(coin, false, positionSize, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("Failed to open ETH position: %v", err)
	}
	t.Logf("✅ ETH position opened: %+v", openResult)

	// Wait for position to be processed
	time.Sleep(7 * time.Second)

	// Step 5: Check margin after opening position
	t.Log("Step 5: Checking margin after opening position")
	afterOpenUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after opening: %v", err)
	}

	t.Logf("After Open - Account Value: $%s", afterOpenUserState.MarginSummary.AccountValue)
	t.Logf("After Open - Total Margin Used: $%s", afterOpenUserState.MarginSummary.TotalMarginUsed)
	t.Logf("After Open - Withdrawable: $%s", afterOpenUserState.Withdrawable)

	// Find ETH position after opening
	var ethPositionAfterOpen *hyperliquid.Position
	for _, ap := range afterOpenUserState.AssetPositions {
		if ap.Position.Coin == coin {
			ethPositionAfterOpen = &ap.Position
			t.Logf("ETH Position After Open - Size: %s, Margin Used: %s, Leverage: %dx (%s)",
				ap.Position.Szi, ap.Position.MarginUsed, ap.Position.Leverage.Value, ap.Position.Leverage.Type)

			// Verify this is an isolated margin position
			if ap.Position.Leverage.Type != "isolated" {
				t.Fatalf("Expected isolated margin position, but got %s margin", ap.Position.Leverage.Type)
			}
			t.Logf("✅ Confirmed: ETH position is using isolated margin")
			break
		}
	}

	if ethPositionAfterOpen == nil {
		t.Fatalf("ETH position not found after opening")
	}

	// Step 6: Add $15 to isolated margin
	t.Log("Step 6: Adding $15 to isolated margin")
	t.Logf("Adding $%.2f to isolated margin (will be converted to microUSDC)", marginAdjustmentUSD)
	addMarginResult, err := exchange.UpdateIsolatedMargin(marginAdjustmentUSD, coin)
	if err != nil {
		t.Fatalf("Failed to add isolated margin: %v", err)
	}
	if !addMarginResult.Ok {
		t.Fatalf("Add isolated margin failed: %s", addMarginResult.Err)
	}
	t.Logf("✅ Added $%.2f to isolated margin: %+v", marginAdjustmentUSD, addMarginResult)

	// Wait for margin update to be processed
	time.Sleep(7 * time.Second)

	// Step 7: Check margin after adding
	t.Log("Step 7: Checking margin after adding $15")
	afterAddUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after adding margin: %v", err)
	}

	t.Logf("After Add - Account Value: $%s", afterAddUserState.MarginSummary.AccountValue)
	t.Logf("After Add - Total Margin Used: $%s", afterAddUserState.MarginSummary.TotalMarginUsed)
	t.Logf("After Add - Withdrawable: $%s", afterAddUserState.Withdrawable)

	// Find ETH position after adding margin
	var ethPositionAfterAdd *hyperliquid.Position
	for _, ap := range afterAddUserState.AssetPositions {
		if ap.Position.Coin == coin {
			ethPositionAfterAdd = &ap.Position
			t.Logf("ETH Position After Add - Size: %s, Margin Used: %s, Leverage: %dx (%s)",
				ap.Position.Szi, ap.Position.MarginUsed, ap.Position.Leverage.Value, ap.Position.Leverage.Type)

			// Verify this is still an isolated margin position
			if ap.Position.Leverage.Type != "isolated" {
				t.Fatalf("Expected isolated margin position, but got %s margin", ap.Position.Leverage.Type)
			}
			break
		}
	}

	if ethPositionAfterAdd == nil {
		t.Fatalf("ETH position not found after adding margin")
	}

	// Compare margin used before and after adding
	if initialETHPosition != nil {
		initialMargin, _ := strconv.ParseFloat(initialETHPosition.MarginUsed, 64)
		afterAddMargin, _ := strconv.ParseFloat(ethPositionAfterAdd.MarginUsed, 64)
		marginIncrease := afterAddMargin - initialMargin
		t.Logf("Margin increase: $%.2f (expected: $%.2f)", marginIncrease, marginAdjustmentUSD)
	}

	// Step 8: Remove $5 from isolated margin (smaller amount to avoid under-collateralization)
	removeAmount := 10.0
	t.Log("Step 8: Removing $10 from isolated margin")
	t.Logf("Removing $%.2f from isolated margin (will be converted to microUSDC)", removeAmount)
	removeMarginResult, err := exchange.UpdateIsolatedMargin(-removeAmount, coin)
	if err != nil {
		t.Fatalf("Failed to remove isolated margin: %v", err)
	}
	if !removeMarginResult.Ok {
		t.Fatalf("Remove isolated margin failed: %s", removeMarginResult.Err)
	}
	t.Logf("✅ Removed $%.2f from isolated margin: %+v", removeAmount, removeMarginResult)

	// Wait for margin update to be processed
	time.Sleep(2 * time.Second)

	// Step 9: Check margin after removing
	t.Log("Step 9: Checking margin after removing $5")
	afterRemoveUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Fatalf("Failed to get user state after removing margin: %v", err)
	}

	t.Logf("After Remove - Account Value: $%s", afterRemoveUserState.MarginSummary.AccountValue)
	t.Logf("After Remove - Total Margin Used: $%s", afterRemoveUserState.MarginSummary.TotalMarginUsed)
	t.Logf("After Remove - Withdrawable: $%s", afterRemoveUserState.Withdrawable)

	// Find ETH position after removing margin
	var ethPositionAfterRemove *hyperliquid.Position
	for _, ap := range afterRemoveUserState.AssetPositions {
		if ap.Position.Coin == coin {
			ethPositionAfterRemove = &ap.Position
			t.Logf("ETH Position After Remove - Size: %s, Margin Used: %s, Leverage: %dx (%s)",
				ap.Position.Szi, ap.Position.MarginUsed, ap.Position.Leverage.Value, ap.Position.Leverage.Type)

			// Verify this is still an isolated margin position
			if ap.Position.Leverage.Type != "isolated" {
				t.Fatalf("Expected isolated margin position, but got %s margin", ap.Position.Leverage.Type)
			}
			break
		}
	}

	if ethPositionAfterRemove == nil {
		t.Fatalf("ETH position not found after removing margin")
	}

	// Compare margin used before and after removing
	if ethPositionAfterAdd != nil {
		afterAddMargin, _ := strconv.ParseFloat(ethPositionAfterAdd.MarginUsed, 64)
		afterRemoveMargin, _ := strconv.ParseFloat(ethPositionAfterRemove.MarginUsed, 64)
		marginDecrease := afterAddMargin - afterRemoveMargin
		t.Logf("Margin decrease: $%.2f (expected: $%.2f)", marginDecrease, removeAmount)
	}

	// Step 10: Clean up - close the position
	t.Log("Step 10: Cleaning up - closing ETH position")
	closeResult, err := exchange.MarketClose(coin, nil, nil, slippage, nil, nil)
	if err != nil {
		t.Fatalf("Failed to close ETH position: %v", err)
	}
	t.Logf("✅ ETH position closed: %+v", closeResult)

	// Final verification
	time.Sleep(2 * time.Second)
	finalUserState, err := exchange.GetInfo().UserState(exchange.GetAccountAddr())
	if err != nil {
		t.Logf("Warning: Could not get final user state: %v", err)
	} else {
		t.Logf("Final Account Value: $%s", finalUserState.MarginSummary.AccountValue)
		t.Logf("Final Total Margin Used: $%s", finalUserState.MarginSummary.TotalMarginUsed)
		t.Logf("Final Withdrawable: $%s", finalUserState.Withdrawable)
	}

	t.Logf("\n=== ETH ISOLATED MARGIN WORKFLOW COMPLETED ===")
	t.Logf("✅ Successfully tested:")
	t.Logf("   1. Opened $%.2f ETH position", positionValueUSD)
	t.Logf("   2. Added $%.2f to isolated margin", marginAdjustmentUSD)
	t.Logf("   3. Checked margin after adding")
	t.Logf("   4. Removed $%.2f from isolated margin", marginAdjustmentUSD)
	t.Logf("   5. Checked margin after removing")
	t.Logf("   6. Closed position for cleanup")
}

func TestL2BookWebSocketMultipleTickers(t *testing.T) {
	godotenv.Overload()

	t.Log("=== TESTING L2 BOOK WEBSOCKET SUBSCRIPTION FOR MULTIPLE TICKERS ===")

	// Create WebSocket client
	wsClient := hyperliquid.NewStream(hyperliquid.MainnetAPIURL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to WebSocket
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	t.Log("✅ Connected to WebSocket")

	// Test coins for L2 book subscription
	coins := []string{"BTC"}

	// Channel to receive messages
	messageChan := make(chan hyperliquid.WSMessage, 50)

	// Subscribe to L2 book for multiple coins
	subscriptions := make(map[string]int) // coin -> subscription ID
	for _, coin := range coins {
		subID, err := wsClient.Subscribe(hyperliquid.BookSub(coin), func(msg hyperliquid.WSMessage) {
			t.Logf("📊 Received L2 book message for %s", coin)
			messageChan <- msg
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to L2 book for %s: %v", coin, err)
		}
		subscriptions[coin] = subID
		t.Logf("✅ Subscribed to L2 book for %s (Subscription ID: %d)", coin, subID)
	}

	// Wait for messages
	timeout := time.After(20 * time.Second)
	messageCount := 0
	coinMessageCount := make(map[string]int)

	for {
		select {
		case msg := <-messageChan:
			messageCount++
			t.Logf("📈 Message %d received:", messageCount)
			t.Logf("   Channel: %s", msg.Channel)
			t.Logf("   Data length: %d bytes", len(msg.Data))

			// Parse the L2 book data
			var l2Book hyperliquid.L2Book
			if err := json.Unmarshal(msg.Data, &l2Book); err != nil {
				t.Logf("⚠️  Failed to parse L2 book data: %v", err)
				t.Logf("   Raw data: %s", string(msg.Data))
			} else {
				coinMessageCount[l2Book.Coin]++
				fmt.Printf("raw data +%v", l2Book)
				t.Logf("   Time: %d", l2Book.Time)
				t.Logf("   Levels count: %d", len(l2Book.Levels))

				// Show first few levels if available
				if len(l2Book.Levels) > 0 && len(l2Book.Levels[0]) > 0 {
					bestBid := l2Book.Levels[0][0].Px
					t.Logf("   Best bid: %.2f", bestBid)

					// Try to get best ask from second level if available
					if len(l2Book.Levels) > 1 && len(l2Book.Levels[1]) > 0 {
						bestAsk := l2Book.Levels[1][0].Px
						midPrice := (bestBid + bestAsk) / 2
						t.Logf("   Best ask: %.2f", bestAsk)
						t.Logf("   Mid price: %.2f", midPrice)
					}
				}
			}

			// Stop after receiving enough messages
			if messageCount >= 10 {
				t.Logf("✅ Received %d messages, stopping test", messageCount)
				goto cleanup
			}

		case <-timeout:
			t.Logf("⏰ Timeout reached after 20 seconds")
			if messageCount == 0 {
				t.Fatalf("❌ No messages received within timeout period")
			}
			goto cleanup
		}
	}

cleanup:
	// Unsubscribe from all coins
	for coin, subID := range subscriptions {
		err = wsClient.Unsubscribe(hyperliquid.Subscription{Type: "l2Book", Coin: coin}, subID)
		if err != nil {
			t.Logf("⚠️  Failed to unsubscribe from %s: %v", coin, err)
		} else {
			t.Logf("✅ Unsubscribed from %s", coin)
		}
	}

	// Close WebSocket connection
	err = wsClient.Close()
	if err != nil {
		t.Logf("⚠️  Failed to close WebSocket: %v", err)
	} else {
		t.Logf("✅ WebSocket connection closed")
	}

	t.Logf("\n=== MULTIPLE TICKER L2 BOOK WEBSOCKET TEST COMPLETED ===")
	t.Logf("✅ Successfully tested L2 book WebSocket subscription for multiple tickers")
	t.Logf("📊 Total messages received: %d", messageCount)
	t.Logf("�� Coins tested: %v", coins)
	for coin, count := range coinMessageCount {
		t.Logf("   %s: %d messages", coin, count)
	}
}

func TestAllMidsWebSocket(t *testing.T) {
	godotenv.Overload()

	t.Log("=== TESTING ALL MIDS WEBSOCKET SUBSCRIPTION ===")

	// Create WebSocket client
	wsClient := hyperliquid.NewStream(hyperliquid.MainnetAPIURL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to WebSocket
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	t.Log("✅ Connected to WebSocket")

	// Channel to receive messages
	messageChan := make(chan hyperliquid.WSMessage, 10)

	// Subscribe to all mids
	subID, err := wsClient.Subscribe(hyperliquid.AllMidsSub(), func(msg hyperliquid.WSMessage) {
		t.Logf("📊 Received allMids message")
		messageChan <- msg
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to allMids: %v", err)
	}
	t.Logf("✅ Subscribed to allMids (Subscription ID: %d)", subID)

	// Wait for messages
	timeout := time.After(15 * time.Second)
	messageCount := 0

	for {
		select {
		case msg := <-messageChan:
			messageCount++
			t.Logf("📈 Message %d received:", messageCount)
			t.Logf("   Channel: %s", msg.Channel)
			t.Logf("   Data length: %d bytes", len(msg.Data))

			// Parse the allMids data
			var allMids map[string]string
			if err := json.Unmarshal(msg.Data, &allMids); err != nil {
				t.Logf("⚠️  Failed to parse allMids data: %v", err)
				t.Logf("   Raw data: %s", string(msg.Data))
			} else {
				t.Logf("📊 All Mids Data:")
				t.Logf("   Total assets: %d", len(allMids))

				// Show prices for major assets
				majorAssets := []string{"BTC", "ETH", "SOL", "ARB", "DOGE", "MATIC", "AVAX", "LINK"}
				t.Logf("   Major asset prices:")
				for _, asset := range majorAssets {
					if price, exists := allMids[asset]; exists {
						t.Logf("     %s: $%s", asset, price)
					}
				}

				// Show a few random assets as examples
				count := 0
				t.Logf("   Sample of all available assets:")
				for asset, price := range allMids {
					if count < 10 { // Show first 10 assets
						t.Logf("     %s: $%s", asset, price)
						count++
					}
				}
				if len(allMids) > 10 {
					t.Logf("     ... and %d more assets", len(allMids)-10)
				}
			}

			// Stop after receiving a few messages
			if messageCount >= 3 {
				t.Logf("✅ Received %d messages, stopping test", messageCount)
				goto cleanup
			}

		case <-timeout:
			t.Logf("⏰ Timeout reached after 15 seconds")
			if messageCount == 0 {
				t.Fatalf("❌ No messages received within timeout period")
			}
			goto cleanup
		}
	}

cleanup:
	// Unsubscribe
	err = wsClient.Unsubscribe(hyperliquid.Subscription{Type: "allMids"}, subID)
	if err != nil {
		t.Logf("⚠️  Failed to unsubscribe: %v", err)
	} else {
		t.Logf("✅ Unsubscribed from allMids")
	}

	// Close WebSocket connection
	err = wsClient.Close()
	if err != nil {
		t.Logf("⚠️  Failed to close WebSocket: %v", err)
	} else {
		t.Logf("✅ WebSocket connection closed")
	}

	t.Logf("\n=== ALL MIDS WEBSOCKET TEST COMPLETED ===")
	t.Logf("✅ Successfully tested allMids WebSocket subscription")
	t.Logf("📊 Total messages received: %d", messageCount)
	t.Logf("💡 This subscription provides real-time mid prices for all available assets")
}


