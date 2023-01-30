package types

import (
	"errors"
	"sort"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var (
	errNoSlotCreators   = errors.New("there are no creators for slot")
	errNoSubnetCreators = errors.New("there are no creators for subnet")
	errNoEpochCreators  = errors.New("there are no creators for epoch")
)

type CreatorsCache struct {
	allCreatorsCache            map[int]common.Address                   // index/creator address
	subnetCreatorsCache         map[uint64][]common.Address              // subnet/creators arrey
	shuffledCreatorsCache       map[uint64][][]common.Address            // epoch/array of creators arrays (slot is the index in it)
	shuffledSubnetCreatorsCache map[uint64]map[uint64][][]common.Address // subnet/epoch/array of creators arrays (slot is the index in it)

	allMu            *sync.Mutex
	subnetMu         *sync.Mutex
	shuffledMu       *sync.Mutex
	shuffledSubnetMu *sync.Mutex
}

func NewCreatorsCache() *CreatorsCache {
	return &CreatorsCache{
		allCreatorsCache:            make(map[int]common.Address),
		subnetCreatorsCache:         make(map[uint64][]common.Address),
		shuffledCreatorsCache:       make(map[uint64][][]common.Address, 0),
		shuffledSubnetCreatorsCache: make(map[uint64]map[uint64][][]common.Address),
		allMu:                       new(sync.Mutex),
		subnetMu:                    new(sync.Mutex),
		shuffledMu:                  new(sync.Mutex),
		shuffledSubnetMu:            new(sync.Mutex),
	}
}

func (c *CreatorsCache) AddAllCreators(creators *[]common.Address) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	for index, creator := range *creators {
		c.allCreatorsCache[index] = creator
	}
}

func (c *CreatorsCache) GetAllCreators() []common.Address {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	creators := make([]common.Address, len(c.allCreatorsCache), len(c.allCreatorsCache))
	for index, creator := range c.allCreatorsCache {
		creators[index] = creator
	}

	return creators
}

func (c *CreatorsCache) GetAllCreatorsIndexes() []int {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	indexes := make([]int, len(c.allCreatorsCache), len(c.allCreatorsCache))
	for index := range c.allCreatorsCache {
		indexes[index] = index
	}

	sort.Ints(indexes)

	return indexes
}

func (c *CreatorsCache) GetCreatorsByIndexes(indexes []int) []common.Address {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	creators := make([]common.Address, 0)
	for _, index := range indexes {
		creators = append(creators, c.allCreatorsCache[index])
	}

	return creators
}

func (c *CreatorsCache) AddSubnetCreators(subnet uint64, creators []common.Address) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	c.subnetCreatorsCache[subnet] = creators
}

func (c *CreatorsCache) GetSubnetCreators(subnet uint64) ([]common.Address, error) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	subnetCreators, ok := c.subnetCreatorsCache[subnet]
	if !ok {
		return nil, errNoSubnetCreators
	}

	return subnetCreators, nil
}

func (c *CreatorsCache) AddShuffledCreators(epoch uint64, shuffledCreators [][]common.Address) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	c.shuffledCreatorsCache[epoch] = shuffledCreators
}

func (c *CreatorsCache) GetShuffledCreatorsByEpoch(epoch uint64) ([][]common.Address, error) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	epochCreators, ok := c.shuffledCreatorsCache[epoch]
	if !ok {
		return nil, errNoEpochCreators
	}

	return epochCreators, nil
}

func (c *CreatorsCache) GetShuffledCreatorsBySlot(epoch, slot uint64) ([]common.Address, error) {
	epochCreators, err := c.GetShuffledCreatorsByEpoch(epoch)
	if err != nil {
		return nil, err
	}

	if slot > uint64(len(epochCreators)) {
		return nil, errNoSlotCreators
	}

	return epochCreators[slot], nil
}

func (c *CreatorsCache) AddShuffledSubnetCreators(subnet, epoch uint64, shuffledCreators [][]common.Address) {
	c.shuffledSubnetMu.Lock()
	defer c.shuffledSubnetMu.Unlock()

	if _, ok := c.shuffledSubnetCreatorsCache[subnet]; !ok {
		c.shuffledSubnetCreatorsCache[subnet] = map[uint64][][]common.Address{}
	}

	c.shuffledSubnetCreatorsCache[subnet][epoch] = shuffledCreators
}

func (c *CreatorsCache) GetShuffledSubnetCreators(subnet uint64) (map[uint64][][]common.Address, error) {
	c.shuffledSubnetMu.Lock()
	defer c.shuffledSubnetMu.Unlock()

	subnetCreators, ok := c.shuffledSubnetCreatorsCache[subnet]
	if !ok {
		return nil, errNoSubnetCreators
	}

	return subnetCreators, nil
}

func (c *CreatorsCache) GetShuffledSubnetCreatorsByEpoch(subnet, epoch uint64) ([][]common.Address, error) {
	subnetCreators, err := c.GetShuffledSubnetCreators(subnet)
	if err != nil {
		return nil, err
	}

	epochCreators, ok := subnetCreators[epoch]
	if !ok {
		return nil, errNoEpochCreators
	}

	return epochCreators, nil
}

func (c *CreatorsCache) GetShuffledSubnetCreatorsBySlot(subnet, epoch, slot uint64) ([]common.Address, error) {
	epochCreators, err := c.GetShuffledSubnetCreatorsByEpoch(subnet, epoch)
	if err != nil {
		return nil, err
	}

	if slot > uint64(len(epochCreators)) {
		return nil, errNoSlotCreators
	}

	return epochCreators[slot], nil
}
