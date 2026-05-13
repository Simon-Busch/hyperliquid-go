package hyperliquid

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Trader is the signed-action surface. Construct it indirectly via New;
// direct construction is not part of the public API.
type Trader struct {
	client       *httpAPI
	privateKey   *ecdsa.PrivateKey
	vault        string
	accountAddr  string
	dex          string // For HIP-3 builder-deployed perps
	info         *Info
	expiresAfter *int64

	// userState caches the most recent UserState snapshot used for
	// position-aware validation. It is refreshed by validate() on every
	// placement attempt unless SkipValidation is in effect, and may be
	// refreshed explicitly via RefreshState.
	userState   *UserState
	userStateMu sync.RWMutex

	// Transfer exposes balance and vault transfer actions.
	Transfer *TransferGroup
	// SubAccount exposes sub-account management.
	SubAccount *SubAccountGroup
	// Stake exposes HYPE staking actions.
	Stake *StakeGroup
	// MultiSig exposes multi-sig conversion and execution helpers.
	MultiSig *MultiSigGroup
}

// effectiveAddr returns the address used for position-state lookups. It
// prefers accountAddr (agent flow), falls back to vault, then derives the
// address from the configured private key.
func (t *Trader) effectiveAddr() string {
	if t.accountAddr != "" {
		return t.accountAddr
	}
	if t.vault != "" {
		return t.vault
	}
	if t.privateKey == nil {
		return ""
	}
	return strings.ToLower(crypto.PubkeyToAddress(t.privateKey.PublicKey).Hex())
}

// RefreshState refreshes the cached UserState snapshot used by
// position-aware validation. The ctx parameter is reserved for future
// cancellation; the underlying HTTP call does not yet honour it.
func (t *Trader) RefreshState(ctx context.Context) error {
	_ = ctx
	addr := t.effectiveAddr()
	if addr == "" {
		return fmt.Errorf("hyperliquid: no address available for RefreshState")
	}
	state, err := t.info.UserState(addr)
	if err != nil {
		return err
	}
	t.userStateMu.Lock()
	t.userState = state
	t.userStateMu.Unlock()
	return nil
}

// cachedUserState returns a snapshot of the cached UserState pointer
// without re-fetching. Callers must treat the result as read-only.
func (t *Trader) cachedUserState() *UserState {
	t.userStateMu.RLock()
	defer t.userStateMu.RUnlock()
	return t.userState
}

// attachSubgroups initialises the Transfer/SubAccount/Stake/MultiSig
// subgroup fields. Called by hl.New after constructing the Trader.
func (t *Trader) attachSubgroups() {
	t.Transfer = &TransferGroup{t: t}
	t.SubAccount = &SubAccountGroup{t: t}
	t.Stake = &StakeGroup{t: t}
	t.MultiSig = &MultiSigGroup{t: t}
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
	return e.client.warmUp()
}
