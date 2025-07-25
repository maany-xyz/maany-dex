package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/x/mint/types"
)

// TestGetInflationProportion sanity checks that inflation
// proportion equals to 1 - developer vesting proportion.
func TestGetInflationProportion(t *testing.T) {
	developerVestingProportion := osmomath.NewDecWithPrec(4, 1)
	expectedInflationProportion := osmomath.OneDec().Sub(developerVestingProportion)

	params := types.Params{
		DistributionProportions: types.DistributionProportions{
			DeveloperRewards: developerVestingProportion,
		},
	}

	actualInflationProportion := params.GetInflationProportion()
	require.Equal(t, expectedInflationProportion, actualInflationProportion)
}

// TestGetDeveloperVestingProportion sanity checks that developer
// vesting proportion equals to the value set by
// parameter for dev rewards.
func TestGetDeveloperVestingProportion(t *testing.T) {
	expectedDevVestingProportion := osmomath.NewDecWithPrec(4, 1)

	params := types.Params{
		DistributionProportions: types.DistributionProportions{
			DeveloperRewards: expectedDevVestingProportion,
		},
	}

	actualDevVestingProportion := params.GetDeveloperVestingProportion()
	require.Equal(t, expectedDevVestingProportion, actualDevVestingProportion)
}
