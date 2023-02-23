package validator

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/cache"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/shuffle"
)

type Consensus interface {
	GetValidators(stateDb *state.StateDB, epoch uint64, activeOnly, needAddresses bool) ([]cache.Validator, []common.Address)
	GetShuffledValidators(stateDb *state.StateDB, firstEpochBlock common.Hash, filter ...uint64) ([]common.Address, error)

	breakByValidatorsBySlotCount(validators []common.Address, validatorsPerSlot uint64) [][]common.Address
}

type consensus struct {
	db         ethdb.Database
	validators *cache.ValidatorsCache
	config     *params.ChainConfig
}

func NewConsensus(db ethdb.Database, config *params.ChainConfig) Consensus {
	return &consensus{
		db:         db,
		validators: cache.New(),
		config:     config,
	}
}

func (c *consensus) ValidatorsStateAddress() *common.Address {
	return c.config.ValidatorsStateAddress
}

// GetValidators return two values: array of Validator and array of Validators addresses.
// If parameter needAddresses is false it return array of Validator and nil value for validators addresses.
// Use parameter activeOnly true if you need only active validators.
func (c *consensus) GetValidators(stateDb *state.StateDB, epoch uint64, activeOnly, needAddresses bool) ([]cache.Validator, []common.Address) {
	var err error
	validators := make([]cache.Validator, 0)

	validators, err = c.validators.GetAllValidatorsByEpoch(epoch)
	if err != nil {
		valList := stateDb.GetValidatorsList(c.ValidatorsStateAddress())
		for _, valAddress := range valList {
			var validator cache.Validator
			valInfo := stateDb.GetValidatorInfo(valAddress)
			err = validator.UnmarshalBinary(valInfo)
			if err != nil {
				log.Error("can`t get validators info from state", "error", err)
				continue
			}

			validators = append(validators, validator)
		}

		c.validators.AddAllValidatorsByEpoch(epoch, validators)

	}

	switch {
	case !activeOnly && !needAddresses:
		return validators, nil
	case !activeOnly && needAddresses:
		return validators, c.validators.GetValidatorsAddresses(epoch, false)
	case activeOnly && !needAddresses:
		return c.validators.GetActiveValidatorsByEpoch(epoch), nil
	case activeOnly && needAddresses:
		return c.validators.GetActiveValidatorsByEpoch(epoch), c.validators.GetValidatorsAddresses(epoch, true)
	default:
		return nil, nil
	}
}

// GetShuffledValidators return shuffled validators addresses from cache.
// Input parameters are array of uint64 (epoch, slot, subnet). Sequence is required!!!
func (c *consensus) GetShuffledValidators(stateDb *state.StateDB, firstEpochBlock common.Hash, filter ...uint64) ([]common.Address, error) {
	// TODO: improve this function for subnet supporting.
	params := make([]uint64, len(filter))
	for i, u := range filter {
		params[i] = u
	}

	validators, err := c.validators.GetShuffledValidators(params)
	if err != nil && err == cache.ErrInvalidValidatorsFilter {
		return nil, err
	} else if err == nil {
		return validators, nil
	}

	valList := make([]cache.Validator, 0)

	stateValidators := stateDb.GetValidatorsList(c.ValidatorsStateAddress())
	for _, valAddress := range stateValidators {
		var validator cache.Validator
		valInfo := stateDb.GetValidatorInfo(valAddress)
		err = validator.UnmarshalBinary(valInfo)
		if err != nil {
			log.Error("can`t get validators info from state", "error", err)
			continue
		}

		valList = append(valList, validator)
	}

	c.validators.AddAllValidatorsByEpoch(params[0], valList)

	activeEpochValidators := c.validators.GetValidatorsAddresses(params[0], true)

	seed, err := c.seed(params[0], firstEpochBlock)
	if err != nil {
		return nil, err
	}

	shuffledValidators, err := shuffle.ShuffleValidators(activeEpochValidators, seed)
	if err != nil {
		return nil, err
	}

	shuffledValidatorsBySlots := c.breakByValidatorsBySlotCount(shuffledValidators, c.config.ValidatorsPerSlot)

	if uint64(len(shuffledValidatorsBySlots)) < c.config.SlotsPerEpoch {
		for uint64(len(shuffledValidatorsBySlots)) < c.config.SlotsPerEpoch {
			shuffledValidators, err = shuffle.ShuffleValidators(shuffledValidators, seed)
			if err != nil {
				return nil, err
			}

			shuffledValidatorsBySlots = append(shuffledValidatorsBySlots, c.breakByValidatorsBySlotCount(shuffledValidators, c.config.ValidatorsPerSlot)...)
		}
	}

	err = c.validators.AddShuffledValidators(shuffledValidatorsBySlots, params[0:1])
	if err != nil {
		log.Error("can`t add shuffled validators to cache", "error", err)
	}

	return c.validators.GetShuffledValidators(params)
}

// seed make Seed for shuffling represents in [32] byte
// Seed = hash of the first finalized block in the epoch finalized two epoch ago + epoch number represents in [32] byte
func (c *consensus) seed(epoch uint64, firstEpochBlockHash common.Hash) (common.Hash, error) {
	epochBytes := shuffle.Bytes8(epoch)

	seed := crypto.Keccak256(append(firstEpochBlockHash.Bytes(), epochBytes...))

	seed32 := common.BytesToHash(seed)

	return seed32, nil

}

// breakByValidatorsBySlotCount splits the list of all validators into sublists for each slot
func (c *consensus) breakByValidatorsBySlotCount(validators []common.Address, validatorsPerSlot uint64) [][]common.Address {
	chunks := make([][]common.Address, 0)

	for i := uint64(0); i+validatorsPerSlot <= uint64(len(validators)); i += validatorsPerSlot {
		end := i + validatorsPerSlot
		slotValidators := make([]common.Address, len(validators[i:end]))
		copy(slotValidators, validators[i:end])
		chunks = append(chunks, slotValidators)
		if len(chunks) == int(c.config.SlotsPerEpoch) {
			return chunks
		}
	}

	return chunks
}
