// Package finalizer implements chain finalization of DAG network:
// - ordering (defines order of blocks in finalized chain)
// - apply transactions
// - state propagation
package finalizer

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	FinalisationDelaySlots = 4
)

// Backend wraps all methods required for finalizing.
type Backend interface {
	BlockChain() *core.BlockChain
	Downloader() *downloader.Downloader
}

// Finalizer creates blocks and searches for proof-of-work values.
type Finalizer struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	eth     Backend
	running int32 // The indicator whether the finalizer is running or not.
	busy    int32 // The indicator whether the finalizer is finalizing blocks.
}

// New create new instance of Finalizer
func New(chainConfig *params.ChainConfig, eth Backend, mux *event.TypeMux) *Finalizer {
	f := &Finalizer{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
	}
	atomic.StoreInt32(&f.running, 0)
	atomic.StoreInt32(&f.busy, 0)

	return f
}

// Finalize start finalization procedure
func (f *Finalizer) Finalize(chain NrHashMap) error {

	if f.isSyncing() {
		return ErrSyncing
	}
	if atomic.LoadInt32(&f.busy) == 1 {
		log.Info("⌛ Finalization is skipped: process busy")
		return ErrBusy
	}

	atomic.StoreInt32(&f.busy, 1)
	defer atomic.StoreInt32(&f.busy, 0)

	if len(chain) == 0 {
		log.Info("⌛ Finalization is skipped: received chain empty")
		return nil
	}

	log.Info("Finalization chain received", "chain", chain)

	if chain.HasGap() {
		return ErrChainGap
	}

	bc := f.eth.BlockChain()
	lastFinBlock := bc.GetLastFinalizedBlock()
	lastFinNr := lastFinBlock.Nr()

	mnr := chain.GetMinNr()
	if mnr == nil {
		log.Info("⌛ Finalization is skipped: no candidates")
		return nil
	}
	minNr := *chain.GetMinNr()
	maxNr := *chain.GetMaxNr()
	// check start from current head number
	if lastFinNr+1 != minNr {
		log.Info("Finalization: reconstructing finalising chain")
		candidates, err := f.GetFinalizingCandidates()
		if err != nil {
			return err
		}
		// try to reconstruct full finalizing chain
		for nr, h := range *candidates {
			if nr < minNr {
				chain[nr] = h
			}
		}
		minNr = *chain.GetMinNr()
		maxNr = *chain.GetMaxNr()
		// check finalizing chain again
		if chain.HasGap() || lastFinNr+1 != minNr {
			return ErrChainGap
		}
	}

	//collect and check finalizing blocks
	hashes := chain.GetHashes()
	blocks := bc.GetBlocksByHashes(*hashes)
	for _, block := range blocks {
		if block == nil {
			return ErrUnknownBlock
		}
		if block.Number() != nil {
			if chain[block.Nr()] != nil && block.Hash() == *chain[block.Nr()] {
				delete(chain, block.Nr())
			} else if block.Nr() <= lastFinNr {
				lastFinNr = block.Nr() - 1
			}
		}
	}

	// todo recalculate state
	////get current state
	//statedb, err := bc.StateAt(lastFinBlock.Root())
	//if err != nil {
	//	log.Error("Bad last finalized state", "nr", lastFinBlock.Nr(), "height", lastFinBlock.Height(), "hash", lastFinBlock.Hash().Hex())
	//	return err
	//}
	//// Enable prefetching to pull in trie node paths while processing transactions
	//statedb.StartPrefetcher("chain")
	////activeState := statedb
	//var recommitBlocks []*types.Block
	//var isRecommit bool

	// blocks finalizing
	for i := minNr; i <= maxNr; i++ {
		hash := chain[i]
		block := blocks[*hash]
		isHead := maxNr == i
		if isHead && block.Height() != i {
			log.Error("Block height mismatch finalizing number", "nr", i, "height", block.Height(), "hash", block.Hash().Hex())
		}

		// todo recalculate state
		//if block.Height() == i {
		//	var blueState *state.StateDB
		//	var stateErr error = nil
		//	//	if blue block - check state
		//	blueState, stateErr = bc.StateAt(block.Root())
		//	if stateErr != nil || isRecommit {
		//		log.Info("Recalculate blue block state : start", "finNr", "nr", i, "height", block.Height(), "hash", block.Hash().Hex())
		//		isRecommit = true
		//		// recommit red blocks transactions
		//		for rnr, bl := range recommitBlocks {
		//			log.Info("RecommitBlockTransactions", "nr", rnr, "height", block.Height(), "hash", block.Hash().Hex())
		//			statedb = bc.RecommitBlockTransactions(bl, statedb)
		//		}
		//		// recalculate state
		//		statedb, err = bc.FinalizingBlueBlock(block, statedb, true)
		//		if err != nil {
		//			log.Error("FinalizingBlueBlock failed", "err", err)
		//			panic("FinalizingBlueBlock failed")
		//		}
		//		log.Info("Recalculate blue block state : end", "nr", i, "height", block.Height(), "hash", block.Hash().Hex())
		//	} else {
		//		statedb = blueState
		//		log.Info("Recalculate blue block state : SET statedb", "nr", i, "height", block.Height(), "hash", block.Hash().Hex())
		//	}
		//	recommitBlocks = []*types.Block{}
		//} else {
		//	//	if red block
		//	recommitBlocks = append(recommitBlocks, block)
		//}

		if err := f.finalizeBlock(i, *block, isHead); err != nil {
			log.Error("block finalization failed", "nr", i, "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
			return err
		}
	}
	lastBlock := blocks[*chain[maxNr]]

	f.updateTips(*chain.GetHashes(), *lastBlock)
	log.Info("⛓ Finalization completed", "blocks", len(chain), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
	return nil
}

// updateTips update tips in accordance of finalized blocks.
func (f *Finalizer) updateTips(finHashes common.HashArray, lastBlock types.Block) {
	bc := f.eth.BlockChain()
	bc.FinalizeTips(finHashes, lastBlock.Hash(), lastBlock.Height())
	//remove stale blockDags
	for _, h := range finHashes {
		bc.DeleteBlockDag(h)
	}
}

// finalizeBlock finalize block
func (f *Finalizer) finalizeBlock(finNr uint64, block types.Block, isHead bool) error {
	bc := f.eth.BlockChain()
	nr := bc.ReadFinalizedNumberByHash(block.Hash())
	if nr != nil && *nr == finNr {
		log.Warn("Block already finalized", "finNr", "nr", nr, "height", block.Height(), "hash", block.Hash().Hex())
		return nil
	}
	//if hash := bc.ReadFinalizedHashByNumber(finNr); hash != (common.Hash{}) && hash != block.Hash() {
	//	return fmt.Errorf(fmt.Sprintf("block already finalised finNr=%d new hash=%v existed=%v", finNr, block.Hash(), hash))
	//}
	if err := bc.WriteFinalizedBlock(finNr, &block, []*types.Receipt{}, []*types.Log{}, &state.StateDB{}, isHead); err != nil {
		return err
	}

	log.Info("🔗 block finalized", "Number", finNr, "Height", block.Height(), "hash", block.Hash().Hex())
	return nil
}

// isSyncing returns true if sync process is running.
func (f *Finalizer) isSyncing() bool {
	return f.eth.Downloader().Synchronising()
}

// RetrieveFinalizingChain retrieves dag blocks in order of finalization
func (f *Finalizer) RetrieveFinalizingChain(tips types.Tips) (*[]types.Block, *types.BlockDAG) {
	bc := f.eth.BlockChain()
	dag := tips.GetFinalizingDag()
	fpts := append(dag.FinalityPoints.Uniq(), dag.Hash)
	fpts = fpts.Difference(common.HashArray{dag.LastFinalizedHash}).Uniq()
	if len(fpts) < FinalisationDelaySlots {
		return nil, dag
	}

	unl, _, _, _, _, _err := bc.ExploreChainRecursive(dag.Hash)
	if _err != nil {
		log.Error("Finalizer failed while retrieving finalizing chain", "err", _err)
		return nil, dag
	}
	if len(unl) > 0 {
		log.Error("Finalizer failed due to unknown blocks detected", "hashes", unl)
		return nil, dag
	}
	finPoints := dag.FinalityPoints.Uniq()

	log.Info("Finalizer collect finalisation points", "finPoints", finPoints)

	fpIndex := len(finPoints) - FinalisationDelaySlots
	if fpIndex < 0 {
		fpIndex = 0
	}
	finPoint := finPoints[fpIndex]
	finOrd := dag.DagChainHashes.Uniq()

	log.Info("Finalizer select candidates", "candidates", finOrd)

	blocks := bc.GetBlocksByHashes(finOrd)
	finBlock := blocks[finPoint]
	if finBlock == nil {
		log.Error("Finalizer failed due to block of finality point not found", "hash", finPoint.Hex())
		return nil, dag
	}
	finChain := []types.Block{}
	for _, h := range finOrd {
		bl := blocks[h]
		if bl == nil {
			return &finChain, dag
		}
		if bl.Hash() == finBlock.Hash() {
			finChain = append(finChain, *bl)
			break
		}
		if bl.Height() >= finBlock.Height() {
			log.Error("Finalizer failed due to unacceptable block height", "bl.Height", bl.Height(), "finBlock.Height", finBlock.Height(), "bl.Hash", bl.Hash(), "finBlock.Hash()", finBlock.Hash())
			return nil, dag
		}
		finChain = append(finChain, *bl)
	}
	return &finChain, dag
}

// GetFinalizingCandidates returns the ordered dag block hashes for finalization
func (f *Finalizer) GetFinalizingCandidates() (*NrHashMap, error) {
	bc := f.eth.BlockChain()
	candidates := NrHashMap{}
	tips, unloaded := bc.ReviseTips()
	if len(unloaded) > 0 || tips == nil || len(*tips) == 0 {
		if tips == nil {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips)
		} else {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips.Print())
		}
		return nil, ErrBadDag
	}
	finChain, finDag := f.RetrieveFinalizingChain(*tips)
	if finChain == nil || len(*finChain) == 0 {
		return &candidates, nil
	}

	lastFinNr := finDag.LastFinalizedHeight
	count := 0
	var lastBlock = (*finChain)[len(*finChain)-1]
	nextFinNr := lastFinNr + 1
	for _, block := range *finChain {
		if block.Number() != nil && block.Nr() < nextFinNr && len(*finChain) > 0 {
			bc.FinalizeTips(common.HashArray{block.Hash()}, common.Hash{}, lastFinNr)
			continue
		}
		if block.Number() == nil {
			hash := block.Hash()
			candidates[nextFinNr] = &hash
		}
		nextFinNr++
		count++
	}
	if lastBlock.Height() != nextFinNr-1 {
		return nil, ErrMismatchFinalisingPosition
	}
	return &candidates, nil
}
