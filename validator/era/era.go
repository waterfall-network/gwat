package era

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

type Blockchain interface {
	GetSlotInfo() *types.SlotInfo
	GetCoordinatedCheckpointEpoch(epoch uint64) uint64
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

func (e *Era) Length() uint64 {
	return e.To - e.From
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
func (ei *EraInfo) IsTransitionEpoch(bc Blockchain, epoch uint64) bool {
	if epoch == (ei.ToEpoch() - bc.GetConfig().TransitionPeriod) {
		return true
	}

	return false
}

func (ei *EraInfo) IsTransitionSlot(bc Blockchain, slot uint64) bool {
	if slot == (ei.ToEpoch() - bc.GetConfig().TransitionPeriod) {
		if bc.GetSlotInfo().IsEpochStart(slot) == true {
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

// IsTransitionPeriod checks if the given slot falls in the transition period between the current era and the next era.
func IsEraTransitionPeriodStart(bc Blockchain, slot uint64) bool {
	currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	lastEpoch := bc.GetEraInfo().ToEpoch()

	// Check if the current epoch is in the transition period (i.e., the last two epochs of the current era)
	if currentEpoch == lastEpoch-bc.GetConfig().TransitionPeriod {
		return bc.GetSlotInfo().IsEpochStart(slot)
	} else {
		return false
	}
}
