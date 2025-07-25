package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils"
)

var MaxPoolId uint64 = 99_999_999_999

// PoolI defines an interface for pools that hold tokens.
type PoolI interface {
	proto.Message

	GetAddress() sdk.AccAddress
	String() string
	GetId() uint64
	// GetSpreadFactor returns the pool's spread factor, based on the current state.
	// Pools may choose to make their spread factors dependent upon state
	// (prior TWAPs, network downtime, other pool states, etc.)
	// hence Context is provided as an argument.
	GetSpreadFactor(ctx sdk.Context) osmomath.Dec
	// Returns whether the pool has swaps enabled at the moment
	IsActive(ctx sdk.Context) bool

	// GetPoolDenoms returns the pool's denoms.
	GetPoolDenoms(sdk.Context) []string

	// Returns the spot price of the 'base asset' in terms of the 'quote asset' in the pool,
	// errors if either baseAssetDenom, or quoteAssetDenom does not exist.
	// For example, if this was a UniV2 50-50 pool, with 2 ETH, and 8000 UST
	// pool.SpotPrice(ctx, "eth", "ust") = 4000.00
	SpotPrice(ctx sdk.Context, quoteAssetDenom string, baseAssetDenom string) (osmomath.BigDec, error)
	// GetType returns the type of the pool (Balancer, Stableswap, Concentrated, etc.)
	GetType() PoolType
	// AsSerializablePool returns the pool in a serializable form (useful when a model wraps the proto)
	AsSerializablePool() PoolI
}

// NewPoolAddress returns an address for a pool from a given id.
func NewPoolAddress(poolId uint64) sdk.AccAddress {
	return osmoutils.NewModuleAddressWithPrefix(ModuleName, "pool", sdk.Uint64ToBigEndian(poolId))
}
