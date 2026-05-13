package hyperliquid

import (
	"fmt"
)

type CreateOrderRequest struct {
	Coin          string
	IsBuy         bool
	Price         float64
	Size          float64
	ReduceOnly    bool
	OrderType     OrderType
	ClientOrderID *string
}

type OrderStatusResting struct {
	Oid      int64  `json:"oid"`
	ClientID string `json:"cid"`
	Status   string `json:"status"`
}

type OrderStatusFilled struct {
	TotalSz string `json:"totalSz"`
	AvgPx   string `json:"avgPx"`
	Oid     int    `json:"oid"`
}

type OrderStatus struct {
	Resting *OrderStatusResting `json:"resting,omitempty"`
	Filled  *OrderStatusFilled  `json:"filled,omitempty"`
	Error   *string             `json:"error,omitempty"`
}

type OrderResponse struct {
	Statuses MixedArray `json:"statuses"`
}

// newCreateOrderAction builds an order action with grouping set to "na"
func newCreateOrderAction(
	e *Trader,
	orders []CreateOrderRequest,
	info *BuilderInfo,
) (OrderAction, error) {
	return newCreateOrderActionWithGrouping(e, orders, info, GroupingNA)
}

// NewCreateOrderActionWithGrouping is the public wrapper for creating order actions
// This is useful for WebSocket POST requests where you need the action before signing
func (e *Trader) NewCreateOrderActionWithGrouping(
	orders []CreateOrderRequest,
	info *BuilderInfo,
	grouping Grouping,
) (OrderAction, error) {
	return newCreateOrderActionWithGrouping(e, orders, info, grouping)
}

// newCreateOrderActionWithGrouping builds an order action allowing a specific grouping
func newCreateOrderActionWithGrouping(
	e *Trader,
	orders []CreateOrderRequest,
	info *BuilderInfo,
	grouping Grouping,
) (OrderAction, error) {
	orderRequests := make([]OrderWire, len(orders))
	for i, order := range orders {
		asset := e.info.AssetID(order.Coin)
		class := ClassifyAsset(asset)

		priceWire, err := PriceToWire(order.Price, asset, e.info, class)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire price for order %d: %w", i, err)
		}

		sizeWire, err := sizeToWireWithAsset(order.Size, asset, e.info)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire size for order %d: %w", i, err)
		}

		var orderTypeWire OrderTypeWire
		if order.OrderType.Limit != nil {
			orderTypeWire.Limit = &LimitOrderTypeWire{Tif: order.OrderType.Limit.Tif}
		} else if order.OrderType.Trigger != nil {
			triggerPxWire, err := PriceToWire(order.OrderType.Trigger.TriggerPx, asset, e.info, class)
			if err != nil {
				return OrderAction{}, fmt.Errorf("failed to wire trigger price for order %d: %w", i, err)
			}
			orderTypeWire.Trigger = &TriggerOrderTypeWire{
				TriggerPx: triggerPxWire,
				IsMarket:  order.OrderType.Trigger.IsMarket,
				Tpsl:      order.OrderType.Trigger.Tpsl,
			}
		}

		orderRequests[i] = OrderWire{
			Asset:      e.info.AssetID(order.Coin),
			IsBuy:      order.IsBuy,
			LimitPx:    priceWire,
			Size:       sizeWire,
			ReduceOnly: order.ReduceOnly,
			OrderType:  orderTypeWire,
			Cloid:      order.ClientOrderID,
		}
	}

	return OrderAction{
		Type:     "order",
		Dex:      e.dex, // Include dex for HIP-3 builder-deployed perps
		Orders:   orderRequests,
		Grouping: string(grouping),
		Builder:  info,
	}, nil
}

func (e *Trader) Order(
	req CreateOrderRequest,
	builder *BuilderInfo,
) (result OrderStatus, err error) {
	resp, err := e.BulkOrders([]CreateOrderRequest{req}, builder)
	if err != nil {
		return
	}

	if !resp.Ok {
		err = fmt.Errorf("failed to create order: %s", resp.Err)
		return
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		err = fmt.Errorf("no order status returned")
		return
	}

	// Parse the first status if it's an object; ignore string tokens like "waitingForTrigger"
	first := data.Statuses[0]
	switch first.Type() {
	case "object":
		var st OrderStatus
		if err := first.Parse(&st); err != nil {
			return OrderStatus{}, fmt.Errorf("failed to parse order status: %w", err)
		}
		return st, nil
	case "string":
		// Return empty with an informational error to signal non-object status
		return OrderStatus{}, fmt.Errorf("order status is token: %s", string(first))
	default:
		return OrderStatus{}, fmt.Errorf("unexpected order status type: %s", first.Type())
	}
}

func (e *Trader) BulkOrders(
	orders []CreateOrderRequest,
	builder *BuilderInfo,
) (result *APIResponse[OrderResponse], err error) {
	action, err := newCreateOrderAction(e, orders, builder)
	if err != nil {
		return nil, err
	}
	err = e.executeAction(action, &result)
	return
}

// BulkOrdersWithGrouping places multiple orders in a single action with the provided grouping
func (e *Trader) BulkOrdersWithGrouping(
	orders []CreateOrderRequest,
	grouping Grouping,
	builder *BuilderInfo,
) (result *APIResponse[OrderResponse], err error) {
	action, err := newCreateOrderActionWithGrouping(e, orders, builder, grouping)
	if err != nil {
		return nil, err
	}
	err = e.executeAction(action, &result)
	return
}

type ModifyOrderRequest struct {
	Oid   any // can be int64 or Cloid
	Order CreateOrderRequest
}

func newModifyOrderAction(
	e *Trader,
	modifyRequest ModifyOrderRequest,
) (ModifyAction, error) {
	asset := e.info.AssetID(modifyRequest.Order.Coin)
	class := ClassifyAsset(asset)

	priceWire, err := PriceToWire(modifyRequest.Order.Price, asset, e.info, class)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire price: %w", err)
	}

	sizeWire, err := sizeToWireWithAsset(modifyRequest.Order.Size, asset, e.info)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire size: %w", err)
	}

	// Build order type with deterministic wire struct
	var orderTypeWire OrderTypeWire
	if modifyRequest.Order.OrderType.Limit != nil {
		orderTypeWire.Limit = &LimitOrderTypeWire{Tif: modifyRequest.Order.OrderType.Limit.Tif}
	} else if modifyRequest.Order.OrderType.Trigger != nil {
		triggerPxWire, err := PriceToWire(modifyRequest.Order.OrderType.Trigger.TriggerPx, asset, e.info, class)
		if err != nil {
			return ModifyAction{}, fmt.Errorf("failed to wire trigger price: %w", err)
		}
		orderTypeWire.Trigger = &TriggerOrderTypeWire{
			TriggerPx: triggerPxWire,
			IsMarket:  modifyRequest.Order.OrderType.Trigger.IsMarket,
			Tpsl:      modifyRequest.Order.OrderType.Trigger.Tpsl,
		}
	}

	return ModifyAction{
		Type: "modify",
		Dex:  e.dex, // Include dex for HIP-3 builder-deployed perps
		Oid:  modifyRequest.Oid,
		Order: OrderWire{
			Asset:      e.info.AssetID(modifyRequest.Order.Coin),
			IsBuy:      modifyRequest.Order.IsBuy,
			LimitPx:    priceWire,
			Size:       sizeWire,
			ReduceOnly: modifyRequest.Order.ReduceOnly,
			OrderType:  orderTypeWire,
			Cloid:      modifyRequest.Order.ClientOrderID,
		},
	}, nil
}

func newModifyOrdersAction(
	e *Trader,
	modifyRequests []ModifyOrderRequest,
) (BatchModifyAction, error) {
	modifies := make([]ModifyAction, len(modifyRequests))
	for i, req := range modifyRequests {
		modify, err := newModifyOrderAction(e, req)
		if err != nil {
			return BatchModifyAction{}, fmt.Errorf("failed to create modify request %d: %w", i, err)
		}
		// Clear type and dex for inner modifies (they go on the outer BatchModifyAction)
		modify.Type = ""
		modify.Dex = ""
		modifies[i] = modify
	}

	return BatchModifyAction{
		Type:     "batchModify",
		Dex:      e.dex, // Include dex for HIP-3 builder-deployed perps
		Modifies: modifies,
	}, nil
}

// ModifyOrder modifies an existing order
func (e *Trader) ModifyOrder(
	req ModifyOrderRequest,
) (result OrderStatus, err error) {
	resp := APIResponse[OrderResponse]{}
	action, err := newModifyOrderAction(e, req)
	if err != nil {
		return result, fmt.Errorf("failed to create modify action: %w", err)
	}

	err = e.executeAction(action, &resp)
	if err != nil {
		return result, fmt.Errorf("failed to modify order: %w", err)
	}

	if !resp.Ok {
		return result, fmt.Errorf("failed to modify order: %s", resp.Err)
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		return result, fmt.Errorf("no status for modified order: %s", resp.Err)
	}

	// Parse first object status
	first := data.Statuses[0]
	if first.Type() != "object" {
		return result, fmt.Errorf("unexpected status type: %s", first.Type())
	}
	var parsed OrderStatus
	if err := first.Parse(&parsed); err != nil {
		return result, fmt.Errorf("failed to parse modified order status: %w", err)
	}
	return parsed, nil
}

// BulkModifyOrders modifies multiple orders
func (e *Trader) BulkModifyOrders(
	modifyRequests []ModifyOrderRequest,
) ([]OrderStatus, error) {
	resp := APIResponse[OrderResponse]{}
	action, err := newModifyOrdersAction(e, modifyRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to create bulk modify action: %w", err)
	}

	err = e.executeAction(action, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to modify orders: %w", err)
	}

	if !resp.Ok {
		return nil, fmt.Errorf("failed to modify orders: %s", resp.Err)
	}

	data := resp.Data
	if len(data.Statuses) == 0 {
		return nil, fmt.Errorf("no status for modified order: %s", resp.Err)
	}
	// Parse only object statuses
	var out []OrderStatus
	for _, mv := range data.Statuses {
		if mv.Type() != "object" {
			continue
		}
		var st OrderStatus
		if err := mv.Parse(&st); err != nil {
			return nil, fmt.Errorf("failed to parse modified status: %w", err)
		}
		out = append(out, st)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no object statuses returned")
	}
	return out, nil
}

// MarketOpen opens a market position
func (e *Trader) MarketOpen(
	name string,
	isBuy bool,
	sz float64,
	px *float64,
	slippage float64,
	cloid *string,
	builder *BuilderInfo,
) (res OrderStatus, err error) {
	slippagePrice, err := e.SlippagePrice(name, isBuy, slippage, px)
	if err != nil {
		return
	}

	orderType := OrderType{
		Limit: &LimitOrderType{
			Tif: TifIoc,
		},
	}

	req := CreateOrderRequest{
		Coin:          name,
		IsBuy:         isBuy,
		Size:          sz,
		Price:         slippagePrice,
		ReduceOnly:    false,
		OrderType:     orderType,
		ClientOrderID: cloid,
	}

	return e.Order(req, builder)
}

// MarketOpenWithSLTP opens a position and places either a Stop-Loss (isTP=false) or Take-Profit (isTP=true)
// trigger in a single grouped action. The trigger is reduce-only and market-on-trigger.
// Full-position size is used for the trigger. For partial size, use MarketOpenWithSLTPPartial.
func (e *Trader) MarketOpenWithSLTP(
	name string,
	isBuy bool,
	sz float64,
	px *float64,
	slippage float64,
	tpslPercent float64, // e.g., 0.10 means 10%
	isTP bool,
	cloidOpen *string,
	cloidTPSL *string,
	builder *BuilderInfo,
) (result *APIResponse[OrderResponse], err error) {
	return e.MarketOpenWithSLTPPartial(name, isBuy, sz, px, slippage, tpslPercent, isTP, nil, cloidOpen, cloidTPSL, builder)
}

// MarketOpenWithSLTPPartial is like MarketOpenWithSLTP but allows specifying a partial TP/SL size via tpslSize.
// If tpslSize is nil, the trigger uses the full position size.
func (e *Trader) MarketOpenWithSLTPPartial(
	name string,
	isBuy bool,
	sz float64,
	px *float64,
	slippage float64,
	tpslPercent float64,
	isTP bool,
	tpslSize *float64,
	cloidOpen *string,
	cloidTPSL *string,
	builder *BuilderInfo,
) (result *APIResponse[OrderResponse], err error) {
	// Compute the intended execution price for opening
	openPx, err := e.SlippagePrice(name, isBuy, slippage, px)
	if err != nil {
		return nil, err
	}

	// Compute trigger price relative to open price
	var triggerPx float64
	if isTP {
		if isBuy {
			triggerPx = openPx * (1 + tpslPercent)
		} else {
			triggerPx = openPx * (1 - tpslPercent)
		}
	} else {
		if isBuy {
			triggerPx = openPx * (1 - tpslPercent)
		} else {
			triggerPx = openPx * (1 + tpslPercent)
		}
	}

	// Decide TP/SL size
	triggerSize := sz
	if tpslSize != nil {
		triggerSize = *tpslSize
	}

	// Build orders: 1) IOC open; 2) TP/SL trigger reduce-only
	openOrder := CreateOrderRequest{
		Coin:          name,
		IsBuy:         isBuy,
		Price:         openPx,
		Size:          sz,
		ReduceOnly:    false,
		OrderType:     OrderType{Limit: &LimitOrderType{Tif: TifIoc}},
		ClientOrderID: cloidOpen,
	}

	tpslTrigger := CreateOrderRequest{
		Coin:          name,
		IsBuy:         !isBuy,    // Close direction
		Price:         triggerPx, // included per wire schema, though ignored when isMarket=true
		Size:          triggerSize,
		ReduceOnly:    true,
		OrderType:     OrderType{Trigger: &TriggerOrderType{TriggerPx: triggerPx, IsMarket: true, Tpsl: map[bool]string{true: "tp", false: "sl"}[isTP]}},
		ClientOrderID: cloidTPSL,
	}

	// Use normalTpsl grouping to align with trigger order expectations
	return e.BulkOrdersWithGrouping([]CreateOrderRequest{openOrder, tpslTrigger}, GroupingNormalTpsl, builder)
}

// MarketClose closes a position
func (e *Trader) MarketClose(
	coin string,
	sz *float64,
	px *float64,
	slippage float64,
	cloid *string,
	builder *BuilderInfo,
) (OrderStatus, error) {
	address := e.accountAddr
	if address == "" {
		address = e.vault
	}

	userState, err := e.info.UserState(address)
	if err != nil {
		return OrderStatus{}, err
	}

	for _, assetPos := range userState.AssetPositions {
		pos := assetPos.Position
		if coin != pos.Coin {
			continue
		}

		szi := parseFloat(pos.Szi)
		var size float64
		if sz != nil {
			size = *sz
		} else {
			size = abs(szi)
		}

		isBuy := szi < 0

		slippagePrice, err := e.SlippagePrice(coin, isBuy, slippage, px)
		if err != nil {
			return OrderStatus{}, err
		}

		orderType := OrderType{
			Limit: &LimitOrderType{Tif: TifIoc},
		}

		return e.Order(CreateOrderRequest{
			Coin:          coin,
			IsBuy:         isBuy,
			Size:          size,
			Price:         slippagePrice,
			OrderType:     orderType,
			ReduceOnly:    true,
			ClientOrderID: cloid,
		}, builder)
	}

	return OrderStatus{}, fmt.Errorf("position not found for coin: %s", coin)
}

