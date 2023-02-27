package storage

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/shuffle"
)

type Storage interface {
	GetValidators(stateDb *state.StateDB, epoch uint64, activeOnly, needAddresses bool) ([]Validator, []common.Address)
	GetShuffledValidators(stateDb *state.StateDB, firstEpochBlock common.Hash, filter ...uint64) ([]common.Address, error)

	SetValidatorInfo(stateDb *state.StateDB, info ValidatorInfo)
	GetValidatorInfo(stateDb *state.StateDB, address common.Address) ValidatorInfo

	SetValidatorsList(stateDb *state.StateDB, list []common.Address)
	GetValidatorsList(stateDb *state.StateDB) []common.Address

	breakByValidatorsBySlotCount(validators []common.Address, validatorsPerSlot uint64) [][]common.Address
}

type storage struct {
	validatorsCache *ValidatorsCache
	db              ethdb.Database
	config          *params.ChainConfig
}

func NewStorage(db ethdb.Database, config *params.ChainConfig) Storage {
	return &storage{
		validatorsCache: NewCache(),
		db:              db,
		config:          config,
	}
}
func (s *storage) ValidatorsStateAddress() *common.Address {
	return s.config.ValidatorsStateAddress
}

func (s *storage) SetValidatorInfo(stateDb *state.StateDB, info ValidatorInfo) {
	stateDb.SetCode(info.GetAddress(), info)
}

func (s *storage) GetValidatorInfo(stateDb *state.StateDB, address common.Address) ValidatorInfo {
	return stateDb.GetCode(address)
}

func (s *storage) SetValidatorsList(stateDb *state.StateDB, list []common.Address) {
	buf := make([]byte, len(list)*common.AddressLength)
	for i, validator := range list {
		beginning := i * common.AddressLength
		end := beginning + common.AddressLength
		copy(buf[beginning:end], validator[:])
	}

	stateDb.SetCode(*s.ValidatorsStateAddress(), buf)
}

func (s *storage) GetValidatorsList(stateDb *state.StateDB) []common.Address {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())

	validators := make([]common.Address, 0)
	for i := 0; i+common.AddressLength <= len(buf); i += common.AddressLength {
		validators = append(validators, common.BytesToAddress(buf[i:i+common.AddressLength]))
	}

	return validators
}

// GetValidators return two values: array of Validator and array of Validators addresses.
// If parameter needAddresses is false it return array of Validator and nil value for validators addresses.
// Use parameter activeOnly true if you need only active validators.
func (s *storage) GetValidators(stateDb *state.StateDB, epoch uint64, activeOnly, needAddresses bool) ([]Validator, []common.Address) {
	var err error
	validators := make([]Validator, 0)

	validators, err = s.validatorsCache.GetAllValidatorsByEpoch(epoch)
	if err != nil {
		valList := s.GetValidatorsList(stateDb)
		for _, valAddress := range valList {
			var validator Validator
			valInfo := s.GetValidatorInfo(stateDb, valAddress)
			err = validator.UnmarshalBinary(valInfo)
			if err != nil {
				log.Error("can`t get validators info from state", "error", err)
				continue
			}

			validators = append(validators, validator)
		}

		s.validatorsCache.AddAllValidatorsByEpoch(epoch, validators)

	}

	switch {
	case !activeOnly && !needAddresses:
		return validators, nil
	case !activeOnly && needAddresses:
		return validators, s.validatorsCache.GetValidatorsAddresses(epoch, false)
	case activeOnly && !needAddresses:
		return s.validatorsCache.GetActiveValidatorsByEpoch(epoch), nil
	case activeOnly && needAddresses:
		return s.validatorsCache.GetActiveValidatorsByEpoch(epoch), s.validatorsCache.GetValidatorsAddresses(epoch, true)
	}

	return nil, nil
}

// GetShuffledValidators return shuffled validators addresses from cache.
// Input parameters are array of uint64 (epoch, slot, subnet). Sequence is required!!!
func (s *storage) GetShuffledValidators(stateDb *state.StateDB, firstEpochBlock common.Hash, filter ...uint64) ([]common.Address, error) {
	// TODO: improve this function for subnet supporting.
	params := make([]uint64, len(filter))
	for i, u := range filter {
		params[i] = u
	}

	validators, err := s.validatorsCache.GetShuffledValidators(params)
	if err != nil && err == ErrInvalidValidatorsFilter {
		return nil, err
	} else if err == nil {
		return validators, nil
	}

	valList := make([]Validator, 0)

	stateValidators := s.GetValidatorsList(stateDb)
	for _, valAddress := range stateValidators {
		var validator Validator
		valInfo := s.GetValidatorInfo(stateDb, valAddress)
		err = validator.UnmarshalBinary(valInfo)
		if err != nil {
			log.Error("can`t get validators info from state", "error", err)
			continue
		}

		valList = append(valList, validator)
	}

	s.validatorsCache.AddAllValidatorsByEpoch(params[0], valList)

	activeEpochValidators := s.validatorsCache.GetValidatorsAddresses(params[0], true)

	seed, err := s.seed(params[0], firstEpochBlock)
	if err != nil {
		return nil, err
	}

	shuffledValidators, err := shuffle.ShuffleValidators(activeEpochValidators, seed)
	if err != nil {
		return nil, err
	}

	shuffledValidatorsBySlots := s.breakByValidatorsBySlotCount(shuffledValidators, s.config.ValidatorsPerSlot)

	if uint64(len(shuffledValidatorsBySlots)) < s.config.SlotsPerEpoch {
		for uint64(len(shuffledValidatorsBySlots)) < s.config.SlotsPerEpoch {
			shuffledValidators, err = shuffle.ShuffleValidators(shuffledValidators, seed)
			if err != nil {
				return nil, err
			}

			shuffledValidatorsBySlots = append(shuffledValidatorsBySlots, s.breakByValidatorsBySlotCount(shuffledValidators, s.config.ValidatorsPerSlot)...)
		}
	}

	err = s.validatorsCache.AddShuffledValidators(shuffledValidatorsBySlots, params[0:1])
	if err != nil {
		log.Error("can`t add shuffled validators to cache", "error", err)
		return nil, err
	}

	return s.validatorsCache.GetShuffledValidators(params)
}

// seed make Seed for shuffling represents in [32] byte
// Seed = hash of the first finalized block in the epoch finalized two epoch ago + epoch number represents in [32] byte
func (s *storage) seed(epoch uint64, firstEpochBlockHash common.Hash) (common.Hash, error) {
	epochBytes := shuffle.Bytes8(epoch)

	seed := crypto.Keccak256(append(firstEpochBlockHash.Bytes(), epochBytes...))

	seed32 := common.BytesToHash(seed)

	return seed32, nil

}

// breakByValidatorsBySlotCount splits the list of all validators into sublists for each slot
func (s *storage) breakByValidatorsBySlotCount(validators []common.Address, validatorsPerSlot uint64) [][]common.Address {
	chunks := make([][]common.Address, 0)

	for i := uint64(0); i+validatorsPerSlot <= uint64(len(validators)); i += validatorsPerSlot {
		end := i + validatorsPerSlot
		slotValidators := make([]common.Address, len(validators[i:end]))
		copy(slotValidators, validators[i:end])
		chunks = append(chunks, slotValidators)
		if len(chunks) == int(s.config.SlotsPerEpoch) {
			return chunks
		}
	}

	return chunks
}
