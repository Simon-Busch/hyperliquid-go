package hyperliquid

import (
	"encoding/json"
	"fmt"
	"time"
)

// pendingRequest tracks an in-flight WS POST awaiting its response.
type pendingRequest struct {
	responseChan chan WsPostResponseData
}

// PostRequest sends a POST-style request over the WebSocket and waits up
// to timeout for the response. Lower-level than PostInfoRequest /
// PostActionRequest; prefer those.
func (w *Stream) PostRequest(
	requestType string,
	payload any,
	timeout time.Duration,
) (*WsPostResponseData, error) {
	if !w.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}

	id := int(w.nextPostID.Add(1))
	responseChan := make(chan WsPostResponseData, 1)

	pending := &pendingRequest{
		responseChan: responseChan,
	}

	w.pendingMu.Lock()
	w.pendingRequests[id] = pending
	w.pendingMu.Unlock()

	defer func() {
		w.pendingMu.Lock()
		delete(w.pendingRequests, id)
		w.pendingMu.Unlock()
	}()

	request := WsPostRequest{
		Method: "post",
		ID:     id,
		Request: WsRequest{
			Type:    requestType,
			Payload: payload,
		},
	}

	if err := w.writeJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case response, ok := <-responseChan:
		if !ok {
			return nil, fmt.Errorf("request cancelled")
		}
		return &response, nil
	case <-timer.C:
		return nil, fmt.Errorf("request timeout")
	}
}

// PostInfoRequest sends an info-style request over the WebSocket. When
// timeout is zero the call waits up to 30s.
func (w *Stream) PostInfoRequest(
	payload map[string]any,
	timeout time.Duration,
) (json.RawMessage, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	resp, err := w.PostRequest("info", payload, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Response.Type == "error" {
		return nil, fmt.Errorf("info request error: %s", string(resp.Response.Payload))
	}

	return resp.Response.Payload, nil
}

// PostActionRequest sends a signed action over the WebSocket. vaultAddress
// is forwarded as-is — supply an empty string to set vaultAddress: null.
// When timeout is zero the call waits up to 30s.
func (w *Stream) PostActionRequest(
	action any,
	signature SignatureResult,
	nonce int64,
	vaultAddress string,
	timeout time.Duration,
) (json.RawMessage, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	payload := map[string]any{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}

	if vaultAddress != "" {
		payload["vaultAddress"] = vaultAddress
	} else {
		payload["vaultAddress"] = nil
	}

	resp, err := w.PostRequest("action", payload, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Response.Type == "error" {
		return nil, fmt.Errorf("action request error: %s", string(resp.Response.Payload))
	}

	return resp.Response.Payload, nil
}
