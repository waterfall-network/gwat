//Package dag implements:
//- consensus functionality
//- finalizing process
//- block creation process

package dag

import (
	"fmt"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/creator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/finalizer"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/headsync"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/slotticker"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

const optimisticSpinesCacheLen = 8

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	Etherbase() (eb common.Address, err error)
	SetEtherbase(etherbase common.Address)
	CreatorAuthorize(creator common.Address) error
	IsDevMode() bool
	AccountManager() *accounts.Manager
}

type blockChain interface {
	SetLastCoordinatedCheckpoint(cp *types.Checkpoint)
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	AppendNotProcessedValidatorSyncData(valSyncData []*types.ValidatorSync)
	GetLastFinalizedHeader() *types.Header
	GetHeaderByHash(hash common.Hash) *types.Header
	WriteLastCoordinatedHash(hash common.Hash)
	GetBlock(hash common.Hash) *types.Block
	GetLastFinalizedBlock() *types.Block
	GetTips() types.Tips
	Database() ethdb.Database
	GetSlotInfo() *types.SlotInfo
	SetSlotInfo(si *types.SlotInfo) error
	Config() *params.ChainConfig
	GetEraInfo() *era.EraInfo
	SetNewEraInfo(newEra era.Era)
	EnterNextEra(root common.Hash) *era.Era
	StartTransitionPeriod()
	SyncEraToSlot(slot uint64)
	ValidatorStorage() valStore.Storage
	StateAt(root common.Hash) (*state.StateDB, error)
	DagMuLock()
	DagMuUnlock()
	AddOptimisticSpineToCache(spine *types.Block)
	GetOptimisticSpinesFromCache(slot uint64) types.Blocks
}

type ethDownloader interface {
	Synchronising() bool
}

type Dag struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	consensusInfoFeed event.Feed

	eth        Backend
	bc         blockChain
	downloader ethDownloader

	//creator
	creator *creator.Creator
	//finalizer
	finalizer *finalizer.Finalizer
	//headsync
	headsync *headsync.Headsync

	busy int32

	exitChan chan struct{}
	errChan  chan error
}

// New creates new instance of Dag
func New(eth Backend, chainConfig *params.ChainConfig, mux *event.TypeMux, creatorConfig *creator.Config, engine consensus.Engine) *Dag {
	fin := finalizer.New(chainConfig, eth, mux)
	d := &Dag{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
		bc:          eth.BlockChain(),
		downloader:  eth.Downloader(),
		creator:     creator.New(creatorConfig, chainConfig, engine, eth, mux),
		finalizer:   fin,
		headsync:    headsync.New(chainConfig, eth, mux, fin),
		exitChan:    make(chan struct{}),
		errChan:     make(chan error),
	}
	atomic.StoreInt32(&d.busy, 0)
	return d
}

// Creator get current creator
func (d *Dag) Creator() *creator.Creator {
	return d.creator
}

// HandleFinalize run blocks finalization procedure
func (d *Dag) HandleFinalize(data *types.FinalizationParams) *types.FinalizationResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	if data.BaseSpine != nil {
		log.Info("Handle Finalize: start",
			"baseSpine", fmt.Sprintf("%#x", data.BaseSpine),
			"spines", data.Spines,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.Spine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.Roor", fmt.Sprintf("%#x", data.Checkpoint.Root),
			"ValSyncData", data.ValSyncData,
			"\u2692", params.BuildId)
	} else {
		log.Info("Handle Finalize: start",
			"baseSpine", nil,
			"spines", data.Spines,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.Spine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.Spine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"ValSyncData", data.ValSyncData,
			"\u2692", params.BuildId)
	}

	if data.ValSyncData != nil {
		for _, vs := range data.ValSyncData {
			log.Info("received validator sync",
				"OpType", vs.OpType,
				"Index", vs.Index,
				"Creator", vs.Creator.Hash(),
				"ProcEpoch", vs.ProcEpoch,
				"Amount", vs.Amount,
			)
		}
	}

	res := &types.FinalizationResult{
		Error: nil,
	}
	// finalization
	if len(data.Spines) > 0 {
		if err := d.finalizer.Finalize(&data.Spines, data.BaseSpine, false); err != nil {
			e := err.Error()
			res.Error = &e
		} else {
			d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
		}
	} else {
		d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
	}

	// handle validator sync data
	d.bc.AppendNotProcessedValidatorSyncData(data.ValSyncData)

	lfHeader := d.bc.GetLastFinalizedHeader()
	if lfHeader.Height != lfHeader.Nr() {
		err := fmt.Sprintf("☠ bad last finalized block: mismatch nr=%d and height=%d", lfHeader.Nr(), lfHeader.Height)
		if res.Error == nil {
			res.Error = &err
		} else {
			mrg := fmt.Sprintf("error[0]=%s\nerror[1]: %s", *res.Error, err)
			res.Error = &mrg
		}
	} else {
		d.bc.WriteLastCoordinatedHash(lfHeader.Hash())
	}
	lfHash := lfHeader.Hash()
	res.LFSpine = &lfHash

	if cp := d.bc.GetLastCoordinatedCheckpoint(); cp != nil {
		res.CpEpoch = &cp.Epoch
		res.CpRoot = &cp.Root
	}

	log.Info("Handle Finalize: response", "result", res)
	return res
}

// HandleCoordinatedState return coordinated state
func (d *Dag) HandleCoordinatedState() *types.FinalizationResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	lfHeader := d.bc.GetLastFinalizedHeader()
	lfHash := lfHeader.Hash()
	res := &types.FinalizationResult{
		Error:   nil,
		LFSpine: &lfHash,
	}
	var cpepoch uint64
	if cp := d.bc.GetLastCoordinatedCheckpoint(); cp != nil {
		res.CpEpoch = &cp.Epoch
		res.CpRoot = &cp.Root
		cpepoch = cp.Epoch
	}
	log.Info("Handle CoordinatedState: response", "epoch", cpepoch, "result", res)
	return res
}

// HandleGetCandidates collect next finalization candidates
func (d *Dag) HandleGetCandidates(slot uint64) *types.CandidatesResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.CandidatesResult{
			Error: &errStr,
		}
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	tstart := time.Now()

	// collect next finalization candidates
	candidates, err := d.finalizer.GetFinalizingCandidates(&slot)
	if len(candidates) == 0 {
		log.Info("No candidates for tips", "tips", d.bc.GetTips().Print())
	}
	log.Info("Handle GetCandidates: get finalizing candidates", "err", err, "candidates", candidates, "elapsed", common.PrettyDuration(time.Since(tstart)), "\u2692", params.BuildId)
	res := &types.CandidatesResult{
		Error:      nil,
		Candidates: candidates,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	log.Info("Handle GetCandidates: response", "result", res, "\u2692", params.BuildId)
	return res
}

func (d *Dag) HandleGetOptimisticSpines(lastFinSpine common.Hash) *types.OptimisticSpinesResult {
	spineBlock := d.bc.GetBlock(lastFinSpine)
	spineSlot := spineBlock.Slot()
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.OptimisticSpinesResult{
			Error: &errStr,
		}
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	tstart := time.Now()

	// collect optimistic spines
	spines, err := d.GetOptimisticSpines(spineSlot)
	if len(spines) == 0 {
		log.Info("No spines for tips", "tips", d.bc.GetTips().Print())
	}

	log.Info("Handle GetOptimisticSpines: get finalizing spines", "err", err, "spines", spines, "elapsed", common.PrettyDuration(time.Since(tstart)), "\u2692", params.BuildId)
	res := &types.OptimisticSpinesResult{
		Error: nil,
		Data:  spines,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	log.Info("Handle GetOptimisticSpines: response", "result", res, "\u2692", params.BuildId)
	return res
}

func (d *Dag) GetOptimisticSpines(gtSlot uint64) ([]common.HashArray, error) {
	currentSlot := d.bc.GetSlotInfo().CurrentSlot()
	if currentSlot <= gtSlot {
		return []common.HashArray{}, nil
	}

	slotsBlocks := make(types.Blocks, 0)

	for i := gtSlot + 1; i <= currentSlot; i++ {
		slotSpinesFromCache := d.bc.GetOptimisticSpinesFromCache(i)
		if slotSpinesFromCache == nil {
			blocksHashes := rawdb.ReadSlotBlocksHashes(d.bc.Database(), i)
			for _, hash := range blocksHashes {
				block := d.bc.GetBlock(hash)
				slotSpinesFromCache = append(slotSpinesFromCache, block)
				d.bc.AddOptimisticSpineToCache(block)
			}
		}

		slotsBlocks = append(slotsBlocks, slotSpinesFromCache...)
	}

	if len(slotsBlocks) == 0 {
		return []common.HashArray{}, nil
	}

	optimisticSpines, err := types.CalculateOptimisticSpines(slotsBlocks)
	if err != nil {
		return []common.HashArray{}, err
	}

	spinesHashes := make([]common.HashArray, 0)
	for _, spines := range optimisticSpines {
		spinesHashes = append(spinesHashes, *spines.GetHashes())
	}

	return spinesHashes, nil
}

// HandleHeadSyncReady set initial state to start head sync with coordinating network.
func (d *Dag) HandleHeadSyncReady(checkpoint *types.ConsensusInfo) (bool, error) {
	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	log.Info("Handle Head Sync Ready", "checkpoint", checkpoint)
	return d.headsync.SetReadyState(checkpoint)
}

// HandleSyncSlotInfo set initial state to start head sync with coordinating network.
func (d *Dag) HandleSyncSlotInfo(slotInfo types.SlotInfo) (bool, error) {
	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	log.Info("Handle Sync Slot info", "params", slotInfo)
	si := d.bc.GetSlotInfo()
	if si.GenesisTime == slotInfo.GenesisTime &&
		si.SecondsPerSlot == slotInfo.SecondsPerSlot &&
		si.SlotsPerEpoch == slotInfo.SlotsPerEpoch {
		return true, nil
	}
	if d.eth.IsDevMode() {
		err := d.bc.SetSlotInfo(&slotInfo)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// HandleHeadSync run head sync with coordinating network.
func (d *Dag) HandleHeadSync(data []types.ConsensusInfo) (bool, error) {
	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	log.Info("Handle Head Sync", "len(data)", len(data), "data", data)
	return d.headsync.Sync(data)
}

// HandleValidateSpines collect next finalization candidates
func (d *Dag) HandleValidateSpines(spines common.HashArray) (bool, error) {
	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	log.Info("Handle Validate Spines", "spines", spines, "\u2692", params.BuildId)
	return d.finalizer.IsValidSequenceOfSpines(spines)
}

// SubscribeConsensusInfoEvent registers a subscription for consensusInfo updated event
func (d *Dag) SubscribeConsensusInfoEvent(ch chan<- types.Tips) event.Subscription {
	return d.consensusInfoFeed.Subscribe(ch)
}

func (d *Dag) StartWork(accounts []common.Address) {
	startTicker := time.NewTicker(500 * time.Millisecond)

	tickSec := 0
	for {
		currentTime := time.Now()
		genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)

		if currentTime.Before(genesisTime) {
			if tickSec != currentTime.Second() && currentTime.Second()%5 == 0 {
				timeRemaining := genesisTime.Sub(currentTime)
				log.Info("Time before start", "hour", timeRemaining.Truncate(time.Second))
			}
		} else {
			log.Info("Chain genesis time reached")
			startTicker.Stop()
			go d.workLoop(accounts)

			return
		}
		tickSec = currentTime.Second()

		<-startTicker.C
	}
}

func (d *Dag) workLoop(accounts []common.Address) {
	secPerSlot := d.bc.GetSlotInfo().SecondsPerSlot
	genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)
	slotTicker := slotticker.NewSlotTicker(genesisTime, secPerSlot)

	for {
		select {
		case <-d.exitChan:
			close(d.exitChan)
			close(d.errChan)
			slotTicker.Done()
			return
		case err := <-d.errChan:
			close(d.errChan)
			close(d.exitChan)
			slotTicker.Done()
			log.Error("dag worker has error", "error", err)
			return
		case slot := <-slotTicker.C():
			if slot == 0 {
				newEra := era.NewEra(0, 0, d.bc.Config().EpochsPerEra-1, common.Hash{})
				d.bc.SetNewEraInfo(*newEra)
				continue
			}
			var (
				err      error
				creators []common.Address
			)

			transitionSlot, err := d.bc.GetSlotInfo().SlotOfEpochStart(d.bc.GetEraInfo().ToEpoch() - d.chainConfig.TransitionPeriod)
			if err != nil {
				log.Error("Error calculating transition slot", "error", err)
				return
			}

			log.Info("New slot",
				"slot", slot,
				"epoch", d.bc.GetSlotInfo().SlotToEpoch(slot),
				"era", d.bc.GetEraInfo().Number(),
				"transEpoch", d.bc.GetEraInfo().ToEpoch()-d.chainConfig.TransitionPeriod,
				"transSlot", transitionSlot)

			d.handleEra(slot)

			// TODO: uncomment this code for subnetwork support, add subnet and get it to the creators getter (line 253)
			//if d.bc.Config().IsForkSlotSubNet1(currentSlot) {
			//	creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, currentSlot,subnet)
			//	if err != nil {
			//		d.errChan <- err
			//	}
			//} else {}
			// TODO: move it to else condition

			creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, slot)
			if err != nil {
				d.errChan <- err
			}

			d.work(slot, creators, accounts)
		}
	}
}

func (d *Dag) work(slot uint64, creators, accounts []common.Address) {
	//skip if synchronising
	if d.downloader.Synchronising() {
		return
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	errs := map[string]string{}

	var (
		err error
	)
	// create block
	tips := d.bc.GetTips()
	dagSlots := d.countDagSlots(&tips)
	log.Info("Creator processing: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"dagSlots", dagSlots,
	)

	if d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit {
		assigned := &creator.Assignment{
			Slot:     slot,
			Creators: creators,
		}

		crtStart := time.Now()
		crtInfo := map[string]string{}
		for _, creator := range assigned.Creators {
			// if received next slot

			coinbase := common.Address{}
			for _, acc := range accounts {
				if creator == acc {
					coinbase = creator
					break
				}
			}
			if coinbase == (common.Address{}) {
				return
			}

			d.eth.SetEtherbase(coinbase)
			if err = d.eth.CreatorAuthorize(coinbase); err != nil {
				log.Error("Creator authorize err", "err", err, "creator", coinbase)
				return
			}
			log.Info("Creator assigned", "creator", coinbase)

			block, crtErr := d.creator.CreateBlock(assigned, &tips)
			if crtErr != nil {
				crtInfo["error"] = crtErr.Error()
			}

			if block != nil {
				crtInfo["newBlock"] = block.Hash().Hex()
			}

			log.Info("Creator processing: create block", "dagSlots", dagSlots, "IsRunning", d.creator.IsRunning(), "crtInfo", crtInfo, "elapsed", common.PrettyDuration(time.Since(crtStart)))
		}
	}
}

// countDagSlots count number of slots in dag chain
// if error returns  -1
func (d *Dag) countDagSlots(tips *types.Tips) int {
	candidates, err := d.finalizer.GetFinalizingCandidates(nil)
	if err != nil {
		return -1
	}
	return len(candidates)
}

func (d *Dag) handleEra(slot uint64) {
	currentEpoch := d.bc.GetSlotInfo().SlotToEpoch(slot)
	newEpoch := d.bc.GetSlotInfo().IsEpochStart(slot)

	if newEpoch {
		// New era
		if d.bc.GetEraInfo().ToEpoch()+1 == currentEpoch {
			// Checkpoint
			checkpoint := d.bc.GetLastCoordinatedCheckpoint()
			spineRoot := common.Hash{}
			if checkpoint != nil {
				header := d.bc.GetHeaderByHash(checkpoint.Spine)
				spineRoot = header.Root
			} else {
				log.Error("Invalid checkpoint: write new era error")
			}

			d.bc.EnterNextEra(spineRoot)

			return
		}

		// Transition period
		if d.bc.GetEraInfo().IsTransitionPeriodStartSlot(d.bc, slot) {
			d.bc.StartTransitionPeriod()
		}
	}

	// Sync era to current slot
	if currentEpoch > (d.bc.GetEraInfo().ToEpoch()-d.chainConfig.TransitionPeriod) && !d.bc.GetEraInfo().IsTransitionPeriodStartSlot(d.bc, slot) {
		d.bc.SyncEraToSlot(slot)
	}
}
