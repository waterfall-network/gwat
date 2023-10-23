package storage

import (
	"encoding/binary"
	"fmt"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/shuffle"
)

type blockchain interface {
	StateAt(root common.Hash) (*state.StateDB, error)
	GetBlock(hash common.Hash) *types.Block
	GetSlotInfo() *types.SlotInfo
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	GetEpoch(epoch uint64) common.Hash
	EpochToEra(uint64) *era.Era
}

type Storage interface {
	GetValidators(bc blockchain, slot uint64, activeOnly, needAddresses bool, tmpFromWhere string) ([]Validator, []common.Address)
	GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error)

	SetValidator(stateDb vm.StateDB, val *Validator) error
	GetValidator(stateDb vm.StateDB, address common.Address) (*Validator, error)

	SetValidatorsList(stateDb vm.StateDB, list []common.Address)
	GetValidatorsList(stateDb vm.StateDB) []common.Address
	AddValidatorToList(stateDB vm.StateDB, index uint64, validator common.Address)

	GetValidatorsStateAddress() *common.Address
	GetDepositCount(stateDb vm.StateDB) uint64
	IncrementDepositCount(stateDb vm.StateDB)
}

type storage struct {
	validatorsCache *ValidatorsCache
	config          *params.ChainConfig
}

func NewStorage(config *params.ChainConfig) Storage {
	return &storage{
		validatorsCache: NewCache(),
		config:          config,
	}
}
func (s *storage) ValidatorsStateAddress() *common.Address {
	return s.config.ValidatorsStateAddress
}

func (s *storage) SetValidator(stateDb vm.StateDB, val *Validator) error {
	valData, err := val.MarshalBinary()
	if err != nil {
		return err
	}

	stateDb.SetCode(val.Address, valData)
	return nil
}

func (s *storage) GetValidator(stateDb vm.StateDB, address common.Address) (*Validator, error) {
	var valData ValidatorBinary

	valData = stateDb.GetCode(address)
	if valData == nil {
		return nil, ErrNoStateValidatorInfo
	}

	return valData.ToValidator()
}

func (s *storage) SetValidatorsList(stateDb vm.StateDB, list []common.Address) {
	newList := make([]byte, len(list)*common.AddressLength+uint64Size)

	currentList := stateDb.GetCode(*s.ValidatorsStateAddress())
	if len(currentList) > 0 {
		depositCount := binary.BigEndian.Uint64(currentList[:uint64Size])
		binary.BigEndian.PutUint64(newList[:uint64Size], depositCount)
	}

	for i, validator := range list {
		beginning := i*common.AddressLength + uint64Size
		end := beginning + common.AddressLength
		copy(newList[beginning:end], validator[:])
	}

	stateDb.SetCode(*s.ValidatorsStateAddress(), newList)
}

func (s *storage) GetValidatorsList(stateDb vm.StateDB) []common.Address {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())

	validators := make([]common.Address, 0)
	for i := uint64Size; i+common.AddressLength <= len(buf); i += common.AddressLength {
		if (common.BytesToAddress(buf[i:i+common.AddressLength]) != common.Address{}) {
			validators = append(validators, common.BytesToAddress(buf[i:i+common.AddressLength]))
		}
	}

	return validators
}

func (s *storage) GetDepositCount(stateDb vm.StateDB) uint64 {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())
	if buf == nil {
		return 0
	}

	return binary.BigEndian.Uint64(buf[:uint64Size])
}

func (s *storage) IncrementDepositCount(stateDb vm.StateDB) {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())

	if buf == nil {
		buf = make([]byte, uint64Size)
	}
	currentCount := binary.BigEndian.Uint64(buf[:uint64Size])
	currentCount++

	binary.BigEndian.PutUint64(buf[:uint64Size], currentCount)
	stateDb.SetCode(*s.ValidatorsStateAddress(), buf)
}

// GetValidators return two values: array of Validator and array of Validators addresses.
// If parameter needAddresses is false it return array of Validator and nil value for validators addresses.
// Use parameter activeOnly true if you need only active validators.
func (s *storage) GetValidators(bc blockchain, slot uint64, activeOnly, needAddresses bool, tmpFromWhere string) ([]Validator, []common.Address) {
	var err error
	var validators []Validator

	slotEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	validators, err = s.validatorsCache.getAllValidatorsByEpoch(slotEpoch)

	if err != nil {
		eraEra := bc.EpochToEra(slotEpoch)
		log.Info("Get validators", "error", err, "epoch", slotEpoch, "era", eraEra.Number, "root", eraEra.Root.Hex())

		stateDb, _ := bc.StateAt(eraEra.Root)

		valList := s.GetValidatorsList(stateDb)
		log.Info("GetValidators from state", "validators", len(valList), "slot", slot, "epoch", slotEpoch, "root", eraEra.Root.Hex())
		for _, valAddress := range valList {
			val, err := s.GetValidator(stateDb, valAddress)
			if err != nil {
				log.Error("can`t get validator from state", "error", err, "address", valAddress.Hex())
				continue
			}
			validators = append(validators, *val)
		}

		s.validatorsCache.addAllValidatorsByEpoch(slotEpoch, validators)
	}

	log.Info("GetValidators", "callFunc", tmpFromWhere, "all", len(validators),
		"active", len(s.validatorsCache.getActiveValidatorsByEpoch(bc, slotEpoch)),
		"slot", slot, "epoch", slotEpoch,
	)

	switch {
	case !activeOnly && !needAddresses:
		return validators, nil
	case !activeOnly && needAddresses:
		return validators, s.validatorsCache.getValidatorsAddresses(bc, slotEpoch, false)
	case activeOnly && !needAddresses:
		return s.validatorsCache.getActiveValidatorsByEpoch(bc, slotEpoch), nil
	case activeOnly && needAddresses:
		return s.validatorsCache.getActiveValidatorsByEpoch(bc, slotEpoch), s.validatorsCache.getValidatorsAddresses(bc, slotEpoch, true)
	}

	return nil, nil
}

// GetCreatorsBySlot return shuffled validators addresses from cache.
// Input parameters are list of uint64 (slot, subnet). Sequence is required!!!
func (s *storage) GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error) {
	// TODO: improve this function for subnet supporting.
	start := time.Now()

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

	allValidators, _ := s.GetValidators(bc, slot, false, false, "GetCreatorsBySlot")
	if len(allValidators) == 0 {
		return nil, errNoValidators
	}

	s.validatorsCache.addAllValidatorsByEpoch(epoch, allValidators)

	activeEpochValidators := s.validatorsCache.getValidatorsAddresses(bc, epoch, true)

	spineEpoch := uint64(0)
	if epoch > 0 {
		spineEpoch = epoch - 1
	}
	epochSpine := bc.GetEpoch(spineEpoch)
	if epochSpine == (common.Hash{}) {
		return nil, fmt.Errorf("epoch not found")
	}
	seed, err := s.seed(epoch, epochSpine)
	if err != nil {
		return nil, err
	}

	log.Info("CheckShuffle - shuffle params", "slot", slot,
		"slotEpoch", epoch,
		"spineEpoch", spineEpoch,
		"spine", epochSpine.Hex(),
		"seed", seed.Hex(),
		"validatorsCount", len(activeEpochValidators),
	)

	shuffledValidators, err := shuffle.ShuffleValidators(activeEpochValidators, seed)
	if err != nil {
		return nil, err
	}

	shuffledValidatorsBySlots := breakByValidatorsBySlotCount(shuffledValidators, s.config.ValidatorsPerSlot, s.config.SlotsPerEpoch)

	if uint64(len(shuffledValidatorsBySlots)) < s.config.SlotsPerEpoch {
		for uint64(len(shuffledValidatorsBySlots)) < s.config.SlotsPerEpoch {
			shuffledValidators, err = shuffle.ShuffleValidators(shuffledValidators, seed)
			if err != nil {
				return nil, err
			}

			shuffledValidatorsBySlots = append(shuffledValidatorsBySlots, breakByValidatorsBySlotCount(shuffledValidators, s.config.ValidatorsPerSlot, s.config.SlotsPerEpoch)...)
		}
	}

	err = s.validatorsCache.addShuffledValidators(shuffledValidatorsBySlots, params[0:1])
	if err != nil {
		log.Error("can`t add shuffled validators to cache", "error", err)
		return nil, err
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "GetCreatorsBySlot",
	)
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
func breakByValidatorsBySlotCount(validators []common.Address, validatorsPerSlot, slotsPerEpoch uint64) [][]common.Address {
	chunks := make([][]common.Address, 0)

	for i := uint64(0); i+validatorsPerSlot <= uint64(len(validators)); i += validatorsPerSlot {
		end := i + validatorsPerSlot
		slotValidators := make([]common.Address, len(validators[i:end]))
		copy(slotValidators, validators[i:end])
		chunks = append(chunks, slotValidators)
		if len(chunks) == int(slotsPerEpoch) {
			return chunks
		}
	}

	return chunks
}

func (s *storage) GetValidatorsStateAddress() *common.Address {
	return s.config.ValidatorsStateAddress
}

func (s *storage) AddValidatorToList(stateDb vm.StateDB, index uint64, validator common.Address) {
	list := s.GetValidatorsList(stateDb)
	for _, address := range list {
		if address == validator {
			return
		}
	}

	if uint64(len(list)) <= index {
		for uint64(len(list)) <= index {
			list = append(list, common.Address{})
		}
	}

	list[index] = validator
	s.SetValidatorsList(stateDb, list)
}
