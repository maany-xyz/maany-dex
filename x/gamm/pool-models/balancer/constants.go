package balancer

import (
	"github.com/neutron-org/neutron/v5/osmomath"
)

var (
	// Pool creators can specify a weight in [1, MaxUserSpecifiedWeight)
	// for every token in the balancer pool.
	//
	// The weight used in the balancer equation is then creator-specified-weight * GuaranteedWeightPrecision.
	// This is done so that LBP's / smooth weight changes can actually happen smoothly,
	// without complex precision loss / edge effects.
	MaxUserSpecifiedWeight osmomath.Int = osmomath.NewIntFromUint64(1 << 20)
	// Scaling factor for every weight. The pool weight is:
	// weight_in_MsgCreateBalancerPool * GuaranteedWeightPrecision
	//
	// This is done so that smooth weight changes have enough precision to actually be smooth.
	GuaranteedWeightPrecision int64 = 1 << 30

	PoolTypeName string = "Balancer"
	oneDec              = osmomath.OneDec()
)
