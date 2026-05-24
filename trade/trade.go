package trade

import (
	"fmt"

	xtransport "github.com/Simon-Busch/hyperliquid-go/internal/transport"
	"github.com/Simon-Busch/hyperliquid-go/types"
)

// place is the shared pipeline for every single-order placement verb. It
// validates the spec, converts it to the wire-level CreateOrderRequest list
// (including any bracket legs), and dispatches the signed order action.
// The returned Result flattens the most useful fields from the underlying
// APIResponse.
func (c *Client) place(spec *types.OrderSpec) (types.Result, error) {
	if err := c.validate(spec); err != nil {
		return types.Result{}, err
	}

	reqs, builder, err := specToRequests(spec)
	if err != nil {
		return types.Result{}, err
	}

	action, err := newCreateOrderActionWithGrouping(c, reqs, builder, bracketGrouping(spec))
	if err != nil {
		return types.Result{}, err
	}
	var resp *xtransport.APIResponse[OrderResponse]
	if err := c.executeAction(action, &resp); err != nil {
		return types.Result{}, err
	}
	return resultFromResponse(resp), nil
}

// placeMany is the shared pipeline for PlaceMany. Each spec is validated
// individually before any signing happens.
func (c *Client) placeMany(specs []types.OrderSpec) (types.BatchResult, error) {
	all := make([]CreateOrderRequest, 0, len(specs))
	var builder *types.BuilderInfo
	for i := range specs {
		s := &specs[i]
		if err := c.validate(s); err != nil {
			return types.BatchResult{}, err
		}
		reqs, b, err := specToRequests(s)
		if err != nil {
			return types.BatchResult{}, err
		}
		if b != nil {
			builder = b
		}
		all = append(all, reqs...)
	}
	action, err := newCreateOrderActionWithGrouping(c, all, builder, types.GroupingNA)
	if err != nil {
		return types.BatchResult{}, err
	}
	var resp *xtransport.APIResponse[OrderResponse]
	if err := c.executeAction(action, &resp); err != nil {
		return types.BatchResult{}, err
	}
	return batchResultFromResponse(resp), nil
}

// specToRequests converts a single OrderSpec to one or more
// CreateOrderRequests (parent + optional bracket legs) and the optional
// BuilderInfo for the action.
func specToRequests(spec *types.OrderSpec) ([]CreateOrderRequest, *types.BuilderInfo, error) {
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
		parent.OrderType = types.OrderType{
			Trigger: &types.TriggerOrderType{
				TriggerPx: spec.TriggerPx,
				IsMarket:  spec.IsMarket,
				Tpsl:      triggerTpsl(spec),
			},
		}
	default:
		parent.OrderType = types.OrderType{
			Limit: &types.LimitOrderType{Tif: string(tifFromMethod(spec))},
		}
	}

	out := []CreateOrderRequest{parent}
	out = append(out, bracketOrders(spec)...)

	var builder *types.BuilderInfo
	if spec.BuilderAddr != "" {
		builder = &types.BuilderInfo{Builder: spec.BuilderAddr, Fee: spec.BuilderFeeBps}
	}
	return out, builder, nil
}

// tifFromMethod maps a placement method to its wire TIF.
func tifFromMethod(spec *types.OrderSpec) types.TIF {
	switch spec.Method {
	case "alo":
		return types.TifAlo
	case "ioc", "market":
		return types.TifIoc
	case "gtc", "close", "modify":
		return types.TifGtc
	default:
		if spec.TIF != "" {
			return spec.TIF
		}
		return types.TifGtc
	}
}

// triggerTpsl picks the TPSL discriminator for a PlaceTrigger spec.
func triggerTpsl(spec *types.OrderSpec) string {
	if spec.Side.IsBuy() {
		return "sl"
	}
	return "tp"
}

// bracketGrouping returns the grouping value for a place() action: if the
// spec has bracket legs, "normalTpsl"; otherwise "na".
func bracketGrouping(spec *types.OrderSpec) types.Grouping {
	if spec.TakeProfit > 0 || spec.StopLoss > 0 {
		return types.GroupingNormalTpsl
	}
	return types.GroupingNA
}

// resultFromResponse flattens an *APIResponse[OrderResponse] into a Result.
func resultFromResponse(resp *xtransport.APIResponse[OrderResponse]) types.Result {
	r := types.Result{}
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
func batchResultFromResponse(resp *xtransport.APIResponse[OrderResponse]) types.BatchResult {
	br := types.BatchResult{}
	if resp == nil {
		return br
	}
	if !resp.Ok {
		br.Error = resp.Err
		return br
	}
	for _, s := range resp.Data.Statuses {
		var single types.Result
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
