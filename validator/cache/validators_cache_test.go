package cache

import (
	"math/rand"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/token/testutils"
)

var (
	addr1 = common.BytesToAddress(testutils.RandomStringInBytes(50))
	addr2 = common.BytesToAddress(testutils.RandomStringInBytes(30))
	addr3 = common.BytesToAddress(testutils.RandomStringInBytes(20))
	addr4 = common.BytesToAddress(testutils.RandomStringInBytes(40))
	addr5 = common.BytesToAddress(testutils.RandomStringInBytes(50))
	addr6 = common.BytesToAddress(testutils.RandomStringInBytes(70))

	inputValidators    = []common.Address{addr1, addr2, addr3, addr4, addr5, addr6}
	shuffledValidators = [][]common.Address{
		{
			addr3,
			addr5,
			addr1,
		},
		{
			addr2,
			addr6,
			addr4,
		},
	}

	subnet = uint64(2)
	epoch  = uint64(1)
	slot   = uint64(0)
)

func TestNewValidatorsCache(t *testing.T) {
	validatorsCache := New()
	if validatorsCache.allValidatorsCache == nil {
		t.Fatal("allValidatorsCache not initialized")
	}
	if validatorsCache.subnetValidatorsCache == nil {
		t.Fatal("subnetValidatorsCache not initialized")
	}
	if validatorsCache.shuffledValidatorsCache == nil {
		t.Fatal("shuffledValidatorsCache not initialized")
	}
	if validatorsCache.shuffledSubnetValidatorsCache == nil {
		t.Fatal("shuffledSubnetValidatorsCache not initialized")
	}
}

func TestAllValidatorsCache(t *testing.T) {
	c := New()

	validators := make([]Validator, len(inputValidators))
	for i, validator := range inputValidators {
		validators[i] = *NewValidator(validator, nil, rand.Uint64(), rand.Uint64(), rand.Uint64(), nil)
	}

	c.AddAllValidatorsByEpoch(epoch, validators)

	outputValidators, err := c.GetAllValidatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, outputValidators, validators)

	outputValidators, err = c.GetAllValidatorsByEpoch(epoch + 1)
	testutils.AssertError(t, err, errNoCachedValidators)
	testutils.AssertNil(t, outputValidators)
}

func TestSubnetValidatorsCache(t *testing.T) {
	c := New()

	validators, err := c.GetSubnetValidators(subnet)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, validators)

	c.AddSubnetValidators(subnet, inputValidators)
	validators, err = c.GetSubnetValidators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, inputValidators, validators)
}

func TestShuffledValidatorsCache(t *testing.T) {
	c := New()

	validators, err := c.GetShuffledValidatorsByEpoch(epoch)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, validators)

	validatorsBySlot, err := c.GetShuffledValidatorsBySlot(epoch, slot)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, validatorsBySlot)

	c.AddShuffledValidators(epoch, shuffledValidators)

	validatorsBySlot, err = c.GetShuffledValidatorsBySlot(epoch, 5)
	testutils.AssertError(t, err, errNoSlotValidators)
	testutils.AssertNil(t, validatorsBySlot)

	validators, err = c.GetShuffledValidatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, [][]common.Address{{addr3, addr5, addr1}, {addr2, addr6, addr4}}, validators)

	validatorsBySlot, err = c.GetShuffledValidatorsBySlot(epoch, slot)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{addr3, addr5, addr1}, validatorsBySlot)
}

func TestShuffledSubnetValidators(t *testing.T) {
	c := New()

	c.AddShuffledSubnetValidators(subnet, epoch, shuffledValidators)

	slotValidators, err := c.GetShuffledSubnetValidatorsBySlot(subnet, epoch, 5)
	testutils.AssertError(t, err, errNoSlotValidators)
	testutils.AssertNil(t, slotValidators)

	slotValidators, err = c.GetShuffledSubnetValidatorsBySlot(subnet, 3, slot)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, slotValidators)

	slotValidators, err = c.GetShuffledSubnetValidatorsBySlot(3, epoch, slot)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, slotValidators)

	epochValidators, err := c.GetShuffledSubnetValidatorsByEpoch(subnet, 3)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, epochValidators)

	epochValidators, err = c.GetShuffledSubnetValidatorsByEpoch(1, epoch)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, epochValidators)

	subnetValidators, err := c.GetShuffledSubnetValidators(1)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, subnetValidators)

	subnetValidators, err = c.GetShuffledSubnetValidators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, map[uint64][][]common.Address{1: shuffledValidators}, subnetValidators)

	epochValidators, err = c.GetShuffledSubnetValidatorsByEpoch(subnet, epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledValidators, epochValidators)

	slotValidators, err = c.GetShuffledSubnetValidatorsBySlot(subnet, epoch, 0)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledValidators[0], slotValidators)
}

func TestGetActiveValidatorsByEpoch(t *testing.T) {
	cache := New()

	validators := cache.GetActiveValidatorsByEpoch(1)
	if validators != nil {
		t.Fatal("Expected nil, got", validators)
	}

	validator1 := Validator{ActivationEpoch: 1, ExitEpoch: 20}
	validator2 := Validator{ActivationEpoch: 5, ExitEpoch: 50}
	validator3 := Validator{ActivationEpoch: 15, ExitEpoch: 30}
	validatorsList := []Validator{validator1, validator2, validator3}
	cache.AddAllValidatorsByEpoch(10, validatorsList)

	expectedValidators := []Validator{validator1, validator2}
	validators = cache.GetActiveValidatorsByEpoch(10)
	testutils.AssertEqual(t, expectedValidators, validators)

	cache.AddAllValidatorsByEpoch(30, validatorsList)
	validators = cache.GetActiveValidatorsByEpoch(30)
	testutils.AssertEqual(t, []Validator{validator2}, validators)
}

func TestValidatorsCacheAddValidator(t *testing.T) {
	c := New()
	validator := Validator{
		Address:         addr1,
		ActivationEpoch: 10,
		ExitEpoch:       20,
	}

	c.AddValidator(validator, 10)
	validators := c.GetActiveValidatorsByEpoch(10)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}
	if validators[0].Address != validator.Address {
		t.Fatalf("Expected address %v but got %v", validator.Address, validators[0].Address)
	}
}

func TestValidatorsCacheDelValidator(t *testing.T) {
	c := New()
	validator := Validator{
		Address:         common.Address{1},
		ActivationEpoch: 10,
		ExitEpoch:       20,
	}

	c.AddValidator(validator, 10)
	validators := c.GetActiveValidatorsByEpoch(10)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}

	c.DelValidator(validator, 10)
	validators = c.GetActiveValidatorsByEpoch(10)
	if len(validators) != 0 {
		t.Fatalf("Expected 0 validators but got %v", len(validators))
	}
}
