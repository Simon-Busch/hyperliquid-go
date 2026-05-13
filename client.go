// Package hyperliquid provides a Go client library for the Hyperliquid
// exchange API. It includes support for both REST API and WebSocket
// connections, allowing users to access market data, manage orders, and
// handle user account operations.
package hyperliquid

import (
	"crypto/ecdsa"
	"errors"
	"net/http"
)

// Client is the top-level Hyperliquid client. It exposes three handles:
//
//	c.Info   — read-only queries
//	c.Trade  — signed actions (requires a private key)
//	c.Stream — websocket subscriptions and POST requests
//
// Construct it with New and a list of options.
type Client struct {
	Info   *Info
	Trade  *Trader
	Stream *Stream
}

// New builds a Client configured by the supplied options. At minimum either
// WithMainnet, WithTestnet, or WithBaseURL must be specified for non-default
// behaviour. Signed actions on Trade require WithPrivateKey.
func New(opts ...Option) (*Client, error) {
	cfg := defaultClientConfig()
	for _, o := range opts {
		o(cfg)
	}

	api := newHTTPAPI(cfg.baseURL, cfg.httpClient)

	info := NewInfo(cfg.baseURL, true, cfg.meta, cfg.spotMeta, cfg.perpDexs, cfg.builderDex)

	c := &Client{Info: info}

	if cfg.privateKey != nil {
		c.Trade = &Trader{
			client:      api,
			privateKey:  cfg.privateKey,
			vault:       cfg.vault,
			accountAddr: cfg.account,
			dex:         cfg.builderDex,
			info:        info,
		}
	}

	if !cfg.skipStream {
		c.Stream = NewStream(cfg.baseURL)
		c.Stream.SetLogger(cfg.logger)
	}

	return c, nil
}

// clientConfig holds the options accumulated by New.
type clientConfig struct {
	baseURL    string
	httpClient *http.Client
	privateKey *ecdsa.PrivateKey
	account    string
	vault      string
	builderDex string
	meta       *Meta
	spotMeta   *SpotMeta
	perpDexs   *MixedArray
	skipStream bool
	logger     Logger
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		baseURL: MainnetAPIURL,
		logger:  nopLogger{},
	}
}

// ErrMissingPrivateKey is returned by Trader methods called on a Client
// constructed without WithPrivateKey.
var ErrMissingPrivateKey = errors.New("hyperliquid: trader requires WithPrivateKey")
