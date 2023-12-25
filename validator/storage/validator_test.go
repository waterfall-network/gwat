package storage

import (
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

var (
	pubKey            common.BlsPubKey
	validatorAddress  common.Address
	withdrawalAddress common.Address
	validatorIndex    uint64
	activationEra     uint64
	exitEpoch         uint64
	testValidator     *Validator
)

func init() {
	pubKey = common.BytesToBlsPubKey(testutils.RandomStringInBytes(48))
	validatorAddress = common.BytesToAddress(testutils.RandomStringInBytes(20))
	withdrawalAddress = common.BytesToAddress(testutils.RandomStringInBytes(20))
	validatorIndex = uint64(testutils.RandomInt(0, 9999999999))
	activationEra = uint64(testutils.RandomInt(0, 9999999999))
	exitEpoch = uint64(testutils.RandomInt(int(activationEra), int(activationEra+999999999)))

	testValidator = NewValidator(pubKey, validatorAddress, &withdrawalAddress)
	testValidator.Index = validatorIndex
	testValidator.ExitEra = exitEpoch
	testValidator.ActivationEra = activationEra

	for i := 0; i < 10; i++ {
		stakeAddress := common.BytesToAddress(testutils.RandomStringInBytes(20))
		stakeSum := big.NewInt(int64(testutils.RandomInt(1, 100000)))
		testValidator.AddStake(stakeAddress, stakeSum)
	}
}

func TestValidator_MarshalingBinary(t *testing.T) {
	data, err := testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	v := new(Validator)
	err = v.UnmarshalBinary(data)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, v.PubKey, testValidator.PubKey)
	testutils.AssertEqual(t, v.Address, testValidator.Address)
	testutils.AssertEqual(t, v.WithdrawalAddress, testValidator.WithdrawalAddress)
	testutils.AssertEqual(t, v.Index, testValidator.Index)
	testutils.AssertEqual(t, v.ActivationEra, testValidator.ActivationEra)
	testutils.AssertEqual(t, v.ExitEra, testValidator.ExitEra)

	testutils.AssertEqual(t, len(v.Stake), len(testValidator.Stake))
	t.Logf("Length of unmarshalled Stake slice: %d, expected length: %d", len(v.Stake), len(testValidator.Stake))
	for i, stake := range v.Stake {
		expectedStake := testValidator.Stake[i]
		testutils.AssertEqual(t, stake.Address, expectedStake.Address)
		testutils.AssertEqual(t, stake.Sum, expectedStake.Sum)

		t.Logf("Stake %d: Address: %s, Expected Address: %s, Sum: %d, Expected Sum: %d",
			i, stake.Address, expectedStake.Address, stake.Sum, expectedStake.Sum)
	}
}

func TestValidatorDelegatingStake_MarshalingBinary(t *testing.T) {
	profitShare, stakeShare, exit, withdrawal := operation.TestParamsDelegatingStakeRules()
	trialPeriod := uint64(321)

	rules, err := operation.NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := operation.NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	dsr, err := operation.NewDelegatingStakeData(rules, trialPeriod, trialRules)
	testutils.AssertNoError(t, err)
	testValidator.DelegatingStake = dsr

	data, err := testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	v := new(Validator)
	err = v.UnmarshalBinary(data)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, v.PubKey, testValidator.PubKey)
	testutils.AssertEqual(t, v.Address, testValidator.Address)
	testutils.AssertEqual(t, v.WithdrawalAddress, testValidator.WithdrawalAddress)
	testutils.AssertEqual(t, v.Index, testValidator.Index)
	testutils.AssertEqual(t, v.ActivationEra, testValidator.ActivationEra)
	testutils.AssertEqual(t, v.ExitEra, testValidator.ExitEra)

	testutils.AssertEqual(t, len(v.Stake), len(testValidator.Stake))
	for i, stake := range v.Stake {
		expectedStake := testValidator.Stake[i]
		testutils.AssertEqual(t, stake.Address, expectedStake.Address)
		testutils.AssertEqual(t, stake.Sum, expectedStake.Sum)
	}

	// delegate stake
	testutils.AssertEqual(t, dsr.Rules, v.DelegatingStake.Rules)
	testutils.AssertEqual(t, dsr.TrialPeriod, v.DelegatingStake.TrialPeriod)
	testutils.AssertEqual(t, dsr.TrialRules, v.DelegatingStake.TrialRules)
}

func TestValidatorSettersGetters(t *testing.T) {
	val := new(Validator)

	val.SetPubKey(pubKey)
	valPubKey := val.GetPubKey()
	testutils.AssertEqual(t, pubKey, valPubKey)

	val.SetAddress(validatorAddress)
	valAddr := val.GetAddress()
	testutils.AssertEqual(t, validatorAddress, valAddr)

	val.SetWithdrawalAddress(&withdrawalAddress)
	valWithdraw := val.GetWithdrawalAddress()
	testutils.AssertEqual(t, withdrawalAddress, *valWithdraw)

	val.SetIndex(validatorIndex)
	valIndex := val.GetIndex()
	testutils.AssertEqual(t, validatorIndex, valIndex)

	val.SetActivationEra(activationEra)
	valActive := val.GetActivationEra()
	testutils.AssertEqual(t, activationEra, valActive)

	val.SetExitEra(exitEpoch)
	valExit := val.GetExitEra()
	testutils.AssertEqual(t, exitEpoch, valExit)
}
