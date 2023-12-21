package storage

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
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
	ctrl := gomock.NewController(t)
	bc := NewMockblockchain(ctrl)

	cache := NewCache()

	validators := cache.getActiveValidatorsByEpoch(bc, 1)
	if validators != nil {
		t.Fatal("Expected nil, got", validators)
	}

	bc.EXPECT().EpochToEra(uint64(10)).Return(&era.Era{
		Number: 10,
		From:   0,
		To:     0,
		Root:   common.Hash{},
	})
	validator1 := Validator{ActivationEra: 1, ExitEra: 20}
	validator2 := Validator{ActivationEra: 5, ExitEra: 50}
	validator3 := Validator{ActivationEra: 15, ExitEra: 30}
	validatorsList := []Validator{validator1, validator2, validator3}
	cache.addAllValidatorsByEpoch(10, validatorsList)

	bc.EXPECT().EpochToEra(uint64(30)).Return(&era.Era{
		Number: 30,
		From:   0,
		To:     0,
		Root:   common.Hash{},
	})
	expectedValidators := []Validator{validator1, validator2}
	validators = cache.getActiveValidatorsByEpoch(bc, 10)
	testutils.AssertEqual(t, expectedValidators, validators)

	cache.addAllValidatorsByEpoch(30, validatorsList)
	validators = cache.getActiveValidatorsByEpoch(bc, 30)
	testutils.AssertEqual(t, []Validator{validator2}, validators)
}

func TestAddValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	bc := NewMockblockchain(ctrl)
	epoch := uint64(10)
	bc.EXPECT().EpochToEra(epoch).Return(&era.Era{Number: 10})

	c := NewCache()
	validator := Validator{
		Address:       testmodels.Addr1,
		ActivationEra: 10,
		ExitEra:       20,
	}

	c.addValidator(validator, epoch)
	validators := c.getActiveValidatorsByEpoch(bc, epoch)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}
	if validators[0].Address != validator.Address {
		t.Fatalf("Expected address %v but got %v", validator.Address, validators[0].Address)
	}
}

func TestDelValidator(t *testing.T) {
	ctrl := gomock.NewController(t)
	bc := NewMockblockchain(ctrl)
	epoch := uint64(10)
	bc.EXPECT().EpochToEra(epoch).AnyTimes().Return(&era.Era{Number: 10})

	c := NewCache()
	validator := Validator{
		Address:       common.Address{1},
		ActivationEra: 10,
		ExitEra:       20,
	}

	c.addValidator(validator, epoch)
	validators := c.getActiveValidatorsByEpoch(bc, epoch)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}

	c.delValidator(validator, epoch)
	validators = c.getActiveValidatorsByEpoch(bc, epoch)
	if len(validators) != 0 {
		t.Fatalf("Expected 0 validators but got %v", len(validators))
	}
}

func TestAllValidatorsCache(t *testing.T) {
	cache := NewCache()
	validatorsList := make([]Validator, len(testmodels.InputValidators))

	for i, inputValidator := range testmodels.InputValidators {
		validatorsList[i] = *NewValidator(common.BlsPubKey{}, inputValidator, nil)
	}

	cache.addAllValidatorsByEpoch(epoch, validatorsList)

	cachedValidatorsList, ok := cache.allValidatorsCache[epoch]
	if !ok {
		t.Errorf("Expected validators list for epoch %d to be cached, but it wasn't", epoch)
	}

	testutils.AssertEqual(t, cachedValidatorsList, validatorsList)

	cachedValidatorsList, err := cache.getAllValidatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cachedValidatorsList, validatorsList)

	cachedValidatorsList, err = cache.getAllValidatorsByEpoch(epoch + 1)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, cachedValidatorsList)
}

func TestSubnetValidatorsCache(t *testing.T) {
	cache := NewCache()

	cache.addSubnetValidators(epoch, subnet, testmodels.InputValidators)
	cachedValidators, ok := cache.subnetValidatorsCache[epoch][subnet]
	if !ok {
		t.Fatalf("Expected validators list for epoch %d, subnet %d to be cached, but it wasn't", epoch, subnet)
	}
	testutils.AssertEqual(t, cachedValidators, testmodels.InputValidators)

	cachedValidators, err := cache.getSubnetValidators(epoch, subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, testmodels.InputValidators, cachedValidators)

	cachedValidators, err = cache.getSubnetValidators(epoch+1, subnet)
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, cachedValidators)

	cachedValidators, err = cache.getSubnetValidators(epoch, subnet+1)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, cachedValidators)
}

func TestGetValidatorsAddresses(t *testing.T) {
	ctrl := gomock.NewController(t)
	bc := NewMockblockchain(ctrl)
	bc.EXPECT().EpochToEra(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(&era.Era{Number: 10})

	cache := NewCache()
	validatorsList := make([]Validator, len(testmodels.InputValidators))

	for i, inputValidator := range testmodels.InputValidators {
		validatorsList[i] = *NewValidator(common.BlsPubKey{}, inputValidator, nil)
	}

	cache.addAllValidatorsByEpoch(epoch, validatorsList)

	addresses := cache.getValidatorsAddresses(bc, epoch, false)
	testutils.AssertEqual(t, testmodels.InputValidators, addresses)

	for i := 0; i < len(validatorsList); i++ {
		currentEpoch := uint64(i)
		cache.addAllValidatorsByEpoch(currentEpoch, validatorsList)
		addresses = cache.getValidatorsAddresses(bc, currentEpoch, true)

		activeValidators := make([]common.Address, 0)
		for _, v := range validatorsList {
			if v.ActivationEra <= currentEpoch && v.ExitEra > currentEpoch {
				activeValidators = append(activeValidators, v.Address)
			}
		}

		testutils.AssertEqual(t, activeValidators, addresses)
	}
}

func TestShuffledValidatorsCache(t *testing.T) {
	cache := NewCache()

	err := cache.addShuffledValidators(shuffledValidators, []uint64{epoch})
	testutils.AssertNoError(t, err)
	shuffleList, err := cache.getShuffledValidators([]uint64{epoch, slot})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffleList, shuffledValidators[slot])

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch + 1, slot})
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, shuffleList)

	for i := uint64(0); i < cacheCapacity*2; i++ {
		err = cache.addShuffledValidators(shuffledValidators, []uint64{i})
		testutils.AssertNoError(t, err)
	}
	err = cache.addShuffledValidators(shuffledValidators, []uint64{epoch})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cacheCapacity, len(cache.shuffledValidatorsCache))

	err = cache.addShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
	testutils.AssertNoError(t, err)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffleList, shuffledValidators[slot])

	err = cache.addShuffledValidators(shuffledValidators, []uint64{epoch, epoch, subnet, subnet})
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)

	cache.shuffledSubnetValidatorsCache[epoch] = make(map[uint64][][]common.Address, cacheCapacity)
	for i := uint64(0); i <= cacheCapacity; i++ {
		err = cache.addShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
		testutils.AssertNoError(t, err)
		epoch = epoch + i
	}

	err = cache.addShuffledValidators(shuffledValidators, []uint64{epoch, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, cacheCapacity, len(cache.shuffledSubnetValidatorsCache))

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledValidators[slot], shuffleList)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch + 1, slot, subnet})
	testutils.AssertError(t, err, errNoEpochValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet + 1})
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet, epoch, subnet, slot})
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, shuffleList)
}