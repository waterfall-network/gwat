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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/dag/creator"
	"github.com/ethereum/go-ethereum/dag/finalizer"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
}

// Finalizer creates blocks and searches for proof-of-work values.
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

func (d *Dag) HandleConsensus(data *ConsensusInfo) *ConsensusResult {
	d.bc.DagMu.Lock()
	defer d.bc.DagMu.Unlock()

	errs := map[string]string{}
	info := map[string]string{}
	tstart := time.Now()

	d.setConsensusInfo(data)

	log.Info("============= HandleConsensus::=============", "data", data)

	// finalization
	if len(data.Finalizing) > 0 {
		if err := d.finalizer.Finalize(*data.Finalizing.Copy()); err != nil {
			errs["finalization"] = err.Error()
		}
	}

	log.Info("============= HandleConsensus::finalized=============", "err", errs["finalization"], "data", data)

	// collect next finalization candidats
	candidates, err := d.finalizer.GetFinalizingCandidates()
	if err != nil {
		errs["candidates"] = err.Error()
	}

	log.Info("============= HandleConsensus::GetFinalizingCandidates=============", "err", err, "candidates", candidates, "elapsed", common.PrettyDuration(time.Since(tstart)))

	// create block
	if d.creator.IsRunning() && len(errs) == 0 {
		assigned := &creator.Assignment{
			Slot:     data.Slot,
			Epoch:    data.Epoch,
			Creators: data.Creators,
		}

		go func() {
			crtStart := time.Now()
			crtInfo := map[string]string{}
			block, crtErr := d.creator.CreateBlock(assigned)
			if crtErr != nil {
				crtInfo["error"] = crtErr.Error()
			}
			if block != nil {
				crtInfo["newBlock"] = block.Hash().Hex()
			}
			log.Info("============= HandleConsensus::create=============", "IsRunning", d.creator.IsRunning(), "crtInfo", crtInfo, "elapsed", common.PrettyDuration(time.Since(crtStart)))
		}()
	}

	info["elapsed"] = fmt.Sprintf("%s", common.PrettyDuration(time.Since(tstart)))
	res := &ConsensusResult{
		Error:      nil,
		Info:       &info,
		Candidates: candidates,
	}
	if len(errs) > 0 {
		strBuf, _ := json.Marshal(errs)
		estr := fmt.Sprintf("%s", strBuf)
		res.Error = &estr
	}
	log.Info("============= HandleConsensus::response=============", "result", res)
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
