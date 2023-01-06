//Package dag implements:
//- consensus functionality
//- finalizing process
//- block creation process

package dag

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/creator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/finalizer"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/headsync"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	Etherbase() (eb common.Address, err error)
	SetEtherbase(etherbase common.Address)
	CreatorAuthorize(creator common.Address) error
	IsDevMode() bool
}

type Dag struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	consensusInfo     *types.ConsensusInfo
	consensusInfoFeed event.Feed

	eth Backend
	bc  *core.BlockChain

	//creator
	creator *creator.Creator
	//finalizer
	finalizer *finalizer.Finalizer
	//headsync
	headsync *headsync.Headsync

	busy int32
}

// New creates new instance of Dag
func New(eth Backend, chainConfig *params.ChainConfig, mux *event.TypeMux, creatorConfig *creator.Config, engine consensus.Engine) *Dag {
	fin := finalizer.New(chainConfig, eth, mux)
	d := &Dag{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
		bc:          eth.BlockChain(),
		creator:     creator.New(creatorConfig, chainConfig, engine, eth, mux),
		finalizer:   fin,
		headsync:    headsync.New(chainConfig, eth, mux, fin),
	}
	atomic.StoreInt32(&d.busy, 0)
	return d
}

// Creator get current creator
func (d *Dag) Creator() *creator.Creator {
	return d.creator
}

// HandleConsensus handles consensus data
// 3. new block creation
// 4. return result
// depracated
func (d *Dag) HandleConsensus(data *types.ConsensusInfo, accounts []common.Address) *types.ConsensusResult {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.ConsensusResult{
			Error: &errStr,
		}
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	errs := map[string]string{}
	info := map[string]string{}
	tstart := time.Now()

	d.setConsensusInfo(data)

	log.Info("Handle Consensus: start", "data", data, "\u2692", params.BuildId)

	var (
		err        error
		candidates = common.HashArray{}
	)
	// create block
	tips := d.bc.GetTips()
	//tips, unloaded := d.bc.ReviseTips()
	dagSlots := d.countDagSlots(&tips)
	log.Info("Handle Consensus: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"dagSlots", dagSlots,
	)

	if d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.CreateDagSlotsLimit {
		assigned := &creator.Assignment{
			Slot:     data.Slot,
			Creators: data.Creators,
		}

		go func() {
			crtStart := time.Now()
			crtInfo := map[string]string{}
			for _, creator := range assigned.Creators {
				// if received next slot
				if d.consensusInfo.Slot > assigned.Slot {
					break
				}

				func() {
					d.eth.BlockChain().DagMu.Lock()
					defer d.eth.BlockChain().DagMu.Unlock()

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
					log.Info("HandleConsensus: create block", "dagSlots", dagSlots, "IsRunning", d.creator.IsRunning(), "crtInfo", crtInfo, "elapsed", common.PrettyDuration(time.Since(crtStart)))
				}()
			}
		}()
	}

	d.bc.WriteCreators(data.Slot, data.Creators)

	info["elapsed"] = common.PrettyDuration(time.Since(tstart)).String()
	res := &types.ConsensusResult{
		Error:      nil,
		Info:       &info,
		Candidates: candidates,
	}
	if len(errs) > 0 {
		strBuf, _ := json.Marshal(errs)
		estr := string(strBuf)
		res.Error = &estr
	}
	log.Info("Handle Consensus: response", "result", res)
	return res
}

// HandleFinalize handles consensus data
// 1. blocks finalization
func (d *Dag) HandleFinalize(data *types.FinalizationParams) *types.FinalizationResult {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.FinalizationResult{
			Error: &errStr,
		}
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	if data.BaseSpine != nil {
		log.Info("Handle Finalize: start", "baseSpine", *data.BaseSpine, "spines", data.Spines, "\u2692", params.BuildId)
	} else {
		log.Info("Handle Finalize: start", "baseSpine", nil, "spines", data.Spines, "\u2692", params.BuildId)
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
			d.bc.WriteLastCoordinatedHash(data.Spines[len(data.Spines)-1])
		}
		lfHeader := d.bc.GetLastFinalizedHeader()
		if lfHeader.Height != lfHeader.Nr() {
			err := fmt.Sprintf("☠ bad last finalized block: mismatch nr=%d and height=%d", lfHeader.Nr(), lfHeader.Height)
			if res.Error == nil {
				res.Error = &err
			} else {
				mrg := fmt.Sprintf("error[0]=%s\nerror[1]: %s", *res.Error, err)
				res.Error = &mrg
			}
		}
		lfHash := lfHeader.Hash()
		res.LFSpine = &lfHash
	}
	log.Info("Handle Finalize: response", "result", res)
	return res
}

// HandleGetCandidates collect next finalization candidates
func (d *Dag) HandleGetCandidates(slot uint64) *types.CandidatesResult {
	//skip if synchronising
	if d.eth.Downloader().Synchronising() {
		errStr := creator.ErrSynchronization.Error()
		return &types.CandidatesResult{
			Error: &errStr,
		}
	}

	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

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

// HandleHeadSyncReady set initial state to start head sync with coordinating network.
func (d *Dag) HandleHeadSyncReady(checkpoint *types.ConsensusInfo) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Head Sync Ready", "checkpoint", checkpoint)
	return d.headsync.SetReadyState(checkpoint)
}

// HandleSyncSlotInfo set initial state to start head sync with coordinating network.
func (d *Dag) HandleSyncSlotInfo(slotInfo types.SlotInfo) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
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
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Head Sync", "len(data)", len(data), "data", data)
	return d.headsync.Sync(data)
}

// HandleValidateSpines collect next finalization candidates
func (d *Dag) HandleValidateSpines(spines common.HashArray) (bool, error) {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()
	log.Info("Handle Validate Spines", "spines", spines, "\u2692", params.BuildId)
	return d.finalizer.IsValidSequenceOfSpines(spines)
}

// GetConsensusInfo returns the last info received from the consensus network
func (d *Dag) GetConsensusInfo() *types.ConsensusInfo {
	if d.consensusInfo == nil {
		return nil
	}
	return d.consensusInfo.Copy()
}

// SetConsensusInfo set info received from the coordinating network
func (d *Dag) setConsensusInfo(dsi *types.ConsensusInfo) {
	d.consensusInfo = dsi
	d.emitDagSyncInfo()
	d.bc.SetLastCoordinatedSlot(dsi.Slot)
}

// SubscribeConsensusInfoEvent registers a subscription for consensusInfo updated event
func (d *Dag) SubscribeConsensusInfoEvent(ch chan<- types.Tips) event.Subscription {
	return d.consensusInfoFeed.Subscribe(ch)
}

// emitDagSyncInfo emit consensusInfo updated event
func (d *Dag) emitDagSyncInfo() bool {
	if dsi := d.GetConsensusInfo(); dsi != nil {
		d.consensusInfoFeed.Send(*dsi)
		return true
	}
	return false
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
