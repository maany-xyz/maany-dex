package types

import (
	"github.com/neutron-org/neutron/v5/osmomath"
)

// ConsensusMinFee is a governance set parameter from prop 354 (https://www.mintscan.io/osmosis/proposals/354)
// Its intended to be .0025 uosmo / gas
var ConsensusMinFee osmomath.Dec = osmomath.NewDecWithPrec(25, 4)
