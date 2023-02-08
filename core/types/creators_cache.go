package types

import (
	"errors"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"sort"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var (
	errNoSlotCreators   = errors.New("there are no creators for slot")
	errNoSubnetCreators = errors.New("there are no creators for subnet")
	errNoEpochCreators  = errors.New("there are no creators for epoch")
)

type ValidatorsCache struct {
	allValidatorsCache            map[uint64][]Validator                   // epoch/array of validators
	subnetValidatorsCache         map[uint64][]common.Address              // subnet/validators arrey
	shuffledValidatorsCache       map[uint64][][]common.Address            // epoch/array of validators arrays (slot is the index in it)
	shuffledSubnetValidatorsCache map[uint64]map[uint64][][]common.Address // subnet/epoch/array of validators arrays (slot is the index in it)

	allMu            *sync.Mutex
	subnetMu         *sync.Mutex
	shuffledMu       *sync.Mutex
	shuffledSubnetMu *sync.Mutex
}

func NewValidatorsCache() *ValidatorsCache {
	return &ValidatorsCache{
		allValidatorsCache:            make(map[uint64][]Validator),
		subnetValidatorsCache:         make(map[uint64][]common.Address),
		shuffledValidatorsCache:       make(map[uint64][][]common.Address, 0),
		shuffledSubnetValidatorsCache: make(map[uint64]map[uint64][][]common.Address),
		allMu:                         new(sync.Mutex),
		subnetMu:                      new(sync.Mutex),
		shuffledMu:                    new(sync.Mutex),
		shuffledSubnetMu:              new(sync.Mutex),
	}
}

func (c *ValidatorsCache) AddAllValidators(epoch uint64, validatorsList []Validator) {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	c.allValidatorsCache[epoch] = validatorsList
}

func (c *ValidatorsCache) GetAllValidatorsByEpoch(epoch uint64) []Validator {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators, ok := c.allValidatorsCache[epoch]
	if !ok {
		log.Error("there are no cached validators", "epoch", epoch)
		return nil
	}
	return validators
}

func (c *ValidatorsCache) GetActiveValidatorsByEpoch(epoch uint64) []Validator {
	validators := make([]Validator, 0)
	validatorsList, ok := c.allValidatorsCache[epoch]
	if !ok {
		log.Error("there are no cached validators", "epoch", epoch)
		return nil
	}

	for _, validator := range validatorsList {
		if validator.ActivationEpoch <= epoch && validator.ExitEpoch > epoch {
			validators = append(validators, validator)
		}
	}

	return validators
}

func (c *ValidatorsCache) GetValidatorsIndexes(validators []Validator) []uint64 {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	indexes := make([]uint64, len(validators))

	for i, validator := range validators {
		indexes[i] = validator.ValidatorIndex
	}

	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})

	return indexes
}

func (c *ValidatorsCache) GetValidatorsByIndexes(epoch uint64, indexes []uint64) []common.Address {
	c.allMu.Lock()
	defer c.allMu.Unlock()

	validators := make([]common.Address, len(indexes))

	epochValidators, ok := c.allValidatorsCache[epoch]
	if !ok {
		log.Error("there are no validators", "epoch", epoch)
		return nil
	}

	for i, index := range indexes {
		for _, validator := range epochValidators {
			if index == validator.ValidatorIndex {
				validators[i] = validator.Address
			}
		}
	}

	return validators
}

func (c *ValidatorsCache) AddSubnetValidators(subnet uint64, validators []common.Address) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	c.subnetValidatorsCache[subnet] = validators
}

func (c *ValidatorsCache) GetSubnetValidators(subnet uint64) ([]common.Address, error) {
	c.subnetMu.Lock()
	defer c.subnetMu.Unlock()

	subnetCreators, ok := c.subnetValidatorsCache[subnet]
	if !ok {
		return nil, errNoSubnetCreators
	}

	return subnetCreators, nil
}

func (c *ValidatorsCache) AddShuffledValidators(epoch uint64, shuffledCreators [][]common.Address) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	c.shuffledValidatorsCache[epoch] = shuffledCreators
}

func (c *ValidatorsCache) GetShuffledValidatorsByEpoch(epoch uint64) ([][]common.Address, error) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	epochCreators, ok := c.shuffledValidatorsCache[epoch]
	if !ok {
		return nil, errNoEpochCreators
	}

	return epochCreators, nil
}

func (c *ValidatorsCache) GetShuffledValidatorsBySlot(epoch, slot uint64) ([]common.Address, error) {
	epochCreators, err := c.GetShuffledValidatorsByEpoch(epoch)
	if err != nil {
		return nil, err
	}

	if slot > uint64(len(epochCreators)) {
		return nil, errNoSlotCreators
	}

	return epochCreators[slot], nil
}

func (c *ValidatorsCache) AddShuffledSubnetValidators(subnet, epoch uint64, shuffledCreators [][]common.Address) {
	c.shuffledSubnetMu.Lock()
	defer c.shuffledSubnetMu.Unlock()

	if _, ok := c.shuffledSubnetValidatorsCache[subnet]; !ok {
		c.shuffledSubnetValidatorsCache[subnet] = map[uint64][][]common.Address{}
	}

	c.shuffledSubnetValidatorsCache[subnet][epoch] = shuffledCreators
}

func (c *ValidatorsCache) GetShuffledSubnetValidators(subnet uint64) (map[uint64][][]common.Address, error) {
	c.shuffledSubnetMu.Lock()
	defer c.shuffledSubnetMu.Unlock()

	subnetCreators, ok := c.shuffledSubnetValidatorsCache[subnet]
	if !ok {
		return nil, errNoSubnetCreators
	}

	return subnetCreators, nil
}

func (c *ValidatorsCache) GetShuffledSubnetValidatorsByEpoch(subnet, epoch uint64) ([][]common.Address, error) {
	subnetCreators, err := c.GetShuffledSubnetValidators(subnet)
	if err != nil {
		return nil, err
	}

	epochCreators, ok := subnetCreators[epoch]
	if !ok {
		return nil, errNoEpochCreators
	}

	return epochCreators, nil
}

func (c *ValidatorsCache) GetShuffledSubnetValidatorsBySlot(subnet, epoch, slot uint64) ([]common.Address, error) {
	epochCreators, err := c.GetShuffledSubnetValidatorsByEpoch(subnet, epoch)
	if err != nil {
		return nil, err
	}

	if slot > uint64(len(epochCreators)) {
		return nil, errNoSlotCreators
	}

	return epochCreators[slot], nil
}
