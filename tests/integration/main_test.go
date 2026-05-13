//go:build integration

// Package integration contains the network-dependent test suite for the
// hyperliquid Go SDK. Every test here is gated behind the `integration`
// build tag so the default `go test ./...` run never reaches the network.
//
// Run with:
//
//	go test -tags=integration -count=1 ./tests/integration/...
//
// The suite reads its configuration from a `.env` file resolved relative
// to the current directory, then walks up two parents — useful when the
// test binary runs from the repo root, from tests/, or from
// tests/integration/.
package integration

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"

	hl "github.com/Simon-Busch/hyperliquid-go"
)

// integrationConfig is the typed view of the .env variables used by the
// suite.
type integrationConfig struct {
	BaseURL        string
	PrivateKeyHex  string
	AccountAddr    string
	TestCoin       string
	TestSize       float64
	BuilderAddr    string
	BuilderFeeBps  int
	SkipTransfer   bool
	SkipWS         bool

	privateKey *ecdsa.PrivateKey
	signerAddr string
}

var (
	cfgOnce sync.Once
	cfg     *integrationConfig
	cfgErr  error
)

func loadConfig() (*integrationConfig, error) {
	cfgOnce.Do(func() {
		// Resolve .env from the current directory upward — works whether the
		// test binary runs from the repo root, tests/, or tests/integration/.
		// godotenv.Load with multiple args tries to load all of them and
		// bails on the first miss, so iterate manually and stop at the
		// first one that exists.
		for _, p := range []string{".env", "../.env", "../../.env"} {
			if err := godotenv.Load(p); err == nil {
				break
			}
		}

		pk := strings.TrimSpace(os.Getenv("HL_PRIVATE_KEY"))
		if pk == "" {
			cfgErr = fmt.Errorf("HL_PRIVATE_KEY is required")
			return
		}
		pk = strings.TrimPrefix(pk, "0x")
		priv, err := crypto.HexToECDSA(pk)
		if err != nil {
			cfgErr = fmt.Errorf("parse HL_PRIVATE_KEY: %w", err)
			return
		}
		signerAddr := strings.ToLower(crypto.PubkeyToAddress(priv.PublicKey).Hex())

		baseURL := strings.TrimSpace(os.Getenv("HL_BASE_URL"))
		if baseURL == "" {
			baseURL = hl.TestnetAPIURL
		}

		coin := strings.TrimSpace(os.Getenv("HL_TEST_COIN"))
		if coin == "" {
			coin = "ETH"
		}

		sizeStr := strings.TrimSpace(os.Getenv("HL_TEST_SIZE"))
		size := 0.01
		if sizeStr != "" {
			v, err := strconv.ParseFloat(sizeStr, 64)
			if err != nil {
				cfgErr = fmt.Errorf("parse HL_TEST_SIZE: %w", err)
				return
			}
			size = v
		}

		feeBps := 1
		if s := strings.TrimSpace(os.Getenv("HL_BUILDER_FEE_BPS")); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil {
				cfgErr = fmt.Errorf("parse HL_BUILDER_FEE_BPS: %w", err)
				return
			}
			feeBps = v
		}

		cfg = &integrationConfig{
			BaseURL:       baseURL,
			PrivateKeyHex: pk,
			AccountAddr:   strings.TrimSpace(os.Getenv("HL_ACCOUNT_ADDRESS")),
			TestCoin:      coin,
			TestSize:      size,
			BuilderAddr:   strings.TrimSpace(os.Getenv("HL_BUILDER_ADDR")),
			BuilderFeeBps: feeBps,
			SkipTransfer:  strings.EqualFold(os.Getenv("HL_SKIP_TRANSFER"), "true"),
			SkipWS:        strings.EqualFold(os.Getenv("HL_SKIP_WS"), "true"),
			privateKey:    priv,
			signerAddr:    signerAddr,
		}
		if cfg.AccountAddr == "" {
			cfg.AccountAddr = signerAddr
		}
	})
	return cfg, cfgErr
}

// TestMain logs configuration health once, then defers per-test
// skipping so `go test -list` can enumerate scenarios even when the
// suite is not configured to run.
func TestMain(m *testing.M) {
	if _, err := loadConfig(); err != nil {
		log.Printf("integration suite not configured: %v", err)
		log.Printf("Required env vars: HL_PRIVATE_KEY. Optional: HL_ACCOUNT_ADDRESS, HL_BASE_URL, HL_TEST_COIN, HL_TEST_SIZE, HL_BUILDER_ADDR, HL_BUILDER_FEE_BPS, HL_SKIP_TRANSFER, HL_SKIP_WS.")
	}
	os.Exit(m.Run())
}
