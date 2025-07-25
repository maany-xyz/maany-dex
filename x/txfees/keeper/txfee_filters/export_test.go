package txfee_filters

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	gammtypes "github.com/neutron-org/neutron/v5/x/gamm/types"
)

func IsArbTxLooseAuthz(msg sdk.Msg, swapInDenom string, lpTypesSeen map[gammtypes.LiquidityChangeType]bool) (string, bool) {
	return isArbTxLooseAuthz(msg, swapInDenom, lpTypesSeen)
}
