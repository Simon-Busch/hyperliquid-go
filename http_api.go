package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Simon-Busch/hyperliquid-go/internal/transport"
)

const (
	// MainnetAPIURL is the production Hyperliquid REST endpoint.
	MainnetAPIURL = "https://api.hyperliquid.xyz"
	// TestnetAPIURL is the testnet Hyperliquid REST endpoint.
	TestnetAPIURL = "https://api.hyperliquid-testnet.xyz"
	// LocalAPIURL is a local development endpoint.
	LocalAPIURL = "http://localhost:3001"

	// httpErrorStatusCode is the minimum status code considered an error.
	httpErrorStatusCode = 400
)

// httpAPI is the low-level HTTP wrapper used by Info and Trader. It is not
// part of the public surface; callers should construct a top-level Client
// via New() and use its Info/Trade/Stream fields.
type httpAPI struct {
	baseURL    string
	httpClient *http.Client
}

// newHTTPAPI builds an httpAPI targeting baseURL with the default transport.
func newHTTPAPI(baseURL string, httpClient *http.Client) *httpAPI {
	if baseURL == "" {
		baseURL = MainnetAPIURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Transport: transport.Default()}
	}
	return &httpAPI{baseURL: baseURL, httpClient: httpClient}
}

func (c *httpAPI) post(path string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if os.Getenv("HL_DEBUG_HTTP") == "true" {
		fmt.Fprintf(os.Stderr, ">>> POST %s%s\n%s\n", c.baseURL, path, jsonData)
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

	if resp.StatusCode >= httpErrorStatusCode {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: string(body),
		}
	}

	return body, nil
}

// warmUp sends a lightweight /info request to establish the HTTP/2 connection
// up front so the first real request doesn't pay the cold-start penalty.
func (c *httpAPI) warmUp() error {
	_, err := c.post("/info", map[string]any{"type": "meta"})
	return err
}
