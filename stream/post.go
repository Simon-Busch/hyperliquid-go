package stream

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Simon-Busch/hyperliquid-go/signing"
)

// pendingRequest tracks an in-flight WS POST awaiting its response.
type pendingRequest struct {
	responseChan chan WsPostResponseData
}

// Post sends a POST-style request over the WebSocket and waits up to
// timeout for the response. Lower-level than PostInfo / PostAction;
// prefer those.
func (c *Client) Post(
	requestType string,
	payload any,
	timeout time.Duration,
) (*WsPostResponseData, error) {
	if !c.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}

	id := int(c.nextPostID.Add(1))
	responseChan := make(chan WsPostResponseData, 1)

	pending := &pendingRequest{
		responseChan: responseChan,
	}

	c.pendingMu.Lock()
	c.pendingRequests[id] = pending
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pendingRequests, id)
		c.pendingMu.Unlock()
	}()

	request := WsPostRequest{
		Method: "post",
		ID:     id,
		Request: WsRequest{
			Type:    requestType,
			Payload: payload,
		},
	}

	if err := c.writeJSON(request); err != nil {
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
func (c *Client) PostInfo(
	payload map[string]any,
	timeout time.Duration,
) (json.RawMessage, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	resp, err := c.Post("info", payload, timeout)
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
func (c *Client) PostAction(
	action any,
	signature signing.SignatureResult,
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

	resp, err := c.Post("action", payload, timeout)
	if err != nil {
		return nil, err
	}

	if resp.Response.Type == "error" {
		return nil, fmt.Errorf("action request error: %s", string(resp.Response.Payload))
	}

	return resp.Response.Payload, nil
}
