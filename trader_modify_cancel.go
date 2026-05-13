package hyperliquid

import "fmt"

// Modify changes the price (or size, or both) of a resting order identified
// by oid. The coin is preserved on the existing order — only the supplied
// fields change. Required: WithLimit(newPx) for a new price, WithSize(x)
// for a new size, or both.
func (t *Trader) Modify(oid int64, opts ...PlaceOpt) (Result, error) {
	spec := OrderSpec{Method: "modify", ModifyOID: oid, TIF: tifGTC}
	for _, o := range opts {
		o(&spec)
	}
	return t.doModify(&spec)
}

// ModifyByCloid changes a resting order identified by its client order id.
func (t *Trader) ModifyByCloid(cloid string, opts ...PlaceOpt) (Result, error) {
	spec := OrderSpec{Method: "modify", ModifyCloid: cloid, TIF: tifGTC}
	for _, o := range opts {
		o(&spec)
	}
	return t.doModify(&spec)
}

func (t *Trader) doModify(spec *OrderSpec) (Result, error) {
	if err := t.validate(spec); err != nil {
		return Result{}, err
	}
	if spec.LimitPrice > 0 {
		spec.Price = spec.LimitPrice
	}
	if spec.OverrideSize > 0 {
		spec.Size = spec.OverrideSize
	}
	// Modify is a cancel + replace under the hood; default the
	// replacement TIF to ALO (post-only) so a far-from-mid resting order
	// stays inside Hyperliquid's price-band rules. A future hl.WithTIF
	// option can let callers override when modifying a GTC/IOC order.
	req := CreateOrderRequest{
		Coin:       spec.Coin,
		IsBuy:      spec.Side.IsBuy(),
		Price:      spec.Price,
		Size:       spec.Size,
		ReduceOnly: spec.ReduceOnly,
		OrderType:  OrderType{Limit: &LimitOrderType{Tif: string(tifALO)}},
	}
	if spec.Cloid != "" {
		c := spec.Cloid
		req.ClientOrderID = &c
	}
	var oidAny any
	if spec.ModifyOID != 0 {
		oidAny = spec.ModifyOID
	} else {
		oidAny = spec.ModifyCloid
	}
	action, err := newModifyOrderAction(t, ModifyOrderRequest{Oid: oidAny, Order: req})
	if err != nil {
		return Result{}, fmt.Errorf("failed to create modify action: %w", err)
	}
	resp := APIResponse[OrderResponse]{}
	if err := t.executeAction(action, &resp); err != nil {
		return Result{}, fmt.Errorf("failed to modify order: %w", err)
	}
	if !resp.Ok {
		return Result{}, fmt.Errorf("failed to modify order: %s", resp.Err)
	}
	if len(resp.Data.Statuses) == 0 {
		return Result{}, fmt.Errorf("no status for modified order: %s", resp.Err)
	}
	first := resp.Data.Statuses[0]
	if first.Type() != "object" {
		return Result{}, fmt.Errorf("unexpected status type: %s", first.Type())
	}
	var status OrderStatus
	if err := first.Parse(&status); err != nil {
		return Result{}, fmt.Errorf("failed to parse modified order status: %w", err)
	}
	r := Result{}
	if status.Resting != nil {
		r.OID = status.Resting.Oid
		r.Cloid = status.Resting.ClientID
		r.Status = status.Resting.Status
	}
	if status.Filled != nil {
		r.OID = int64(status.Filled.Oid)
		r.AvgPx = status.Filled.AvgPx
		r.TotalSz = status.Filled.TotalSz
		r.Status = "filled"
	}
	if status.Error != nil {
		r.Error = *status.Error
	}
	return r, nil
}

// CancelAll cancels every open order across the supplied coins. With no
// coins supplied it cancels everything across every asset.
func (t *Trader) CancelAll(coins ...string) (BatchCancelResult, error) {
	addr := t.accountAddr
	if addr == "" {
		addr = t.vault
	}
	orders, err := t.info.OpenOrders(addr)
	if err != nil {
		return BatchCancelResult{}, err
	}
	keep := func(coin string) bool {
		if len(coins) == 0 {
			return true
		}
		for _, c := range coins {
			if c == coin {
				return true
			}
		}
		return false
	}
	reqs := make([]CancelOrderRequest, 0, len(orders))
	for _, o := range orders {
		if !keep(o.Coin) {
			continue
		}
		reqs = append(reqs, CancelOrderRequest{Coin: o.Coin, OrderID: o.Oid})
	}
	if len(reqs) == 0 {
		return BatchCancelResult{}, nil
	}
	resp, err := t.bulkCancel(reqs)
	if err != nil {
		return BatchCancelResult{}, err
	}
	return cancelBatchFromResponse(resp), nil
}

// cancelBatchFromResponse maps a bulk-cancel APIResponse into a
// BatchCancelResult, one entry per status returned by the server.
func cancelBatchFromResponse(resp *APIResponse[CancelOrderResponse]) BatchCancelResult {
	br := BatchCancelResult{}
	if resp == nil {
		return br
	}
	if !resp.Ok {
		br.Error = resp.Err
		return br
	}
	for _, s := range resp.Data.Statuses {
		br.Results = append(br.Results, CancelResult{Status: string(s)})
	}
	return br
}
