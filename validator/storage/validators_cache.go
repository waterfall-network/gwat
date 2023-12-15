package storage

import (
	"errors"
	"math"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

var (
	ErrInvalidValidatorsFilter = errors.New("invalid validators filter")
	ErrNoStateValidatorInfo    = errors.New("there is no validator in the state")
	errNoSubnetValidators      = errors.New("there are no validators for subnet")
	errNoEpochValidators       = errors.New("there are no validators for epoch")
	errNoValidators            = errors.New("there ara no validators")
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

func NewCache() *ValidatorsCache {
	return &ValidatorsCache{
		allValidatorsCache:            make(map[uint64][]Validator),
		subnetValidatorsCache:         make(map[uint64]map[uint64][]common.Address),
		shuffledValidatorsCache:       make(map[uint64][][]common.Address),
		shuffledSubnetValidatorsCache: make(map[uint64]map[uint64][][]common.Address),
		allMu:                         new(sync.Mutex),
		subnetMu:                      new(sync.Mutex),
		shuffledMu:                    new(sync.Mutex),
		shuffledSubnetMu:              new(sync.Mutex),
	}
}

func (c *ValidatorsCache) addAllValidatorsByEpoch(epoch uint64, validatorsList []Validator) {
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

func (c *ValidatorsCache) getAllValidatorsByEpoch(epoch uint64) ([]Validator, error) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators, ok := c.allValidatorsCache[epoch]
	if !ok || validators == nil {
		return nil, errNoEpochValidators
	}

	return validators, nil
}

func (c *ValidatorsCache) getActiveValidatorsByEpoch(bc blockchain, epoch uint64) []Validator {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators := make([]Validator, 0)
	validatorsList, ok := c.allValidatorsCache[epoch]
	if !ok {
		log.Warn(errNoEpochValidators.Error(), "epoch", epoch)
		return nil
	}

	era := bc.EpochToEra(epoch)

	for _, validator := range validatorsList {
		if validator.ActivationEra <= era.Number && validator.ExitEra > era.Number {
			validators = append(validators, validator)
		}
	}

	return validators
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
		return nil, errNoEpochValidators
	}

	subnetValidators, ok := epochValidators[subnet]
	if !ok {
		return nil, errNoSubnetValidators
	}

	return subnetValidators, nil
}

//nolint:unused // subnets support
func (c *ValidatorsCache) addValidator(validator Validator, epoch uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	c.allValidatorsCache[epoch] = append(c.allValidatorsCache[epoch], validator)
}

//nolint:unused // subnets support
func (c *ValidatorsCache) delValidator(validator Validator, epoch uint64) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	for i, v := range c.allValidatorsCache[epoch] {
		if v.Address == validator.Address {
			c.allValidatorsCache[epoch] = append(c.allValidatorsCache[epoch][:i], c.allValidatorsCache[epoch][i+1:]...)
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
			return nil, errNoEpochValidators
		}

		return epochValidators[slot], nil
	case 3:
		epoch = filter[0]
		slot = filter[1]
		subnet = filter[2]
		epochValidators, ok := c.shuffledSubnetValidatorsCache[epoch]
		if !ok {
			return nil, errNoEpochValidators
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

func (c *ValidatorsCache) getValidatorsAddresses(bc blockchain, epoch uint64, activeOnly bool) []common.Address {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	addresses := make([]common.Address, 0)
	validators := c.allValidatorsCache[epoch]

	if !activeOnly {
		for _, validator := range validators {
			addresses = append(addresses, validator.Address)
		}

		return addresses
	}

	era := bc.EpochToEra(epoch)

	for _, validator := range validators {
		if validator.ActivationEra <= era.Number && validator.ExitEra > era.Number {
			addresses = append(addresses, validator.Address)
		}
	}

	return addresses
}
