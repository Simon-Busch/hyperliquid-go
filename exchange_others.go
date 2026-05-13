package hyperliquid

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func (e *Trader) UpdateLeverage(leverage int, name string, isCross bool) (*UserState, error) {
	action := UpdateLeverageAction{
		Type:     "updateLeverage",
		Asset:    e.info.NameToAsset(name),
		IsCross:  isCross,
		Leverage: leverage,
	}

	var result UserState
	if err := e.executeAction(action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DefaultResponse represents a simple API response with type "default"
type DefaultResponse struct {
	Type string `json:"type"`
}

func (e *Trader) UpdateIsolatedMargin(amount float64, name string) (*APIResponse[DefaultResponse], error) {
	action := UpdateIsolatedMarginAction{
		Type:  "updateIsolatedMargin",
		Asset: e.info.NameToAsset(name),
		IsBuy: true,
		Ntli:  int64(math.Round(amount * 1e6)),
	}
	var result APIResponse[DefaultResponse]
	if err := e.executeAction(action, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetExpiresAfter sets the expiration time for actions
// If expiresAfter is nil, actions will not have an expiration time
// If expiresAfter is set, actions will include this expiration timestamp
func (e *Trader) SetExpiresAfter(expiresAfter *int64) {
	e.expiresAfter = expiresAfter
}

// SlippagePrice calculates the slippage price for market orders
func (e *Trader) SlippagePrice(
	name string,
	isBuy bool,
	slippage float64,
	px *float64,
) (float64, error) {
	var price float64

	if px != nil {
		price = *px
	} else {
		// Get midprice
		mids, err := e.info.AllMids()
		if err != nil {
			return 0, err
		}
		if midPriceStr, ok := mids[name]; ok {
			price, err = strconv.ParseFloat(midPriceStr, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse midprice for %s: %w", name, err)
			}
		} else {
			return 0, fmt.Errorf("no midprice found for %s", name)
		}
	}

	// Apply slippage
	if isBuy {
		price = price * (1 + slippage)
	} else {
		price = price * (1 - slippage)
	}

	// Get asset info for proper price formatting
	asset := e.info.NameToAsset(name)
	szDecimals := e.info.assetToDecimal[asset]
	class := ClassifyAsset(asset)

	// Apply proper price formatting according to Hyperliquid docs:
	// - Up to 5 significant figures
	// - No more than MAX_DECIMALS - szDecimals decimal places
	// - MAX_DECIMALS is 6 for perps, 8 for spot, 3 for HIP-4 outcome markets
	price = formatPriceToTickSize(price, szDecimals, class)

	// Validate and adjust price to meet tick size requirements
	adjustedPrice, err := validateAndAdjustPrice(price, asset)
	if err != nil {
		return 0, fmt.Errorf("failed to validate price for tick size: %w", err)
	}

	return adjustedPrice, nil
}

// formatPriceToTickSize formats price according to Hyperliquid tick size rules.
// Two constraints apply, in order:
//  1. Up to 5 significant figures
//  2. No more than (MAX_DECIMALS - szDecimals) decimal places
//
// See: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
func formatPriceToTickSize(price float64, szDecimals int, class AssetClass) float64 {
	// Constraint 1: enforce 5 significant figures first.
	sigFigsRounded, err := roundToSignificantFigures(price, 5)
	if err != nil {
		return price
	}

	// Constraint 2: round to allowed decimal places.
	maxPriceDecimals := class.MaxPriceDecimals() - szDecimals
	if maxPriceDecimals < 0 {
		maxPriceDecimals = 0
	}
	multiplier := math.Pow(10, float64(maxPriceDecimals))
	return math.Round(sigFigsRounded*multiplier) / multiplier
}

// roundToTickSize rounds a price to the nearest tick size
func roundToTickSize(price, tickSize float64) float64 {
	return math.Round(price/tickSize) * tickSize
}

// getAssetTickSize returns the tick size for a specific asset
// This is a fallback function that provides hardcoded tick sizes
// The actual implementation should use getAssetTickSizeFromMetadata which calculates dynamically
func getAssetTickSize(assetID int) float64 {
	// Perp assets (0-9999) have different tick sizes based on price ranges
	if assetID < 10000 {
		// Common tick sizes from Hyperliquid docs and testing:
		switch assetID {
		case 0: // BTC
			return 0.1
		case 1: // ETH
			return 0.01
		case 2: // SOL
			return 0.01
		default:
			// For other assets, use a reasonable default
			// This should be replaced with dynamic calculation
			return 0.01 // Default to 0.01 for most perp assets
		}
	}

	// Spot assets (10000+) typically have smaller tick sizes
	return 0.0001 // Default to 0.0001 for spot assets
}

// validateAndAdjustPrice ensures the price meets tick size requirements
// Based on Hyperliquid docs: https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/tick-and-lot-size
func validateAndAdjustPrice(price float64, assetID int) (float64, error) {
	// This function is called from contexts where we don't have access to the Trader instance
	// So we use the fallback method for now
	tickSize := getAssetTickSize(assetID)

	// Round to nearest tick size first to handle floating-point precision issues
	adjustedPrice := roundToTickSize(price, tickSize)

	// Check if the adjusted price is significantly different from the original
	// Use a small epsilon to account for floating-point precision
	epsilon := tickSize * 0.0001 // Very small tolerance
	if math.Abs(adjustedPrice-price) > epsilon {
		// Log the adjustment for debugging
		fmt.Printf("WARNING: Price %.8f adjusted to %.8f to meet tick size %.8f for asset %d\n",
			price, adjustedPrice, tickSize, assetID)
	}

	return adjustedPrice, nil
}

// ScheduleCancel schedules cancellation of all open orders
func (e *Trader) ScheduleCancel(scheduleTime *int64) (*ScheduleCancelResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := ScheduleCancelAction{
		Type: "scheduleCancel",
		Time: scheduleTime,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result ScheduleCancelResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetReferrer sets a referral code
func (e *Trader) SetReferrer(code string) (*SetReferrerResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SetReferrerAction{
		Type: "setReferrer",
		Code: code,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address for referrer
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result SetReferrerResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateSubAccount creates a new sub-account
func (e *Trader) CreateSubAccount(name string) (*CreateSubAccountResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := CreateSubAccountAction{
		Type: "createSubAccount",
		Name: name,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address for sub-account creation
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result CreateSubAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UsdClassTransfer transfers between USD classes (perps <-> spot).
func (e *Trader) UsdClassTransfer(amount float64, toPerp bool) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	amountStr := formatUsdAmount(amount)
	if e.vault != "" {
		amountStr += " subaccount:" + e.vault
	}
	action := map[string]any{
		"type":   "usdClassTransfer",
		"amount": amountStr,
		"toPerp": toPerp,
		"nonce":  nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, usdClassTransferSignTypes,
		"HyperliquidTransaction:UsdClassTransfer", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubAccountTransfer transfers funds to/from sub-account
func (e *Trader) SubAccountTransfer(
	subAccountUser string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SubAccountTransferAction{
		Type:           "subAccountTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Usd:            usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// VaultUsdTransfer transfers to/from vault
func (e *Trader) VaultUsdTransfer(
	vaultAddress string,
	isDeposit bool,
	usd int,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := VaultUsdTransferAction{
		Type:         "vaultTransfer",
		VaultAddress: vaultAddress,
		IsDeposit:    isDeposit,
		Usd:          usd,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UsdTransfer sends USD to another address.
func (e *Trader) UsdTransfer(amount float64, destination string) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "usdSend",
		"destination": destination,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, usdSendSignTypes,
		"HyperliquidTransaction:UsdSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// SpotTransfer sends spot tokens to another address.
func (e *Trader) SpotTransfer(
	amount float64,
	destination, token string,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "spotSend",
		"destination": destination,
		"token":       token,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, spotTransferSignTypes,
		"HyperliquidTransaction:SpotSend", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// UseBigBlocks enables or disables big blocks
func (e *Trader) UseBigBlocks(enable bool) (*ApprovalResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := UseBigBlocksAction{
		Type:           "evmUserModify",
		UsingBigBlocks: enable,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		"", // No vault address
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result ApprovalResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PerpDexClassTransfer transfers tokens between perp dex classes
func (e *Trader) PerpDexClassTransfer(
	dex, token string,
	amount float64,
	toPerp bool,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := PerpDexClassTransferAction{
		Type:   "perpDexClassTransfer",
		Dex:    dex,
		Token:  token,
		Amount: amount,
		ToPerp: toPerp,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubAccountSpotTransfer transfers spot tokens to/from sub-account
func (e *Trader) SubAccountSpotTransfer(
	subAccountUser string,
	isDeposit bool,
	token string,
	amount float64,
) (*TransferResponse, error) {
	timestamp := time.Now().UnixMilli()

	action := SubAccountSpotTransferAction{
		Type:           "subAccountSpotTransfer",
		SubAccountUser: subAccountUser,
		IsDeposit:      isDeposit,
		Token:          token,
		Amount:         amount,
	}

	sig, err := SignL1Action(
		e.privateKey,
		action,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(action, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TokenDelegate delegates (or undelegates) HYPE stake.
func (e *Trader) TokenDelegate(
	validator string,
	wei int,
	isUndelegate bool,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":         "tokenDelegate",
		"validator":    validator,
		"wei":          int64(wei),
		"isUndelegate": isUndelegate,
		"nonce":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, tokenDelegateSignTypes,
		"HyperliquidTransaction:TokenDelegate", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// WithdrawFromBridge withdraws USDC to the configured destination on L1.
func (e *Trader) WithdrawFromBridge(
	amount float64,
	destination string,
) (*TransferResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":        "withdraw3",
		"destination": destination,
		"amount":      formatUsdAmount(amount),
		"time":        nonce,
	}
	var result TransferResponse
	if err := e.executeUserSignedAction(
		action, withdrawSignTypes,
		"HyperliquidTransaction:Withdraw", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveAgent generates a fresh agent key and approves it to act on the
// user's behalf. Returns the result and the agent private key hex.
func (e *Trader) ApproveAgent(name *string) (*AgentApprovalResponse, string, error) {
	agentBytes := make([]byte, 32)
	if _, err := rand.Read(agentBytes); err != nil {
		return nil, "", fmt.Errorf("generate agent key: %w", err)
	}
	agentKey := "0x" + hex.EncodeToString(agentBytes)
	pk, err := crypto.HexToECDSA(agentKey[2:])
	if err != nil {
		return nil, "", fmt.Errorf("parse agent key: %w", err)
	}
	agentAddress := strings.ToLower(crypto.PubkeyToAddress(pk.PublicKey).Hex())

	nonce := time.Now().UnixMilli()
	agentName := ""
	if name != nil {
		agentName = *name
	}
	action := map[string]any{
		"type":         "approveAgent",
		"agentAddress": agentAddress,
		"agentName":    agentName,
		"nonce":        nonce,
	}
	var result AgentApprovalResponse
	if err := e.executeUserSignedAction(
		action, approveAgentSignTypes,
		"HyperliquidTransaction:ApproveAgent", nonce, &result,
	); err != nil {
		return nil, "", err
	}
	return &result, agentKey, nil
}

// ApproveBuilderFee approves a builder address to charge up to maxFeeRate.
// maxFeeRate must be a percent string like "0.1%".
func (e *Trader) ApproveBuilderFee(builder string, maxFeeRate string) (*ApprovalResponse, error) {
	nonce := time.Now().UnixMilli()
	action := map[string]any{
		"type":       "approveBuilderFee",
		"maxFeeRate": maxFeeRate,
		"builder":    strings.ToLower(builder),
		"nonce":      nonce,
	}
	var result ApprovalResponse
	if err := e.executeUserSignedAction(
		action, approveBuilderFeeSignTypes,
		"HyperliquidTransaction:ApproveBuilderFee", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// ConvertToMultiSigUser converts the account to a multi-sig user with the
// given authorized signers and approval threshold.
func (e *Trader) ConvertToMultiSigUser(
	authorizedUsers []string,
	threshold int,
) (*MultiSigConversionResponse, error) {
	nonce := time.Now().UnixMilli()
	sort.Strings(authorizedUsers)
	signersJSON, err := json.Marshal(map[string]any{
		"authorizedUsers": authorizedUsers,
		"threshold":       threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal signers: %w", err)
	}
	action := map[string]any{
		"type":    "convertToMultiSigUser",
		"signers": string(signersJSON),
		"nonce":   nonce,
	}
	var result MultiSigConversionResponse
	if err := e.executeUserSignedAction(
		action, convertToMultiSigUserSignTypes,
		"HyperliquidTransaction:ConvertToMultiSigUser", nonce, &result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

// Spot Deploy Methods



func (e *Trader) MultiSig(
	action map[string]any,
	signers []string,
	signatures []string,
) (*MultiSigResponse, error) {
	timestamp := time.Now().UnixMilli()

	multiSigAction := map[string]any{
		"type":       "multiSig",
		"action":     action,
		"signers":    signers,
		"signatures": signatures,
	}

	sig, err := SignL1Action(
		e.privateKey,
		multiSigAction,
		e.vault,
		timestamp,
		e.expiresAfter,
		e.client.baseURL == MainnetAPIURL,
	)
	if err != nil {
		return nil, err
	}

	resp, err := e.postAction(multiSigAction, sig, timestamp)
	if err != nil {
		return nil, err
	}

	var result MultiSigResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
