package era

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

type blockchain interface {
	GetSlotInfo() *types.SlotInfo
	GetEraInfo() *EraInfo
	GetConfig() *params.ChainConfig
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

func NextEra(bc blockchain, root common.Hash, numValidators uint64) *Era {
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
func (ei *EraInfo) FirstSlot(bc blockchain) uint64 {
	slot, err := bc.GetSlotInfo().SlotOfEpochStart(ei.FirstEpoch())
	if err != nil {
		return 0
	}

	return slot
}

func (ei *EraInfo) LastEpoch() uint64 {
	return ei.ToEpoch()
}

func (ei *EraInfo) LastSlot(bc blockchain) uint64 {
	slot, err := bc.GetSlotInfo().SlotOfEpochEnd(ei.LastEpoch())
	if err != nil {
		return 0
	}

	return slot
}

func (ei *EraInfo) IsTransitionPeriodEpoch(bc blockchain, epoch uint64) bool {
	return epoch >= ei.ToEpoch()-bc.GetConfig().TransitionPeriod && epoch <= ei.ToEpoch()
}

func (ei *EraInfo) IsTransitionPeriodStartEpoch(bc blockchain, epoch uint64) bool {
	return epoch == (ei.ToEpoch() - bc.GetConfig().TransitionPeriod)
}

func (ei *EraInfo) IsTransitionPeriodStartSlot(bc blockchain, slot uint64) bool {
	transitionEpoch := (ei.ToEpoch() - bc.GetConfig().TransitionPeriod)
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

func (ei *EraInfo) NextEraFirstSlot(bc blockchain) uint64 {
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

func EstimateEraLength(bc blockchain, numberOfValidators uint64) (eraLength uint64) {
	var (
		epochsPerEra  = bc.GetConfig().EpochsPerEra
		slotsPerEpoch = bc.GetSlotInfo().SlotsPerEpoch
	)

	eraLength = epochsPerEra * (1 + (numberOfValidators / (epochsPerEra * slotsPerEpoch)))

	return
}
