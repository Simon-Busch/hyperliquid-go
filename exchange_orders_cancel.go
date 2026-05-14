package hyperliquid

import (
	"encoding/json"
	"fmt"
)

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

// Cancel cancels the resting order with the given exchange oid. A
// venue-side per-order error (already cancelled, filled, never placed)
// surfaces as a typed Go error rather than being buried in the result
// Status field.
func (t *Trader) Cancel(coin string, oid int64) (CancelResult, error) {
	resp, err := t.bulkCancel([]CancelOrderRequest{{Coin: coin, OrderID: oid}})
	if err != nil {
		return CancelResult{}, err
	}
	return cancelResultOrError(resp)
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
// Per-order errors are surfaced as Go errors, same as Cancel.
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
	return cancelResultOrError(res)
}

// cancelResultOrError inspects the first per-order status from a cancel
// response. A bare string status ("success") becomes a CancelResult with
// no error; an object status with an "error" field becomes a Go error
// carrying the venue's message verbatim, so callers can distinguish
// idempotent re-cancels from genuine successes.
func cancelResultOrError(resp *APIResponse[CancelOrderResponse]) (CancelResult, error) {
	if resp == nil {
		return CancelResult{}, fmt.Errorf("cancel: empty response")
	}
	if !resp.Ok {
		return CancelResult{Error: resp.Err}, fmt.Errorf("cancel: %s", resp.Err)
	}
	if len(resp.Data.Statuses) == 0 {
		return CancelResult{}, nil
	}
	first := resp.Data.Statuses[0]
	// Try to decode as an error object first; on failure treat as a
	// plain string status.
	var errObj struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(first, &errObj); err == nil && errObj.Error != "" {
		return CancelResult{Status: string(first), Error: errObj.Error},
			fmt.Errorf("cancel: %s", errObj.Error)
	}
	return CancelResult{Status: string(first)}, nil
}
