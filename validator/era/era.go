package era

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

const transitionPeriod = 2 // The number of epochs in the transition period

type blockchain interface {
	GetSlotInfo() *types.SlotInfo
	GetCoordinatedCheckpointEpoch(epoch uint64) uint64
	GetEraInfo() *EraInfo
	//ValidatorStorage() valStore.Storage
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
func (e *EraInfo) FirstSlot() uint64 {
	return 0
}

func (ei *EraInfo) LastEpoch() uint64 {
	return ei.ToEpoch()
}

func (ei *EraInfo) LastSlot() uint64 {
	return ei.LastEpoch()
}
func (ei *EraInfo) IsTransitionEpoch(epoch uint64) bool {
	if epoch == (ei.ToEpoch() - transitionPeriod) {
		return true
	}
	return false
}

func (ei *EraInfo) IsTransitionSlot(slot uint64) bool {
	if slot == (ei.ToEpoch() - transitionPeriod) {
		return true
	}
	return false
}

func (ei *EraInfo) NextEraFirstEpoch() uint64 {
	return ei.ToEpoch() + 1
}

// to_epoch + 1
func (ei *EraInfo) NextEraFirstSlot() uint64 {
	return ei.NextEraFirstEpoch()
} // NextEraFirstEpoch() first slot

func (ei *EraInfo) LenEpochs() uint64 {
	return ei.length
}

func (ei *EraInfo) LenSlots() uint64 {
	return ei.length * 32
}

// IsTransitionPeriod checks if the given slot falls in the transition period between the current era and the next era.
func IsEraTransitionPeriodStart(bc blockchain, slot uint64) bool {
	currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	lastEpoch := bc.GetEraInfo().ToEpoch()

	// Check if the current epoch is in the transition period (i.e., the last two epochs of the current era)
	if currentEpoch == lastEpoch-transitionPeriod {
		return bc.GetSlotInfo().IsEpochStart(slot)
	} else {
		return false
	}
}
