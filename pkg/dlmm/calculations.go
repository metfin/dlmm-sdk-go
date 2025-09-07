package dlmm

import (
	"math/big"

	lb_clmm "github.com/metfin/dlmm-sdk-go/pkg/generated"
)

// CalculateBinAmounts calculates token amounts for a specific bin
func CalculateBinAmounts(bin *lb_clmm.Bin, share *big.Int, liqSupply *big.Int) (*big.Int, *big.Int) {
	if share.Sign() == 0 || liqSupply.Sign() == 0 {
		return big.NewInt(0), big.NewInt(0)
	}

	// amountX * share / liqSupply
	xPart := new(big.Int).SetUint64(bin.AmountX)
	xPart.Mul(xPart, share)
	xPart.Quo(xPart, liqSupply)

	yPart := new(big.Int).SetUint64(bin.AmountY)
	yPart.Mul(yPart, share)
	yPart.Quo(yPart, liqSupply)

	return xPart, yPart
}

// CalculateBinFees calculates claimable fees for a specific bin
func CalculateBinFees(bin *lb_clmm.Bin, feeInfo *lb_clmm.FeeInfo, share *big.Int, scaleSquared *big.Int) (*big.Int, *big.Int) {
	feeXBig := big.NewInt(0)
	feeYBig := big.NewInt(0)

	// Fees: compute delta of per-liquidity accumulators and apply liquidity share
	// deltaX = bin.FeeAmountXPerTokenStored - position.FeeXPerTokenComplete
	deltaX := new(big.Int).Sub(bin.FeeAmountXPerTokenStored.BigInt(), feeInfo.FeeXPerTokenComplete.BigInt())
	if deltaX.Sign() > 0 {
		// (deltaX * share) >> 128  because both are Q64.64
		fx := new(big.Int).Mul(deltaX, share)
		fx.Quo(fx, scaleSquared)
		feeXBig.Add(feeXBig, fx)
	}

	// deltaY
	deltaY := new(big.Int).Sub(bin.FeeAmountYPerTokenStored.BigInt(), feeInfo.FeeYPerTokenComplete.BigInt())
	if deltaY.Sign() > 0 {
		fy := new(big.Int).Mul(deltaY, share)
		fy.Quo(fy, scaleSquared)
		feeYBig.Add(feeYBig, fy)
	}

	// Add pending leftovers
	feeXBig.Add(feeXBig, new(big.Int).SetUint64(feeInfo.FeeXPending))
	feeYBig.Add(feeYBig, new(big.Int).SetUint64(feeInfo.FeeYPending))

	return feeXBig, feeYBig
}
