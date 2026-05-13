package hyperliquid

import (
	"fmt"
)

// place is the shared pipeline for every single-order placement verb. It
// validates the spec, converts it to the wire-level CreateOrderRequest list
// (including any bracket legs), and dispatches via the existing BulkOrders
// path. The returned Result flattens the most useful fields from the
// underlying APIResponse.
func (t *Trader) place(spec *OrderSpec) (Result, error) {
	if err := validate(spec, t.info); err != nil {
		return Result{}, err
	}

	reqs, builder, err := specToRequests(spec)
	if err != nil {
		return Result{}, err
	}

	resp, err := t.BulkOrdersWithGrouping(reqs, bracketGrouping(spec), builder)
	if err != nil {
		return Result{}, err
	}
	return resultFromResponse(resp), nil
}

// placeMany is the shared pipeline for PlaceMany. Each spec is validated
// individually before any signing happens.
func (t *Trader) placeMany(specs []OrderSpec) (BatchResult, error) {
	all := make([]CreateOrderRequest, 0, len(specs))
	var builder *BuilderInfo
	for i := range specs {
		s := &specs[i]
		if err := validate(s, t.info); err != nil {
			return BatchResult{}, err
		}
		reqs, b, err := specToRequests(s)
		if err != nil {
			return BatchResult{}, err
		}
		if b != nil {
			builder = b
		}
		all = append(all, reqs...)
	}
	resp, err := t.BulkOrders(all, builder)
	if err != nil {
		return BatchResult{}, err
	}
	return batchResultFromResponse(resp), nil
}

// specToRequests converts a single OrderSpec to one or more
// CreateOrderRequests (parent + optional bracket legs) and the optional
// BuilderInfo for the action.
func specToRequests(spec *OrderSpec) ([]CreateOrderRequest, *BuilderInfo, error) {
	if spec.Coin == "" {
		return nil, nil, fmt.Errorf("OrderSpec.Coin is required")
	}

	parent := CreateOrderRequest{
		Coin:       spec.Coin,
		IsBuy:      spec.Side.IsBuy(),
		Price:      spec.Price,
		Size:       spec.Size,
		ReduceOnly: spec.ReduceOnly,
	}
	if spec.Cloid != "" {
		cloid := spec.Cloid
		parent.ClientOrderID = &cloid
	}

	switch spec.Method {
	case "trigger":
		parent.OrderType = OrderType{
			Trigger: &TriggerOrderType{
				TriggerPx: spec.TriggerPx,
				IsMarket:  spec.IsMarket,
				Tpsl:      triggerTpsl(spec),
			},
		}
	default:
		parent.OrderType = OrderType{
			Limit: &LimitOrderType{Tif: string(tifFromMethod(spec))},
		}
	}

	out := []CreateOrderRequest{parent}
	out = append(out, bracketOrders(spec)...)

	var builder *BuilderInfo
	if spec.BuilderAddr != "" {
		builder = &BuilderInfo{Builder: spec.BuilderAddr, Fee: spec.BuilderFeeBps}
	}
	return out, builder, nil
}

// tifFromMethod maps a placement method to its wire TIF.
func tifFromMethod(spec *OrderSpec) TIF {
	switch spec.Method {
	case "alo":
		return tifALO
	case "ioc", "market":
		return tifIOC
	case "gtc", "close", "modify":
		return tifGTC
	default:
		if spec.TIF != "" {
			return spec.TIF
		}
		return tifGTC
	}
}

// triggerTpsl picks the TPSL discriminator for a PlaceTrigger spec.
func triggerTpsl(spec *OrderSpec) string {
	if spec.Side.IsBuy() {
		return "sl"
	}
	return "tp"
}

// bracketGrouping returns the grouping value for a place() action: if the
// spec has bracket legs, "normalTpsl"; otherwise "na".
func bracketGrouping(spec *OrderSpec) Grouping {
	if spec.TakeProfit > 0 || spec.StopLoss > 0 {
		return GroupingNormalTpsl
	}
	return GroupingNA
}

// resultFromResponse flattens an *APIResponse[OrderResponse] into a Result.
func resultFromResponse(resp *APIResponse[OrderResponse]) Result {
	r := Result{}
	if resp == nil {
		return r
	}
	if !resp.Ok {
		r.Error = resp.Err
		return r
	}
	if len(resp.Data.Statuses) == 0 {
		return r
	}
	first := resp.Data.Statuses[0]
	switch first.Type() {
	case "object":
		var st OrderStatus
		if err := first.Parse(&st); err == nil {
			if st.Resting != nil {
				r.OID = st.Resting.Oid
				r.Cloid = st.Resting.ClientID
				r.Status = st.Resting.Status
			}
			if st.Filled != nil {
				r.OID = int64(st.Filled.Oid)
				r.AvgPx = st.Filled.AvgPx
				r.TotalSz = st.Filled.TotalSz
				r.Status = "filled"
			}
			if st.Error != nil {
				r.Error = *st.Error
			}
		}
	case "string":
		r.Status = string(first)
	}
	return r
}

// batchResultFromResponse flattens an *APIResponse[OrderResponse] into a
// BatchResult, one Result per status.
func batchResultFromResponse(resp *APIResponse[OrderResponse]) BatchResult {
	br := BatchResult{}
	if resp == nil {
		return br
	}
	if !resp.Ok {
		br.Error = resp.Err
		return br
	}
	for _, s := range resp.Data.Statuses {
		var single Result
		switch s.Type() {
		case "object":
			var st OrderStatus
			if err := s.Parse(&st); err == nil {
				if st.Resting != nil {
					single.OID = st.Resting.Oid
					single.Cloid = st.Resting.ClientID
					single.Status = st.Resting.Status
				}
				if st.Filled != nil {
					single.OID = int64(st.Filled.Oid)
					single.AvgPx = st.Filled.AvgPx
					single.TotalSz = st.Filled.TotalSz
					single.Status = "filled"
				}
				if st.Error != nil {
					single.Error = *st.Error
				}
			}
		case "string":
			single.Status = string(s)
		}
		br.Results = append(br.Results, single)
	}
	return br
}
