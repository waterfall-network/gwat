package storage

import (
	"encoding/binary"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	pubKey            common.BlsPubKey
	validatorAddress  common.Address
	withdrawalAddress common.Address
	validatorIndex    uint64
	activationEpoch   uint64
	exitEpoch         uint64
	balance           *big.Int
	testValidator     *Validator
)

func init() {
	pubKey = common.BytesToBlsPubKey(testutils.RandomStringInBytes(48))
	validatorAddress = common.BytesToAddress(testutils.RandomStringInBytes(20))
	withdrawalAddress = common.BytesToAddress(testutils.RandomStringInBytes(20))
	validatorIndex = uint64(testutils.RandomInt(0, 9999999999))
	activationEpoch = uint64(testutils.RandomInt(0, 9999999999))
	exitEpoch = uint64(testutils.RandomInt(int(activationEpoch), int(activationEpoch+999999999)))
	balance = new(big.Int)
	balance.SetString("319992450932200 000 000 000 000 000 000 000 000", 10)

	testValidator = NewValidator(pubKey, validatorAddress, &withdrawalAddress)
	testValidator.Index = validatorIndex
	testValidator.ExitEpoch = exitEpoch
	testValidator.ActivationEpoch = activationEpoch
	testValidator.Balance = balance
}

func TestValidatorMarshalBinary(t *testing.T) {
	data, err := testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	expectedData := make([]byte, common.BlsPubKeyLength+common.AddressLength*2+uint64Size*4+len(balance.Bytes()))
	copy(expectedData[:common.BlsPubKeyLength], pubKey[:])
	copy(expectedData[creatorAddressOffset:creatorAddressOffset+common.AddressLength], validatorAddress[:])
	copy(expectedData[withdrawalAddressOffset:validatorIndexOffset], withdrawalAddress[:])
	binary.BigEndian.PutUint64(expectedData[validatorIndexOffset:activationEpochOffset], validatorIndex)
	binary.BigEndian.PutUint64(expectedData[activationEpochOffset:exitEpochOffset], activationEpoch)
	binary.BigEndian.PutUint64(expectedData[exitEpochOffset:balanceOffset], exitEpoch)
	binary.BigEndian.PutUint64(expectedData[balanceLengthOffset:balanceOffset], uint64(len(balance.Bytes())))
	copy(expectedData[balanceOffset:balanceOffset+len(balance.Bytes())], balance.Bytes())

	testutils.AssertEqual(t, data, expectedData)
}

func TestValidatorUnmarshalBinary(t *testing.T) {
	data, err := testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	v := new(Validator)
	err = v.UnmarshalBinary(data)
	testutils.AssertNoError(t, err)

	//testutils.AssertEqual(t, v.PubKey, pubKey)
	testutils.AssertEqual(t, v.Address, validatorAddress)
	testutils.AssertEqual(t, *v.WithdrawalAddress, withdrawalAddress)
	testutils.AssertEqual(t, v.Index, validatorIndex)
	testutils.AssertEqual(t, v.ActivationEpoch, activationEpoch)
	testutils.AssertEqual(t, v.ExitEpoch, exitEpoch)
	testutils.AssertEqual(t, v.Balance, balance)
}

func TestValidatorInfoGetters(t *testing.T) {
	var (
		valInfo ValidatorInfo
		err     error
	)

	valInfo, err = testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	valPubKey := valInfo.GetPubKey()
	testutils.AssertEqual(t, valPubKey, pubKey)

	valAddress := valInfo.GetAddress()
	testutils.AssertEqual(t, valAddress, validatorAddress)

	valWithdrawal := valInfo.GetWithdrawalAddress()
	testutils.AssertEqual(t, valWithdrawal, withdrawalAddress)

	valIndex := valInfo.GetIndex()
	testutils.AssertEqual(t, valIndex, validatorIndex)

	valActiveEpoch := valInfo.GetActivationEpoch()
	testutils.AssertEqual(t, valActiveEpoch, activationEpoch)

	valExitEpoch := valInfo.GetExitEpoch()
	testutils.AssertEqual(t, valExitEpoch, exitEpoch)

	valBalance := valInfo.GetBalance()
	testutils.AssertEqual(t, valBalance, balance)
}

func TestValidatorInfoSetters(t *testing.T) {
	minLenValInfo := balanceOffset
	val := make(ValidatorInfo, minLenValInfo)

	val.SetPubKey(pubKey)
	valPubKey := val.GetPubKey()
	testutils.AssertEqual(t, pubKey, valPubKey)

	val.SetAddress(validatorAddress)
	valAddr := val.GetAddress()
	testutils.AssertEqual(t, valAddr, validatorAddress)

	val.SetWithdrawalAddress(withdrawalAddress)
	valWithdraw := val.GetWithdrawalAddress()
	testutils.AssertEqual(t, valWithdraw, withdrawalAddress)

	val.SetIndex(validatorIndex)
	valIndex := val.GetIndex()
	testutils.AssertEqual(t, valIndex, validatorIndex)

	val.SetActivationEpoch(activationEpoch)
	valActive := val.GetActivationEpoch()
	testutils.AssertEqual(t, valActive, activationEpoch)

	val.SetExitEpoch(exitEpoch)
	valExit := val.GetExitEpoch()
	testutils.AssertEqual(t, valExit, exitEpoch)

	val = SetValidatorBalance(val, balance)
	valBalance := val.GetBalance()
	testutils.AssertEqual(t, balance, valBalance)
}
