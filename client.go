// Package hyperliquid provides a Go client library for the Hyperliquid exchange API.
// It includes support for both REST API and WebSocket connections, allowing users to
// access market data, manage orders, and handle user account operations.
package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Simon-Busch/hyperliquid-go/internal/transport"
)

const (
	MainnetAPIURL = "https://api.hyperliquid.xyz"
	TestnetAPIURL = "https://api.hyperliquid-testnet.xyz"
	LocalAPIURL   = "http://localhost:3001"

	// httpErrorStatusCode is the minimum status code considered an error
	httpErrorStatusCode = 400
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a low-level HTTP client targeting baseURL (defaults to
// mainnet). It is primarily an internal helper; prefer hyperliquid.New.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport.Default(),
		},
	}
}

// NewClientWithHTTPClient creates a Client with a caller-provided http.Client,
// allowing full control over transport, timeouts, and connection pooling.
func NewClientWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Transport: transport.Default()}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) post(path string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: string(body),
		}
	}

	return body, nil
}

// WarmUp sends a lightweight request to establish and warm the HTTP/2 connection
// (TCP + TLS handshake + ALPN negotiation). Call this once at startup so the first
// real order doesn't pay the cold-connection penalty.
func (c *Client) WarmUp() error {
	_, err := c.post("/info", map[string]any{"type": "meta"})
	return err
}
