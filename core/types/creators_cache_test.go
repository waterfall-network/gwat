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
	creatorsCache := NewCreatorsCache()
	if creatorsCache.allCreatorsCache == nil {
		t.Fatal("allCreatorsCache not initialized")
	}
	if creatorsCache.subnetCreatorsCache == nil {
		t.Fatal("subnetCreatorsCache not initialized")
	}
	if creatorsCache.shuffledCreatorsCache == nil {
		t.Fatal("shuffledCreatorsCache not initialized")
	}
	if creatorsCache.shuffledSubnetCreatorsCache == nil {
		t.Fatal("shuffledSubnetCreatorsCache not initialized")
	}
}

func TestAllCreatorsCache(t *testing.T) {
	c := NewCreatorsCache()

	c.AddAllCreators(&inputCreators)

	creators := c.GetAllCreators()
	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creators)

	indexes := c.GetAllCreatorsIndexes()
	testutils.AssertEqual(t, []int{0, 1, 2}, indexes)

	creatorsByIndexes := c.GetCreatorsByIndexes([]int{1, 2})
	testutils.AssertEqual(t, []common.Address{addr2, addr3}, creatorsByIndexes)
}

func TestSubnetCreatorsCache(t *testing.T) {
	c := NewCreatorsCache()

	creators, err := c.GetSubnetCreators(subnet)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, creators)

	c.AddSubnetCreators(subnet, inputCreators)
	creators, err = c.GetSubnetCreators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creators)
}

func TestShuffledCreatorsCache(t *testing.T) {
	c := NewCreatorsCache()

	creators, err := c.GetShuffledCreatorsByEpoch(epoch)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, creators)

	creatorsBySlot, err := c.GetShuffledCreatorsBySlot(epoch, slot)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, creatorsBySlot)

	c.AddShuffledCreators(epoch, shuffledCreators)

	creatorsBySlot, err = c.GetShuffledCreatorsBySlot(epoch, 5)
	testutils.AssertError(t, err, errNoSlotCreators)
	testutils.AssertNil(t, creatorsBySlot)

	creators, err = c.GetShuffledCreatorsByEpoch(epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, [][]common.Address{{addr1, addr2, addr3}, {addr4, addr5, addr6}}, creators)

	creatorsBySlot, err = c.GetShuffledCreatorsBySlot(epoch, slot)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{addr1, addr2, addr3}, creatorsBySlot)
}

func TestShuffledSubnetCreators(t *testing.T) {
	c := NewCreatorsCache()

	c.AddShuffledSubnetCreators(subnet, epoch, shuffledCreators)

	slotCreators, err := c.GetShuffledSubnetCreatorsBySlot(subnet, epoch, 5)
	testutils.AssertError(t, err, errNoSlotCreators)
	testutils.AssertNil(t, slotCreators)

	slotCreators, err = c.GetShuffledSubnetCreatorsBySlot(subnet, 3, slot)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, slotCreators)

	slotCreators, err = c.GetShuffledSubnetCreatorsBySlot(3, epoch, slot)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, slotCreators)

	epochCreators, err := c.GetShuffledSubnetCreatorsByEpoch(subnet, 3)
	testutils.AssertError(t, err, errNoEpochCreators)
	testutils.AssertNil(t, epochCreators)

	epochCreators, err = c.GetShuffledSubnetCreatorsByEpoch(1, epoch)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, epochCreators)

	subnetCreators, err := c.GetShuffledSubnetCreators(1)
	testutils.AssertError(t, err, errNoSubnetCreators)
	testutils.AssertNil(t, subnetCreators)

	subnetCreators, err = c.GetShuffledSubnetCreators(subnet)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, map[uint64][][]common.Address{1: shuffledCreators}, subnetCreators)

	epochCreators, err = c.GetShuffledSubnetCreatorsByEpoch(subnet, epoch)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledCreators, epochCreators)

	slotCreators, err = c.GetShuffledSubnetCreatorsBySlot(subnet, epoch, 0)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, shuffledCreators[0], slotCreators)
}
