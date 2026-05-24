package hyperliquid

import "encoding/json"

// WsMsg represents a WebSocket message with a channel and data payload.
type WsMsg struct {
	Channel string         `json:"channel"`
	Data    map[string]any `json:"data"`
}

// CancelRequest names a single order to cancel by exchange oid.
type CancelRequest struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

// CancelByCloidRequest names a single order to cancel by client order id.
type CancelByCloidRequest struct {
	Coin  string `json:"coin"`
	Cloid string `json:"cloid"`
}

// Trade is a single trade record (used in the trades subscription feed).
type Trade struct {
	Coin  string   `json:"coin"`
	Side  string   `json:"side"`
	Px    string   `json:"px"`
	Sz    string   `json:"sz"`
	Time  int64    `json:"time"`
	Hash  string   `json:"hash"`
	Tid   int64    `json:"tid"`
	Users []string `json:"users"`
}

// BulkOrderResponse is the legacy response shape for a bulk order action.
type BulkOrderResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// CancelResponse is the wire response for a single-cancel action.
type CancelResponse struct {
	Status string     `json:"status"`
	Data   *OpenOrder `json:"data,omitempty"`
	Error  string     `json:"error,omitempty"`
}

// BulkCancelResponse is the wire response for a bulk-cancel action.
type BulkCancelResponse struct {
	Status string      `json:"status"`
	Data   []OpenOrder `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// ModifyResponse is the legacy response shape for an order modify action.
type ModifyResponse struct {
	Status string        `json:"status"`
	Data   []OrderStatus `json:"data,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// TransferResponse is the response shape returned by transfer-style
// signed actions (usdSend, spotSend, vaultTransfer, etc.) and the
// HIP-4 userOutcome action family. Hyperliquid encodes failure as
// {"status":"err","response":"<message>"}; Response captures the raw
// payload so callers can extract the reason without re-parsing the
// wire bytes.
type TransferResponse struct {
	Status   string          `json:"status"`
	TxHash   string          `json:"txHash,omitempty"`
	Error    string          `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

// ApprovalResponse is the response shape returned by approval-style
// actions (approveBuilderFee, evmUserModify, ...).
type ApprovalResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CreateSubAccountResponse is returned by the createSubAccount action.
type CreateSubAccountResponse struct {
	Status string      `json:"status"`
	Data   *SubAccount `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// SetReferrerResponse is returned by the setReferrer action.
type SetReferrerResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ScheduleCancelResponse is returned by the scheduleCancel action.
type ScheduleCancelResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// AgentApprovalResponse is returned by the approveAgent action.
// Hyperliquid encodes failure as {"status":"err","response":"<message>"};
// success as {"status":"ok","response":{...}}. The Response field
// captures whichever form was returned so callers can surface the
// rejection reason verbatim.
type AgentApprovalResponse struct {
	Status   string          `json:"status"`
	TxHash   string          `json:"txHash,omitempty"`
	Error    string          `json:"error,omitempty"`
	Response json.RawMessage `json:"response,omitempty"`
}

// MultiSigConversionResponse is returned by the
// convertToMultiSigUser action.
type MultiSigConversionResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SpotDeployResponse is returned by HIP-2 / HIP-3 spot deploy actions.
type SpotDeployResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ValidatorResponse is returned by CValidator and CSigner actions.
type ValidatorResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// MultiSigResponse is returned by the multiSig action envelope.
type MultiSigResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// PerpDeployResponse is returned by HIP-3 perp deploy actions; the inner
// statuses array reports per-asset outcomes.
type PerpDeployResponse struct {
	Status string `json:"status"`
	Data   struct {
		Statuses []TxStatus `json:"statuses"`
	} `json:"data"`
}

// TxStatus is one per-asset outcome inside PerpDeployResponse.
type TxStatus struct {
	Coin   string `json:"coin"`
	Status string `json:"status"`
}
