package types

import (
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/neutron-org/neutron/v5/osmomath"
	"github.com/neutron-org/neutron/v5/osmoutils"
)

// constants
const (
	TypeMsgSetValidatorSetPreference = "set_validator_set_preference"
)

var _ sdk.Msg = &MsgSetValidatorSetPreference{}

// NewMsgCreateValidatorSetPreference creates a msg to create a validator-set preference.
func NewMsgSetValidatorSetPreference(delegator sdk.AccAddress, preferences []ValidatorPreference) *MsgSetValidatorSetPreference {
	return &MsgSetValidatorSetPreference{
		Delegator:   delegator.String(),
		Preferences: preferences,
	}
}

func (m MsgSetValidatorSetPreference) Route() string { return RouterKey }
func (m MsgSetValidatorSetPreference) Type() string  { return TypeMsgSetValidatorSetPreference }
func (m MsgSetValidatorSetPreference) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid delegator address (%s)", err)
	}

	totalWeight := osmomath.ZeroDec()
	validatorAddrs := []string{}
	for _, validator := range m.Preferences {
		_, err := sdk.ValAddressFromBech32(validator.ValOperAddress)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid validator address (%s)", err)
		}

		// all the weights should be positive
		if !(validator.Weight.IsPositive()) {
			return fmt.Errorf("Invalid weight, validator weight needs to be positive, got %d", validator.Weight)
		}

		totalWeight = totalWeight.Add(validator.Weight)
		validatorAddrs = append(validatorAddrs, validator.ValOperAddress)
	}

	// check that all the validator address are unique
	containsDuplicate := osmoutils.ContainsDuplicate(validatorAddrs)
	if containsDuplicate {
		return errors.New("The validator operator address are duplicated")
	}

	// Round to 2 digit after the decimal. For ex: 0.999 = 1.0, 0.874 = 0.87, 0.5123 = 0.51
	roundedValue := osmomath.SigFigRound(totalWeight, osmomath.NewDec(10).Power(2).TruncateInt())

	// check if the total validator distribution weights equal 1
	if !roundedValue.Equal(osmomath.OneDec()) {
		return fmt.Errorf("The weights allocated to the validators do not add up to 1, Got: %f", roundedValue)
	}

	return nil
}

// GetSigners takes a create validator-set message and returns the delegator in a byte array.
func (m MsgSetValidatorSetPreference) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgDelegateToValidatorSet = "delegate_to_validator_set"
)

var _ sdk.Msg = &MsgDelegateToValidatorSet{}

// NewMsgMsgStakeToValidatorSet creates a msg to stake to a validator set.
func NewMsgDelegateToValidatorSet(delegator sdk.AccAddress, coin sdk.Coin) *MsgDelegateToValidatorSet {
	return &MsgDelegateToValidatorSet{
		Delegator: delegator.String(),
		Coin:      coin,
	}
}

func (m MsgDelegateToValidatorSet) Route() string { return RouterKey }
func (m MsgDelegateToValidatorSet) Type() string  { return TypeMsgDelegateToValidatorSet }
func (m MsgDelegateToValidatorSet) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if !m.Coin.IsValid() {
		return errors.New("The stake coin is not valid")
	}

	return nil
}

func (m MsgDelegateToValidatorSet) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgUndelegateFromValidatorSet = "undelegate_from_validator_set"
)

var _ sdk.Msg = &MsgUndelegateFromValidatorSet{}

// NewMsgMsgStakeToValidatorSet creates a msg to stake to a validator.
func NewMsgUndelegateFromValidatorSet(delegator sdk.AccAddress, coin sdk.Coin) *MsgUndelegateFromValidatorSet {
	return &MsgUndelegateFromValidatorSet{
		Delegator: delegator.String(),
		Coin:      coin,
	}
}

func (m MsgUndelegateFromValidatorSet) Route() string { return RouterKey }
func (m MsgUndelegateFromValidatorSet) Type() string  { return TypeMsgUndelegateFromValidatorSet }
func (m MsgUndelegateFromValidatorSet) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if !m.Coin.IsValid() {
		return errors.New("The stake coin is not valid")
	}

	return nil
}

func (m MsgUndelegateFromValidatorSet) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgUndelegateFromRebalancedValidatorSet = "undelegate_from_rebalanced_validator_set"
)

var _ sdk.Msg = &MsgUndelegateFromRebalancedValidatorSet{}

// NewMsgUndelegateFromRebalancedValidatorSet creates a msg to undelegated from a rebalanced validator set.
func NewMsgUndelegateFromRebalancedValidatorSet(delegator sdk.AccAddress, coin sdk.Coin) *MsgUndelegateFromRebalancedValidatorSet {
	return &MsgUndelegateFromRebalancedValidatorSet{
		Delegator: delegator.String(),
		Coin:      coin,
	}
}

func (m MsgUndelegateFromRebalancedValidatorSet) Route() string { return RouterKey }
func (m MsgUndelegateFromRebalancedValidatorSet) Type() string {
	return TypeMsgUndelegateFromRebalancedValidatorSet
}
func (m MsgUndelegateFromRebalancedValidatorSet) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if !m.Coin.IsValid() {
		return errors.New("The stake coin is not valid")
	}

	return nil
}

func (m MsgUndelegateFromRebalancedValidatorSet) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgRedelegateValidatorSet = "redelegate_validator_set"
)

var _ sdk.Msg = &MsgRedelegateValidatorSet{}

// NewMsgMsgStakeToValidatorSet creates a msg to stake to a validator.
func NewMsgRedelegateValidatorSet(delegator sdk.AccAddress, preferences []ValidatorPreference) *MsgRedelegateValidatorSet {
	return &MsgRedelegateValidatorSet{
		Delegator:   delegator.String(),
		Preferences: preferences,
	}
}

func (m MsgRedelegateValidatorSet) Route() string { return RouterKey }
func (m MsgRedelegateValidatorSet) Type() string  { return TypeMsgRedelegateValidatorSet }
func (m MsgRedelegateValidatorSet) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid delegator address (%s)", err)
	}

	totalWeight := osmomath.NewDec(0)
	validatorAddrs := []string{}
	for _, validator := range m.Preferences {
		_, err := sdk.ValAddressFromBech32(validator.ValOperAddress)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid validator address (%s)", err)
		}

		totalWeight = totalWeight.Add(validator.Weight)
		validatorAddrs = append(validatorAddrs, validator.ValOperAddress)
	}

	// check that all the validator address are unique
	containsDuplicate := osmoutils.ContainsDuplicate(validatorAddrs)
	if containsDuplicate {
		return errors.New("The validator operator address are duplicated")
	}

	// Round to 2 digit after the decimal. For ex: 0.999 = 1.0, 0.874 = 0.87, 0.5123 = 0.51
	roundedValue := osmomath.SigFigRound(totalWeight, osmomath.NewDec(10).Power(2).TruncateInt())

	// check if the total validator distribution weights equal 1
	if !roundedValue.Equal(osmomath.OneDec()) {
		return fmt.Errorf("The weights allocated to the validators do not add up to 1, Got: %f", roundedValue)
	}

	return nil
}

func (m MsgRedelegateValidatorSet) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgWithdrawDelegationRewards = "withdraw_delegation_rewards"
)

var _ sdk.Msg = &MsgWithdrawDelegationRewards{}

// NewMsgMsgStakeToValidatorSet creates a msg to stake to a validator.
func NewMsgWithdrawDelegationRewards(delegator sdk.AccAddress) *MsgWithdrawDelegationRewards {
	return &MsgWithdrawDelegationRewards{
		Delegator: delegator.String(),
	}
}

func (m MsgWithdrawDelegationRewards) Route() string { return RouterKey }
func (m MsgWithdrawDelegationRewards) Type() string  { return TypeMsgWithdrawDelegationRewards }
func (m MsgWithdrawDelegationRewards) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	return nil
}

func (m MsgWithdrawDelegationRewards) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}

// constants
const (
	TypeMsgDelegateBondedTokens = "delegate_bonded_tokens"
)

var _ sdk.Msg = &MsgDelegateBondedTokens{}

// NewMsgMsgStakeToValidatorSet creates a msg to stake to a validator.
func NewMsgDelegateBondedTokens(delegator sdk.AccAddress, lockId uint64) *MsgDelegateBondedTokens {
	return &MsgDelegateBondedTokens{
		Delegator: delegator.String(),
		LockID:    lockId,
	}
}

func (m MsgDelegateBondedTokens) Type() string { return TypeMsgDelegateBondedTokens }
func (m MsgDelegateBondedTokens) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(m.Delegator)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "Invalid sender address (%s)", err)
	}

	if m.LockID <= 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "lock id should be bigger than 1 (%s)", err)
	}

	return nil
}

func (m MsgDelegateBondedTokens) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(m.Delegator)
	return []sdk.AccAddress{delegator}
}
