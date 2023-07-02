package era

import (
	"errors"
	"math"

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
	EnterNextEra(*types.Checkpoint, common.Hash) *Era
	StartTransitionPeriod(cp *types.Checkpoint, spineRoot common.Hash)
	//SyncEraToSlot(slot uint64)
}

type Era struct {
	Number uint64      `json:"number"`
	From   uint64      `json:"fromEpoch"`
	To     uint64      `json:"toEpoch"`
	Root   common.Hash `json:"root"`
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
	nextEraLength := EstimateEraLength(bc.Config(), numValidators)
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
	transitionEpoch := (ei.ToEpoch() + 1 - bc.Config().TransitionPeriod)
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

func EstimateEraLength(chainConfig *params.ChainConfig, numberOfValidators uint64) (eraLength uint64) {
	var (
		epochsPerEra      = chainConfig.EpochsPerEra
		slotsPerEpoch     = float64(chainConfig.SlotsPerEpoch)
		validatorsPerSlot = float64(chainConfig.ValidatorsPerSlot)
	)

	eraLength = epochsPerEra * roundUp(float64(numberOfValidators)/(float64(epochsPerEra)*slotsPerEpoch*validatorsPerSlot))

	return
}

func roundUp(num float64) uint64 {
	return uint64(math.Ceil(num))
}

func HandleEra(bc Blockchain, cp *types.Checkpoint) error {
	log.Info("ERA started for new cp", "cp", cp.Epoch, "finEpoch", cp.FinEpoch, "root", cp.Spine.Hex())
	//currentEpoch := bc.GetSlotInfo().SlotToEpoch(slot)
	//newEpoch := bc.GetSlotInfo().IsEpochStart(slot)
	var spineRoot common.Hash
	// if cp != nil {
	header := bc.GetHeaderByHash(cp.Spine)
	if header != nil {
		spineRoot = header.Root
	} else {
		log.Error("Ccheckpoint spine header not found", "err", ErrCheckpointInvalid)
		return ErrCheckpointInvalid
	}

	//if newEpoch {
	// New era
	if bc.GetEraInfo().ToEpoch()+1 <= cp.FinEpoch {
		// Checkpoint
		// checkpoint := bc.GetLastCoordinatedCheckpoint()

		bc.EnterNextEra(cp, spineRoot)

		log.Info("######## HandleEra", "cpEpoch", cp.Epoch,
			"cpFinEpoch", cp.FinEpoch,
			"curEpoch", bc.GetSlotInfo().SlotInEpoch(bc.GetSlotInfo().CurrentSlot()),
			"curSlot", bc.GetSlotInfo().CurrentSlot(),
			"bc.GetEraInfo().ToEpoch", bc.GetEraInfo().ToEpoch(),
			"bc.GetEraInfo().FromEpoch", bc.GetEraInfo().FromEpoch(),
			"bc.GetEraInfo().Number", bc.GetEraInfo().Number(),
		)

		return nil
	} else if (bc.GetEraInfo().ToEpoch()+1)-bc.Config().TransitionPeriod == cp.FinEpoch && cp.FinEpoch <= bc.GetEraInfo().ToEpoch()+1 {
		bc.StartTransitionPeriod(cp, spineRoot)
	}

	return nil
}
