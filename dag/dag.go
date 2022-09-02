//Package dag implements:
//- consensus functionality
//- finalizing process
//- block creation process

package dag

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/consensus"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/dag/creator"
	"github.com/waterfall-foundation/gwat/dag/finalizer"
	"github.com/waterfall-foundation/gwat/eth/downloader"
	"github.com/waterfall-foundation/gwat/event"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/params"
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	Etherbase() (eb common.Address, err error)
	SetEtherbase(etherbase common.Address)
	CreatorAuthorize(creator common.Address) error
}

type Dag struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	consensusInfo     *ConsensusInfo
	consensusInfoFeed event.Feed

	eth Backend
	bc  *core.BlockChain

	//creator
	creator *creator.Creator

	//finalizer
	finalizer *finalizer.Finalizer

	busy int32
}

// New creates new instance of Dag
func New(eth Backend, chainConfig *params.ChainConfig, mux *event.TypeMux, creatorConfig *creator.Config, engine consensus.Engine) *Dag {
	d := &Dag{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
		bc:          eth.BlockChain(),
		creator:     creator.New(creatorConfig, chainConfig, engine, eth, mux),
		finalizer:   finalizer.New(chainConfig, eth, mux),
	}
	atomic.StoreInt32(&d.busy, 0)

	return d
}

// Creator get current creator
func (d *Dag) Creator() *creator.Creator {
	return d.creator
}

// HandleConsensus handles consensus data
// 1. block finalization
// 2. collect next finalization candidates
// 3. new block creation
// 4. return result
func (d *Dag) HandleConsensus(data *ConsensusInfo, accounts []common.Address) *ConsensusResult {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	errs := map[string]string{}
	info := map[string]string{}
	tstart := time.Now()

	d.setConsensusInfo(data)

	log.Info("Handle Consensus: start", "data", data)

	// finalization
	if len(data.Finalizing) > 0 {
		if err := d.finalizer.Finalize(&data.Finalizing); err != nil {
			errs["finalization"] = err.Error()
		}
	}

	//log.Info("Handle Consensus: finalized", "err", errs["finalization"], "data", data)

	// collect next finalization candidates
	const slotsDelay = uint64(2) // number of slots of delay to retrieve candidates
	candidatesSlot := data.Slot - slotsDelay
	if candidatesSlot < 0 {
		candidatesSlot = 0
	}
	candidates, err := d.finalizer.GetFinalizingCandidates(&candidatesSlot)
	if err != nil {
		errs["candidates"] = err.Error()
	}

	if len(candidates) == 0 {
		log.Info("No candidates for tips", "tips", d.bc.GetTips().Print())
	}

	log.Info("Handle Consensus: get finalizing candidates", "err", err, "candidates", candidates, "elapsed", common.PrettyDuration(time.Since(tstart)))

	// create block
	tips, unloaded := d.bc.ReviseTips()
	dagSlots := d.countDagSlots(tips)

	log.Info("Handle Consensus: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.FinalisationDelaySlots+1,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"unloaded", unloaded,
		"dagSlots", dagSlots,
	)

	if d.creator.IsRunning() && len(errs) == 0 && len(unloaded) == 0 && dagSlots != -1 && dagSlots <= finalizer.FinalisationDelaySlots+1 {
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

					block, crtErr := d.creator.CreateBlock(assigned, tips)
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

	info["elapsed"] = common.PrettyDuration(time.Since(tstart)).String()
	res := &ConsensusResult{
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
// 2. new block creation
func (d *Dag) HandleFinalize(data *ConsensusInfo, accounts []common.Address) *FinalizationResult {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	errs := map[string]string{}

	d.setConsensusInfo(data)

	log.Info("Handle Consensus: start", "data", data)

	// finalization
	if len(data.Finalizing) > 0 {
		if err := d.finalizer.Finalize(&data.Finalizing); err != nil {
			errs["finalization"] = err.Error()
		}
	}

	//log.Info("Handle Consensus: finalized", "err", errs["finalization"], "data", data)

	// create block
	tips, unloaded := d.bc.ReviseTips()
	dagSlots := d.countDagSlots(tips)

	log.Info("Handle Consensus: create condition",
		"condition", d.creator.IsRunning() && len(errs) == 0 && dagSlots != -1 && dagSlots <= finalizer.FinalisationDelaySlots+1,
		"IsRunning", d.creator.IsRunning(),
		"errs", errs,
		"unloaded", unloaded,
		"dagSlots", dagSlots,
	)

	if d.creator.IsRunning() && len(errs) == 0 && len(unloaded) == 0 && dagSlots != -1 && dagSlots <= finalizer.FinalisationDelaySlots+1 {
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
					if err := d.eth.CreatorAuthorize(coinbase); err != nil {
						log.Error("Creator authorize err", "err", err, "creator", coinbase)
						return
					}
					log.Info("Creator assigned", "creator", coinbase)

					block, crtErr := d.creator.CreateBlock(assigned, tips)
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

	res := &FinalizationResult{
		Error: nil,
	}
	if len(errs) > 0 {
		strBuf, _ := json.Marshal(errs)
		estr := string(strBuf)
		res.Error = &estr
	}
	log.Info("Handle Finalize: response", "result", res)
	return res
}

// HandleGetCandidates collect next finalization candidates
func (d *Dag) HandleGetCandidates(slot uint64) *CandidatesResult {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	tstart := time.Now()

	// collect next finalization candidates
	candidates, err := d.finalizer.GetFinalizingCandidates(&slot)
	if len(candidates) == 0 {
		log.Info("No candidates for tips", "tips", d.bc.GetTips().Print())
	}
	log.Info("Handle Consensus: get finalizing candidates", "err", err, "candidates", candidates, "elapsed", common.PrettyDuration(time.Since(tstart)))
	res := &CandidatesResult{
		Error:      nil,
		Candidates: candidates,
	}
	if err != nil {
		estr := err.Error()
		res.Error = &estr
	}
	log.Info("Handle GetCandidates: response", "result", res)
	return res
}

// GetConsensusInfo returns the last info received from the consensus network
func (d *Dag) GetConsensusInfo() *ConsensusInfo {
	if d.consensusInfo == nil {
		return nil
	}
	return d.consensusInfo.Copy()
}

// SetConsensusInfo set info received from the coordinating network
func (d *Dag) setConsensusInfo(dsi *ConsensusInfo) {
	d.consensusInfo = dsi
	d.emitDagSyncInfo()
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
