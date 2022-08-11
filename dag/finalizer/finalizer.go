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
	"github.com/waterfall-foundation/gwat/dag/finalizer/interfaces"
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

	orderedChain := spinesToChain(&spinesMap, f.eth.BlockChain())
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

// RetrieveFinalizingChain retrieves dag blocks in order of finalization
func (f *Finalizer) RetrieveFinalizingChain(tips types.Tips) (types.Blocks, *types.BlockDAG) {
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
	finChain := make(types.Blocks, 0)
	for _, h := range finOrd {
		bl := blocks[h]
		if bl == nil {
			return finChain, dag
		}
		if bl.Hash() == finBlock.Hash() {
			finChain = append(finChain, bl)
			break
		}
		if bl.Height() >= finBlock.Height() {
			log.Error("Finalizer failed due to unacceptable block height", "bl.Height", bl.Height(), "finBlock.Height", finBlock.Height(), "bl.Hash", bl.Hash(), "finBlock.Hash()", finBlock.Hash())
			return nil, dag
		}
		finChain = append(finChain, bl)
	}
	return finChain, dag
}

func (f *Finalizer) GetFinalizingCandidatesSpines() (types.SlotSpineHashMap, error) {
	bc := f.eth.BlockChain()
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
	if finChain == nil || len(finChain) == 0 {
		return nil, nil
	}

	lastFinNr := finDag.LastFinalizedHeight
	nextFinNr := lastFinNr + 1
	candidates := types.Blocks{}
	for _, block := range finChain {
		if block.Number() != nil && block.Nr() < nextFinNr && len(finChain) > 0 {
			bc.FinalizeTips(common.HashArray{block.Hash()}, common.Hash{}, lastFinNr)
			continue
		}
		if block.Number() == nil {
			candidates = append(candidates, block)
		}
		nextFinNr++
	}

	spines, err := CalculateSpines(candidates)
	if err != nil {
		return nil, err
	}

	return spines, nil
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
	finChain, finDag := f.RetrieveFinalizingChain(*tips)
	if finChain == nil || len(finChain) == 0 {
		return &candidates, nil
	}

	spines, err := CalculateSpines(finChain)
	if err != nil {
		return nil, err
	}

	finChain = spinesToChain(&spines, bc)

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

func CalculateSpine(blocks types.Blocks) *types.Block {
	if len(blocks) == 0 {
		return nil
	}

	if len(blocks) == 1 {
		return blocks[0]
	}

	maxHeightBlocks := blocks.GetMaxHeightBlocks()
	if len(maxHeightBlocks) == 1 {
		return maxHeightBlocks[0]
	}

	maxParentHashesBlocks := maxHeightBlocks.GetMaxParentHashesLenBlocks()
	if len(maxParentHashesBlocks) == 1 {
		return maxParentHashesBlocks[0]
	}

	var maxHashIndex int
	for i := range maxParentHashesBlocks {
		if maxParentHashesBlocks[i].Hash().String() > maxParentHashesBlocks[maxHashIndex].Hash().String() {
			maxHashIndex = i
		}
	}

	return maxParentHashesBlocks[maxHashIndex]
}

func CalculateSpines(blocks types.Blocks) (types.SlotSpineHashMap, error) {
	blocksBySlot, err := blocks.GroupBySlot()
	if err != nil {
		return nil, err
	}

	spines := make(types.SlotSpineHashMap)
	for slot, bs := range blocksBySlot {
		spines[slot] = CalculateSpine(bs)
	}
	return spines, nil
}

func OrderChain(blocks types.Blocks, bc interfaces.BlockChain) (types.Blocks, error) {
	spines, err := CalculateSpines(blocks)
	if err != nil {
		return nil, err
	}

	orderedChain := spinesToChain(&spines, bc)
	return orderedChain, nil
}

func spinesToChain(spines *types.SlotSpineHashMap, bc interfaces.BlockChain) types.Blocks {
	candidatesInChain := make(map[common.Hash]struct{})
	maxSlot := spines.GetMaxSlot()
	minSlot := spines.GetMinSlot()

	chain := make(types.Blocks, 0)
	for slot := int(minSlot); slot <= int(maxSlot); slot++ {
		if _, ok := (*spines)[uint64(slot)]; !ok {
			continue
		}
		spine := (*spines)[uint64(slot)]

		processBlock(spine, candidatesInChain, bc, &chain)
	}
	return chain
}

func calculateChain(block *types.Block, candidatesInChain map[common.Hash]struct{}, bc interfaces.BlockChain) types.Blocks {
	chain := make(types.Blocks, 0, len(block.ParentHashes()))

	processBlock(block, candidatesInChain, bc, &chain)

	return chain
}

func processBlock(block *types.Block, candidatesInChain map[common.Hash]struct{}, bc interfaces.BlockChain, chain *types.Blocks) {
	if _, wasProcessed := candidatesInChain[block.Hash()]; wasProcessed {
		return
	}
	orderedHashes := core.GetOrderedParentHashes(block, bc)
	for _, ph := range orderedHashes {
		parent := bc.GetBlockByHash(ph)
		if _, wasProcessed := candidatesInChain[ph]; !wasProcessed && parent.Number() == nil {
			candidatesInChain[block.Hash()] = struct{}{}
			if chainPart := calculateChain(parent, candidatesInChain, bc); len(chainPart) != 0 {
				*chain = append(*chain, chainPart...)
			}
			candidatesInChain[ph] = struct{}{}
		}
	}
	*chain = append(*chain, block)
}
