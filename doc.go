// Package hyperliquid is a Go client for the Hyperliquid exchange.
//
// Construct a Client via [New] with the [Option] functions in this package:
//
//	c, err := hyperliquid.New(
//	    hyperliquid.WithMainnet(),
//	    hyperliquid.WithPrivateKey(pk),
//	    hyperliquid.WithLogger(myLogger),
//	)
//	if err != nil {
//	    return err
//	}
//
//	// Read-only queries.
//	state, err := c.Info.UserState(addr)
//
//	// Signed actions.
//	res, err := c.Trade.PlaceGTC("ETH", hyperliquid.Buy, 0.1, 3000)
//
//	// Websocket subscriptions.
//	sub, err := c.Stream.Subscribe(hyperliquid.Trades("ETH"), handler)
//
// The Trade surface returns [Result] / [BatchResult] / [CancelResult] /
// [BatchCancelResult] for the order verbs, and [TransferResponse] /
// [ApprovalResponse] / [ValidatorResponse] for the account-management
// verbs. Validation failures return *[ValidationError]; server-side
// failures return [APIError].
package hyperliquid
