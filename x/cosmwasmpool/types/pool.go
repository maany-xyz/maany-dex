package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	poolmanagertypes "github.com/neutron-org/neutron/v5/x/poolmanager/types"
)

// CosmWasmExtension
type CosmWasmExtension interface {
	poolmanagertypes.PoolI

	GetCodeId() uint64

	GetInstantiateMsg() []byte

	GetContractAddress() string

	SetContractAddress(contractAddress string)

	GetStoreModel() poolmanagertypes.PoolI

	SetWasmKeeper(wasmKeeper WasmKeeper)

	GetTotalPoolLiquidity(ctx sdk.Context) sdk.Coins

	SetCodeId(codeId uint64)
}
