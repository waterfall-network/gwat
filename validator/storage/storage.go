package storage

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/shuffle"
)

type blockchain interface {
	StateAt(root common.Hash) (*state.StateDB, error)
	GetBlock(hash common.Hash) *types.Block
	GetSlotInfo() *types.SlotInfo
	GetCoordinatedCheckpointEpoch(epoch uint64) uint64
	SearchFirstEpochBlockHashRecursive(epoch uint64) (common.Hash, bool)
}

type Storage interface {
	GetValidators(bc blockchain, slot uint64, activeOnly, needAddresses bool) ([]Validator, []common.Address)
	GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error)

	SetValidatorInfo(stateDb *state.StateDB, info ValidatorInfo)
	GetValidatorInfo(stateDb *state.StateDB, address common.Address) ValidatorInfo

	SetValidatorsList(stateDb *state.StateDB, list []common.Address)
	GetValidatorsList(stateDb *state.StateDB) []common.Address

	EstimateEraLength(bc blockchain, slot uint64) (epochsPerEra uint64)

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
func (s *storage) GetValidators(bc blockchain, slot uint64, activeOnly, needAddresses bool) ([]Validator, []common.Address) {
	var err error
	validators := make([]Validator, 0)

	currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)

	checkpointEpoch := bc.GetCoordinatedCheckpointEpoch(currentEpoch)

	validators, err = s.validatorsCache.getAllValidatorsByEpoch(checkpointEpoch)
	if err != nil {
		firstEpochBlockHash, ok := bc.SearchFirstEpochBlockHashRecursive(checkpointEpoch)
		if !ok {
			rawdb.WriteFirstEpochBlockHash(s.db, checkpointEpoch, firstEpochBlockHash)
		}

		firstEpochBlock := bc.GetBlock(firstEpochBlockHash)

		stateDb, err := bc.StateAt(firstEpochBlock.Root())

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

		s.validatorsCache.addAllValidatorsByEpoch(currentEpoch, validators)
	}

	switch {
	case !activeOnly && !needAddresses:
		return validators, nil
	case !activeOnly && needAddresses:
		return validators, s.validatorsCache.getValidatorsAddresses(currentEpoch, false)
	case activeOnly && !needAddresses:
		return s.validatorsCache.getActiveValidatorsByEpoch(currentEpoch), nil
	case activeOnly && needAddresses:
		return s.validatorsCache.getActiveValidatorsByEpoch(currentEpoch), s.validatorsCache.getValidatorsAddresses(currentEpoch, true)
	}

	return nil, nil
}

// GetCreatorsBySlot return shuffled validators addresses from cache.
// Input parameters are list of uint64 (slot, subnet). Sequence is required!!!
func (s *storage) GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error) {
	// TODO: improve this function for subnet supporting.
	if len(filter) > 2 {
		return nil, ErrInvalidValidatorsFilter
	}

	slot := filter[0]

	params := make([]uint64, 0)
	epoch := bc.GetSlotInfo().SlotToEpoch(slot)
	params = append(params, epoch)

	slotInEpoch := bc.GetSlotInfo().SlotInEpoch(slot)
	params = append(params, slotInEpoch)

	if len(filter) == 2 {
		params = append(params, filter[1])
	}

	validators, err := s.validatorsCache.getShuffledValidators(params)
	if err != nil && err == ErrInvalidValidatorsFilter {
		return nil, err
	} else if err == nil {
		return validators, nil
	}

	allValidators, _ := s.GetValidators(bc, slot, false, false)

	s.validatorsCache.addAllValidatorsByEpoch(epoch, allValidators)

	activeEpochValidators := s.validatorsCache.getValidatorsAddresses(epoch, true)

	checkpointEpoch := bc.GetCoordinatedCheckpointEpoch(epoch)
	seedBlockHash, ok := bc.SearchFirstEpochBlockHashRecursive(checkpointEpoch)
	if !ok {
		rawdb.WriteFirstEpochBlockHash(s.db, checkpointEpoch, seedBlockHash)
	}

	seed, err := s.seed(epoch, seedBlockHash)
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

	err = s.validatorsCache.addShuffledValidators(shuffledValidatorsBySlots, params[0:1])
	if err != nil {
		log.Error("can`t add shuffled validators to cache", "error", err)
		return nil, err
	}

	return s.validatorsCache.getShuffledValidators(params)
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

func (s *storage) EstimateEraLength(bc blockchain, slot uint64) (eraLength uint64) {
	var (
		epochsPerEra       = s.config.EpochsPerEra
		slotsPerEpoch      = bc.GetSlotInfo().SlotsPerEpoch
		validators, _      = s.GetValidators(bc, slot, true, false)
		numberOfValidators = uint64(len(validators))
	)

	eraLength = epochsPerEra * (1 + (numberOfValidators / (epochsPerEra * slotsPerEpoch)))

	return
}
