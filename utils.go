package hyperliquid

import (
	"github.com/Simon-Busch/hyperliquid-go/internal/wire"
)

// roundToDecimals rounds value to the given number of decimal places.
func roundToDecimals(value float64, decimals int) float64 {
	return wire.RoundToDecimals(value, decimals)
}

// parseFloat parses s as float64; returns 0 on parse failure.
func parseFloat(s string) float64 {
	return wire.ParseFloat(s)
}

// abs returns the absolute value of x.
func abs(x float64) float64 {
	return wire.Abs(x)
}

// formatFloat formats f with six decimal places.
func formatFloat(f float64) string {
	return wire.FormatFloat(f)
}

// formatUsdAmount renders an amount the way the Python SDK does for
// user-signed actions.
func formatUsdAmount(f float64) string {
	return wire.FormatUsdAmount(f)
}

// sizeToWire converts x to a wire-compatible string at the default
// three-decimal lot precision.
func sizeToWire(x float64) (string, error) {
	return wire.SizeToWire(x)
}

// sizeToWireWithAsset converts x to a wire-compatible string using the
// asset-specific size-decimals precision.
func sizeToWireWithAsset(x float64, asset int, info *Info) (string, error) {
	szDecimals, exists := info.AssetToDecimalMap()[asset]
	if !exists {
		return wire.SizeToWire(x)
	}
	return wire.SizeToWireDecimals(x, szDecimals)
}

// PriceToWire converts x to a wire-compatible price string for the supplied
// asset and asset class, applying Hyperliquid's five-significant-figure cap
// and the MaxPriceDecimals(class) - szDecimals decimal-place limit.
func PriceToWire(x float64, asset int, info *Info, class AssetClass) (string, error) {
	szDecimals, exists := info.AssetToDecimalMap()[asset]
	if !exists {
		return wire.FloatToWire(x)
	}
	allowed := class.MaxPriceDecimals() - szDecimals
	return wire.PriceToWireDecimals(x, allowed)
}

// floatToWire converts x to a wire-compatible string with up to eight
// decimal places, returning an error if rounding would change the value.
func floatToWire(x float64) (string, error) {
	return wire.FloatToWire(x)
}

// roundToSignificantFigures rounds x to n significant figures.
func roundToSignificantFigures(x float64, n int) (float64, error) {
	return wire.RoundToSignificantFigures(x, n)
}
