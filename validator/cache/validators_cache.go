package cache

import (
	"errors"
	"math"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

var (
	ErrInvalidValidatorsFilter = errors.New("invalid validators filter")
	errNoSlotValidators        = errors.New("there are no validators for slot")
	errNoSubnetValidators      = errors.New("there are no validators for subnet")
	errNoEpochValidators       = errors.New("there are no validators for epoch")
	errNoCachedValidators      = errors.New("there are no cached validators")
)

const (
	cacheCapacity = 10
)

type ValidatorsCache struct {
	allValidatorsCache            map[uint64][]Validator                   // epoch/array of validators
	subnetValidatorsCache         map[uint64]map[uint64][]common.Address   // epoch/subnet/validators array
	shuffledValidatorsCache       map[uint64][][]common.Address            // epoch/array of validators arrays (slot is the index in array)
	shuffledSubnetValidatorsCache map[uint64]map[uint64][][]common.Address // epoch/subnet/array of validators arrays (slot is the index in array)

	allMu            *sync.Mutex
	subnetMu         *sync.Mutex
	shuffledMu       *sync.Mutex
	shuffledSubnetMu *sync.Mutex
}

func New() *ValidatorsCache {
	return &ValidatorsCache{
		allValidatorsCache:            make(map[uint64][]Validator),
		subnetValidatorsCache:         make(map[uint64]map[uint64][]common.Address),
		shuffledValidatorsCache:       make(map[uint64][][]common.Address, 0),
		shuffledSubnetValidatorsCache: make(map[uint64]map[uint64][][]common.Address),
		allMu:                         new(sync.Mutex),
		subnetMu:                      new(sync.Mutex),
		shuffledMu:                    new(sync.Mutex),
		shuffledSubnetMu:              new(sync.Mutex),
	}
}

func (c *ValidatorsCache) AddAllValidatorsByEpoch(epoch uint64, validatorsList []Validator) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	if len(c.allValidatorsCache) == cacheCapacity {
		needDel := uint64(math.MaxUint64)
		for e := range c.allValidatorsCache {
			if e < needDel {
				needDel = e
			}
		}

		delete(c.allValidatorsCache, needDel)
	}

	c.allValidatorsCache[epoch] = validatorsList
}

func (c *ValidatorsCache) GetAllValidatorsByEpoch(epoch uint64) ([]Validator, error) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators, ok := c.allValidatorsCache[epoch]
	if !ok {
		return nil, errNoCachedValidators
	}

	return validators, nil
}

func (c *ValidatorsCache) GetActiveValidatorsByEpoch(epoch uint64) []Validator {
	validators := make([]Validator, 0)
	validatorsList, ok := c.allValidatorsCache[epoch]
	if !ok {
		log.Error(errNoEpochValidators.Error(), "epoch", epoch)
		return nil
	}

	for _, validator := range validatorsList {
		if validator.ActivationEpoch <= epoch && validator.ExitEpoch > epoch {
			validators = append(validators, validator)
		}
	}

	return validators
}

func (c *ValidatorsCache) AddSubnetValidators(epoch, subnet uint64, validators []common.Address) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	c.subnetValidatorsCache[epoch][subnet] = validators
}

func (c *ValidatorsCache) GetSubnetValidators(epoch, subnet uint64) ([]common.Address, error) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	epochValidators, ok := c.subnetValidatorsCache[epoch]
	if !ok {
		return nil, errNoSubnetValidators
	}

	return epochValidators[subnet], nil
}

func (c *ValidatorsCache) AddValidator(validator Validator, epoch uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	c.allValidatorsCache[epoch] = append(c.allValidatorsCache[epoch], validator)
}

func (c *ValidatorsCache) DelValidator(validator Validator, epoch uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	for i, v := range c.allValidatorsCache[epoch] {
		if v.Address == validator.Address {
			c.allValidatorsCache[epoch] = append(c.allValidatorsCache[epoch][:i], c.allValidatorsCache[epoch][i+1:]...)

		}
	}
}

// AddShuffledValidators set shuffled validators addresses to the cache.
// Input parameters are array of uint64 (epoch, subnet). Sequence is required!!!
func (c *ValidatorsCache) AddShuffledValidators(shuffledValidators [][]common.Address, filter []uint64) error {

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
		if len(c.shuffledSubnetValidatorsCache[filter[0]]) == cacheCapacity {
			needDel := uint64(math.MaxUint64)
			for e := range c.shuffledSubnetValidatorsCache {
				if e < needDel {
					needDel = e
				}
			}

			delete(c.shuffledSubnetValidatorsCache[filter[0]], needDel)
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

// GetShuffledValidators return shuffled validators addresses from cache.
// Input parameters are array of uint64 (epoch, slot, subnet). Sequence is required!!!
func (c *ValidatorsCache) GetShuffledValidators(filter []uint64) ([]common.Address, error) {
	var epoch, slot, subnet uint64

	switch len(filter) {
	case 2:
		epoch = filter[0]
		slot = filter[1]

		epochValidators, ok := c.shuffledValidatorsCache[epoch]
		if !ok {
			return nil, errNoEpochValidators
		}

		return epochValidators[slot], nil
	case 3:
		epoch = filter[0]
		slot = filter[1]
		subnet = filter[2]
		subnetValidators, ok := c.shuffledSubnetValidatorsCache[subnet]
		if !ok {
			return nil, errNoSubnetValidators
		}

		epochValidators, ok := subnetValidators[epoch]
		if !ok {
			return nil, errNoEpochValidators
		}

		return epochValidators[slot], nil
	default:
		return nil, ErrInvalidValidatorsFilter
	}
}

func (c *ValidatorsCache) GetValidatorsAddresses(epoch uint64, activeOnly bool) []common.Address {
	addresses := make([]common.Address, 0)
	validators := c.allValidatorsCache[epoch]

	if !activeOnly {
		for _, validator := range validators {
			addresses = append(addresses, validator.Address)
		}

		return addresses
	}

	for _, validator := range validators {
		if validator.ActivationEpoch <= epoch && validator.ExitEpoch > epoch {
			addresses = append(addresses, validator.Address)
		}
	}

	return addresses
}
