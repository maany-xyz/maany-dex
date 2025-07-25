package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/neutron-org/neutron/v5/osmomath"
)

var (
	errNilEpochProvisions      = errors.New("epoch provisions was nil in genesis")
	errNegativeEpochProvisions = errors.New("epoch provisions should be non-negative")
)

// NewMinter returns a new Minter object with the given epoch
// provisions values.
func NewMinter(epochProvisions osmomath.Dec) Minter {
	return Minter{
		EpochProvisions: epochProvisions,
	}
}

// InitialMinter returns an initial Minter object.
func InitialMinter() Minter {
	return NewMinter(osmomath.NewDec(0))
}

// DefaultInitialMinter returns a default initial Minter object for a new chain.
func DefaultInitialMinter() Minter {
	return InitialMinter()
}

// Validate validates minter. Returns nil on success, error otherwise.
func (m Minter) Validate() error {
	if m.EpochProvisions.IsNil() {
		return errNilEpochProvisions
	}

	if m.EpochProvisions.IsNegative() {
		return errNegativeEpochProvisions
	}
	return nil
}

// NextEpochProvisions returns the epoch provisions.
func (m Minter) NextEpochProvisions(params Params) osmomath.Dec {
	return m.EpochProvisions.Mul(params.ReductionFactor)
}

// EpochProvision returns the provisions for a block based on the epoch
// provisions rate.
func (m Minter) EpochProvision(params Params) sdk.Coin {
	provisionAmt := m.EpochProvisions
	return sdk.NewCoin(params.MintDenom, provisionAmt.TruncateInt())
}

// GetInflationProvisions returns the inflation provisions.
// These are calculated as the current epoch provisions * (1 - developer rewards proportion)
// The returned denom is taken from input parameters.
func (m Minter) GetInflationProvisions(params Params) sdk.DecCoin {
	provisionAmt := m.EpochProvisions.Mul(params.GetInflationProportion())
	return sdk.NewDecCoinFromDec(params.MintDenom, provisionAmt)
}

// GetDeveloperVestingEpochProvisions returns the developer vesting provisions.
// These are calculated as the current epoch provisions * developer rewards proportion
// The returned denom is taken from input parameters.
func (m Minter) GetDeveloperVestingEpochProvisions(params Params) sdk.DecCoin {
	provisionAmt := m.EpochProvisions.Mul(params.GetDeveloperVestingProportion())
	return sdk.NewDecCoinFromDec(params.MintDenom, provisionAmt)
}
