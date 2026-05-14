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
	resp, err := t.bulkCancel([]CancelOrderRequest{{Coin: coin, OrderID: oid}})
	if err != nil {
		return CancelResult{}, err
	}
	return firstCancelResult(resp), nil
}

// bulkCancel signs and submits a cancel action for every request.
func (t *Trader) bulkCancel(requests []CancelOrderRequest) (*APIResponse[CancelOrderResponse], error) {
	cancels := make([]CancelOrderWire, len(requests))
	for i, req := range requests {
		cancels[i] = CancelOrderWire{
			Asset:   t.info.AssetID(req.Coin),
			OrderID: req.OrderID,
		}
	}

	action := CancelAction{
		Type:    "cancel",
		Cancels: cancels,
	}

	var res *APIResponse[CancelOrderResponse]
	if err := t.executeAction(action, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// CancelOrderRequestByCloid identifies a single order to cancel by client id.
type CancelOrderRequestByCloid struct {
	Coin  string
	Cloid string
}

// CancelByCloid cancels the resting order with the given client order id.
func (t *Trader) CancelByCloid(coin, cloid string) (CancelResult, error) {
	cancels := []CancelByCloidWire{{Asset: t.info.AssetID(coin), ClientID: cloid}}
	action := CancelByCloidAction{
		Type:    "cancelByCloid",
		Cancels: cancels,
	}
	var res *APIResponse[CancelOrderResponse]
	if err := t.executeAction(action, &res); err != nil {
		return CancelResult{}, err
	}
	return firstCancelResult(res), nil
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
