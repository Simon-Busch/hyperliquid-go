package hyperliquid

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

// Cached EIP-712 domain separator — identical for every Hyperliquid signature.
// Computed once to avoid repeated ABI encoding + Keccak256 on every sign.
var cachedDomainSeparator []byte

// Shared EIP-712 types (never changes)
var eip712Types = apitypes.Types{
	"Agent": []apitypes.Type{
		{Name: "source", Type: "string"},
		{Name: "connectionId", Type: "bytes32"},
	},
	"EIP712Domain": []apitypes.Type{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	},
}

var eip712Domain apitypes.TypedDataDomain

func init() {
	chainId := math.HexOrDecimal256(*big.NewInt(1337))
	eip712Domain = apitypes.TypedDataDomain{
		ChainId:           &chainId,
		Name:              "Exchange",
		Version:           "1",
		VerifyingContract: "0x0000000000000000000000000000000000000000",
	}

	td := apitypes.TypedData{
		Domain:      eip712Domain,
		Types:       eip712Types,
		PrimaryType: "Agent",
		Message:     map[string]any{"source": "a", "connectionId": "0x" + strings.Repeat("00", 32)},
	}
	domainSep, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		panic(fmt.Sprintf("failed to compute domain separator: %v", err))
	}
	cachedDomainSeparator = domainSep
}

// convertStr16ToStr8 is currently a no-op: vmihailenco/msgpack v5 with
// UseCompactInts already emits str8 for strings 32–255 bytes, matching
// Python's eth_account msgpack output. Kept as a hook in case a future
// dependency upgrade reintroduces the str16 mismatch.
func convertStr16ToStr8(data []byte) []byte {
	return data
}

// userSignedChainID is the wallet-side chainId Hyperliquid prescribes for
// user-signed actions ("0x66eee" = 421614, Arbitrum Sepolia). The same value
// is used on mainnet and testnet — hyperliquidChain distinguishes them.
const userSignedChainIDHex = "0x66eee"

var userSignedChainID = func() *big.Int {
	v, _ := new(big.Int).SetString(strings.TrimPrefix(userSignedChainIDHex, "0x"), 16)
	return v
}()

// hashStructLenient is HashStruct that silently drops message fields not
// declared in payloadTypes. Mirrors Python eth_account's behavior where
// extra keys like "type" / "signatureChainId" appear in the JSON sent to
// /exchange but are not part of the EIP-712 typed-data schema.
//
// Also normalizes uint64 fields to *big.Int, which apitypes.HashStruct
// requires (it cannot consume native uint64 from a map[string]any).
func hashStructLenient(td apitypes.TypedData, primaryType string, msg map[string]any) ([]byte, error) {
	types := td.Types[primaryType]
	filtered := make(map[string]any, len(types))
	for _, t := range types {
		v, ok := msg[t.Name]
		if !ok {
			return nil, fmt.Errorf("missing field %q for primary type %q", t.Name, primaryType)
		}
		if t.Type == "uint64" {
			n, err := toBigUint64(v, t.Name)
			if err != nil {
				return nil, err
			}
			filtered[t.Name] = n
			continue
		}
		filtered[t.Name] = v
	}
	return td.HashStruct(primaryType, filtered)
}

func toBigUint64(v any, fieldName string) (*big.Int, error) {
	switch x := v.(type) {
	case *big.Int:
		return x, nil
	case uint64:
		return new(big.Int).SetUint64(x), nil
	case int64:
		if x < 0 {
			return nil, fmt.Errorf("%s: negative int64 %d", fieldName, x)
		}
		return new(big.Int).SetUint64(uint64(x)), nil
	case int:
		if x < 0 {
			return nil, fmt.Errorf("%s: negative int %d", fieldName, x)
		}
		return new(big.Int).SetUint64(uint64(x)), nil
	case float64:
		// JSON unmarshal turns numbers into float64. Reject if non-integral.
		if x < 0 || x > 1<<63 || float64(uint64(x)) != x {
			return nil, fmt.Errorf("%s: float64 %g not a valid uint64", fieldName, x)
		}
		return new(big.Int).SetUint64(uint64(x)), nil
	case string:
		n, ok := new(big.Int).SetString(x, 10)
		if !ok {
			return nil, fmt.Errorf("%s: cannot parse string %q as uint64", fieldName, x)
		}
		return n, nil
	}
	return nil, fmt.Errorf("%s: unsupported type %T for uint64", fieldName, v)
}

// SignUserSignedAction signs an EIP-712 user-signed action using the
// "HyperliquidSignTransaction" domain (chainId 421614). Unlike L1 actions,
// these are not msgpack-hashed; the action map is hashed directly per its
// EIP-712 typed-data schema. Caller passes the per-action payloadTypes and
// primary type name (e.g. "HyperliquidTransaction:UsdClassTransfer").
//
// Mutates action by adding hyperliquidChain + signatureChainId so the JSON
// sent to /exchange matches what was signed (Python SDK does the same).
func SignUserSignedAction(
	privateKey *ecdsa.PrivateKey,
	action map[string]any,
	payloadTypes []apitypes.Type,
	primaryType string,
	isMainnet bool,
) (SignatureResult, error) {
	if isMainnet {
		action["hyperliquidChain"] = "Mainnet"
	} else {
		action["hyperliquidChain"] = "Testnet"
	}
	action["signatureChainId"] = userSignedChainIDHex

	chainID := math.HexOrDecimal256(*userSignedChainID)
	td := apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			ChainId:           &chainID,
			Name:              "HyperliquidSignTransaction",
			Version:           "1",
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Types: apitypes.Types{
			primaryType: payloadTypes,
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: primaryType,
		Message:     action,
	}

	domainSep, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return SignatureResult{}, fmt.Errorf("hash domain: %w", err)
	}
	msgHash, err := hashStructLenient(td, primaryType, action)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("hash message: %w", err)
	}

	var raw [66]byte
	raw[0] = 0x19
	raw[1] = 0x01
	copy(raw[2:34], domainSep)
	copy(raw[34:66], msgHash)
	digest := crypto.Keccak256Hash(raw[:])

	sig, err := crypto.Sign(digest.Bytes(), privateKey)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("sign: %w", err)
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	return SignatureResult{
		R: hexutil.EncodeBig(r),
		S: hexutil.EncodeBig(s),
		V: int(sig[64]) + 27,
	}, nil
}

// addressToBytes converts a hex address to bytes, matching Python's address_to_bytes
func addressToBytes(address string) []byte {
	address = strings.TrimPrefix(address, "0x")
	bytes, _ := hex.DecodeString(address)
	return bytes
}

// actionHash implements the same logic as Python's action_hash function
func actionHash(action any, vaultAddress string, nonce int64, expiresAfter *int64) []byte {
	// Pack action using msgpack (like Python's msgpack.packb)
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.SetSortMapKeys(true)
	enc.UseCompactInts(true)
	
	err := enc.Encode(action)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal action: %v", err))
	}
	data := convertStr16ToStr8(buf.Bytes())

	// Add nonce as 8 bytes big endian
	if nonce < 0 {
		panic(fmt.Sprintf("nonce cannot be negative: %d", nonce))
	}
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	data = append(data, nonceBytes...)

	// Add vault address
	if vaultAddress == "" {
		data = append(data, 0x00)
	} else {
		data = append(data, 0x01)
		data = append(data, addressToBytes(vaultAddress)...)
	}

	// Add expires_after if provided
	if expiresAfter != nil {
		if *expiresAfter < 0 {
			panic(fmt.Sprintf("expiresAfter cannot be negative: %d", *expiresAfter))
		}
		data = append(data, 0x00)
		expiresAfterBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(expiresAfterBytes, uint64(*expiresAfter))
		data = append(data, expiresAfterBytes...)
	}

	// Return keccak256 hash
	return crypto.Keccak256(data)
}

// constructPhantomAgent implements the same logic as Python's construct_phantom_agent
func constructPhantomAgent(hash []byte, isMainnet bool) map[string]any {
	source := "b" // testnet
	if isMainnet {
		source = "a" // mainnet
	}
	return map[string]any{
		"source":       source,
		"connectionId": "0x" + hex.EncodeToString(hash),
	}
}

// l1Payload implements the same logic as Python's l1_payload
func l1Payload(phantomAgent map[string]any) apitypes.TypedData {
	return apitypes.TypedData{
		Domain:      eip712Domain,
		Types:       eip712Types,
		PrimaryType: "Agent",
		Message:     phantomAgent,
	}
}

// SignatureResult represents the structured signature result
type SignatureResult struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// signInner implements the same logic as Python's sign_inner
func signInner(
	privateKey *ecdsa.PrivateKey,
	typedData apitypes.TypedData,
) (SignatureResult, error) {
	// Message hash (only part that changes per sign)
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to hash typed data: %w", err)
	}

	// EIP-712: 0x19 0x01 || domainSeparator || messageHash
	var rawData [66]byte // 2 + 32 + 32, stack-allocated
	rawData[0] = 0x19
	rawData[1] = 0x01
	copy(rawData[2:34], cachedDomainSeparator)
	copy(rawData[34:66], typedDataHash)
	msgHash := crypto.Keccak256Hash(rawData[:])

	signature, err := crypto.Sign(msgHash.Bytes(), privateKey)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to sign message: %w", err)
	}

	// Extract r, s, v components
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	v := int(signature[64]) + 27

	return SignatureResult{
		R: hexutil.EncodeBig(r),
		S: hexutil.EncodeBig(s),
		V: v,
	}, nil
}

// SignL1Action implements the same logic as Python's sign_l1_action
func SignL1Action(
	privateKey *ecdsa.PrivateKey,
	action any,
	vaultAddress string,
	timestamp int64,
	expiresAfter *int64,
	isMainnet bool,
) (SignatureResult, error) {
	// Step 1: Create action hash
	hash := actionHash(action, vaultAddress, timestamp, expiresAfter)

	// Step 2: Construct phantom agent
	phantomAgent := constructPhantomAgent(hash, isMainnet)

	// Step 3: Create l1 payload
	typedData := l1Payload(phantomAgent)

	// Step 4: Sign using EIP-712
	return signInner(privateKey, typedData)
}

// User-signed action payload schemas, copied from the official Python SDK
// (hyperliquid/utils/signing.py). The order of fields must match Python.

var (
	usdSendSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	spotTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	withdrawSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}
	usdClassTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}
	tokenDelegateSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "validator", Type: "address"},
		{Name: "wei", Type: "uint64"},
		{Name: "isUndelegate", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}
	convertToMultiSigUserSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	approveAgentSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
	approveBuilderFeeSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}
)

// Utility function to convert float to USD integer representation
func FloatToUsdInt(value float64) int {
	// Convert float USD to integer representation (assuming 6 decimals for USDC)
	return int(value * 1e6)
}

// GetTimestampMs returns current timestamp in milliseconds
func GetTimestampMs() int64 {
	return time.Now().UnixMilli()
}
