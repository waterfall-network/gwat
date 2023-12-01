//Package dag implements:
//- consensus functionality
//- finalizing process
//- block creation process

package dag

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/creator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/finalizer"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/slotticker"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	// ErrSynchronization throws if synchronization process running
	errSynchronization = errors.New("synchronization")
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	CreatorAuthorize(creator common.Address) error
	IsDevMode() bool
	AccountManager() *accounts.Manager
}

type blockChain interface {
	GetEpoch(epoch uint64) common.Hash
	SetLastCoordinatedCheckpoint(cp *types.Checkpoint)
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	AppendNotProcessedValidatorSyncData(valSyncData []*types.ValidatorSync)
	GetLastFinalizedHeader() *types.Header
	GetHeaderByHash(common.Hash) *types.Header
	GetBlock(hash common.Hash) *types.Block
	GetBlockByHash(hash common.Hash) *types.Block
	GetLastFinalizedNumber() uint64
	GetBlocksByHashes(hashes common.HashArray) types.BlockMap
	GetLastFinalizedBlock() *types.Block
	GetTips() types.Tips
	Database() ethdb.Database
	GetSlotInfo() *types.SlotInfo
	SetSlotInfo(si *types.SlotInfo) error
	Config() *params.ChainConfig
	GetEraInfo() *era.EraInfo
	SetNewEraInfo(newEra era.Era)
	EnterNextEra(nextEraEpochFrom uint64, root common.Hash) *era.Era
	StartTransitionPeriod(cp *types.Checkpoint, spineRoot common.Hash)
	//SyncEraToSlot(slot uint64)
	ValidatorStorage() valStore.Storage
	StateAt(root common.Hash) (*state.StateDB, error)
	DagMuLock()
	DagMuUnlock()
	SetOptimisticSpinesToCache(slot uint64, spines common.HashArray)
	GetOptimisticSpinesFromCache(slot uint64) common.HashArray
	GetOptimisticSpines(gtSlot uint64) ([]common.HashArray, error)
	ExploreChainRecursive(common.Hash, ...core.ExploreResultMap) (common.HashArray, common.HashArray, common.HashArray, *types.GraphDag, core.ExploreResultMap, error)
	EpochToEra(uint64) *era.Era
	Genesis() *types.Block

	SetIsSynced(synced bool)
	IsSynced() bool

	CollectAncestorsAftCpByParents(common.HashArray, common.Hash) (bool, types.HeaderMap, common.HashArray, error)
	IsCheckpointOutdated(*types.Checkpoint) bool
	GetCoordinatedCheckpoint(cpSpine common.Hash) *types.Checkpoint
	SetSyncCheckpointCache(cp *types.Checkpoint)
	ResetSyncCheckpointCache()
	RemoveTips(hashes common.HashArray)
	WriteCurrentTips()
	GetBlockHashesBySlot(slot uint64) common.HashArray
	HaveEpochBlocks(epoch uint64) (bool, error)
}

type ethDownloader interface {
	Synchronising() bool
	MainSync(baseSpine common.Hash, spines common.HashArray) error
	DagSync(baseSpine common.Hash, spines common.HashArray) error
	Terminate()
}

type Dag struct {
	eth        Backend
	bc         blockChain
	downloader ethDownloader

	//creator
	creator *creator.Creator
	//finalizer
	finalizer *finalizer.Finalizer

	lastFinApiSlot uint64

	exitChan chan struct{}
	enddChan chan struct{}
	errChan  chan error

	checkpoint *types.Checkpoint
}

// New creates new instance of Dag
func New(eth Backend, mux *event.TypeMux, creatorConfig *creator.Config) *Dag {
	fin := finalizer.New(eth)
	d := &Dag{
		eth:        eth,
		bc:         eth.BlockChain(),
		downloader: eth.Downloader(),
		creator:    creator.New(creatorConfig, eth, mux),
		finalizer:  fin,
		exitChan:   make(chan struct{}),
		enddChan:   make(chan struct{}),
		errChan:    make(chan error),
	}
	return d
}

// Creator get current creator
func (d *Dag) Creator() *creator.Creator {
	return d.creator
}

// HandleFinalize run blocks finalization procedure
func (d *Dag) HandleFinalize(data *types.FinalizationParams) *types.FinalizationResult {
	res := &types.FinalizationResult{
		Error: nil,
	}
	start := time.Now()

	if d.bc.GetSlotInfo() == nil {
		errStr := "no slot info"
		res.Error = &errStr
		log.Error("Handle Finalize: response (no slot info)", "result", res, "err", errStr)
		return res
	}
	d.setLastFinalizeApiSlot()

	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := errSynchronization.Error()
		res.Error = &errStr
		log.Error("Handle Finalize: response (busy)", "result", res, "err", errStr)
		// 		return res
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	if data.BaseSpine != nil {
		log.Info("Handle Finalize: start",
			"data.SyncMode", data.SyncMode,
			"cp.FinEpoch", data.Checkpoint.FinEpoch,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.Root", fmt.Sprintf("%#x", data.Checkpoint.Root),
			"baseSpine", fmt.Sprintf("%#x", data.BaseSpine),
			"spines", data.Spines,
			"ValSyncData", data.ValSyncData,
			"\u2692", params.BuildId)
	} else {
		log.Info("Handle Finalize: start",
			"data.SyncMode", data.SyncMode,
			"cp.FinEpoch", data.Checkpoint.FinEpoch,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"baseSpine", nil,
			"spines", data.Spines,
			"ValSyncData", data.ValSyncData,
			"\u2692", params.BuildId)
	}

	if data.ValSyncData != nil {
		for _, vs := range data.ValSyncData {
			log.Info("received validator sync",
				"OpType", vs.OpType,
				"Index", vs.Index,
				"Creator", vs.Creator.Hex(),
				"ProcEpoch", vs.ProcEpoch,
				"Amount", vs.Amount,
				"InitTxHash", vs.InitTxHash.Hex(),
			)
		}
	}

	var err error
	baseSpine := *data.BaseSpine
	spines := data.Spines
	// if baseSpine is in spines - remove
	if bi := spines.IndexOf(baseSpine); bi >= 0 {
		spines = spines[bi+1:]
	}
	//forward finalization
	spines, baseSpine, err = d.finalizer.ForwardFinalization(spines, baseSpine)
	if err != nil {
		e := err.Error()
		res.Error = &e
		log.Error("Handle Finalize: forward finalization failed", "syncMode", data.SyncMode, "err", err)
		return res
	}

	switch data.SyncMode {
	case types.NoSync:
		if err = d.handleSyncUnloadedBlocks(baseSpine, spines, data.Checkpoint); err != nil {
			strErr := err.Error()
			res.Error = &strErr
			log.Error("Handle Finalize: response (sync failed)", "syncMode", data.SyncMode, "result", res, "err", err)
			return res
		}
	case types.MainSync:
		if err = d.downloader.MainSync(baseSpine, spines); err != nil {
			strErr := err.Error()
			res.Error = &strErr
			log.Error("Handle Finalize: response (sync failed)", "syncMode", data.SyncMode, "result", res, "err", err)
			return res
		}
	case types.HeadSync:
		if err = d.downloader.DagSync(baseSpine, spines); err != nil {
			strErr := err.Error()
			res.Error = &strErr
			log.Error("Handle Finalize: response (sync failed)", "syncMode", data.SyncMode, "result", res, "err", err)
			return res
		}
	default:
		e := fmt.Sprintf("bad sync mode=%d", data.SyncMode)
		res.Error = &e
		log.Error("Handle Finalize: bad sync mode", "syncMode", data.SyncMode, "result", res)
		return res
	}

	// finalization
	if len(spines) > 0 {
		if err = d.finalizer.Finalize(&spines, &baseSpine); err != nil {
			if err == core.ErrInsertUncompletedDag || err == finalizer.ErrSpineNotFound {
				log.Error("Handle Finalize: response (finalize failed)", "result", res, "err", err)
				return res
			}
			e := err.Error()
			res.Error = &e
		} else {
			d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
			if err := era.HandleEra(d.bc, data.Checkpoint); err != nil {
				strErr := err.Error()
				res.Error = &strErr
				log.Error("Handle Finalize: update era failed 1", "syncMode", data.SyncMode, "result", res, "err", err)
				return res
			}
		}
	} else {
		d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
		if err := era.HandleEra(d.bc, data.Checkpoint); err != nil {
			strErr := err.Error()
			res.Error = &strErr
			log.Error("Handle Finalize: update era failed 2", "syncMode", data.SyncMode, "result", res, "err", err)
			return res
		}
	}

	for i, vs := range data.ValSyncData {
		log.Info("Handle Finalize: valSync", "i", i, "valSyncData", vs.Print())
	}

	// handle validator sync data
	d.bc.AppendNotProcessedValidatorSyncData(data.ValSyncData)

	lfHeader := d.bc.GetLastFinalizedHeader()

	lfHash := lfHeader.Hash()
	res.LFSpine = &lfHash

	if cp := d.bc.GetLastCoordinatedCheckpoint(); cp != nil {
		res.CpEpoch = &cp.FinEpoch
		res.CpRoot = &cp.Root
	}
	if data.SyncMode == types.NoSync || data.SyncMode == types.HeadSync {
		d.bc.SetIsSynced(true)
		log.Info("HandleFinalize SetIsSynced true", "SyncMode", data.SyncMode)
	}

	log.Info("Handle Finalize: response",
		"resEpoch", *res.CpEpoch,
		"resSpine", res.LFSpine.Hex(),
		"resRoot", res.CpRoot.Hex(),
	)
	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "Finalize",
	)
	return res
}

// handleSyncUnloadedBlocks:
// 1. check is synchronization required
// 2. switch on sync mode
// 3. start sync process
// 4. if chain head reached - switch off sync mode
func (d *Dag) handleSyncUnloadedBlocks(baseSpine common.Hash, spines common.HashArray, cp *types.Checkpoint) error {
	defer func(start time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "dag.handleSyncUnloadedBlocks",
		)
	}(time.Now())

	if len(spines) == 0 {
		return nil
	}
	baseHeader := d.bc.GetHeaderByHash(baseSpine)
	if baseHeader == nil || baseHeader.Nr() == 0 && baseHeader.Height > 0 {
		return downloader.ErrInvalidBaseSpine
	}
	isSync, err := d.hasUnloadedBlocks(spines)
	if err != nil {
		return err
	}
	if !isSync {
		return nil
	}
	d.bc.SetIsSynced(false)

	lfNr := d.bc.GetLastFinalizedNumber()
	err = d.finalizer.SetSpineState(&baseSpine, lfNr)
	if err != nil {
		return err
	}

	d.bc.SetSyncCheckpointCache(cp)
	defer d.bc.ResetSyncCheckpointCache()
	if err = d.downloader.DagSync(baseSpine, spines); err != nil {
		return err
	}
	return nil
}

func (d *Dag) hasUnloadedBlocks(spines common.HashArray) (bool, error) {
	defer func(start time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "dag.hasUnloadedBlocks",
		)
	}(time.Now())

	var (
		unl common.HashArray
		err error
	)
	for _, spine := range spines.Reverse() {
		spHeader := d.bc.GetHeaderByHash(spine)
		if spHeader == nil {
			return true, nil
		}
		_, _, unl, err = d.bc.CollectAncestorsAftCpByParents(spHeader.ParentHashes, spHeader.CpHash)
		if err != nil {
			return false, err
		}
		if len(unl) > 0 {
			return true, nil
		}
	}

	return false, nil
}

// HandleCoordinatedState return coordinated state
func (d *Dag) HandleCoordinatedState() *types.FinalizationResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := errSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	lfHeader := d.bc.GetLastFinalizedHeader()
	lfHash := lfHeader.Hash()
	res := &types.FinalizationResult{
		Error:   nil,
		LFSpine: &lfHash,
	}
	var cpepoch uint64
	if cp := d.bc.GetLastCoordinatedCheckpoint(); cp != nil {
		res.CpEpoch = &cp.FinEpoch
		res.CpRoot = &cp.Root
		cpepoch = cp.FinEpoch
	}
	log.Info("Handle CoordinatedState: response", "epoch", cpepoch, "result", res)
	return res
}

// HandleGetCandidates collect next finalization candidates
func (d *Dag) HandleGetCandidates(slot uint64) *types.CandidatesResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := errSynchronization.Error()
		return &types.CandidatesResult{
			Error: &errStr,
		}
	}

	defer func(tstart time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(tstart)),
			"func:", "GetCandidates",
		)
	}(time.Now())
	// collect next finalization candidates
	//candidates, err := d.finalizer.GetFinalizingCandidates(&slot)
	fromSlot := d.bc.GetLastFinalizedHeader().Slot
	optSpines, err := d.bc.GetOptimisticSpines(fromSlot)
	if len(optSpines) == 0 {
		log.Info("No spines found", "tips", d.bc.GetTips().Print(), "slot", slot)
	}

	// take the first element of each record in the input slice.
	var candidates common.HashArray
	for _, candidate := range optSpines {
		if len(candidate) > 0 {
			header := d.bc.GetHeaderByHash(candidate[0])
			if header.Slot <= slot {
				candidates = append(candidates, candidate[0])
			}
		}
	}

	if len(candidates) == 0 {
		log.Info("No candidates found", "slot", slot)
	}

	log.Debug("Candidates HandleGetCandidates: get finalizing candidates", "err", err, "toSlot", slot, "fromSlot", fromSlot, "candidates", candidates, "\u2692", params.BuildId)
	res := &types.CandidatesResult{
		Error:      nil,
		Candidates: candidates,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	return res
}

func (d *Dag) HandleGetOptimisticSpines(fromSpine common.Hash) *types.OptimisticSpinesResult {
	//skip if synchronising
	if d.downloader.Synchronising() {
		errStr := errSynchronization.Error()
		return &types.OptimisticSpinesResult{
			Error: &errStr,
		}
	}

	tstart := time.Now()

	log.Info("Handle GetOptimisticSpines: start", "fromSpine", fromSpine.Hex())

	spineBlock := d.bc.GetHeaderByHash(fromSpine)
	if spineBlock == nil {
		err := errors.New("bad params: base spine not found").Error()
		return &types.OptimisticSpinesResult{
			Error: &err,
			Data:  nil,
		}
	}

	// collect optimistic spines
	spines, err := d.bc.GetOptimisticSpines(spineBlock.Slot)
	if len(spines) == 0 {
		log.Info("No spines for tips", "tips", d.bc.GetTips().Print())
	}

	res := &types.OptimisticSpinesResult{
		Error: nil,
		Data:  spines,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	log.Info("Handle GetOptimisticSpines: response", "result", len(res.Data), "elapsed", common.PrettyDuration(time.Since(tstart)), "\u2692", params.BuildId)
	return res
}

// HandleSyncSlotInfo set initial state to start head sync with coordinating network.
func (d *Dag) HandleSyncSlotInfo(slotInfo types.SlotInfo) (bool, error) {
	log.Info("Handle Sync Slot info", "params", slotInfo)
	si := d.bc.GetSlotInfo()
	if si == nil {
		err := d.bc.SetSlotInfo(&slotInfo)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if si.GenesisTime == slotInfo.GenesisTime &&
		si.SecondsPerSlot == slotInfo.SecondsPerSlot &&
		si.SlotsPerEpoch == slotInfo.SlotsPerEpoch {
		return true, nil
	} else if d.eth.IsDevMode() {
		err := d.bc.SetSlotInfo(&slotInfo)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// HandleValidateFinalization validate given spines sequence of finalization.
// Checks existence and order by slot of finalization sequence.
func (d *Dag) HandleValidateFinalization(spines common.HashArray) (bool, error) {
	log.Info("Handle validate finalisation: start", "spines", spines)

	headers := d.bc.GetBlocksByHashes(spines)
	lastSlot := uint64(0)

	for _, s := range spines {
		spine := headers[s]
		if spine == nil {
			err := fmt.Errorf("spine not found: %#x", s)
			log.Warn("Handle validate finalisation: failed", "err", err)
			return false, err
		}
		if lastSlot >= spine.Slot() {
			err := fmt.Errorf("bad order by slot: %#x slot=%d prevSlot=%d", s, spine.Slot(), lastSlot)
			log.Warn("Handle validate finalisation: failed", "err", err)
			return false, err
		}
	}
	return true, nil
}

// HandleValidateSpines collect next finalization candidates
func (d *Dag) HandleValidateSpines(spines common.HashArray) (bool, error) {
	log.Debug("Candidates HandleValidateSpines req", "candidates", spines, "elapsed", "\u2692", params.BuildId)
	return d.finalizer.IsValidSequenceOfSpines(spines)
}

func (d *Dag) StartWork() {
	startTicker := time.NewTicker(2000 * time.Millisecond)

	tickSec := 0
	for {
		select {
		case <-d.exitChan:
			d.exitProcedurre()
			return
		case <-startTicker.C:
			si := d.bc.GetSlotInfo()
			currentTime := time.Now()
			if si != nil {
				genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)

				if currentTime.Before(genesisTime) {
					if tickSec != currentTime.Second() && currentTime.Second()%5 == 0 {
						timeRemaining := genesisTime.Sub(currentTime)
						log.Info("Time before start", "hour", timeRemaining.Truncate(time.Second))
					}
				} else {
					log.Info("Chain genesis time reached", "curSlot", si.CurrentSlot())
					startTicker.Stop()
					go d.workLoop()

					return
				}
				tickSec = currentTime.Second()
			}
		}
	}
}

func (d *Dag) StopWork() {
	d.exitChan <- struct{}{}
	<-d.enddChan
	close(d.enddChan)
}

func (d *Dag) exitProcedurre() {
	d.creator.Stop()
	d.downloader.Terminate()
	d.bc.DagMuLock()
	d.bc.DagMuUnlock()
	close(d.exitChan)
	close(d.errChan)
	d.enddChan <- struct{}{}
}

func (d *Dag) workLoop() {
	secPerSlot := d.bc.GetSlotInfo().SecondsPerSlot
	genesisTime := time.Unix(int64(d.bc.GetSlotInfo().GenesisTime), 0)
	slotTicker := slotticker.NewSlotTicker(genesisTime, secPerSlot)

	for {
		select {
		case <-d.exitChan:
			slotTicker.Done()
			d.exitProcedurre()
			return
		case err := <-d.errChan:
			close(d.errChan)
			close(d.exitChan)
			slotTicker.Done()
			log.Error("Dag worker stopped with error", "error", err)
			return
		case slot := <-slotTicker.C():
			if slot == 0 {
				d.bc.SetIsSynced(true)
				d.resetCheckpoint()
				continue
			}
			if !d.bc.IsSynced() {
				log.Info("dag workloop !d.bc.IsSynced()", "slot", slot, "IsSynced", d.bc.IsSynced())
				d.resetCheckpoint()
				continue
			}
			if d.isCoordinatorConnectionLost() {
				d.bc.SetIsSynced(false)
				log.Info("Detected coordinator skipped slot handling: sync mode on", "slot", slot, "coordSlot", d.getLastFinalizeApiSlot())
				continue
			}
			currentEpoch := d.bc.GetSlotInfo().SlotToEpoch(d.bc.GetSlotInfo().CurrentSlot())
			log.Debug("######### curEpoch to eraInfo toEpoch", "epoch", currentEpoch, "d.bc.GetEraInfo().ToEpoch()", d.bc.GetEraInfo().ToEpoch())

			var (
				err          error
				slotCreators []common.Address
			)

			startTransitionSlot, err := d.bc.GetSlotInfo().SlotOfEpochStart(d.bc.GetEraInfo().ToEpoch() + 1 - d.bc.Config().TransitionPeriod)
			if err != nil {
				log.Error("Error calculating start transition slot", "error", err)
				return
			}

			endTransitionSlot, err := d.bc.GetSlotInfo().SlotOfEpochEnd(d.bc.GetEraInfo().ToEpoch())

			log.Info("New slot",
				"slot", slot,
				"currentEpoch", d.bc.GetSlotInfo().SlotToEpoch(slot),
				"era", d.bc.GetEraInfo().Number(),
				"startTransEpoch", d.bc.GetEraInfo().ToEpoch()+1-d.bc.Config().TransitionPeriod,
				"startTransSlot", startTransitionSlot,
				"endTransEpoch", d.bc.GetEraInfo().ToEpoch(),
				"endTransSlot", endTransitionSlot,
			)

			// TODO: uncomment this code for subnetwork support, add subnet and get it to the creators getter (line 253)
			//if d.bc.Config().IsForkSlotSubNet1(currentSlot) {
			//	creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, currentSlot,subnet)
			//	if err != nil {
			//		d.errChan <- err
			//	}
			//} else {}
			// TODO: move it to else condition

			slotCreators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, slot)
			if err != nil {
				d.errChan <- err
			}

			// todo check
			log.Info("CheckShuffle - dag SlotCreators", "slot", slot, "creators", slotCreators)

			go d.work(slot, slotCreators)
		}
	}
}

func (d *Dag) work(slot uint64, slotCreators []common.Address) {
	if !d.bc.IsSynced() {
		return
	}
	if d.isSlotLocked(slot) {
		return
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	if err := d.removeTipsWithOutdatedCp(); err != nil {
		return
	}

	if d.Creator().IsRunning() {
		var canCreate bool
		for _, account := range slotCreators {
			if d.Creator().IsCreatorActive(account) {
				canCreate = true
				break
			}
		}
		if canCreate {
			tips := d.bc.GetTips()
			checkpoint := d.getCheckpoint()
			if checkpoint == nil {
				checkpoint = d.bc.GetLastCoordinatedCheckpoint()
			}

			err := d.Creator().RunBlockCreation(slot, slotCreators, tips, checkpoint)
			if err != nil {
				log.Error("Create block error", "error", err)
			}
		}
	} else {
		log.Warn("Creator stopped")
	}

	d.saveCheckpoint(d.bc.GetLastCoordinatedCheckpoint())
}

// getLastFinalizeApiSlot returns the slot of last HandleFinalize api call.
func (d *Dag) getLastFinalizeApiSlot() uint64 {
	return atomic.LoadUint64(&d.lastFinApiSlot)
}

// setLastFinalizeApiSlot set the slot of last HandleFinalize api call.
func (d *Dag) setLastFinalizeApiSlot() {
	slot := uint64(0)
	if si := d.bc.GetSlotInfo(); si != nil {
		slot = si.CurrentSlot()
	}
	atomic.StoreUint64(&d.lastFinApiSlot, slot)
}

// isCoordinatorConnectionLost returns true if the difference of
// current slot and the last coord slot grater than 1.
func (d *Dag) isCoordinatorConnectionLost() bool {
	slot := uint64(0)
	si := d.bc.GetSlotInfo()
	if si == nil {
		return true
	}
	slot = si.CurrentSlot()
	return (slot - d.getLastFinalizeApiSlot()) > si.SlotsPerEpoch
}

// removeTipsWithOutdatedCp remove tips with outdated cp.
func (d *Dag) removeTipsWithOutdatedCp() error {
	tips := d.bc.GetTips()
	rmTips := common.HashArray{}
	for th, tip := range tips.Copy() {
		if th == d.bc.Genesis().Hash() {
			continue
		}
		cp := d.bc.GetCoordinatedCheckpoint(tip.CpHash)
		if cp == nil {
			err := errors.New("tips checkpoint not found")
			log.Error("Removing tips with outdated cp failed", "err", err, "tip.CpHash", tip.CpHash.Hex(), "tip.Hash", th.Hex())
			return err
		}
		if d.bc.IsCheckpointOutdated(cp) {
			rmTips = append(rmTips, th)
			log.Warn("Creator detect outdated cp", "tip.CpHash", tip.CpHash.Hex(), "tip.Hash", th.Hex())
		}
	}
	if len(rmTips) > 0 {
		d.bc.RemoveTips(rmTips)
		d.bc.WriteCurrentTips()
	}
	return nil
}

func (d *Dag) getCheckpoint() *types.Checkpoint {
	return d.checkpoint
}

func (d *Dag) saveCheckpoint(cp *types.Checkpoint) {
	d.checkpoint = cp
}

func (d *Dag) resetCheckpoint() {
	d.checkpoint = nil
}

// isSlotLocked compare incoming epoch/slot with the latest epoch/slot of chain.
func (d *Dag) isSlotLocked(slot uint64) bool {
	if slot <= d.bc.GetLastFinalizedHeader().Slot {
		return true
	}

	return false
}
