package hyperliquid

import (
	"testing"
)

func TestClassifyAsset(t *testing.T) {
	cases := []struct {
		name  string
		asset int
		want  AssetClass
	}{
		// Default perp range: 0..9_999
		{"perp BTC", 0, AssetClassPerp},
		{"perp ETH", 1, AssetClassPerp},
		{"perp upper boundary", 9999, AssetClassPerp},

		// Spot range: 10_000..99_999
		{"spot lower boundary", 10000, AssetClassSpot},
		{"spot PURR/USDC", 10000, AssetClassSpot},
		{"spot mid range", 50000, AssetClassSpot},
		{"spot upper boundary", 99999, AssetClassSpot},

		// Builder perp range: 100_000..99_999_999 (HIP-3)
		{"builder perp lower boundary", 100000, AssetClassBuilderPerp},
		// dex 1, idx 0 → 100000 + 1*10000 + 0 = 110000
		{"builder perp dex1 idx0", 110000, AssetClassBuilderPerp},
		// dex 1, idx 5 → 110005 (the previously-misclassified case)
		{"builder perp dex1 idx5", 110005, AssetClassBuilderPerp},
		{"builder perp upper boundary", 99999999, AssetClassBuilderPerp},

		// Outcome range: 100_000_000+ (HIP-4)
		{"outcome lower boundary", 100000000, AssetClassOutcome},
		// outcome 123, side 0 → 100_000_000 + 1230 = 100_001_230
		{"outcome 123 yes", 100001230, AssetClassOutcome},
		// outcome 123, side 1 → 100_000_000 + 1231 = 100_001_231
		{"outcome 123 no", 100001231, AssetClassOutcome},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyAsset(tc.asset)
			if got != tc.want {
				t.Errorf("ClassifyAsset(%d) = %v, want %v", tc.asset, got, tc.want)
			}
		})
	}
}

func TestMaxPriceDecimals(t *testing.T) {
	cases := []struct {
		class AssetClass
		want  int
	}{
		{AssetClassPerp, 6},
		{AssetClassBuilderPerp, 6},
		{AssetClassSpot, 8},
		{AssetClassOutcome, 6}, // empirically: 5 decimal max prices with szDecimals=1
	}

	for _, tc := range cases {
		got := tc.class.MaxPriceDecimals()
		if got != tc.want {
			t.Errorf("(%v).MaxPriceDecimals() = %d, want %d", tc.class, got, tc.want)
		}
	}
}

// TestHIP3MisclassificationFix is the regression test for the pre-existing bug
// where `isSpot := asset >= 10000` treated HIP-3 builder perps (>=100_000) as
// spot, giving them spot's 8-decimal max instead of perp's 6-decimal max.
func TestHIP3MisclassificationFix(t *testing.T) {
	hip3Asset := 110005 // dex 1, idx 5
	class := ClassifyAsset(hip3Asset)
	if class == AssetClassSpot {
		t.Fatalf("HIP-3 builder perp asset %d misclassified as spot", hip3Asset)
	}
	if class != AssetClassBuilderPerp {
		t.Fatalf("HIP-3 builder perp asset %d classified as %v, want AssetClassBuilderPerp", hip3Asset, class)
	}
	if got := class.MaxPriceDecimals(); got != 6 {
		t.Errorf("HIP-3 builder perp MaxPriceDecimals = %d, want 6 (was incorrectly 8 before fix)", got)
	}
}

func TestIsSpotLike(t *testing.T) {
	if !AssetClassSpot.IsSpotLike() {
		t.Error("AssetClassSpot.IsSpotLike() = false, want true")
	}
	for _, c := range []AssetClass{AssetClassPerp, AssetClassBuilderPerp, AssetClassOutcome} {
		if c.IsSpotLike() {
			t.Errorf("(%v).IsSpotLike() = true, want false", c)
		}
	}
}

// TestPriceToWireOutcomeDecimals verifies that an outcome asset with
// szDecimals=0 produces up to 6 decimal price formatting (capped at 5
// significant figures, which is the binding constraint for prices < 1).
// szDecimals=0 is empirically confirmed: the exchange rejected size=16.4
// with "Order has invalid size" but accepted size=16.
func TestPriceToWireOutcomeDecimals(t *testing.T) {
	outcomeAsset := outcomeAssetBase + 40 // outcome 4, YES (the live mainnet market)
	info := &Info{
		coinToAsset:    map[string]int{"#40": outcomeAsset},
		nameToCoin:     map[string]string{"#40": "#40"},
		assetToDecimal: map[int]int{outcomeAsset: 0},
	}

	cases := []struct {
		price float64
		want  string
	}{
		// 5 sig figs cap is the binding constraint for prices in (0, 1).
		// Values pulled from a real mainnet snapshot of asset #40.
		{0.66782, "0.66782"},
		{0.6689, "0.6689"},
		{0.5, "0.5"},
		{0.001, "0.001"}, // minimum
		{0.999, "0.999"}, // maximum (exchange bounds)
		// 5 sf of 0.7501234 → 0.75012
		{0.7501234, "0.75012"},
	}
	for _, tc := range cases {
		got, err := PriceToWire(tc.price, outcomeAsset, info, AssetClassOutcome)
		if err != nil {
			t.Errorf("PriceToWire(%f) error: %v", tc.price, err)
			continue
		}
		if got != tc.want {
			t.Errorf("PriceToWire(%f, outcome) = %q, want %q", tc.price, got, tc.want)
		}
	}
}

// TestFormatPriceToTickSize verifies the function applies the two Hyperliquid
// price constraints in order: (1) 5 significant figures, (2) max decimal
// places = MAX_DECIMALS - szDecimals.
func TestFormatPriceToTickSize(t *testing.T) {
	cases := []struct {
		name       string
		price      float64
		szDecimals int
		class      AssetClass
		want       float64
	}{
		// Perp/builder-perp: maxPriceDecimals = 6 - szDecimals
		// 5 sf of 50000.12345 → 50000; allowed=4 → 50000
		{"perp 50000.12345 sz2", 50000.12345, 2, AssetClassPerp, 50000},
		// 5 sf of 2500.123 → 2500.1; allowed=4 → 2500.1
		{"perp 2500.123 sz2", 2500.123, 2, AssetClassPerp, 2500.1},
		// 5 sf of 100.001 → 100.00; allowed=4 → 100
		{"perp 100.001 sz2", 100.001, 2, AssetClassPerp, 100},
		// Builder perp must behave identically to perp (the HIP-3 fix invariant)
		{"builder perp 50000.123 sz2", 50000.123, 2, AssetClassBuilderPerp, 50000},
		{"builder perp 100.001 sz2", 100.001, 2, AssetClassBuilderPerp, 100},
		// Spot: maxPriceDecimals = 8 - szDecimals
		// Previously zeroed-out due to broken sig-fig branch — now correct.
		// 5 sf of 0.123456789 → 0.12346; allowed=6 → 0.12346
		{"spot 0.123456789 sz2", 0.123456789, 2, AssetClassSpot, 0.12346},
		// 5 sf of 1.23456 → 1.2346; allowed=6 → 1.2346
		{"spot 1.23456 sz2", 1.23456, 2, AssetClassSpot, 1.2346},
		// Outcome (HIP-4): MaxPriceDecimals = 6, szDecimals = 0 (integer
		// contracts). 5 sig figs is the binding cap for prices in (0, 1).
		// Values pulled from the live mainnet binary BTC market.
		{"outcome 0.66782 sz0", 0.66782, 0, AssetClassOutcome, 0.66782},
		{"outcome 0.6689 sz0", 0.6689, 0, AssetClassOutcome, 0.6689},
		{"outcome 0.001 min sz0", 0.001, 0, AssetClassOutcome, 0.001},
		{"outcome 0.5 sz0", 0.5, 0, AssetClassOutcome, 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatPriceToTickSize(tc.price, tc.szDecimals, tc.class)
			// Compare with epsilon since we're dealing with floats
			if abs(got-tc.want) > 1e-9 {
				t.Errorf("formatPriceToTickSize(%f, sz=%d, class=%v) = %v, want %v",
					tc.price, tc.szDecimals, tc.class, got, tc.want)
			}
		})
	}
}

// TestPriceToWireHIP3BugFix — the actual production-code regression test
// for the HIP-3 fix. With szDecimals=2, a builder-perp asset should now
// allow 4 price decimals (6-2), not 6 (8-2). Verifies via observable
// wire output, not enum lookup.
func TestPriceToWireHIP3BugFix(t *testing.T) {
	hip3Asset := 110005 // dex 1, idx 5; previously misclassified as spot
	info := &Info{
		assetToDecimal: map[int]int{hip3Asset: 2},
	}
	// 0.123456: with old (buggy) spot rules, allowed=6 → "0.12346"
	//           with new builder-perp rules, allowed=4 → "0.1235" (5 sf cap)
	got, err := PriceToWire(0.123456, hip3Asset, info, AssetClassBuilderPerp)
	if err != nil {
		t.Fatalf("PriceToWire error: %v", err)
	}
	// 5 sf of 0.123456 → 0.12346; allowed=4 → "0.1235"
	if got != "0.1235" {
		t.Errorf("HIP-3 builder perp PriceToWire(0.123456) = %q, want %q "+
			"(if got %q, the HIP-3 fix is broken — that was the old buggy spot behavior)",
			got, "0.1235", "0.12346")
	}
}

// TestPriceToWireBackwardsCompatPerpSpot ensures the signature change didn't
// regress perp/spot wire formatting.
func TestPriceToWireBackwardsCompatPerpSpot(t *testing.T) {
	// Perp BTC: szDecimals=5 → maxDecimals(perp)=6 → allowed=1
	perpAsset := 0
	info := &Info{
		assetToDecimal: map[int]int{perpAsset: 5},
	}
	got, err := PriceToWire(50000.123, perpAsset, info, AssetClassPerp)
	if err != nil {
		t.Fatalf("perp PriceToWire error: %v", err)
	}
	// 5 sig figs of 50000.123 → 50000; allowed=1 → "50000"
	if got != "50000" {
		t.Errorf("perp PriceToWire = %q, want %q", got, "50000")
	}

	// Spot: szDecimals=2 → maxDecimals(spot)=8 → allowed=6
	spotAsset := 10000
	info2 := &Info{
		assetToDecimal: map[int]int{spotAsset: 2},
	}
	got, err = PriceToWire(0.12345678, spotAsset, info2, AssetClassSpot)
	if err != nil {
		t.Fatalf("spot PriceToWire error: %v", err)
	}
	// 5 sig figs of 0.12345678 → 0.12346; allowed=6 → "0.12346"
	if got != "0.12346" {
		t.Errorf("spot PriceToWire = %q, want %q", got, "0.12346")
	}
}
