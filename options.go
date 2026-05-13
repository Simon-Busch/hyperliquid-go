package hyperliquid

import (
	"crypto/ecdsa"
	"net/http"
)

// Option configures the top-level Client. Pass options to New.
type Option func(*clientConfig)

// WithMainnet targets the production Hyperliquid endpoint.
func WithMainnet() Option {
	return func(c *clientConfig) { c.baseURL = MainnetAPIURL }
}

// WithTestnet targets the Hyperliquid testnet endpoint.
func WithTestnet() Option {
	return func(c *clientConfig) { c.baseURL = TestnetAPIURL }
}

// WithBaseURL targets the supplied REST endpoint. Use for custom or local
// development servers.
func WithBaseURL(url string) Option {
	return func(c *clientConfig) { c.baseURL = url }
}

// WithPrivateKey provides the ECDSA key used to sign all Trader actions.
// Without it, c.Trade will be nil.
func WithPrivateKey(pk *ecdsa.PrivateKey) Option {
	return func(c *clientConfig) { c.privateKey = pk }
}

// WithAccount sets the account address Trader actions are submitted on
// behalf of (used by the agent flow when the signer differs from the owner).
func WithAccount(addr string) Option {
	return func(c *clientConfig) { c.account = addr }
}

// WithVault sets the vault address Trader actions act on.
func WithVault(addr string) Option {
	return func(c *clientConfig) { c.vault = addr }
}

// WithBuilderDex pins the client to a HIP-3 builder-deployed perp dex.
func WithBuilderDex(dex string) Option {
	return func(c *clientConfig) { c.builderDex = dex }
}

// WithHTTPClient supplies a caller-owned *http.Client for full transport
// control (custom timeouts, connection pooling, proxies, etc.).
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = httpClient }
}

// WithMeta supplies pre-fetched metadata (perp meta, spot meta, perp dex
// list) so New does not need to round-trip to /info during construction.
func WithMeta(meta *Meta, spotMeta *SpotMeta, perpDexs *MixedArray) Option {
	return func(c *clientConfig) {
		c.meta = meta
		c.spotMeta = spotMeta
		c.perpDexs = perpDexs
	}
}

// WithSkipStream disables construction of the websocket Stream. Useful for
// REST-only callers that don't want a background goroutine.
func WithSkipStream(skip bool) Option {
	return func(c *clientConfig) { c.skipStream = skip }
}

// WithLogger plugs in a Logger. The default is a no-op logger.
func WithLogger(l Logger) Option {
	return func(c *clientConfig) {
		if l != nil {
			c.logger = l
		}
	}
}
