package types

import (
	fmt "fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/neutron-org/neutron/v5/osmomath"
)

// Parameter store keys.
var (
	KeyMinimumRiskFactor     = []byte("MinimumRiskFactor")
	defaultMinimumRiskFactor = osmomath.NewDecWithPrec(5, 1) // 50%
)

// ParamTable for minting module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(minimumRiskFactor osmomath.Dec) Params {
	return Params{
		MinimumRiskFactor: minimumRiskFactor,
	}
}

// default minting module parameters.
func DefaultParams() Params {
	return Params{
		MinimumRiskFactor: defaultMinimumRiskFactor, // 5%
	}
}

// validate params.
func (p Params) Validate() error {
	return nil
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinimumRiskFactor, &p.MinimumRiskFactor, ValidateMinimumRiskFactor),
	}
}

func ValidateMinimumRiskFactor(i interface{}) error {
	v, ok := i.(osmomath.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() || v.GT(osmomath.NewDec(100)) {
		return fmt.Errorf("minimum risk factor should be between 0 - 100: %s", v.String())
	}

	return nil
}

func ValidateUnbondingDuration(i interface{}) error {
	v, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("unbonding duration should be positive: %s", v.String())
	}

	return nil
}
