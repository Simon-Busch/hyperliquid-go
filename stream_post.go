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

// Post sends a POST-style request over the WebSocket and waits up to
// timeout for the response. Lower-level than PostInfo / PostAction;
// prefer those.
func (s *Stream) Post(
	requestType string,
	payload any,
	timeout time.Duration,
) (*WsPostResponseData, error) {
	if !s.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}

	id := int(s.nextPostID.Add(1))
	responseChan := make(chan WsPostResponseData, 1)

	pending := &pendingRequest{
		responseChan: responseChan,
	}

	s.pendingMu.Lock()
	s.pendingRequests[id] = pending
	s.pendingMu.Unlock()

	defer func() {
		s.pendingMu.Lock()
		delete(s.pendingRequests, id)
		s.pendingMu.Unlock()
	}()

	request := WsPostRequest{
		Method: "post",
		ID:     id,
		Request: WsRequest{
			Type:    requestType,
			Payload: payload,
		},
	}

	if err := s.writeJSON(request); err != nil {
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

// PostInfo sends an info-style request over the WebSocket. When timeout
// is zero the call waits up to 30s.
func (s *Stream) PostInfo(
	payload map[string]any,
	timeout time.Duration,
) (json.RawMessage, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	resp, err := s.Post("info", payload, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Response.Type == "error" {
		return nil, fmt.Errorf("info request error: %s", string(resp.Response.Payload))
	}

	return resp.Response.Payload, nil
}

// PostAction sends a signed action over the WebSocket. vaultAddress is
// forwarded as-is -- supply an empty string to set vaultAddress: null.
// When timeout is zero the call waits up to 30s.
func (s *Stream) PostAction(
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

	resp, err := s.Post("action", payload, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Response.Type == "error" {
		return nil, fmt.Errorf("action request error: %s", string(resp.Response.Payload))
	}

	return resp.Response.Payload, nil
}
