package swapstrategy

import (
	"fmt"

	"github.com/neutron-org/neutron/v5/osmomath"
)

type spreadFactorOverOneMinusSpreadFactorGetter func() osmomath.Dec

// computeSpreadRewardChargePerSwapStepOutGivenIn returns the total spread factor charge per swap step given the parameters.
// Assumes swapping for token out given token in.
//
// - hasReachedTarget is the boolean flag indicating whether the sqrtPriceTarget has been reached during the swap step.
//   - the sqrtPriceTarget can be one of:
//   - sqrtPriceLimit
//   - nextTickSqrtPrice
//
// - amountIn the amount of token in to be consumed during the swap step
//
// - amountSpecifiedRemaining is the total remaining amount of token in that needs to be consumed to complete the swap.

// - spreadFactor the spread factor to be charged.
//
// If spread factor is negative, it panics.
// If spread factor is 0, returns 0. Otherwise, computes and returns the spread factor charge per step.
func computeSpreadRewardChargePerSwapStepOutGivenIn(hasReachedTarget bool, amountIn, amountSpecifiedRemaining, spreadFactor osmomath.Dec, SfOveroneMinSf spreadFactorOverOneMinusSpreadFactorGetter) osmomath.Dec {
	if spreadFactor.IsZero() {
		return osmomath.ZeroDec()
	} else if spreadFactor.IsNegative() {
		// This should never happen but is added as a defense-in-depth measure.
		panic(fmt.Errorf("spread factor must be non-negative, was (%s), if figure is more than -1 this panic is known and not an issue", spreadFactor))
	}

	var spreadRewardChargeTotal osmomath.Dec
	if hasReachedTarget {
		// This branch implies two options:
		// 1) either sqrtPriceNextTick is reached
		// 2) or sqrtPriceLimit is reached
		// In both cases, we charge the spread factor on the amount in actually consumed before
		// hitting the target.
		spreadRewardChargeTotal = computeSpreadRewardChargeFromAmountIn(amountIn, SfOveroneMinSf())
	} else {
		// Otherwise, the current tick had enough liquidity to fulfill the swap
		// and we ran out of amount remaining before reaching either the next tick or the limit.
		// As a result, the spread factor is the difference between
		// the amount needed to fulfill and the actual amount we ended up charging.
		spreadRewardChargeTotal = amountSpecifiedRemaining.Sub(amountIn)
	}

	if spreadRewardChargeTotal.IsNegative() {
		// This should never happen but is added as a defense-in-depth measure.
		panic(fmt.Errorf("spread factor charge must be non-negative, was (%s), if figure is more than -1 this panic is known and not an issue", spreadRewardChargeTotal))
	}

	return spreadRewardChargeTotal
}

// computeSpreadRewardChargeFromAmountIn returns the spread factor charge given the amount in and spread factor.
// Computes amountIn * spreadFactor / (1 - spreadFactor) where math operations round up
// at precision end. This is necessary to ensure that the spread factor charge is always
// rounded in favor of the pool.
// TODO: Change this fn to take in 1 - spreadFactor as it should already have been computed.
func computeSpreadRewardChargeFromAmountIn(amountIn osmomath.Dec, spreadFactorOverOneMinusSpreadFactor osmomath.Dec) osmomath.Dec {
	return amountIn.MulRoundUp(spreadFactorOverOneMinusSpreadFactor)
}
