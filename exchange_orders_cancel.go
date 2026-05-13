package hyperliquid

type (
	// CancelOrderRequest identifies a single order to cancel by exchange oid.
	CancelOrderRequest struct {
		Coin    string
		OrderID int64
	}

	// CancelOrderResponse holds the per-cancel statuses returned by the
	// /exchange cancel action.
	CancelOrderResponse struct {
		Statuses MixedArray
	}
)

// Cancel cancels the resting order with the given exchange oid.
func (t *Trader) Cancel(coin string, oid int64) (CancelResult, error) {
	resp, err := t.BulkCancel([]CancelOrderRequest{{Coin: coin, OrderID: oid}})
	if err != nil {
		return CancelResult{}, err
	}
	return firstCancelResult(resp), nil
}

// BulkCancel cancels every order in requests as a single signed action.
func (t *Trader) BulkCancel(
	requests []CancelOrderRequest,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := make([]CancelOrderWire, len(requests))
	for i, req := range requests {
		cancels[i] = CancelOrderWire{
			Asset:   t.info.AssetID(req.Coin),
			OrderID: req.OrderID,
		}
	}

	action := CancelAction{
		Type:    "cancel",
		Dex:     t.dex,
		Cancels: cancels,
	}

	if err = t.executeAction(action, &res); err != nil {
		return
	}
	return
}

// CancelOrderRequestByCloid identifies a single order to cancel by client id.
type CancelOrderRequestByCloid struct {
	Coin  string
	Cloid string
}

// CancelByCloid cancels the resting order with the given client order id.
func (t *Trader) CancelByCloid(coin, cloid string) (CancelResult, error) {
	resp, err := t.BulkCancelByCloids([]CancelOrderRequestByCloid{{Coin: coin, Cloid: cloid}})
	if err != nil {
		return CancelResult{}, err
	}
	return firstCancelResult(resp), nil
}

// BulkCancelByCloids cancels every order identified by client id as one
// signed action.
func (t *Trader) BulkCancelByCloids(
	requests []CancelOrderRequestByCloid,
) (res *APIResponse[CancelOrderResponse], err error) {
	cancels := make([]CancelByCloidWire, len(requests))
	for i, req := range requests {
		cancels[i] = CancelByCloidWire{
			Asset:    t.info.AssetID(req.Coin),
			ClientID: req.Cloid,
		}
	}

	action := CancelByCloidAction{
		Type:    "cancelByCloid",
		Dex:     t.dex,
		Cancels: cancels,
	}

	if err = t.executeAction(action, &res); err != nil {
		return
	}
	return
}

// firstCancelResult extracts the first CancelResult from a bulk-cancel
// APIResponse.
func firstCancelResult(resp *APIResponse[CancelOrderResponse]) CancelResult {
	if resp == nil {
		return CancelResult{}
	}
	if !resp.Ok {
		return CancelResult{Error: resp.Err}
	}
	if len(resp.Data.Statuses) == 0 {
		return CancelResult{}
	}
	first := resp.Data.Statuses[0]
	return CancelResult{Status: string(first)}
}
