package hyperliquid

import (
	"github.com/Simon-Busch/hyperliquid-go/internal/wire"
)

// abs returns the absolute value of x. Retained as a tiny adapter used by
// root-package tests; production callers use the wire package directly.
func abs(x float64) float64 {
	return wire.Abs(x)
}
