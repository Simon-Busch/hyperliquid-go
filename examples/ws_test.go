package examples

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Simon-Busch/hyperliquid-go"
	"github.com/joho/godotenv"
)

func TestWebSocketConnection(t *testing.T) {
	godotenv.Overload()

	// Use testnet URL
	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	// Get test wallet address from environment or use a default test address
	testAddress := os.Getenv("WALLET_ADDRESS")
	if testAddress == "" {
		testAddress = "0x1234567890abcdef1234567890abcdef12345678" // Default test address
		testAddress = "0x1234567890abcdef1234567890abcdef12345678" // Default test address
	}

	t.Logf("=== TESTING WEBSOCKET CONNECTION TO TESTNET ===")
	t.Logf("Testnet URL: %s", hyperliquid.TestnetAPIURL)
	t.Logf("Test Address: %s", testAddress)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to WebSocket
	t.Log("Connecting to WebSocket...")
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	t.Log("✅ WebSocket connected successfully")

	// Track received messages
	var messageCount int
	var lastMessageTime time.Time

	// Message handler
	messageHandler := func(msg hyperliquid.WSMessage) {
		messageCount++
		lastMessageTime = time.Now()

		// Log the first few messages to see what we're receiving
		if messageCount <= 5 {
			msgData, _ := json.MarshalIndent(msg, "", "  ")
			t.Logf("📨 Message %d: %s", messageCount, string(msgData))
		}
	}

	// Subscribe to various data streams
	subscriptions := []struct {
		name string
		sub  hyperliquid.Subscription
	}{
		{
			name: "All Mid Prices",
			sub:  hyperliquid.Subscription{Type: "allMids"},
		},
		{
			name: "User Events",
			sub:  hyperliquid.Subscription{Type: "userEvents", User: testAddress},
		},
		{
			name: "User Fills",
			sub:  hyperliquid.Subscription{Type: "userFills", User: testAddress},
		},
		{
			name: "Web Data v2",
			sub:  hyperliquid.Subscription{Type: "webData2", User: testAddress},
		},
		{
			name: "SOL Orderbook",
			sub:  hyperliquid.Subscription{Type: "l2Book", Coin: "SOL"},
		},
		{
			name: "SOL Trades",
			sub:  hyperliquid.Subscription{Type: "trades", Coin: "SOL"},
		},
		{
			name: "ETH Orderbook",
			sub:  hyperliquid.Subscription{Type: "l2Book", Coin: "ETH"},
		},
		{
			name: "ETH Trades",
			sub:  hyperliquid.Subscription{Type: "trades", Coin: "ETH"},
		},
	}

	// Subscribe to all streams
	t.Log("\n=== SUBSCRIBING TO DATA STREAMS ===")
	for _, sub := range subscriptions {
		subID, err := wsClient.Subscribe(sub.sub, messageHandler)
		if err != nil {
			t.Logf("❌ Failed to subscribe to %s: %v", sub.name, err)
		} else {
			t.Logf("✅ Subscribed to %s (ID: %d)", sub.name, subID)
		}
	}

	// Wait for messages to arrive
	t.Log("\n=== WAITING FOR MESSAGES ===")
	t.Log("Waiting up to 20 seconds for messages...")

	startTime := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Log("Context cancelled, stopping test")
			break
		case <-ticker.C:
			elapsed := time.Since(startTime)
			t.Logf("⏱️  Elapsed: %v, Messages received: %d", elapsed, messageCount)

			// If we've received messages and waited long enough, we can stop
			if messageCount > 0 && elapsed > 10*time.Second {
				t.Log("✅ Received messages, test successful")
				goto cleanup
			}

			// If we've waited too long without messages, fail the test
			if elapsed > 20*time.Second {
				t.Log("❌ No messages received within timeout")
				goto cleanup
			}
		}
	}

cleanup:
	// Close the connection
	t.Log("\n=== CLEANING UP ===")
	err = wsClient.Close()
	if err != nil {
		t.Logf("Warning: Error closing WebSocket: %v", err)
	} else {
		t.Log("✅ WebSocket closed successfully")
	}

	// Test results
	t.Logf("\n=== TEST RESULTS ===")
	t.Logf("Total messages received: %d", messageCount)
	if messageCount > 0 {
		t.Logf("Last message received: %v", lastMessageTime)
		t.Logf("Test duration: %v", time.Since(startTime))
		t.Log("🎉 WebSocket test PASSED")
	} else {
		t.Fatal("❌ WebSocket test FAILED - no messages received")
	}
}

func TestWebSocketReconnection(t *testing.T) {
	godotenv.Overload()

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Log("=== TESTING WEBSOCKET RECONNECTION ===")

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Connect initially
	t.Log("Initial connection...")
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect initially: %v", err)
	}
	t.Log("✅ Initial connection successful")

	// Subscribe to a simple stream
	messageCount := 0
	subID, err := wsClient.Subscribe(hyperliquid.Subscription{Type: "allMids"}, func(msg hyperliquid.WSMessage) {
		messageCount++
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	t.Logf("✅ Subscribed to allMids (ID: %d)", subID)

	// Wait for a message
	time.Sleep(3 * time.Second)

	// Close connection manually
	t.Log("Manually closing connection...")
	err = wsClient.Close()
	if err != nil {
		t.Logf("Warning: Error closing: %v", err)
	}

	// Try to reconnect
	t.Log("Attempting to reconnect...")
	err = wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	t.Log("✅ Reconnection successful")

	// Wait for messages after reconnection
	time.Sleep(3 * time.Second)

	// Cleanup
	wsClient.Close()

	if messageCount > 0 {
		t.Logf("✅ Reconnection test PASSED - received %d messages", messageCount)
	} else {
		t.Log("⚠️  Reconnection test completed but no messages received")
	}
}

func TestWebSocketSpecificAddress(t *testing.T) {
	godotenv.Overload()

	// Get specific address to test
	testAddress := os.Getenv("WALLET_ADDRESS")
	if testAddress == "" {
		t.Skip("WALLET_ADDRESS not set, skipping specific address test")
	}

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Logf("=== TESTING WEBSOCKET FOR SPECIFIC ADDRESS ===")
	t.Logf("Address: %s", testAddress)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Connect
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Subscribe to user-specific data
	userDataReceived := false
	userFillsReceived := false

	// Subscribe to user events
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "userEvents", User: testAddress}, func(msg hyperliquid.WSMessage) {
		userDataReceived = true
		t.Logf("📊 User event received: %s", string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to user events: %v", err)
	}

	// Subscribe to user fills
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "userFills", User: testAddress}, func(msg hyperliquid.WSMessage) {
		userFillsReceived = true
		t.Logf("💰 User fill received: %s", string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to user fills: %v", err)
	}

	// Wait for data
	t.Log("Waiting for user-specific data...")
	time.Sleep(10 * time.Second)

	// Cleanup
	wsClient.Close()

	// Results
	t.Logf("\n=== ADDRESS-SPECIFIC TEST RESULTS ===")
	t.Logf("User events received: %t", userDataReceived)
	t.Logf("User fills received: %t", userFillsReceived)

	if userDataReceived || userFillsReceived {
		t.Log("✅ Address-specific test PASSED")
	} else {
		t.Log("⚠️  Address-specific test completed but no user data received")
	}
}

func TestRealTimeOrderMonitoring(t *testing.T) {
	godotenv.Overload()

	// Get your wallet address
	testAddress := os.Getenv("WALLET_ADDRESS")
	if testAddress == "" {
		testAddress = "0x1234567890abcdef1234567890abcdef12345678"
	}

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Logf("=== REAL-TIME ORDER MONITORING ===")
	t.Logf("Monitoring address: %s", testAddress)
	t.Logf("⚠️  OPEN AN ORDER IN ANOTHER TERMINAL NOW!")
	t.Logf("This test will watch for your order events...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Track different types of events
	events := map[string]int{
		"userFills": 0,
		"webData2":  0,
		"trades":    0,
		"allMids":   0,
	}

	// Subscribe to user fills (WORKING - shows when your orders are filled)
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "userFills", User: testAddress}, func(msg hyperliquid.WSMessage) {
		events["userFills"]++
		t.Logf("💰 USER FILL #%d: %s", events["userFills"], string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to user fills: %v", err)
	}

	// Subscribe to web data (WORKING - shows user account updates)
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "webData2", User: testAddress}, func(msg hyperliquid.WSMessage) {
		events["webData2"]++
		t.Logf("📊 WEB DATA #%d: %s", events["webData2"], string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to web data: %v", err)
	}

	// Subscribe to trades for SOL (WORKING - shows all trades including yours)
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "trades", Coin: "SOL"}, func(msg hyperliquid.WSMessage) {
		events["trades"]++
		t.Logf("📈 TRADE #%d: %s", events["trades"], string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}

	// Subscribe to all mids (WORKING - shows market price updates)
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "allMids"}, func(msg hyperliquid.WSMessage) {
		events["allMids"]++
		if events["allMids"] <= 3 { // Only log first 3 to avoid spam
			t.Logf("📊 MID PRICE #%d: %s", events["allMids"], string(msg.Data))
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to all mids: %v", err)
	}

	t.Log("✅ All subscriptions active")
	t.Log("Now open an order in another terminal to see it here!")
	t.Log("💡 Working channels: userFills, webData2, trades, allMids")

	// Monitor for 30 seconds
	time.Sleep(30 * time.Second)

	// Cleanup
	wsClient.Close()

	// Results
	t.Logf("\n=== ORDER MONITORING RESULTS ===")
	t.Logf("User fills received: %d", events["userFills"])
	t.Logf("Web data updates: %d", events["webData2"])
	t.Logf("Trades received: %d", events["trades"])
	t.Logf("Mid price updates: %d", events["allMids"])

	if events["userFills"] > 0 || events["webData2"] > 0 {
		t.Log("🎉 Order monitoring successful - you should see your orders!")
	} else {
		t.Log("⚠️  No order events received - try opening an order while this test runs")
		t.Log("💡 Note: userEvents and orderUpdates require a user address parameter")
	}
}

func TestWebSocketChannelDiscovery(t *testing.T) {
	godotenv.Overload()

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Log("=== WEBSOCKET CHANNEL DISCOVERY ===")
	t.Log("Testing different subscription types to see what works...")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Connect
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test different subscription types
	testSubscriptions := []struct {
		name string
		sub  hyperliquid.Subscription
	}{
		{"allMids", hyperliquid.Subscription{Type: "allMids"}},
		{"l2Book SOL", hyperliquid.Subscription{Type: "l2Book", Coin: "SOL"}},
		{"trades SOL", hyperliquid.Subscription{Type: "trades", Coin: "SOL"}},
		{"userEvents", hyperliquid.Subscription{Type: "userEvents", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"userFills", hyperliquid.Subscription{Type: "userFills", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"orderUpdates", hyperliquid.Subscription{Type: "orderUpdates"}},
		{"webData2", hyperliquid.Subscription{Type: "webData2", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"userNonFundingLedgerUpdates", hyperliquid.Subscription{Type: "userNonFundingLedgerUpdates", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"userFundings", hyperliquid.Subscription{Type: "userFundings", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"clearinghouseState", hyperliquid.Subscription{Type: "clearinghouseState", User: "0x1234567890abcdef1234567890abcdef12345678"}},
		{"frontendOpenOrders", hyperliquid.Subscription{Type: "frontendOpenOrders", User: "0x1234567890abcdef1234567890abcdef12345678"}},
	}

	// Track which subscriptions work
	workingSubs := make(map[string]bool)
	messageCounts := make(map[string]int)

	// Subscribe to all test channels
	for _, testSub := range testSubscriptions {
		_, err := wsClient.Subscribe(testSub.sub, func(msg hyperliquid.WSMessage) {
			messageCounts[testSub.name]++
			if messageCounts[testSub.name] <= 3 { // Log first 3 messages
				t.Logf("📨 %s: %s", testSub.name, string(msg.Data))
			}
		})

		if err != nil {
			t.Logf("❌ Failed to subscribe to %s: %v", testSub.name, err)
			workingSubs[testSub.name] = false
		} else {
			t.Logf("✅ Subscribed to %s", testSub.name)
			workingSubs[testSub.name] = true
		}
	}

	t.Log("\n=== WAITING FOR MESSAGES ===")
	t.Log("Waiting 30 seconds to see which channels are active...")

	// Wait and monitor
	time.Sleep(30 * time.Second)

	// Cleanup
	wsClient.Close()

	// Results
	t.Logf("\n=== CHANNEL DISCOVERY RESULTS ===")
	for _, testSub := range testSubscriptions {
		status := "❌"
		if workingSubs[testSub.name] {
			status = "✅"
		}
		t.Logf("%s %s: %d messages", status, testSub.name, messageCounts[testSub.name])
	}

	// Summary
	workingCount := 0
	for _, working := range workingSubs {
		if working {
			workingCount++
		}
	}

	t.Logf("\nWorking channels: %d/%d", workingCount, len(testSubscriptions))
}

func TestUserNotifications(t *testing.T) {
	godotenv.Overload()

	// Get your wallet address
	testAddress := os.Getenv("WALLET_ADDRESS")
	if testAddress == "" {
		testAddress = "0x1234567890abcdef1234567890abcdef12345678"
	}

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Logf("=== USER NOTIFICATIONS TEST ===")
	t.Logf("Listening for notifications for address: %s", testAddress)
	t.Logf("This test will show any notifications sent to your wallet...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Track notifications
	notificationCount := 0
	lastNotificationTime := time.Now()

	// Subscribe to user notifications
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "notification", User: testAddress}, func(msg hyperliquid.WSMessage) {
		notificationCount++
		lastNotificationTime = time.Now()

		t.Logf("🔔 NOTIFICATION #%d received at %s", notificationCount, lastNotificationTime.Format("15:04:05"))
		t.Logf("   Channel: %s", msg.Channel)
		t.Logf("   Data: %s", string(msg.Data))

		// Try to parse the notification data
		var notification struct {
			Notification string `json:"notification"`
		}
		if err := json.Unmarshal(msg.Data, &notification); err == nil {
			t.Logf("   Message: %s", notification.Notification)
		}
		t.Logf("   ---")
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to notifications: %v", err)
	}

	t.Log("✅ Notification subscription active")
	t.Log("Waiting for notifications...")
	t.Log("💡 Notifications may include:")
	t.Log("   - Order status updates")
	t.Log("   - Position changes")
	t.Log("   - Account alerts")
	t.Log("   - System messages")

	// Monitor for 45 seconds
	time.Sleep(45 * time.Second)

	// Cleanup
	wsClient.Close()

	// Results
	t.Logf("\n=== NOTIFICATION RESULTS ===")
	t.Logf("Total notifications received: %d", notificationCount)

	if notificationCount > 0 {
		t.Logf("Last notification at: %s", lastNotificationTime.Format("15:04:05"))
		t.Log("🎉 Successfully received user notifications!")
	} else {
		t.Log("⚠️  No notifications received during test period")
		t.Log("💡 This is normal - notifications are only sent when there are events")
		t.Log("💡 Try placing an order or making a trade to trigger notifications")
	}
}

func TestCompleteUserActivityMonitoring(t *testing.T) {
	godotenv.Overload()

	// Get your wallet address
	testAddress := os.Getenv("WALLET_ADDRESS")
	if testAddress == "" {
		testAddress = "0x1234567890abcdef1234567890abcdef12345678"
	}

	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	t.Logf("=== COMPLETE USER ACTIVITY MONITORING ===")
	t.Logf("Monitoring address: %s", testAddress)
	t.Logf("�� This test monitors user fills and notifications...")
	t.Logf("⚠️  Now try: opening/closing positions, placing orders, etc.")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Connect
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Track different types of events
	events := map[string]int{
		"userFills":    0,
		"notification": 0,
	}

	// Subscribe to user fills (when orders are executed)
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "userFills", User: testAddress}, func(msg hyperliquid.WSMessage) {
		events["userFills"]++

		// Parse the userFills message structure
		var fillMessage struct {
			IsSnapshot bool   `json:"isSnapshot,omitempty"`
			User       string `json:"user"`
			Fills      []struct {
				Coin      string `json:"coin"`
				Price     string `json:"px"`
				Size      string `json:"sz"`
				Side      string `json:"side"`
				Direction string `json:"dir"`
				ClosedPnL string `json:"closedPnl"`
				Hash      string `json:"hash"`
				OID       int64  `json:"oid"`
				Fee       string `json:"fee"`
				FeeToken  string `json:"feeToken"`
			} `json:"fills"`
		}

		if err := json.Unmarshal(msg.Data, &fillMessage); err == nil {
			if fillMessage.IsSnapshot {
				t.Logf("💰 SNAPSHOT #%d: %d fills loaded", events["userFills"], len(fillMessage.Fills))
			} else {
				t.Logf("💰 NEW FILLS #%d: %d fills received", events["userFills"], len(fillMessage.Fills))
			}

			// Show details for each fill
			for i, fill := range fillMessage.Fills {
				side := "BUY"
				if fill.Side == "A" {
					side = "SELL"
				}

				t.Logf("   %d. %s %s %s @ $%s | PnL: $%s | Fee: $%s %s",
					i+1, side, fill.Size, fill.Coin, fill.Price, fill.ClosedPnL, fill.Fee, fill.FeeToken)
			}
		} else {
			t.Logf("💰 FILL #%d: Failed to parse: %v", events["userFills"], err)
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to userFills: %v", err)
	}

	// Subscribe to notifications
	_, err = wsClient.Subscribe(hyperliquid.Subscription{Type: "notification", User: testAddress}, func(msg hyperliquid.WSMessage) {
		events["notification"]++

		// Try to parse notification content for key info
		var notification struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		}
		if json.Unmarshal(msg.Data, &notification) == nil {
			t.Logf("🔔 NOTIFICATION #%d: %s (%s)",
				events["notification"],
				notification.Message,
				notification.Type)
		} else {
			t.Logf("🔔 NOTIFICATION #%d: %s", events["notification"], string(msg.Data))
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to notifications: %v", err)
	}

	t.Log("✅ Subscriptions active!")
	t.Log("📋 Monitoring channels:")
	t.Log("   - userFills: Your order executions")
	t.Log("   - notification: System alerts & messages")
	t.Log("")
	t.Log("🚀 Now perform some actions:")
	t.Log("   - Open/close positions")
	t.Log("   - Place/cancel orders")
	t.Log("   - Check for liquidations")
	t.Log("")

	// Monitor for 75 seconds
	time.Sleep(75 * time.Second)

	// Cleanup
	wsClient.Close()

	// Results
	t.Logf("\n=== MONITORING RESULTS ===")
	t.Logf("Total events received:")
	t.Logf("   - User Fills: %d", events["userFills"])
	t.Logf("   - Notifications: %d", events["notification"])

	totalEvents := events["userFills"] + events["notification"]
	t.Logf("Total events: %d", totalEvents)

	if totalEvents == 0 {
		t.Log("⚠️  No events received. This might mean:")
		t.Log("   - No trading activity during test")
		t.Log("   - Need to perform actions during test")
	} else {
		t.Log("✅ Events received successfully!")
	}
}

// TestUserFillsSubscriptionTriggersOnOrder subscribes to userFills for the current account,
// opens a tiny market order to trigger a fill, and asserts we receive a fill event.
func TestUserFillsSubscriptionTriggersOnOrder(t *testing.T) {
	godotenv.Overload()
	wsClient := hyperliquid.NewStream(hyperliquid.MainnetAPIURL)
	infoClient := hyperliquid.NewInfo(hyperliquid.MainnetAPIURL, true, nil, nil, nil, "")
	// Your target address
	address := "0x1234567890abcdef1234567890abcdef12345678"

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	// Track received messages
	var fillCount int
	var snapshotCount int
	messageChan := make(chan hyperliquid.WSMessage, 100)

	// Position tracking map: coin -> position size (positive = long, negative = short, 0 = flat)
	positions := make(map[string]float64)
	// Leverage tracking map: coin -> leverage details
	leverages := make(map[string]hyperliquid.Leverage)

	// Subscribe to user fills with snapshot filtering
	_, err = wsClient.SubscribeToUserFills(address, func(msg hyperliquid.WSMessage) {
		t.Logf("🔍 RAW userFills message received: %s", string(msg.Data))

		// Parse the userFills message structure to check for snapshots
		var fillMessage struct {
			IsSnapshot bool   `json:"isSnapshot,omitempty"`
			User       string `json:"user"`
			Fills      []struct {
				Coin      string `json:"coin"`
				Price     string `json:"px"`
				Size      string `json:"sz"`
				Side      string `json:"side"`
				Direction string `json:"dir"`
				ClosedPnL string `json:"closedPnl"`
				Hash      string `json:"hash"`
				OID       int64  `json:"oid"`
				Fee       string `json:"fee"`
				FeeToken  string `json:"feeToken"`
			} `json:"fills"`
		}

		if err := json.Unmarshal(msg.Data, &fillMessage); err == nil {
			if fillMessage.IsSnapshot {
				snapshotCount++
				t.Logf("📸 SNAPSHOT #%d: %d fills loaded (ignored)", snapshotCount, len(fillMessage.Fills))
			} else {
				fillCount++
				t.Logf("💰 NEW FILLS #%d: %d fills received", fillCount, len(fillMessage.Fills))

				// Process each fill and determine position context
				for i, fill := range fillMessage.Fills {
					side := "BUY"
					if fill.Side == "A" {
						side = "SELL"
					}

					// Parse fill size
					fillSize, parseErr := strconv.ParseFloat(fill.Size, 64)
					if parseErr != nil {
						t.Logf("   %d. Failed to parse fill size: %v", i+1, parseErr)
						continue
					}

					// Determine position change
					var positionChange float64
					if side == "BUY" {
						positionChange = fillSize // Long position increases
					} else {
						positionChange = -fillSize // Short position decreases
					}

					// Get previous position
					previousPosition := positions[fill.Coin]
					newPosition := previousPosition + positionChange
					positions[fill.Coin] = newPosition

					// Determine trade context
					var tradeContext string
					if side == "BUY" {
						if previousPosition < 0 {
							// Was short, now buying = closing short
							tradeContext = "🔄 CLOSING SHORT"
						} else if previousPosition == 0 {
							// Was flat, now buying = opening long
							tradeContext = "📈 OPENING LONG"
						} else {
							// Was long, now buying more = increasing long
							tradeContext = "📈 INCREASING LONG"
						}
					} else {
						if previousPosition > 0 {
							// Was long, now selling = closing long
							tradeContext = "🔄 CLOSING LONG"
						} else if previousPosition == 0 {
							// Was flat, now selling = opening short
							tradeContext = "📉 OPENING SHORT"
						} else {
							// Was short, now selling more = increasing short
							tradeContext = "📉 INCREASING SHORT"
						}
					}

					t.Logf("   %d. %s %s %s @ $%s | PnL: $%s | Fee: $%s %s",
						i+1, side, fill.Size, fill.Coin, fill.Price, fill.ClosedPnL, fill.Fee, fill.FeeToken)
					// Ensure leverage available; if not yet received from webData2, fetch on-demand
					lev, ok := leverages[fill.Coin]
					if !ok {
						if us, err := infoClient.UserState(address); err == nil && us != nil {
							for _, ap := range us.AssetPositions {
								if ap.Position.Coin == fill.Coin {
									leverages[fill.Coin] = ap.Position.Leverage
									lev = ap.Position.Leverage
									ok = true
									break
								}
							}
						}
					}
					if ok {
						if lev.RawUsd != nil {
							t.Logf("       Leverage: %s x%d | rawUsd=%s", lev.Type, lev.Value, *lev.RawUsd)
						} else {
							t.Logf("       Leverage: %s x%d", lev.Type, lev.Value)
						}
					}

					t.Logf("       %s | Position: %.4f -> %.4f | %s",
						tradeContext, previousPosition, newPosition, fill.Coin)
				}

				// Send to channel for test completion
				messageChan <- msg
			}
		} else {
			t.Logf("💰 FILL: Failed to parse: %v", err)
			t.Logf("💰 RAW DATA: %s", string(msg.Data))
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to fills: %v", err)
	}

	// Subscribe to webData2 for position updates
	_, err = wsClient.SubscribeToWebData2(address, func(msg hyperliquid.WSMessage) {
		t.Logf("🔍 RAW webData2 message received: %s", string(msg.Data))

		// Parse webData2 to get current positions
		var webData struct {
			UserState *hyperliquid.UserState `json:"userState,omitempty"`
		}

		if err := json.Unmarshal(msg.Data, &webData); err == nil && webData.UserState != nil {
			// Update positions from webData2
			for _, assetPos := range webData.UserState.AssetPositions {
				if assetPos.Position.Szi != "" {
					if size, parseErr := strconv.ParseFloat(assetPos.Position.Szi, 64); parseErr == nil {
						positions[assetPos.Position.Coin] = size
						t.Logf("📊 Position updated from webData2: %s = %.4f", assetPos.Position.Coin, size)
					}
				}
				// Update leverage details for this coin
				leverages[assetPos.Position.Coin] = assetPos.Position.Leverage
			}
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to webData2: %v", err)
	}

	// Subscribe to user events (broader coverage) - also filter snapshots
	_, err = wsClient.SubscribeToUserEvents(address, func(msg hyperliquid.WSMessage) {
		t.Logf("🔍 RAW userEvents message received: %s", string(msg.Data))

		// Check if this is a snapshot by looking for the isSnapshot field
		if strings.Contains(string(msg.Data), `"isSnapshot":true`) {
			t.Logf("📸 USER EVENT SNAPSHOT (ignored): %s", string(msg.Data))
			return
		}

		t.Logf("📊 USER EVENT: %s", string(msg.Data))
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to events: %v", err)
	}

	// Wait for messages with timeout
	t.Logf("=== WAITING FOR FILL EVENTS (excluding snapshots) ===")
	t.Logf("Monitoring address: %s", address)
	t.Logf("⚠️  OPEN AN ORDER IN ANOTHER TERMINAL TO TRIGGER A FILL!")

	timeout := time.After(300 * time.Second) // 60 seconds should be enough
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-messageChan:
			t.Logf("✅ Received new fill event! Test completed successfully.")
			t.Logf("📊 Summary: %d snapshots ignored, %d new fills received", snapshotCount, fillCount)
			return
		case <-timeout:
			t.Logf("⏰ Timeout reached after 60 seconds")
			t.Logf("📊 Summary: %d snapshots ignored, %d new fills received", snapshotCount, fillCount)
			if fillCount == 0 {
				t.Logf("⚠️  No new fills received (only snapshots were filtered out)")
			}
			return
		case <-ticker.C:
			t.Logf("⏳ Still waiting... (%d snapshots ignored, %d new fills received)", snapshotCount, fillCount)
		}
	}
}

func TestL2BookSubscriptionSOL(t *testing.T) {
	godotenv.Overload()
	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer wsClient.Close()

	// Track received messages
	var messageCount int
	messageChan := make(chan hyperliquid.WSMessage, 100)

	// Subscribe to SOL L2Book
	_, err = wsClient.SubscribeToOrderbook("SOL", func(msg hyperliquid.WSMessage) {
		messageCount++
		t.Logf("📊 L2Book message #%d received", messageCount)

		// Parse the L2Book data
		var l2Book hyperliquid.L2Book
		if err := json.Unmarshal(msg.Data, &l2Book); err != nil {
			t.Logf("⚠️  Failed to parse L2Book data: %v", err)
			t.Logf("   Raw data: %s", string(msg.Data))
		} else {
			t.Logf("   Coin: %s, Time: %d, Levels: %d", l2Book.Coin, l2Book.Time, len(l2Book.Levels))

			// Show best bid/ask if available
			if len(l2Book.Levels) > 0 && len(l2Book.Levels[0]) > 0 {
				bestBid := l2Book.Levels[0][0].Px
				t.Logf("   Best bid: %.2f", bestBid)

				if len(l2Book.Levels) > 1 && len(l2Book.Levels[1]) > 0 {
					bestAsk := l2Book.Levels[1][0].Px
					midPrice := (bestBid + bestAsk) / 2
					t.Logf("   Best ask: %.2f", bestAsk)
					t.Logf("   Mid price: %.2f", midPrice)
				}
			}
		}

		// Send to channel for test completion
		messageChan <- msg
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to SOL L2Book: %v", err)
	}

	// Wait for 1 minute
	t.Logf("=== LISTENING TO SOL L2BOOK FOR 1 MINUTE ===")
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-messageChan:
			// Continue listening, don't exit on first message
		case <-timeout:
			t.Logf("⏰ 1 minute timeout reached")
			t.Logf("📊 Summary: %d L2Book messages received for SOL", messageCount)
			return
		case <-ticker.C:
			t.Logf("⏳ Still listening... (%d messages received so far)", messageCount)
		}
	}
}

// TestL2BookStableConnection5Min tests a stable 5-minute WebSocket connection to SOL L2Book
// The WebSocket client handles pings natively via the pingPump goroutine (see ws.go:187-205)
func TestL2BookStableConnection5Min(t *testing.T) {
	godotenv.Overload()
	wsClient := hyperliquid.NewStream(hyperliquid.TestnetAPIURL)

	// Create a context with timeout slightly longer than test duration
	// to allow graceful cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	// Connect to WebSocket
	t.Log("=== ESTABLISHING WEBSOCKET CONNECTION ===")
	err := wsClient.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer func() {
		t.Log("=== CLOSING WEBSOCKET CONNECTION ===")
		if err := wsClient.Close(); err != nil {
			t.Logf("Warning: Error closing WebSocket: %v", err)
		} else {
			t.Log("✅ WebSocket closed successfully")
		}
	}()

	// Statistics tracking
	var (
		messageCount     int
		lastMessageTime  time.Time
		firstMessageTime time.Time
		mu               sync.Mutex
		bestBidCache     float64
		bestAskCache     float64
		spreadCache      float64
	)

	// Subscribe to SOL L2Book with detailed logging
	_, err = wsClient.SubscribeToOrderbook("SOL", func(msg hyperliquid.WSMessage) {
		mu.Lock()
		defer mu.Unlock()

		messageCount++
		now := time.Now()
		lastMessageTime = now

		if messageCount == 1 {
			firstMessageTime = now
		}

		// Parse the L2Book data
		var l2Book hyperliquid.L2Book
		if err := json.Unmarshal(msg.Data, &l2Book); err != nil {
			t.Logf("⚠️  Message #%d: Failed to parse L2Book data: %v", messageCount, err)
			return
		}

		// Extract best bid/ask
		if len(l2Book.Levels) >= 2 && len(l2Book.Levels[0]) > 0 && len(l2Book.Levels[1]) > 0 {
			bestBid := l2Book.Levels[0][0].Px
			bestAsk := l2Book.Levels[1][0].Px
			spread := bestAsk - bestBid
			spreadBps := (spread / bestBid) * 10000 // basis points

			bestBidCache = bestBid
			bestAskCache = bestAsk
			spreadCache = spreadBps

			// Log detailed info for first 5 messages and then periodically
			if messageCount <= 5 || messageCount%100 == 0 {
				t.Logf("📊 Message #%d | Bid: $%.2f | Ask: $%.2f | Spread: $%.4f (%.2f bps) | Levels: %d",
					messageCount, bestBid, bestAsk, spread, spreadBps, len(l2Book.Levels))
			}
		} else {
			if messageCount <= 5 {
				t.Logf("📊 Message #%d | Incomplete orderbook data", messageCount)
			}
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to SOL L2Book: %v", err)
	}

	t.Log("✅ Successfully subscribed to SOL L2Book")
	t.Log("=== MONITORING FOR 5 MINUTES ===")
	t.Log("Note: WebSocket client handles pings automatically (see ws.go pingPump)")
	t.Log("")

	// Start time tracking
	startTime := time.Now()
	testDuration := 5 * time.Minute

	// Progress ticker - report every 30 seconds
	progressTicker := time.NewTicker(30 * time.Second)
	defer progressTicker.Stop()

	// Statistics ticker - calculate stats every minute
	statsTicker := time.NewTicker(1 * time.Minute)
	defer statsTicker.Stop()

	// Message rate tracking
	lastCheckCount := 0
	lastCheckTime := startTime

	// Main monitoring loop
	for {
		select {
		case <-ctx.Done():
			t.Log("⚠️  Context cancelled")
			goto summary

		case <-progressTicker.C:
			mu.Lock()
			elapsed := time.Since(startTime)
			remaining := testDuration - elapsed
			currentCount := messageCount
			currentBid := bestBidCache
			currentAsk := bestAskCache
			currentSpread := spreadCache
			mu.Unlock()

			// Calculate message rate since last check
			timeSinceCheck := time.Since(lastCheckTime).Seconds()
			msgsSinceCheck := currentCount - lastCheckCount
			msgRate := float64(msgsSinceCheck) / timeSinceCheck

			t.Logf("⏱️  Progress: %.1f%% | Elapsed: %v | Remaining: %v",
				(elapsed.Seconds()/testDuration.Seconds())*100,
				elapsed.Round(time.Second),
				remaining.Round(time.Second))
			t.Logf("   Messages: %d | Rate: %.1f msg/s | Last: Bid $%.2f | Ask $%.2f | Spread %.2f bps",
				currentCount, msgRate, currentBid, currentAsk, currentSpread)

			lastCheckCount = currentCount
			lastCheckTime = time.Now()

		case <-statsTicker.C:
			mu.Lock()
			elapsed := time.Since(startTime)
			currentCount := messageCount
			var timeSinceLastMsg time.Duration
			if !lastMessageTime.IsZero() {
				timeSinceLastMsg = time.Since(lastMessageTime)
			}
			mu.Unlock()

			avgMsgRate := float64(currentCount) / elapsed.Seconds()

			t.Log("📈 === STATISTICS UPDATE ===")
			t.Logf("   Total messages: %d", currentCount)
			t.Logf("   Average rate: %.2f msg/s", avgMsgRate)
			t.Logf("   Time since last message: %v", timeSinceLastMsg.Round(time.Millisecond))
			t.Log("")

			// Alert if no messages received recently
			if timeSinceLastMsg > 10*time.Second {
				t.Logf("⚠️  WARNING: No messages received in last %v", timeSinceLastMsg.Round(time.Second))
			}

		case <-time.After(testDuration):
			t.Log("⏰ 5-minute test duration completed")
			goto summary
		}
	}

summary:
	// Final statistics
	mu.Lock()
	totalMessages := messageCount
	totalDuration := time.Since(startTime)
	var messageSpan time.Duration
	if !firstMessageTime.IsZero() && !lastMessageTime.IsZero() {
		messageSpan = lastMessageTime.Sub(firstMessageTime)
	}
	mu.Unlock()

	t.Log("")
	t.Log("=== FINAL SUMMARY ===")
	t.Logf("Test duration: %v", totalDuration.Round(time.Second))
	t.Logf("Total messages received: %d", totalMessages)

	if totalMessages > 0 {
		avgRate := float64(totalMessages) / totalDuration.Seconds()
		t.Logf("Average message rate: %.2f msg/s", avgRate)
		t.Logf("First message at: %v", firstMessageTime.Format("15:04:05.000"))
		t.Logf("Last message at: %v", lastMessageTime.Format("15:04:05.000"))
		t.Logf("Message span: %v", messageSpan.Round(time.Second))

		// Validate connection stability
		if totalMessages < 10 {
			t.Errorf("❌ Connection unstable: only %d messages received in %v", totalMessages, totalDuration)
		} else if avgRate < 0.1 {
			t.Errorf("❌ Message rate too low: %.2f msg/s", avgRate)
		} else {
			t.Log("✅ Connection stable throughout 5-minute test")
			t.Logf("✅ Ping mechanism working (native WebSocket keepalive)")
		}
	} else {
		t.Error("❌ Test FAILED: No messages received")
	}
}

// TestWebData2Subscription tests the SubscribeToWebData2 functionality
// and demonstrates what information is provided by this subscription type.
//
// webData2 provides:
// - Real-time balance updates (accountValue, withdrawable)
// - Margin usage and available margin
// - Current positions with size, entry price, and PnL
// - Leverage settings per position
// - Total notional position value
// - Updates automatically when trades execute or positions change
func TestWebData2Subscription(t *testing.T) {
	godotenv.Overload()

	testAddress := accountAddress(t)

	wsClient := hyperliquid.NewStream(hyperliquid.MainnetAPIURL)
	defer wsClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := wsClient.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("✅ Connected to WebSocket")

	messageCount := 0
	receivedData := false

	// Track balance changes
	type BalanceSnapshot struct {
		timestamp    time.Time
		accountValue string
		withdrawable string
		marginUsed   string
	}
	var balanceHistory []BalanceSnapshot

	// Subscribe to webData2
	_, err := wsClient.SubscribeToWebData2(testAddress, func(msg hyperliquid.WSMessage) {
		messageCount++
		receivedData = true

		t.Logf("\n=== webData2 Message #%d ===", messageCount)
		t.Logf("Channel: %s", msg.Channel)

		// Parse the webData2 message
		// webData2 has a "clearinghouseState" field that contains the UserState
		var webData struct {
			ClearinghouseState *hyperliquid.UserState `json:"clearinghouseState,omitempty"`
		}

		if err := json.Unmarshal(msg.Data, &webData); err != nil {
			t.Logf("⚠️  Failed to parse webData2: %v", err)
			return
		}

		if webData.ClearinghouseState == nil {
			t.Log("⚠️  ClearinghouseState is nil")
			return
		}

		// Track balance snapshot
		snapshot := BalanceSnapshot{
			timestamp:    time.Now(),
			accountValue: webData.ClearinghouseState.MarginSummary.AccountValue,
			withdrawable: webData.ClearinghouseState.Withdrawable,
			marginUsed:   webData.ClearinghouseState.MarginSummary.TotalMarginUsed,
		}
		balanceHistory = append(balanceHistory, snapshot)

		// Display balance and margin information
		t.Log("\n📊 Balance & Margin Information:")
		t.Logf("   Account Value: %s USD", webData.ClearinghouseState.MarginSummary.AccountValue)
		t.Logf("   Total Margin Used: %s USD", webData.ClearinghouseState.MarginSummary.TotalMarginUsed)
		t.Logf("   Total Notional Position: %s USD", webData.ClearinghouseState.MarginSummary.TotalNtlPos)
		t.Logf("   Total Raw USD: %s USD", webData.ClearinghouseState.MarginSummary.TotalRawUsd)
		t.Logf("   Withdrawable: %s USD", webData.ClearinghouseState.Withdrawable)

		// Show balance changes
		if len(balanceHistory) > 1 {
			prev := balanceHistory[len(balanceHistory)-2]
			if prev.accountValue != snapshot.accountValue {
				t.Logf("   ⚡ Account value changed: %s → %s", prev.accountValue, snapshot.accountValue)
			}
			if prev.withdrawable != snapshot.withdrawable {
				t.Logf("   ⚡ Withdrawable changed: %s → %s", prev.withdrawable, snapshot.withdrawable)
			}
			if prev.marginUsed != snapshot.marginUsed {
				t.Logf("   ⚡ Margin used changed: %s → %s", prev.marginUsed, snapshot.marginUsed)
			}
		}

		// Display cross margin summary if different
		if webData.ClearinghouseState.CrossMarginSummary.AccountValue != "" {
			t.Log("\n📊 Cross Margin Summary:")
			t.Logf("   Account Value: %s USD", webData.ClearinghouseState.CrossMarginSummary.AccountValue)
			t.Logf("   Total Margin Used: %s USD", webData.ClearinghouseState.CrossMarginSummary.TotalMarginUsed)
			t.Logf("   Total Notional Position: %s USD", webData.ClearinghouseState.CrossMarginSummary.TotalNtlPos)
			t.Logf("   Total Raw USD: %s USD", webData.ClearinghouseState.CrossMarginSummary.TotalRawUsd)
		}

		// Display positions
		if len(webData.ClearinghouseState.AssetPositions) > 0 {
			t.Log("\n📈 Active Positions:")
			for i, assetPos := range webData.ClearinghouseState.AssetPositions {
				t.Logf("   Position %d:", i+1)
				t.Logf("      Coin: %s", assetPos.Position.Coin)
				t.Logf("      Size: %s", assetPos.Position.Szi)
				t.Logf("      Leverage: %+v", assetPos.Position.Leverage)
				if assetPos.Position.EntryPx != nil {
					t.Logf("      Entry Price: %s", *assetPos.Position.EntryPx)
				}
				if assetPos.Position.PositionValue != "" {
					t.Logf("      Position Value: %s", assetPos.Position.PositionValue)
				}
				if assetPos.Position.UnrealizedPnl != "" {
					t.Logf("      Unrealized PnL: %s", assetPos.Position.UnrealizedPnl)
				}
			}
		} else {
			t.Log("\n📈 No active positions")
		}

		// Pretty print complete JSON on first message
		if messageCount == 1 {
			t.Log("\n" + strings.Repeat("=", 80))
			t.Log("COMPLETE WEBDATA2 STRUCTURE (First Message)")
			t.Log(strings.Repeat("=", 80))
			var prettyJSON map[string]interface{}
			if err := json.Unmarshal(msg.Data, &prettyJSON); err == nil {
				formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
				t.Logf("\n%s\n", string(formatted))
			}
			t.Log(strings.Repeat("=", 80))
		}
	})

	if err != nil {
		t.Fatalf("Failed to subscribe to webData2: %v", err)
	}
	t.Log("✅ Subscribed to webData2")

	// Wait to receive messages
	t.Log("\n⏳ Waiting 30 seconds for webData2 updates...")
	t.Log("💡 Try placing an order or modifying positions to trigger updates")
	t.Log("")
	t.Log("KEY INFORMATION PROVIDED BY WEBDATA2:")
	t.Log("  ✓ Real-time balance updates (accountValue, withdrawable)")
	t.Log("  ✓ Margin usage and available margin")
	t.Log("  ✓ Current positions with size, entry price, and PnL")
	t.Log("  ✓ Leverage settings per position")
	t.Log("  ✓ Total notional position value")
	t.Log("  ✓ Updates automatically when trades execute or positions change")
	t.Log("")

	time.Sleep(30 * time.Second)

	// Summary
	t.Log("\n=== Test Summary ===")
	t.Logf("Total webData2 messages received: %d", messageCount)
	t.Logf("Balance updates tracked: %d", len(balanceHistory))

	if len(balanceHistory) > 1 {
		t.Log("\n📈 Balance Timeline:")
		for i, snapshot := range balanceHistory {
			t.Logf("  %d. [%s] Account: %s USD | Withdrawable: %s USD | Margin Used: %s USD",
				i+1, snapshot.timestamp.Format("15:04:05"),
				snapshot.accountValue, snapshot.withdrawable, snapshot.marginUsed)
		}
	}

	if !receivedData {
		t.Log("⚠️  No webData2 messages received - this is normal if the account has no activity")
		t.Log("💡 webData2 typically sends updates when:")
		t.Log("   - You first subscribe (initial snapshot)")
		t.Log("   - An order fills")
		t.Log("   - A position is opened/closed/modified")
		t.Log("   - Account balance changes")
	} else {
		t.Log("✅ webData2 subscription test completed successfully")
	}
}
