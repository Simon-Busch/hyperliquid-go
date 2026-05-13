package hyperliquid

import (
	"fmt"
)

// CreateOrderRequest is the wire-adjacent order request consumed by the
// internal order action builders.
type CreateOrderRequest struct {
	Coin          string
	IsBuy         bool
	Price         float64
	Size          float64
	ReduceOnly    bool
	OrderType     OrderType
	ClientOrderID *string
}

// OrderStatusResting describes a resting (post-only or unfilled) order entry
// returned in an /exchange response.
type OrderStatusResting struct {
	Oid      int64  `json:"oid"`
	ClientID string `json:"cid"`
	Status   string `json:"status"`
}

// OrderStatusFilled describes a filled order entry returned in an /exchange
// response.
type OrderStatusFilled struct {
	TotalSz string `json:"totalSz"`
	AvgPx   string `json:"avgPx"`
	Oid     int    `json:"oid"`
}

// OrderStatus is one element of an /exchange response's statuses array.
type OrderStatus struct {
	Resting *OrderStatusResting `json:"resting,omitempty"`
	Filled  *OrderStatusFilled  `json:"filled,omitempty"`
	Error   *string             `json:"error,omitempty"`
}

// OrderResponse is the data shape returned by /exchange for an order action.
type OrderResponse struct {
	Statuses MixedArray `json:"statuses"`
}

// NewCreateOrderActionWithGrouping is the public wrapper for creating order
// actions. Use when you need the signed action bytes before dispatching --
// for example to forward through Stream.PostAction.
func (t *Trader) NewCreateOrderActionWithGrouping(
	orders []CreateOrderRequest,
	info *BuilderInfo,
	grouping Grouping,
) (OrderAction, error) {
	return newCreateOrderActionWithGrouping(t, orders, info, grouping)
}

// newCreateOrderActionWithGrouping builds an order action allowing a specific grouping.
func newCreateOrderActionWithGrouping(
	t *Trader,
	orders []CreateOrderRequest,
	info *BuilderInfo,
	grouping Grouping,
) (OrderAction, error) {
	orderRequests := make([]OrderWire, len(orders))
	for i, order := range orders {
		asset := t.info.AssetID(order.Coin)
		class := ClassifyAsset(asset)

		priceWire, err := PriceToWire(order.Price, asset, t.info, class)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire price for order %d: %w", i, err)
		}

		sizeWire, err := sizeToWireWithAsset(order.Size, asset, t.info)
		if err != nil {
			return OrderAction{}, fmt.Errorf("failed to wire size for order %d: %w", i, err)
		}

		var orderTypeWire OrderTypeWire
		if order.OrderType.Limit != nil {
			orderTypeWire.Limit = &LimitOrderTypeWire{Tif: order.OrderType.Limit.Tif}
		} else if order.OrderType.Trigger != nil {
			triggerPxWire, err := PriceToWire(order.OrderType.Trigger.TriggerPx, asset, t.info, class)
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
			Asset:      asset,
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
		Dex:      t.dex,
		Orders:   orderRequests,
		Grouping: string(grouping),
		Builder:  info,
	}, nil
}

// ModifyOrderRequest pairs an existing order identifier with the
// replacement order definition.
type ModifyOrderRequest struct {
	Oid   any // can be int64 or Cloid
	Order CreateOrderRequest
}

// newModifyOrderAction builds a ModifyAction from a single ModifyOrderRequest.
func newModifyOrderAction(
	t *Trader,
	modifyRequest ModifyOrderRequest,
) (ModifyAction, error) {
	asset := t.info.AssetID(modifyRequest.Order.Coin)
	class := ClassifyAsset(asset)

	priceWire, err := PriceToWire(modifyRequest.Order.Price, asset, t.info, class)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire price: %w", err)
	}

	sizeWire, err := sizeToWireWithAsset(modifyRequest.Order.Size, asset, t.info)
	if err != nil {
		return ModifyAction{}, fmt.Errorf("failed to wire size: %w", err)
	}

	var orderTypeWire OrderTypeWire
	if modifyRequest.Order.OrderType.Limit != nil {
		orderTypeWire.Limit = &LimitOrderTypeWire{Tif: modifyRequest.Order.OrderType.Limit.Tif}
	} else if modifyRequest.Order.OrderType.Trigger != nil {
		triggerPxWire, err := PriceToWire(modifyRequest.Order.OrderType.Trigger.TriggerPx, asset, t.info, class)
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
		Dex:  t.dex,
		Oid:  modifyRequest.Oid,
		Order: OrderWire{
			Asset:      asset,
			IsBuy:      modifyRequest.Order.IsBuy,
			LimitPx:    priceWire,
			Size:       sizeWire,
			ReduceOnly: modifyRequest.Order.ReduceOnly,
			OrderType:  orderTypeWire,
			Cloid:      modifyRequest.Order.ClientOrderID,
		},
	}, nil
}
