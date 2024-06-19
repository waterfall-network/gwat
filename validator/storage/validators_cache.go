package storage

import (
	"errors"
	"math"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var (
	ErrInvalidValidatorsFilter = errors.New("invalid validators filter")
	ErrNoStateValidatorInfo    = errors.New("there is no validator in the state")
	errNoSubnetValidators      = errors.New("there are no validators for subnet")
	errNoEraValidators         = errors.New("there are no validators for era")
	errNoValidators            = errors.New("there ara no validators")
	errBadBinaryData           = errors.New("bad binary data")
)

const (
	cacheCapacity = 3
)

type ValidatorsCache struct {
	allActiveValidatorsCache      map[uint64][]common.Address              // era/array of validators
	subnetValidatorsCache         map[uint64]map[uint64][]common.Address   // epoch/subnet/validators array
	shuffledValidatorsCache       map[uint64][][]common.Address            // epoch/array of validators arrays (slot is the index in array)
	shuffledSubnetValidatorsCache map[uint64]map[uint64][][]common.Address // epoch/subnet/array of validators arrays (slot is the index in array)

	allMu            *sync.Mutex
	subnetMu         *sync.Mutex
	shuffledMu       *sync.Mutex
	shuffledSubnetMu *sync.Mutex
}

func NewCache() *ValidatorsCache {
	return &ValidatorsCache{
		allActiveValidatorsCache:      make(map[uint64][]common.Address),
		subnetValidatorsCache:         make(map[uint64]map[uint64][]common.Address),
		shuffledValidatorsCache:       make(map[uint64][][]common.Address),
		shuffledSubnetValidatorsCache: make(map[uint64]map[uint64][][]common.Address),
		allMu:                         new(sync.Mutex),
		subnetMu:                      new(sync.Mutex),
		shuffledMu:                    new(sync.Mutex),
		shuffledSubnetMu:              new(sync.Mutex),
	}
}

func (c *ValidatorsCache) addAllActiveValidatorsByEra(era uint64, validatorsList []common.Address) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	if len(c.allActiveValidatorsCache) == cacheCapacity {
		needDel := uint64(math.MaxUint64)
		for e := range c.allActiveValidatorsCache {
			if e < needDel {
				needDel = e
			}
		}

		delete(c.allActiveValidatorsCache, needDel)
	}

	c.allActiveValidatorsCache[era] = validatorsList
}

func (c *ValidatorsCache) getAllActiveValidatorsByEra(era uint64) ([]common.Address, error) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators, ok := c.allActiveValidatorsCache[era]
	if !ok || validators == nil {
		return nil, errNoEraValidators
	}

	return validators, nil
}

//nolint:unused // subnets support
func (c *ValidatorsCache) addSubnetValidators(epoch, subnet uint64, validators []common.Address) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	_, ok := c.subnetValidatorsCache[epoch]
	if !ok {
		c.subnetValidatorsCache[epoch] = make(map[uint64][]common.Address)
	}

	c.subnetValidatorsCache[epoch][subnet] = validators
}

//nolint:unused // subnets support
func (c *ValidatorsCache) getSubnetValidators(epoch, subnet uint64) ([]common.Address, error) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	epochValidators, ok := c.subnetValidatorsCache[epoch]
	if !ok {
		return nil, errNoEraValidators
	}

	subnetValidators, ok := epochValidators[subnet]
	if !ok {
		return nil, errNoSubnetValidators
	}

	return subnetValidators, nil
}

//nolint:unused // subnets support
func (c *ValidatorsCache) addValidator(addr common.Address, era uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	c.allActiveValidatorsCache[era] = append(c.allActiveValidatorsCache[era], addr)
}

//nolint:unused // subnets support
func (c *ValidatorsCache) delValidator(addr common.Address, era uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	for i, address := range c.allActiveValidatorsCache[era] {
		if address == addr {
			c.allActiveValidatorsCache[era] = append(c.allActiveValidatorsCache[era][:i], c.allActiveValidatorsCache[era][i+1:]...)
		}
	}
}

// addShuffledValidators set shuffled validators addresses to the cache.
// Input parameters are array of uint64 (epoch, subnet). Sequence is required!!!
func (c *ValidatorsCache) addShuffledValidators(shuffledValidators [][]common.Address, filter []uint64) error {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	switch len(filter) {
	case 1:
		if len(c.shuffledValidatorsCache) == cacheCapacity {
			needDel := uint64(math.MaxUint64)
			for e := range c.shuffledValidatorsCache {
				if e < needDel {
					needDel = e
				}
			}

			delete(c.shuffledValidatorsCache, needDel)
		}

		c.shuffledValidatorsCache[filter[0]] = shuffledValidators
	case 2:
		if len(c.shuffledSubnetValidatorsCache) == cacheCapacity {
			needDel := uint64(math.MaxUint64)
			for e := range c.shuffledSubnetValidatorsCache {
				if e < needDel {
					needDel = e
				}
			}

			delete(c.shuffledSubnetValidatorsCache, needDel)
		}
		if c.shuffledSubnetValidatorsCache[filter[0]] == nil {
			c.shuffledSubnetValidatorsCache[filter[0]] = make(map[uint64][][]common.Address)
		}

		c.shuffledSubnetValidatorsCache[filter[0]][filter[1]] = shuffledValidators
	default:
		return ErrInvalidValidatorsFilter
	}

	return nil
}

// getShuffledValidators return shuffled validators addresses from cache.
// Input parameters are array of uint64 (epoch, slot, subnet). Sequence is required!!!
func (c *ValidatorsCache) getShuffledValidators(filter []uint64) ([]common.Address, error) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	var epoch, slot, subnet uint64

	switch len(filter) {
	case 2:
		epoch = filter[0]
		slot = filter[1]

		epochValidators, ok := c.shuffledValidatorsCache[epoch]
		if !ok {
			return nil, errNoEraValidators
		}

		return epochValidators[slot], nil
	case 3:
		epoch = filter[0]
		slot = filter[1]
		subnet = filter[2]
		epochValidators, ok := c.shuffledSubnetValidatorsCache[epoch]
		if !ok {
			return nil, errNoEraValidators
		}

		subnetValidators, ok := epochValidators[subnet]
		if !ok {
			return nil, errNoSubnetValidators
		}

		return subnetValidators[slot], nil
	default:
		return nil, ErrInvalidValidatorsFilter
	}
}
