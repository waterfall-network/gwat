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
	testValidator.ExitEra = exitEpoch
	testValidator.ActivationEra = activationEpoch
	testValidator.Balance = balance
}

func TestValidatorMarshalBinary(t *testing.T) {
	data, err := testValidator.MarshalBinary()
	testutils.AssertNoError(t, err)

	expectedData := make([]byte, common.BlsPubKeyLength+common.AddressLength*2+uint64Size*4+len(balance.Bytes()))
	copy(expectedData[:common.BlsPubKeyLength], pubKey[:])
	copy(expectedData[creatorAddressOffset:creatorAddressOffset+common.AddressLength], validatorAddress[:])
	copy(expectedData[withdrawalAddressOffset:validatorIndexOffset], withdrawalAddress[:])
	binary.BigEndian.PutUint64(expectedData[validatorIndexOffset:activationEraOffset], validatorIndex)
	binary.BigEndian.PutUint64(expectedData[activationEraOffset:exitEraOffset], activationEpoch)
	binary.BigEndian.PutUint64(expectedData[exitEraOffset:balanceOffset], exitEpoch)
	binary.BigEndian.PutUint64(expectedData[balanceLengthOffset:balanceOffset], uint64(len(balance.Bytes())))
	copy(expectedData[balanceOffset:balanceOffset+len(balance.Bytes())], balance.Bytes())

	testutils.AssertEqual(t, expectedData, data)
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
	testutils.AssertEqual(t, v.ActivationEra, activationEpoch)
	testutils.AssertEqual(t, v.ExitEra, exitEpoch)
	testutils.AssertEqual(t, v.Balance, balance)
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

	val.SetActivationEra(activationEpoch)
	valActive := val.GetActivationEra()
	testutils.AssertEqual(t, activationEpoch, valActive)

	val.SetExitEra(exitEpoch)
	valExit := val.GetExitEra()
	testutils.AssertEqual(t, exitEpoch, valExit)

	val.SetBalance(balance)
	valBalance := val.GetBalance()
	testutils.AssertEqual(t, balance, valBalance)
}
