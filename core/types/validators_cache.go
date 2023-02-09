package types

import (
	"errors"
	"sync"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

var (
	errNoSlotCreators   = errors.New("there are no creators for slot")
	errNoSubnetCreators = errors.New("there are no creators for subnet")
	errNoEpochCreators  = errors.New("there are no creators for epoch")
)

type ValidatorsCache struct {
	allValidatorsCache            map[uint64][]Validator                   // epoch/array of validators
	subnetValidatorsCache         map[uint64][]common.Address              // subnet/validators array
	shuffledValidatorsCache       map[uint64][][]common.Address            // epoch/array of validators arrays (slot is the index in array)
	shuffledSubnetValidatorsCache map[uint64]map[uint64][][]common.Address // subnet/epoch/array of validators arrays (slot is the index in array)

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

func (c *ValidatorsCache) AddAllValidatorsByEpoch(epoch uint64, validatorsList []Validator) {
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

func (c *ValidatorsCache) AddShuffledValidators(epoch uint64, shuffledValidators [][]common.Address) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	c.shuffledValidatorsCache[epoch] = shuffledValidators
}

func (c *ValidatorsCache) GetShuffledValidatorsByEpoch(epoch uint64) ([][]common.Address, error) {
	c.shuffledMu.Lock()
	defer c.shuffledMu.Unlock()

	epochValidators, ok := c.shuffledValidatorsCache[epoch]
	if !ok {
		return nil, errNoEpochCreators
	}

	return epochValidators, nil
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

	subnetValidators, ok := c.shuffledSubnetValidatorsCache[subnet]
	if !ok {
		return nil, errNoSubnetCreators
	}

	return subnetValidators, nil
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
