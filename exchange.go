package hyperliquid

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type Trader struct {
	client       *Client
	privateKey   *ecdsa.PrivateKey
	vault        string
	accountAddr  string
	dex          string // For HIP-3 builder-deployed perps
	info         *Info
	expiresAfter *int64
}

// NewTrader creates a new Trader instance.
// perpDexs is optional - pass nil for the default perp dex.
// perpDexName is optional - set to empty string for the default perp dex,
// or provide a builder dex name (e.g., "flx") for HIP-3 builder-deployed perps.
func NewTrader(
	privateKey *ecdsa.PrivateKey,
	baseURL string,
	meta *Meta,
	vaultAddr, accountAddr string,
	spotMeta *SpotMeta,
	perpDexs *MixedArray,
	perpDexName string,
) *Trader {
	return &Trader{
		client:      NewClient(baseURL),
		privateKey:  privateKey,
		vault:       vaultAddr,
		accountAddr: accountAddr,
		dex:         perpDexName,
		info:        NewInfo(baseURL, true, meta, spotMeta, perpDexs, perpDexName),
	}
}

// PerpDex returns the configured builder perp dex name (e.g. "flx"), or empty string for default dex.
func (e *Trader) PerpDex() string {
	return e.dex
}

// executeAction executes an action and unmarshals the response into the given result
func (e *Trader) executeAction(action any, result any) error {
	timestamp := time.Now().UnixMilli()

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resp, result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// userSignedActionTypes are user-signed (EIP-712) actions for which
// /exchange expects vaultAddress: null in the envelope.
var userSignedActionTypes = map[string]bool{
	"usdClassTransfer":      true,
	"usdSend":               true,
	"spotSend":              true,
	"withdraw3":             true,
	"approveAgent":          true,
	"approveBuilderFee":     true,
	"tokenDelegate":         true,
	"convertToMultiSigUser": true,
}

func actionTypeOf(action any) string {
	if m, ok := action.(map[string]any); ok {
		if t, _ := m["type"].(string); t != "" {
			return t
		}
	}
	b, err := json.Marshal(action)
	if err != nil {
		return ""
	}
	var peek struct{ Type string }
	_ = json.Unmarshal(b, &peek)
	return peek.Type
}

func (e *Trader) postAction(
	action any,
	signature SignatureResult,
	nonce int64,
) ([]byte, error) {
	payload := map[string]any{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}
	if userSignedActionTypes[actionTypeOf(action)] {
		payload["vaultAddress"] = nil
	} else if e.vault != "" {
		payload["vaultAddress"] = e.vault
	}
	return e.client.post("/exchange", payload)
}

// executeUserSignedAction signs a user-signed action with the proper
// HyperliquidSignTransaction EIP-712 domain and POSTs to /exchange.
func (e *Trader) executeUserSignedAction(
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	nonce int64,
	result any,
) error {
	sig, err := SignUserSignedAction(
		e.privateKey, action, payloadTypes, primaryType,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return err
	}
	resp, err := e.postAction(action, sig, nonce)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// GetAccountAddr returns the account address
func (e *Trader) GetAccountAddr() string {
	return e.accountAddr
}

// GetInfo returns the info instance
func (e *Trader) GetInfo() *Info {
	return e.info
}

// WarmUp pre-establishes the HTTP/2 connection so the first order doesn't pay
// the cold-start penalty (TCP + TLS + ALPN). Call once after creating the Trader.
func (e *Trader) WarmUp() error {
	return e.client.WarmUp()
}
