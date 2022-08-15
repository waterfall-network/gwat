// Package finalizer implements chain finalization of DAG network:
// - ordering (defines order of blocks in finalized chain)
// - apply transactions
// - state propagation
package finalizer

import (
	"sync/atomic"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/eth/downloader"
	"github.com/waterfall-foundation/gwat/event"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/params"
)

const (
	FinalisationDelaySlots = 3
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
func (f *Finalizer) Finalize(spines *common.HashArray) error {

	if f.isSyncing() {
		return ErrSyncing
	}
	if atomic.LoadInt32(&f.busy) == 1 {
		log.Info("⌛ Finalization is skipped: process busy")
		return ErrBusy
	}

	atomic.StoreInt32(&f.busy, 1)
	defer atomic.StoreInt32(&f.busy, 0)

	if len(*spines) == 0 {
		log.Info("⌛ Finalization is skipped: received spines empty")
		return nil
	}

	log.Info("Finalization spines received", "spines", spines)

	bc := f.eth.BlockChain()
	lastFinBlock := bc.GetLastFinalizedBlock()
	lastFinNr := lastFinBlock.Nr()

	//collect and check finalizing blocks
	spinesMap := make(types.SlotSpineHashMap, len(*spines))
	for _, spineHash := range *spines {
		block := bc.GetBlockByHash(spineHash)
		if block == nil {
			log.Error("unknown spine hash", "spineHash", spineHash)
			return ErrUnknownHash
		}
		spinesMap[block.Slot()] = block
	}

	orderedChain := types.SpinesToChain(f.eth.BlockChain(), &spinesMap)
	if len(orderedChain) == 0 {
		log.Info("⌛ Finalization is skipped: received spines finalized")
		return nil
	}

	blocks := bc.GetBlocksByHashes(*orderedChain.GetHashes())

	for _, block := range orderedChain {
		if block == nil || block.Header() == nil {
			return ErrUnknownBlock
		}
	}

	// blocks finalizing
	for i, blck := range orderedChain {
		nr := lastFinNr + uint64(i) + 1

		block := blocks[blck.Hash()]
		block.SetNumber(&nr)
		isHead := i == len(orderedChain)-1
		if err := f.finalizeBlock(nr, *block, isHead); err != nil {
			log.Error("block finalization failed", "nr", i, "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
			return err
		}
	}
	lastBlock := blocks[orderedChain[len(blocks)-1].Hash()]

	f.updateTips(*orderedChain.GetHashes(), *lastBlock)
	log.Info("⛓ Finalization completed", "blocks", len(*spines), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
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

// GetFinalizingCandidates returns the ordered dag block hashes for finalization
func (f *Finalizer) GetFinalizingCandidates() (*common.HashArray, error) {
	bc := f.eth.BlockChain()
	candidates := common.HashArray{}
	tips, unloaded := bc.ReviseTips()
	if len(unloaded) > 0 || tips == nil || len(*tips) == 0 {
		if tips == nil {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips)
		} else {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips.Print())
		}
		return nil, ErrBadDag
	}
	finChain := bc.GetBlocksByHashes(tips.GetOrderedDagChainHashes().Uniq()).ToArray()
	spines, err := types.CalculateSpines(finChain)
	if err != nil {
		return nil, err
	}

	finChain = types.SpinesToChain(bc, &spines)

	topSpine := spines[spines.GetMaxSlot()]
	finDag := tips.Get(topSpine.Hash())

	lastFinNr := bc.GetBlockByHash(finDag.LastFinalizedHash).Number()
	for i, block := range finChain {
		if block.Number() != nil && len(finChain) > 0 {
			bc.FinalizeTips(common.HashArray{block.Hash()}, common.Hash{}, *lastFinNr)
			continue
		}

		nr := *lastFinNr + uint64(i) + 1
		block.SetNumber(&nr)
		candidates = append(candidates, block.Hash())
	}

	return spines.GetHashes(), nil
}
