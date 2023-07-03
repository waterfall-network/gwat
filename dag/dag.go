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
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
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

var errWrongInputSlot = errors.New("input slot is greater that current slot")

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
	EnterNextEra(cp *types.Checkpoint, root common.Hash) *era.Era
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
}

type ethDownloader interface {
	Synchronising() bool
	SyncChainBySpines(baseSpine common.Hash, spines common.HashArray, finEpoch uint64) (fullySynced bool, err error)
}

type Dag struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	eth        Backend
	bc         blockChain
	downloader ethDownloader

	//creator
	creator *creator.Creator
	//finalizer
	finalizer *finalizer.Finalizer

	busy           int32
	lastFinApiSlot uint64

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
		errStr := creator.ErrSynchronization.Error()
		res.Error = &errStr
		log.Error("Handle Finalize: response (busy)", "result", res, "err", errStr)
		return res
	}

	d.bc.DagMuLock()
	defer d.bc.DagMuUnlock()

	if data.BaseSpine != nil {
		log.Info("Handle Finalize: start",
			"baseSpine", fmt.Sprintf("%#x", data.BaseSpine),
			"spines", data.Spines,
			"cp.FinEpoch", data.Checkpoint.FinEpoch,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.Root", fmt.Sprintf("%#x", data.Checkpoint.Root),
			"ValSyncData", data.ValSyncData,
			"\u2692", params.BuildId)
	} else {
		log.Info("Handle Finalize: start",
			"baseSpine", nil,
			"spines", data.Spines,
			"cp.FinEpoch", data.Checkpoint.FinEpoch,
			"cp.Epoch", data.Checkpoint.Epoch,
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
			"cp.BaseSpine", fmt.Sprintf("%#x", data.Checkpoint.Spine),
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

	baseSpine := *data.BaseSpine
	spines := data.Spines
	// if baseSpine is in spines - remove
	if bi := spines.IndexOf(baseSpine); bi >= 0 {
		spines = spines[bi+1:]
	}
	//forward finalization
	spines, baseSpine = d.finalizer.ForwardFinalization(spines, baseSpine)

	if err := d.handleSyncUnloadedBlocks(baseSpine, spines, data.Checkpoint.FinEpoch); err != nil {
		strErr := err.Error()
		res.Error = &strErr
		log.Error("Handle Finalize: response (sync failed)", "result", res, "err", err)
		return res
	}

	// finalization
	if len(data.Spines) > 0 {
		if err := d.finalizer.Finalize(&spines, &baseSpine); err != nil {
			if err == core.ErrInsertUncompletedDag || err == finalizer.ErrSpineNotFound {
				// Start syncing if spine or parent is unloaded
				//d.synchronizeUnloadedBlocks(data.pines)
				log.Error("Handle Finalize: response (finalize failed)", "result", res, "err", err)
				return res
			}
			e := err.Error()
			res.Error = &e
		} else {
			d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
			go era.HandleEra(d.bc, data.Checkpoint)
		}
	} else {
		d.bc.SetLastCoordinatedCheckpoint(data.Checkpoint)
		go era.HandleEra(d.bc, data.Checkpoint)
	}

	// handle validator sync data
	d.bc.AppendNotProcessedValidatorSyncData(data.ValSyncData)

	lfHeader := d.bc.GetLastFinalizedHeader()

	lfHash := lfHeader.Hash()
	res.LFSpine = &lfHash

	if cp := d.bc.GetLastCoordinatedCheckpoint(); cp != nil {
		res.CpEpoch = &cp.FinEpoch
		res.CpRoot = &cp.Root

		// is cp finalized epoch reached the current epoch - node is synced
		si := d.bc.GetSlotInfo()
		currEpoch := si.SlotToEpoch(si.CurrentSlot())
		if currEpoch == cp.FinEpoch {
			d.bc.SetIsSynced(true)
			log.Info("HandleFinalize SetIsSynced curEpoch == finEpoch", "currEpoch", currEpoch, "finEpoch", data.Checkpoint.FinEpoch)
		}
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
func (d *Dag) handleSyncUnloadedBlocks(baseSpine common.Hash, spines common.HashArray, finEpoch uint64) error {
	if len(spines) == 0 {
		return nil
	}

	start := time.Now()
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

	var fullySynced bool
	if fullySynced, err = d.downloader.SyncChainBySpines(baseSpine, spines, finEpoch); err != nil {
		return err
	}
	if fullySynced {
		d.bc.SetIsSynced(true)
		log.Info("Node fully synced: head reached")
	}
	log.Debug("handleSyncUnloadedBlocks", "elapsed", common.PrettyDuration(time.Since(start)))
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
		errStr := creator.ErrSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	//d.bc.DagMuLock()
	//defer d.bc.DagMuUnlock()

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
		errStr := creator.ErrSynchronization.Error()
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

	//d.bc.DagMuLock()
	//defer d.bc.DagMuUnlock()

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
		errStr := creator.ErrSynchronization.Error()
		return &types.OptimisticSpinesResult{
			Error: &errStr,
		}
	}

	//d.bc.DagMuLock()
	//defer d.bc.DagMuUnlock()

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
	//d.bc.DagMuLock()
	//defer d.bc.DagMuUnlock()

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
	//d.bc.DagMuLock()
	//defer d.bc.DagMuUnlock()

	log.Debug("Candidates HandleValidateSpines req", "candidates", spines, "elapsed", "\u2692", params.BuildId)
	return d.finalizer.IsValidSequenceOfSpines(spines)
}

func (d *Dag) StartWork(accounts []common.Address) {
	startTicker := time.NewTicker(2000 * time.Millisecond)

	tickSec := 0
	for {
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
				log.Info("Chain genesis time reached")
				startTicker.Stop()
				go d.workLoop(accounts)

				return
			}
			tickSec = currentTime.Second()
		}
		//else {
		//	lcp := d.bc.GetLastCoordinatedCheckpoint()
		//	// revert to LastCoordinatedCheckpoint
		//	if lcp.Spine != d.bc.GetLastFinalizedBlock().Hash() {
		//		spineBlock := d.bc.GetBlock(lcp.Spine)
		//		if err := d.finalizer.SetSpineState(&lcp.Spine, spineBlock.Nr()); err != nil {
		//			log.Error("Revert to checkpoint spine error", "error", err)
		//		}
		//		//d.bc..SetLastFinalisedHeader(genesisHeader, genesisHeight)
		//		//d.bc.Set
		//	}
		//
		//	log.Info("Waiting for slot info from coordinator")
		//}

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
				d.bc.SetIsSynced(true)
				continue
			}

			if !d.bc.IsSynced() {
				log.Info("dag workloop !d.bc.IsSynced()", "IsSynced", d.bc.IsSynced())
				continue
			}
			if d.isCoordinatorConnectionLost() {
				d.bc.SetIsSynced(false)
				log.Info("Detected coordinator skipped slot handling: sync mode on", "slot", slot, "coordSlot", d.getLastFinalizeApiSlot())
				continue
			}
			epoch := d.bc.GetSlotInfo().SlotToEpoch(d.bc.GetSlotInfo().CurrentSlot())
			log.Debug("######### curEpoch to eraInfo toEpoch", "epoch", epoch, "d.bc.GetEraInfo().ToEpoch()", d.bc.GetEraInfo().ToEpoch())
			//if d.bc.GetEraInfo().ToEpoch() >= epoch { // TODO: check ???

			var (
				err      error
				creators []common.Address
			)

			startTransitionSlot, err := d.bc.GetSlotInfo().SlotOfEpochStart(d.bc.GetEraInfo().ToEpoch() + 1 - d.chainConfig.TransitionPeriod)
			if err != nil {
				log.Error("Error calculating start transition slot", "error", err)
				return
			}

			endTransitionSlot, err := d.bc.GetSlotInfo().SlotOfEpochEnd(d.bc.GetEraInfo().ToEpoch())

			//era.HandleEra(d.bc, slot)

			log.Info("New slot",
				"slot", slot,
				"epoch", d.bc.GetSlotInfo().SlotToEpoch(slot),
				"era", d.bc.GetEraInfo().Number(),
				"startTransEpoch", d.bc.GetEraInfo().ToEpoch()+1-d.chainConfig.TransitionPeriod,
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

			creators, err = d.bc.ValidatorStorage().GetCreatorsBySlot(d.bc, slot)
			if err != nil {
				d.errChan <- err
			}

			// todo check
			go d.work(slot, creators, accounts)
			//}
		}
	}
}

func (d *Dag) work(slot uint64, creators, accounts []common.Address) {
	//skip if synchronising
	//if d.downloader.Synchronising() {
	if !d.bc.IsSynced() {
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
	log.Info("Creator processing: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"creators", creators,
		"accounts", accounts,
	)

	if d.creator.IsRunning() && len(errs) == 0 {
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
				continue
			}

			d.eth.SetEtherbase(coinbase)
			if err = d.eth.CreatorAuthorize(coinbase); err != nil {
				log.Error("Creator authorize err", "err", err, "creator", coinbase)
				continue
			}
			log.Info("Creator assigned", "creator", coinbase)

			block, crtErr := d.creator.CreateBlock(assigned, &tips)
			if crtErr != nil {
				crtInfo["error"] = crtErr.Error()
			}

			txCount := 0
			if block != nil {
				crtInfo["newBlock"] = block.Hash().Hex()
				txCount = len(block.Transactions())
			}

			log.Info("Creator processing: create block",
				//"IsRunning", d.creator.IsRunning(),
				//"crtInfo", crtInfo,
				"txs", txCount,
				"elapsed", common.PrettyDuration(time.Since(crtStart)),
			)
		}
	}
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
	if si.IsEpochStart(slot) {
		return (slot - d.getLastFinalizeApiSlot()) > 1
	}
	return (slot - d.getLastFinalizeApiSlot()) > si.SlotsPerEpoch
}
