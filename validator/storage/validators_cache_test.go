package storage

import (
	"math"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/testmodels"
)

var (
	shuffledValidators = [][]common.Address{
		{
			testmodels.Addr1,
			testmodels.Addr5,
			testmodels.Addr1,
		},
		{
			testmodels.Addr2,
			testmodels.Addr6,
			testmodels.Addr4,
		},
		{
			testmodels.Addr12,
			testmodels.Addr7,
			testmodels.Addr9,
		},
		{
			testmodels.Addr8,
			testmodels.Addr11,
			testmodels.Addr10,
		},
	}

	subnet = uint64(2)
	epoch  = uint64(1)
	slot   = uint64(0)
)

func TestNewValidatorsCache(t *testing.T) {
	cache := NewCache()

	if cache.allValidatorsCache == nil {
		t.Fatal("Expected allValidatorsCache to be initialized, but it was nil")
	}

	if cache.subnetValidatorsCache == nil {
		t.Fatal("Expected subnetValidatorsCache to be initialized, but it was nil")
	}

	if cache.shuffledValidatorsCache == nil {
		t.Fatal("Expected shuffledValidatorsCache to be initialized, but it was nil")
	}

	if cache.shuffledSubnetValidatorsCache == nil {
		t.Fatal("Expected shuffledSubnetValidatorsCache to be initialized, but it was nil")
	}

	if cache.allMu == nil {
		t.Fatal("Expected allMu to be initialized, but it was nil")
	}

	if cache.subnetMu == nil {
		t.Fatal("Expected subnetMu to be initialized, but it was nil")
	}

	if cache.shuffledMu == nil {
		t.Fatal("Expected shuffledMu to be initialized, but it was nil")
	}

	if cache.shuffledSubnetMu == nil {
		t.Fatal("Expected shuffledSubnetMu to be initialized, but it was nil")
	}
}

func TestGetActiveValidatorsByEpoch(t *testing.T) {
	cache := NewCache()

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

func TestAddValidator(t *testing.T) {
	c := NewCache()
	validator := Validator{
		Address:         testmodels.Addr1,
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

func TestDelValidator(t *testing.T) {
	c := NewCache()
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

func TestAllValidatorsCache(t *testing.T) {
	cache := NewCache()
	validatorsList := make([]Validator, len(testmodels.InputValidators))

	for i, inputValidator := range testmodels.InputValidators {
		validatorsList[i] = *NewValidator(inputValidator, nil, uint64(i), 0, math.MaxUint64, nil)
	}

	cache.AddAllValidatorsByEpoch(epoch, validatorsList)

	cachedValidatorsList, ok := cache.allValidatorsCache[epoch]
	if !ok {
		t.Errorf("Expected validators list for epoch %d to be cached, but it wasn't", epoch)
	}

	testutils.AssertEqual(t, cachedValidatorsList, validatorsList)

	cachedValidatorsList, err := cache.GetAllValidatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cachedValidatorsList, validatorsList)

	cachedValidatorsList, err = cache.GetAllValidatorsByEpoch(epoch + 1)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, cachedValidatorsList)
}

func TestSubnetValidatorsCache(t *testing.T) {
	cache := NewCache()

	cache.AddSubnetValidators(epoch, subnet, testmodels.InputValidators)
	cachedValidators, ok := cache.subnetValidatorsCache[epoch][subnet]
	if !ok {
		t.Fatalf("Expected validators list for epoch %d, subnet %d to be cached, but it wasn't", epoch, subnet)
	}
	testutils.AssertEqual(t, cachedValidators, testmodels.InputValidators)

	cachedValidators, err := cache.GetSubnetValidators(epoch, subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, testmodels.InputValidators, cachedValidators)

	cachedValidators, err = cache.GetSubnetValidators(epoch+1, subnet)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, cachedValidators)

	cachedValidators, err = cache.GetSubnetValidators(epoch, subnet+1)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, cachedValidators)
}

func TestGetValidatorsAddresses(t *testing.T) {
	cache := NewCache()
	validatorsList := make([]Validator, len(testmodels.InputValidators))

	for i, inputValidator := range testmodels.InputValidators {
		validatorsList[i] = *NewValidator(inputValidator, nil, uint64(i), uint64(i), math.MaxUint64, nil)
	}

	cache.AddAllValidatorsByEpoch(epoch, validatorsList)

	addresses := cache.GetValidatorsAddresses(epoch, false)
	testutils.AssertEqual(t, testmodels.InputValidators, addresses)

	for i := 0; i < len(validatorsList); i++ {
		currentEpoch := uint64(i)
		cache.AddAllValidatorsByEpoch(currentEpoch, validatorsList)
		addresses = cache.GetValidatorsAddresses(currentEpoch, true)

		activeValidators := make([]common.Address, 0)
		for _, v := range validatorsList {
			if v.ActivationEpoch <= currentEpoch && v.ExitEpoch > currentEpoch {
				activeValidators = append(activeValidators, v.Address)
			}
		}

		testutils.AssertEqual(t, activeValidators, addresses)
	}
}

func TestShuffledValidatorsCache(t *testing.T) {
	cache := NewCache()

	err := cache.AddShuffledValidators(shuffledValidators, []uint64{epoch})
	testutils.AssertNoError(t, err)
	shuffleList, err := cache.GetShuffledValidators([]uint64{epoch, slot})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffleList, shuffledValidators[slot])

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch + 1, slot})
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, shuffleList)

	for i := uint64(0); i < cacheCapacity*2; i++ {
		err = cache.AddShuffledValidators(shuffledValidators, []uint64{i})
		testutils.AssertNoError(t, err)
	}
	err = cache.AddShuffledValidators(shuffledValidators, []uint64{epoch})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cacheCapacity, len(cache.shuffledValidatorsCache))

	err = cache.AddShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
	testutils.AssertNoError(t, err)

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch, slot, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffleList, shuffledValidators[slot])

	err = cache.AddShuffledValidators(shuffledValidators, []uint64{epoch, epoch, subnet, subnet})
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)

	cache.shuffledSubnetValidatorsCache[epoch] = make(map[uint64][][]common.Address, cacheCapacity)
	for i := uint64(0); i <= cacheCapacity; i++ {
		err = cache.AddShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
		testutils.AssertNoError(t, err)
		epoch = epoch + i
	}

	err = cache.AddShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cacheCapacity, len(cache.shuffledSubnetValidatorsCache))

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch, slot, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledValidators[slot], shuffleList)

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch + 1, slot, subnet})
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch, slot, subnet + 1})
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.GetShuffledValidators([]uint64{epoch, slot, subnet, epoch, subnet, slot})
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, shuffleList)
}
