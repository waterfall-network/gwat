// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"encoding/binary"
	"errors"
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
	GetBlock(ctx context.Context, hash common.Hash) *types.Block
	GetSlotInfo() *types.SlotInfo
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	GetEpoch(epoch uint64) common.Hash
	EpochToEra(uint64) *era.Era
}

type Storage interface {
	GetValidators(bc blockchain, slot uint64, tmpFromWhere string) ([]common.Address, error)
	GetCreatorsBySlot(bc blockchain, filter ...uint64) ([]common.Address, error)
	GetActiveValidatorsCount(bc blockchain, slot uint64) (uint64, error)

	SetValidator(stateDb vm.StateDB, val *Validator) error
	GetValidator(stateDb vm.StateDB, address common.Address) (*Validator, error)

	SetValidatorsList(stateDb vm.StateDB, list []common.Address)
	GetValidatorsList(stateDb vm.StateDB) []common.Address
	AddValidatorToList(stateDB vm.StateDB, index uint64, validator common.Address)

	GetValidatorsStateAddress() *common.Address
	GetDepositCount(stateDb vm.StateDB) uint64
	IncrementDepositCount(stateDb vm.StateDB)

	PrepareNextEraValidators(bc blockchain, era *era.Era)
}

type storage struct {
	validatorsCache   *ValidatorsCache
	config            *params.ChainConfig
	processTransition map[uint64]struct{}
}

func NewStorage(config *params.ChainConfig) Storage {
	return &storage{
		validatorsCache:   NewCache(),
		config:            config,
		processTransition: make(map[uint64]struct{}),
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
	var valData ValidatorBinary = stateDb.GetCode(address)
	if valData == nil {
		return nil, ErrNoStateValidatorInfo
	}

	return valData.ToValidator()
}

func (s *storage) SetValidatorsList(stateDb vm.StateDB, list []common.Address) {
	newList := make([]byte, len(list)*common.AddressLength+common.Uint64Size)

	currentList := stateDb.GetCode(*s.ValidatorsStateAddress())
	if len(currentList) > 0 {
		depositCount := binary.BigEndian.Uint64(currentList[:common.Uint64Size])
		binary.BigEndian.PutUint64(newList[:common.Uint64Size], depositCount)
	}

	for i, validator := range list {
		beginning := i*common.AddressLength + common.Uint64Size
		end := beginning + common.AddressLength
		copy(newList[beginning:end], validator[:])
	}

	stateDb.SetCode(*s.ValidatorsStateAddress(), newList)
}

func (s *storage) GetValidatorsList(stateDb vm.StateDB) []common.Address {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())

	validators := make([]common.Address, 0)
	for i := common.Uint64Size; i+common.AddressLength <= len(buf); i += common.AddressLength {
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

	return binary.BigEndian.Uint64(buf[:common.Uint64Size])
}

func (s *storage) IncrementDepositCount(stateDb vm.StateDB) {
	buf := stateDb.GetCode(*s.ValidatorsStateAddress())

	if buf == nil {
		buf = make([]byte, common.Uint64Size)
	}
	currentCount := binary.BigEndian.Uint64(buf[:common.Uint64Size])
	currentCount++

	binary.BigEndian.PutUint64(buf[:common.Uint64Size], currentCount)
	stateDb.SetCode(*s.ValidatorsStateAddress(), buf)
}

// GetValidators return two values: array of Validator and array of Validators addresses.
// If parameter needAddresses is false it return array of Validator and nil value for validators addresses.
// Use parameter activeOnly true if you need only active validators.
func (s *storage) GetValidators(bc blockchain, slot uint64, tmpFromWhere string) ([]common.Address, error) {
	slotEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	slotEra := bc.EpochToEra(slotEpoch)

	err := s.checkTransitionProcessing(slotEra.Number)
	if err != nil {
		return nil, err
	}

	validators := s.validatorsCache.getAllActiveValidatorsByEra(slotEra.Number)
	if validators != nil {
		return validators, nil
	}
	log.Info("Get validators", "epoch", slotEpoch, "era", slotEra.Number, "root", slotEra.Root.Hex())

	stateDb, _ := bc.StateAt(slotEra.Root)
	valList := s.GetValidatorsList(stateDb)
	log.Info("GetValidators from state", "validators", len(valList), "slot", slot, "epoch", slotEpoch, "root", slotEra.Root.Hex())
	eraValidators := make([]common.Address, 0)
	for _, valAddress := range valList {
		val, err := s.GetValidator(stateDb, valAddress)
		if err != nil {
			log.Error("can`t get validator from state", "error", err, "address", valAddress.Hex())
			continue
		}
		if val.ActivationEra <= slotEra.Number && val.ExitEra > slotEra.Number {
			eraValidators = append(eraValidators, val.GetAddress())
		}
	}

	s.validatorsCache.addAllActiveValidatorsByEra(slotEra.Number, eraValidators)

	log.Info("GetValidators", "callFunc", tmpFromWhere, "all", len(validators),
		"active", len(eraValidators),
		"slot", slot, "epoch", slotEpoch,
	)

	validators = make([]common.Address, len(eraValidators))
	copy(validators, eraValidators)

	return validators, nil
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
	slotEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	params = append(params, slotEpoch)
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

	allValidators, err := s.GetValidators(bc, slot, "GetCreatorsBySlot")
	if err != nil || len(allValidators) == 0 {
		return nil, err
	}

	spineEpoch := uint64(0)
	if slotEpoch > 0 {
		spineEpoch = slotEpoch - 1
	}
	epochSpine := bc.GetEpoch(spineEpoch)
	if epochSpine == (common.Hash{}) {
		return nil, fmt.Errorf("epoch not found")
	}
	seed, err := s.seed(slotEpoch, epochSpine)
	if err != nil {
		return nil, err
	}

	log.Info("CheckShuffle - shuffle params", "slot", slot,
		"slotEpoch", slotEpoch,
		"spineEpoch", spineEpoch,
		"spine", epochSpine.Hex(),
		"seed", seed.Hex(),
		"validatorsCount", len(allValidators),
	)

	shuffledValidators, err := shuffle.ShuffleValidators(allValidators, seed)
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

func (s *storage) GetActiveValidatorsCount(bc blockchain, slot uint64) (uint64, error) {
	slotEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	slotEra := bc.EpochToEra(slotEpoch)

	err := s.checkTransitionProcessing(slotEra.Number)
	if err != nil {
		return 0, err
	}

	vals, ok := s.validatorsCache.allActiveValidatorsCache[slotEra.Number]
	if ok {
		return uint64(len(vals)), nil
	}

	count := uint64(0)
	stateDb, _ := bc.StateAt(slotEra.Root)
	valList := s.GetValidatorsList(stateDb)
	for _, valAddress := range valList {
		val, err := s.GetValidator(stateDb, valAddress)
		if err != nil {
			log.Error("can`t get validator from state", "error", err, "address", valAddress.Hex())
			continue
		}

		if val.ActivationEra <= slotEra.Number+1 && val.ExitEra > slotEra.Number {
			count++
		}
	}

	return count, nil
}

func (s *storage) PrepareNextEraValidators(bc blockchain, era *era.Era) {
	s.processTransition[era.Number] = struct{}{}
	stateDb, _ := bc.StateAt(era.Root)

	valList := s.GetValidatorsList(stateDb)
	for _, valAddress := range valList {
		val, err := s.GetValidator(stateDb, valAddress)
		if err != nil {
			log.Error("can`t get validator from state", "error", err, "address", valAddress.Hex())
			continue
		}

		if val.ActivationEra <= era.Number && val.ExitEra > era.Number {
			s.validatorsCache.addValidator(val.Address, era.Number)
		}
	}

	log.Info("Prepare next era validators",
		"eraNumber", era.Number,
		"eraRoot", era.Root,
		"eraBlockHash", era.BlockHash,
		"validatorsCount", len(s.validatorsCache.allActiveValidatorsCache[era.Number]),
	)

	delete(s.processTransition, era.Number)
}

func (s *storage) checkTransitionProcessing(era uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, transitionProcessing := s.processTransition[era]
	if transitionProcessing {
		log.Info("Era process transition period, waiting", "era", era)
		for transitionProcessing {
			time.Sleep(time.Millisecond * 200)
			select {
			case <-ctx.Done():
				return errors.New("check transition processing failed, timeout expired")
			default:
				_, transitionProcessing = s.processTransition[era]
			}
		}
	}

	return nil
}
