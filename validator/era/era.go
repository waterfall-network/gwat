package era

import (
	"errors"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

var (
	ErrCheckpointInvalid = errors.New("invalid checkpoint")
)

type Blockchain interface {
	GetSlotInfo() *types.SlotInfo
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	GetEraInfo() *EraInfo
	Config() *params.ChainConfig
	GetHeaderByHash(common.Hash) *types.Header
	EnterNextEra(common.Hash) *Era
	StartTransitionPeriod()
	SyncEraToSlot(slot uint64)
}

type Era struct {
	Number uint64
	From   uint64
	To     uint64
	Root   common.Hash
}

// New function to create a new Era instance
func NewEra(number, from, to uint64, root common.Hash) *Era {
	return &Era{
		Number: number,
		From:   from,
		To:     to,
		Root:   root,
	}
}

func NextEra(bc Blockchain, root common.Hash, numValidators uint64) *Era {
	nextEraNumber := bc.GetEraInfo().Number() + 1
	nextEraLength := EstimateEraLength(bc, numValidators)
	nextEraBegin := bc.GetEraInfo().ToEpoch() + 1
	nextEraEnd := bc.GetEraInfo().ToEpoch() + nextEraLength

	return NewEra(nextEraNumber, nextEraBegin, nextEraEnd, root)
}

func (e *Era) Length() uint64 {
	return e.To - e.From
}

func (e *Era) IsContainsEpoch(epoch uint64) bool {
	log.Info("@@@@@@@@@ IsContainsEpoch valEra", "epoch", epoch, "eraNum", e.Number, "to", e.To, "from", e.From)
	return epoch >= e.From && epoch <= e.To
}

type EraInfo struct {
	currentEra *Era
	length     uint64
}

func NewEraInfo(era Era) EraInfo {
	return EraInfo{
		currentEra: &era,
		length:     era.Length(),
	}
}

func (ei *EraInfo) Number() uint64 {
	return ei.currentEra.Number
}

func (ei *EraInfo) GetEra() *Era {
	return ei.currentEra
}

func (ei *EraInfo) ToEpoch() uint64 {
	return ei.GetEra().To
}

func (ei *EraInfo) FromEpoch() uint64 {
	return ei.GetEra().From
}

func (ei *EraInfo) EpochsPerEra() uint64 {
	return ei.GetEra().To - ei.GetEra().From + 1
}

func (ei *EraInfo) FirstEpoch() uint64 {
	return ei.FromEpoch()
}
func (ei *EraInfo) FirstSlot(bc Blockchain) uint64 {
	slot, err := bc.GetSlotInfo().SlotOfEpochStart(ei.FirstEpoch())
	if err != nil {
		return 0
	}

	return slot
}

func (ei *EraInfo) LastEpoch() uint64 {
	return ei.ToEpoch()
}

func (ei *EraInfo) LastSlot(bc Blockchain) uint64 {
	slot, err := bc.GetSlotInfo().SlotOfEpochEnd(ei.LastEpoch())
	if err != nil {
		return 0
	}

	return slot
}

func (ei *EraInfo) IsTransitionPeriodEpoch(bc Blockchain, epoch uint64) bool {
	return epoch >= ei.ToEpoch()-bc.Config().TransitionPeriod && epoch <= ei.ToEpoch()
}

func (ei *EraInfo) IsTransitionPeriodStartEpoch(bc Blockchain, epoch uint64) bool {
	return epoch == (ei.ToEpoch() - bc.Config().TransitionPeriod)
}

func (ei *EraInfo) IsTransitionPeriodStartSlot(bc Blockchain, slot uint64) bool {
	transitionEpoch := (ei.ToEpoch() - bc.Config().TransitionPeriod)
	currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)

	if currentEpoch == transitionEpoch {
		if bc.GetSlotInfo().IsEpochStart(slot) {
			return true
		}
	}

	return false
}

func (ei *EraInfo) NextEraFirstEpoch() uint64 {
	return ei.ToEpoch() + 1
}

func (ei *EraInfo) NextEraFirstSlot(bc Blockchain) uint64 {
	slot, err := bc.GetSlotInfo().SlotOfEpochStart(ei.NextEraFirstEpoch())
	if err != nil {
		return 0
	}

	return slot
}

func (ei *EraInfo) LenEpochs() uint64 {
	return ei.length
}

func (ei *EraInfo) LenSlots() uint64 {
	return ei.length * 32
}

func (ei *EraInfo) IsContainsEpoch(epoch uint64) bool {
	if epoch >= ei.FromEpoch() && epoch <= ei.ToEpoch() {
		return true
	}
	return false
}

func EstimateEraLength(bc Blockchain, numberOfValidators uint64) (eraLength uint64) {
	var (
		epochsPerEra  = bc.Config().EpochsPerEra
		slotsPerEpoch = bc.GetSlotInfo().SlotsPerEpoch
	)

	eraLength = epochsPerEra * (1 + (numberOfValidators / (epochsPerEra * slotsPerEpoch)))

	return
}

func HandleEra(bc Blockchain, slot uint64) error {
	currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	newEpoch := bc.GetSlotInfo().IsEpochStart(slot)

	if newEpoch {
		// New era
		if bc.GetEraInfo().ToEpoch()+1 == currentEpoch {
			// Checkpoint
			checkpoint := bc.GetLastCoordinatedCheckpoint()
			var spineRoot common.Hash
			if checkpoint != nil {
				header := bc.GetHeaderByHash(checkpoint.Spine)
				spineRoot = header.Root
			} else {
				log.Error("Write new era error", "err", ErrCheckpointInvalid)
				return ErrCheckpointInvalid
			}
			log.Info("######## HandleEra", "currentEpoch", currentEpoch,
				"bc.GetEraInfo().ToEpoch", bc.GetEraInfo().ToEpoch(),
				"bc.GetEraInfo().FromEpoch", bc.GetEraInfo().FromEpoch(),
				"bc.GetEraInfo().Number", bc.GetEraInfo().Number(),
			)
			bc.EnterNextEra(spineRoot)
			return nil
		}
		// Transition period
		if bc.GetEraInfo().IsTransitionPeriodStartSlot(bc, slot) {
			bc.StartTransitionPeriod()
		}
	}
	// Sync era to current slot
	if currentEpoch > (bc.GetEraInfo().ToEpoch()-bc.Config().TransitionPeriod) && !bc.GetEraInfo().IsTransitionPeriodStartSlot(bc, slot) {
		bc.SyncEraToSlot(slot)
	}
	return nil
}
