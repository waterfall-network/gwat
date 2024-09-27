// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
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

	if cache.allActiveValidatorsCache == nil {
		t.Fatal("Expected allActiveValidatorsCache to be initialized, but it was nil")
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

func TestGetActiveValidatorsByEra(t *testing.T) {
	cache := NewCache()

	validators := cache.getAllActiveValidatorsByEra(1)
	if validators != nil {
		t.Fatal("Expected nil, got", validators)
	}

	validatorsList := []common.Address{testmodels.Addr1, testmodels.Addr2, testmodels.Addr3}
	cache.addAllActiveValidatorsByEra(10, validatorsList)

	expectedValidators := validatorsList
	validators = cache.getAllActiveValidatorsByEra(10)
	testutils.AssertEqual(t, expectedValidators, validators)

	cache.addAllActiveValidatorsByEra(30, validatorsList)
	validators = cache.getAllActiveValidatorsByEra(30)
	testutils.AssertEqual(t, expectedValidators, validators)
}

func TestAddValidator(t *testing.T) {
	era := uint64(10)

	c := NewCache()

	c.addValidator(testmodels.Addr1, era)
	validators := c.getAllActiveValidatorsByEra(era)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}
	if validators[0] != testmodels.Addr1 {
		t.Fatalf("Expected address %v but got %v", testmodels.Addr1, validators[0])
	}
}

func TestDelValidator(t *testing.T) {
	era := uint64(10)

	c := NewCache()
	c.addValidator(testmodels.Addr10, era)

	validators := c.getAllActiveValidatorsByEra(era)
	if len(validators) != 1 {
		t.Fatalf("Expected 1 validator but got %v", len(validators))
	}

	c.delValidator(testmodels.Addr10, era)
	validators = c.getAllActiveValidatorsByEra(era)
	if len(validators) != 0 {
		t.Fatalf("Expected 0 validators but got %v", len(validators))
	}
}

func TestAllValidatorsCache(t *testing.T) {
	cache := NewCache()
	cache.addAllActiveValidatorsByEra(epoch, testmodels.InputValidators)

	cachedValidatorsList, ok := cache.allActiveValidatorsCache[epoch]
	if !ok {
		t.Errorf("Expected validators list for epoch %d to be cached, but it wasn't", epoch)
	}

	testutils.AssertEqual(t, cachedValidatorsList, testmodels.InputValidators)

	cachedValidatorsList = cache.getAllActiveValidatorsByEra(epoch)
	testutils.AssertEqual(t, cachedValidatorsList, testmodels.InputValidators)

	cachedValidatorsList = cache.getAllActiveValidatorsByEra(epoch + 1)
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
	testutils.AssertError(t, err, errNoEraValidators)
	testutils.AssertNil(t, cachedValidators)

	cachedValidators, err = cache.getSubnetValidators(epoch, subnet+1)
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, cachedValidators)
}

func TestShuffledValidatorsCache(t *testing.T) {
	cache := NewCache()

	err := cache.addShuffledValidators(shuffledValidators, []uint64{epoch})
	testutils.AssertNoError(t, err)
	shuffleList, err := cache.getShuffledValidators([]uint64{epoch, slot})
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffleList, shuffledValidators[slot])

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch + 1, slot})
	testutils.AssertError(t, err, errNoEraValidators)
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
	testutils.AssertError(t, err, errNoEraValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet + 1})
	testutils.AssertError(t, err, errNoSubnetValidators)
	testutils.AssertNil(t, shuffleList)

	shuffleList, err = cache.getShuffledValidators([]uint64{epoch, slot, subnet, epoch, subnet, slot})
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, shuffleList)
}
