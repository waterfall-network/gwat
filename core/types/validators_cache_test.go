package types

import (
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

	inputCreators    = []common.Address{addr1, addr2, addr3}
	shuffledCreators = [][]common.Address{
		{
			addr1,
			addr2,
			addr3,
		},
		{
			addr4,
			addr5,
			addr6,
		},
	}

	subnet = uint64(2)
	epoch  = uint64(1)
	slot   = uint64(0)
)

func TestNewCreatorsCache(t *testing.T) {
	creatorsCache := NewValidatorsCache()
	if creatorsCache.allValidatorsCache == nil {
		t.Fatal("allValidatorsCache not initialized")
	}
	if creatorsCache.subnetValidatorsCache == nil {
		t.Fatal("subnetValidatorsCache not initialized")
	}
	if creatorsCache.shuffledValidatorsCache == nil {
		t.Fatal("shuffledValidatorsCache not initialized")
	}
	if creatorsCache.shuffledSubnetValidatorsCache == nil {
		t.Fatal("shuffledSubnetValidatorsCache not initialized")
	}
}

// TODO: FIX TEST
//func TestAllCreatorsCache(t *testing.T) {
//	c := NewValidatorsCache()
//
//	c.AddAllValidatorsByEpoch(epoch,&inputCreators)
//
//	creators := c.GetAllValidatorsByEpoch()
//	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creators)
//
//	indexes := c.GetValidatorsIndexes()
//	testutils.AssertEqual(t, []int{0, 1, 2}, indexes)
//
//	creatorsByIndexes := c.GetValidatorsByIndexes([]int{1, 2})
//	testutils.AssertEqual(t, []common.Address{addr2, addr3}, creatorsByIndexes)
//}

func TestSubnetCreatorsCache(t *testing.T) {
	c := NewValidatorsCache()

	creators, err := c.GetSubnetValidators(subnet)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, creators)

	c.AddSubnetValidators(subnet, inputCreators)
	creators, err = c.GetSubnetValidators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creators)
}

func TestShuffledCreatorsCache(t *testing.T) {
	c := NewValidatorsCache()

	creators, err := c.GetShuffledValidatorsByEpoch(epoch)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, creators)

	creatorsBySlot, err := c.GetShuffledValidatorsBySlot(epoch, slot)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, creatorsBySlot)

	c.AddShuffledValidators(epoch, shuffledCreators)

	creatorsBySlot, err = c.GetShuffledValidatorsBySlot(epoch, 5)
	testutils.AssertError(t, err, errNoSlotCreators)
	testutils.AssertNil(t, creatorsBySlot)

	creators, err = c.GetShuffledValidatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, [][]common.Address{{addr1, addr2, addr3}, {addr4, addr5, addr6}}, creators)

	creatorsBySlot, err = c.GetShuffledValidatorsBySlot(epoch, slot)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creatorsBySlot)
}

func TestShuffledSubnetCreators(t *testing.T) {
	c := NewValidatorsCache()

	c.AddShuffledSubnetValidators(subnet, epoch, shuffledCreators)

	slotCreators, err := c.GetShuffledSubnetValidatorsBySlot(subnet, epoch, 5)
	testutils.AssertError(t, err, errNoSlotCreators)
	testutils.AssertNil(t, slotCreators)

	slotCreators, err = c.GetShuffledSubnetValidatorsBySlot(subnet, 3, slot)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, slotCreators)

	slotCreators, err = c.GetShuffledSubnetValidatorsBySlot(3, epoch, slot)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, slotCreators)

	epochCreators, err := c.GetShuffledSubnetValidatorsByEpoch(subnet, 3)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, epochCreators)

	epochCreators, err = c.GetShuffledSubnetValidatorsByEpoch(1, epoch)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, epochCreators)

	subnetCreators, err := c.GetShuffledSubnetValidators(1)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, subnetCreators)

	subnetCreators, err = c.GetShuffledSubnetValidators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, map[uint64][][]common.Address{1: shuffledCreators}, subnetCreators)

	epochCreators, err = c.GetShuffledSubnetValidatorsByEpoch(subnet, epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledCreators, epochCreators)

	slotCreators, err = c.GetShuffledSubnetValidatorsBySlot(subnet, epoch, 0)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledCreators[0], slotCreators)
}
