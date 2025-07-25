package keeper

import (
	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	osmoutils "github.com/neutron-org/neutron/v5/osmoutils"
)

// ConvertToBaseToken converts a fee amount in a whitelisted fee token to the base fee token amount.
func (k Keeper) ConvertToBaseToken(ctx sdk.Context, inputFee sdk.Coin) (sdk.Coin, error) {
	baseDenom, err := k.GetBaseDenom(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	if inputFee.Denom == baseDenom {
		return inputFee, nil
	}

	feeToken, err := k.GetFeeToken(ctx, inputFee.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	spotPrice, err := k.CalcFeeSpotPrice(ctx, feeToken.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	// Note: spotPrice truncation is done here for maintaining state-compatibility with v19.x
	// It should be changed to support full spot price precision before
	// https://github.com/osmosis-labs/osmosis/issues/6064 is complete
	return sdk.NewCoin(baseDenom, spotPrice.Dec().MulIntMut(inputFee.Amount).RoundInt()), nil
}

// CalcFeeSpotPrice converts the provided tx fees into their equivalent value in the base denomination.
// Spot Price Calculation: spotPrice / (1 - spreadFactor),
// where spotPrice is defined as:
// (tokenBalanceIn / tokenWeightIn) / (tokenBalanceOut / tokenWeightOut)
func (k Keeper) CalcFeeSpotPrice(ctx sdk.Context, inputDenom string) (osmomath.BigDec, error) {
	baseDenom, err := k.GetBaseDenom(ctx)
	if err != nil {
		return osmomath.BigDec{}, err
	}

	feeToken, err := k.GetFeeToken(ctx, inputDenom)
	if err != nil {
		return osmomath.BigDec{}, err
	}

	spotPrice, err := k.poolManager.RouteCalculateSpotPrice(ctx, feeToken.PoolID, baseDenom, feeToken.Denom)
	if err != nil {
		return osmomath.BigDec{}, err
	}
	return spotPrice, nil
}

// GetFeeToken returns the fee token record for a specific denom,
// In our case the baseDenom is uosmo.
func (k Keeper) GetBaseDenom(ctx sdk.Context) (denom string, err error) {
	store := ctx.KVStore(k.storeKey)

	if !store.Has(types.BaseDenomKey) {
		return "", types.ErrNoBaseDenom
	}

	bz := store.Get(types.BaseDenomKey)

	return string(bz), nil
}

// SetBaseDenom sets the base fee denom for the chain. Should only be used once.
func (k Keeper) SetBaseDenom(ctx sdk.Context, denom string) error {
	store := ctx.KVStore(k.storeKey)

	err := sdk.ValidateDenom(denom)
	if err != nil {
		return err
	}

	store.Set(types.BaseDenomKey, []byte(denom))
	return nil
}

// ValidateFeeToken validates that a fee token record is valid
// It checks:
// - The denom exists
// - The denom is not the base denom
// - The gamm pool exists
// - The gamm pool includes the base token and fee token.
func (k Keeper) ValidateFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	baseDenom, err := k.GetBaseDenom(ctx)
	if err != nil {
		return err
	}
	if baseDenom == feeToken.Denom {
		return errorsmod.Wrap(types.ErrInvalidFeeToken, "cannot add basedenom as a whitelisted fee token")
	}
	// This not returning an error implies that:
	// - feeToken.Denom exists
	// - feeToken.PoolID exists
	// - feeToken.PoolID has both feeToken.Denom and baseDenom
	_, err = k.poolManager.RouteCalculateSpotPrice(ctx, feeToken.PoolID, feeToken.Denom, baseDenom)

	return err
}

// GetFeeToken returns a unique fee token record for a specific denom.
// If the denom doesn't exist, returns an error.
func (k Keeper) GetFeeToken(ctx sdk.Context, denom string) (types.FeeToken, error) {
	prefixStore := k.GetFeeTokensStore(ctx)
	if !prefixStore.Has([]byte(denom)) {
		return types.FeeToken{}, errorsmod.Wrapf(types.ErrInvalidFeeToken, "%s", denom)
	}
	bz := prefixStore.Get([]byte(denom))

	feeToken := types.FeeToken{}
	err := proto.Unmarshal(bz, &feeToken)
	if err != nil {
		return types.FeeToken{}, err
	}

	return feeToken, nil
}

// setFeeToken sets a new fee token record for a specific denom.
// PoolID is just the pool to swap rate between alt fee token and native fee token.
// If the feeToken pool ID is 0, deletes the fee Token entry.
func (k Keeper) setFeeToken(ctx sdk.Context, feeToken types.FeeToken) error {
	prefixStore := k.GetFeeTokensStore(ctx)

	if feeToken.PoolID == 0 {
		if prefixStore.Has([]byte(feeToken.Denom)) {
			prefixStore.Delete([]byte(feeToken.Denom))
		}
		return nil
	}

	err := k.ValidateFeeToken(ctx, feeToken)
	if err != nil {
		return err
	}

	bz, err := proto.Marshal(&feeToken)
	if err != nil {
		return err
	}

	prefixStore.Set([]byte(feeToken.Denom), bz)
	return nil
}

func (k Keeper) GetFeeTokens(ctx sdk.Context) (feetokens []types.FeeToken) {
	prefixStore := k.GetFeeTokensStore(ctx)

	// this entire store just contains FeeTokens, so iterate over all entries.
	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	feeTokens := []types.FeeToken{}

	for ; iterator.Valid(); iterator.Next() {
		feeToken := types.FeeToken{}

		err := proto.Unmarshal(iterator.Value(), &feeToken)
		if err != nil {
			panic(err)
		}

		feeTokens = append(feeTokens, feeToken)
	}
	return feeTokens
}

func (k Keeper) SetFeeTokens(ctx sdk.Context, feetokens []types.FeeToken) error {
	for _, feeToken := range feetokens {
		err := k.setFeeToken(ctx, feeToken)
		if err != nil {
			return err
		}
	}
	return nil
}

// SenderValidationSetFeeTokens first checks to see if the sender is whitelisted to set fee tokens.
// If the sender is whitelisted, it sets the fee tokens.
// If the sender is not whitelisted, it returns an error.
func (k Keeper) SenderValidationSetFeeTokens(ctx sdk.Context, sender string, feetokens []types.FeeToken) error {
	whitelistedAddresses := k.GetParams(ctx).WhitelistedFeeTokenSetters

	isWhitelisted := osmoutils.Contains(whitelistedAddresses, sender)
	if !isWhitelisted {
		return errorsmod.Wrapf(types.ErrNotWhitelistedFeeTokenSetter, "%s", sender)
	}

	return k.SetFeeTokens(ctx, feetokens)
}
