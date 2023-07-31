package storage

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"reflect"
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

	for i := 0; i < 10; i++ {
		stakeAddress := common.BytesToAddress(testutils.RandomStringInBytes(20))
		stakeSum := big.NewInt(int64(testutils.RandomInt(1, 100000)))
		testValidator.AddStake(stakeAddress, stakeSum)
	}
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

	var stakeBuffer bytes.Buffer
	for _, stake := range testValidator.Stake {
		stakeData := stake.MarshalBinary()

		// Include the length of each stake before its data
		stakeLenBytes := make([]byte, uint64Size)
		binary.BigEndian.PutUint64(stakeLenBytes, uint64(len(stakeData)))
		stakeBuffer.Write(stakeLenBytes)

		stakeBuffer.Write(stakeData)
	}

	stakeData := stakeBuffer.Bytes()

	expectedData = append(expectedData, stakeData...)

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

func TestStakeByAddress_MarshalUnmarshal(t *testing.T) {
	original := &StakeByAddress{
		Address: common.BytesToAddress(testutils.RandomStringInBytes(20)),
		Sum:     big.NewInt(1234567890),
	}

	data := original.MarshalBinary()

	unmarshalled := &StakeByAddress{}
	err := unmarshalled.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary error: %v", err)
	}

	if !reflect.DeepEqual(original, unmarshalled) {
		t.Errorf("Unmarshalled StakeByAddress does not match original. Original: %+v, Unmarshalled: %+v", original, unmarshalled)
	}
}
