package creator

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus/misc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/validatorsync"
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	CreatorAuthorize(creator common.Address) error
	AccountManager() *accounts.Manager
}

// Config is the configuration parameters of block creation.
type Config struct {
	Etherbase  common.Address `toml:",omitempty"` // Public address for block creation rewards (default = first account)
	Notify     []string       `toml:",omitempty"` // HTTP URL list to be notified of new work packages (only useful in ethash).
	NotifyFull bool           `toml:",omitempty"` // Notify with pending block headers instead of work packages
	ExtraData  hexutil.Bytes  `toml:",omitempty"` // Block extra data set by the miner
	GasFloor   uint64         // Target gas floor for mined blocks.
	GasCeil    uint64         // Target gas ceiling for mined blocks.
	GasPrice   *big.Int       // Minimum gas price for mining a transaction
	Recommit   time.Duration  // The time interval for creator to re-create block creation work.
	Noverify   bool           // Disable remote block creation solution verification(only useful in ethash).
}

// environment is the Creator's current environment and holds all of the current state information.
type environment struct {
	signer types.Signer

	gasPool *core.GasPool // available gas used to pack transactions

	txsMu *sync.Mutex
	txs   map[common.Address]*txsWithCumulativeGas
}

type txsWithCumulativeGas struct {
	cumulativeGas uint64
	txs           types.Transactions
}

// Creator is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type Creator struct {
	config *Config
	engine consensus.Engine
	eth    Backend
	bc     *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Subscriptions
	mux *event.TypeMux

	current *environment // An environment for current running cycle.

	mu    sync.RWMutex // The lock used to protect the coinbase and extra fields
	extra []byte

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.
}

// New creates new Creator instance
func New(config *Config, engine consensus.Engine, eth Backend, mux *event.TypeMux) *Creator {
	creator := &Creator{
		config: config,
		engine: engine,
		eth:    eth,
		mux:    mux,
		bc:     eth.BlockChain(),
	}

	return creator
}

// API

// Start prepare to create new blocks
// sets the running status as 1
func (c *Creator) Start() {
	atomic.StoreInt32(&c.running, 1)
}

// Stop sets the running status as 0.
func (c *Creator) Stop() {
	atomic.StoreInt32(&c.running, 0)
}

// IsRunning returns an indicator whether Creator is running or not.
func (c *Creator) IsRunning() bool {
	return atomic.LoadInt32(&c.running) == 1
}

// Pending returns the currently pending block and associated state.
func (c *Creator) Pending() (*types.Block, *state.StateDB) {
	// return a snapshot to avoid contention on currentMu mutex
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()

	block := c.bc.GetLastFinalizedBlock()
	state, err := c.bc.StateAt(block.Root())
	if err != nil {
		log.Error("Get pending block and state failed", "err", err)
		return nil, nil
	}
	return block, state.Copy()
}

// PendingBlock returns the currently pending block.
//
// Note, to access both the pending block and the pending state
// simultaneously, please use Pending(), as the pending state can
// change between multiple method calls
func (c *Creator) PendingBlock() *types.Block {
	// return a snapshot to avoid contention on currentMu mutex
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()
	return c.snapshotBlock
}

// PendingBlockAndReceipts returns the currently pending block and corresponding receipts.
func (c *Creator) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	c.snapshotMu.RLock()
	defer c.snapshotMu.RUnlock()
	return c.snapshotBlock, c.snapshotReceipts
}

// SubscribePendingLogs starts delivering logs from pending transactions
// to the given channel.
func (c *Creator) SubscribePendingLogs(ch chan<- []*types.Log) event.Subscription {
	return c.pendingLogsFeed.Subscribe(ch)
}

// Hashrate retrieve current hashrate
func (c *Creator) Hashrate() uint64 {
	if pow, ok := c.engine.(consensus.PoW); ok {
		return uint64(pow.Hashrate())
	}
	return 0
}

// SetExtra sets the content used to initialize the block extra field.
func (c *Creator) SetExtra(extra []byte) error {
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra exceeds max length. %d > %v", len(extra), params.MaximumExtraDataSize)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.extra = extra
	return nil
}

// SetGasCeil sets the gaslimit to strive for when block creation post 1559.
// For pre-1559 blocks, it sets the ceiling.
func (c *Creator) SetGasCeil(ceil uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.GasCeil = ceil
}

// RunBlockCreation starts process of block creation
func (c *Creator) RunBlockCreation(slot uint64, creators []common.Address, accounts []common.Address, tips types.Tips, checkpoint *types.Checkpoint) error {
	if !c.IsRunning() {
		log.Warn("Creator stopped")
		return ErrCreatorStopped
	}

	if c.isSyncing() {
		log.Warn("Creator skipping due to synchronization")
		return ErrSynchronization
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	assigned := &Assignment{
		Slot:     slot,
		Creators: creators,
	}

	tipsBlocks, err := c.reorgTips(assigned.Slot, tips)
	if err != nil {
		return err
	}

	header, err := c.prepareBlockHeader(assigned, tipsBlocks, tips, checkpoint)
	if err != nil {
		return err
	}

	// Could potentially happen if starting to mine in an odd state.
	err = c.makeCurrent()
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		return err
	}

	wg := new(sync.WaitGroup)
	for _, account := range accounts {
		if c.isCreatorActive(account, assigned) {
			wg.Add(1)
			c.current.txsMu.Lock()
			c.current.txs[account] = &txsWithCumulativeGas{}
			c.current.txsMu.Unlock()
			go c.createNewBlock(account, assigned.Creators, types.CopyHeader(header), wg)
		}
	}

	wg.Wait()
	return nil
}

func (c *Creator) prepareBlockHeader(assigned *Assignment, tipsBlocks types.BlockMap, tips types.Tips, checkpoint *types.Checkpoint) (*types.Header, error) {
	// if max slot of parents is less or equal to last finalized block slot
	// - add last finalized block to parents
	lastFinBlock := c.bc.GetLastFinalizedBlock()
	maxParentSlot := uint64(0)
	for _, blk := range tipsBlocks {
		if blk.Slot() > maxParentSlot {
			maxParentSlot = blk.Slot()
		}
	}
	if maxParentSlot <= lastFinBlock.Slot() {
		tipsBlocks[lastFinBlock.Hash()] = lastFinBlock
	}

	parentHashes := tipsBlocks.Hashes().Sort()

	cpHeader := c.bc.GetHeader(checkpoint.Spine)
	newHeight, err := c.bc.CalcBlockHeightByTips(tips, cpHeader.Hash())
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		return nil, err
	}

	log.Info("Creator calculate block height", "newHeight", newHeight)

	era := c.bc.GetEraInfo().Number()
	if c.bc.GetSlotInfo().SlotToEpoch(c.bc.GetSlotInfo().CurrentSlot()) >= c.bc.GetEraInfo().NextEraFirstEpoch() {
		era++
	}

	header := &types.Header{
		ParentHashes: parentHashes,
		Slot:         assigned.Slot,
		Era:          era,
		Height:       newHeight,
		GasLimit:     core.CalcGasLimit(tipsBlocks.AvgGasLimit(), c.config.GasCeil),
		Extra:        c.extra,
		Time:         uint64(time.Now().Unix()),
		// Checkpoint spine block
		CpHash:        cpHeader.Hash(),
		CpNumber:      cpHeader.Nr(),
		CpBaseFee:     cpHeader.BaseFee,
		CpBloom:       cpHeader.Bloom,
		CpGasUsed:     cpHeader.GasUsed,
		CpReceiptHash: cpHeader.ReceiptHash,
		CpRoot:        cpHeader.Root,
	}

	// Get active validators number
	creatorsPerSlotCount := c.bc.Config().ValidatorsPerSlot
	if creatorsPerSlot, err := c.bc.ValidatorStorage().GetCreatorsBySlot(c.bc, header.Slot); err == nil {
		creatorsPerSlotCount = uint64(len(creatorsPerSlot))
	}
	validators, _ := c.bc.ValidatorStorage().GetValidators(c.bc, header.Slot, true, false, "RunBlockCreation")
	header.BaseFee = misc.CalcSlotBaseFee(c.bc.Config(), creatorsPerSlotCount, uint64(len(validators)), c.bc.Genesis().GasLimit())

	return header, nil
}

func (c *Creator) reorgTips(slot uint64, tips types.Tips) (types.BlockMap, error) {
	genesis := c.bc.Genesis().Hash()

	tipsBlocks := c.bc.GetBlocksByHashes(tips.GetHashes())
	for _, block := range tipsBlocks {
		if block.Slot() >= slot {
			for _, hash := range block.ParentHashes() {
				dagBlock := c.bc.GetBlockDag(hash)
				if dagBlock == nil {
					parentBlock := c.bc.GetHeader(hash)
					cpHeader := c.bc.GetHeader(parentBlock.CpHash)
					if parentBlock == nil {
						log.Warn("Creator reorg tips failed: bad parent in dag", "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "parent", hash.Hex())
						continue
					}
					dagChainHashes := common.HashArray{}
					//if block not finalized
					var (
						isCpAncestor bool
						ancestors    types.HeaderMap
						err          error
						unl          common.HashArray
					)
					log.Warn("Creator reorg tips: active BlockDag not found", "parent", hash.Hex(), "parent.slot", parentBlock.Slot, "parent.height", parentBlock.Height, "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex())
					isCpAncestor, ancestors, unl, err = c.bc.CollectAncestorsAftCpByParents(block.ParentHashes(), block.CpHash())
					if err != nil {
						return nil, err
					}
					if len(unl) > 0 {
						log.Error("Creator reorg tips: should never happen",
							"err", core.ErrInsertUncompletedDag,
							"parent", hash.Hex(),
							"parent.slot", parentBlock.Slot,
							"parent.height", parentBlock.Height,
							"slot", block.Slot(),
							"height", block.Height(),
							"hash", block.Hash().Hex(),
						)
						return nil, core.ErrInsertUncompletedDag
					}
					if !isCpAncestor {
						log.Error("Creator reorg tips: should never happen",
							"err", core.ErrCpIsnotAncestor,
							"parent", hash.Hex(),
							"parent.slot", parentBlock.Slot,
							"parent.height", parentBlock.Height,
							"slot", block.Slot(),
							"height", block.Height(),
							"hash", block.Hash().Hex(),
						)
						return nil, core.ErrCpIsnotAncestor
					}
					delete(ancestors, cpHeader.Hash())
					dagChainHashes = ancestors.Hashes()
					dagBlock = &types.BlockDAG{
						Hash:           hash,
						Height:         parentBlock.Height,
						Slot:           parentBlock.Slot,
						CpHash:         parentBlock.CpHash,
						CpHeight:       cpHeader.Height,
						DagChainHashes: dagChainHashes,
					}
				}
				dagBlock.DagChainHashes = dagBlock.DagChainHashes.Difference(common.HashArray{genesis})
				tips.Add(dagBlock)
			}
			delete(tips, block.Hash())
			log.Info("Creator reorg tips", "blSlot", block.Slot(), "blHeight", block.Height(), "blHash", block.Hash().Hex(), "tips", tips.Print())
		}
	}
	tipsBlocks = c.bc.GetBlocksByHashes(tips.GetHashes())

	// check tips in ancestors other tips [a->b->c , c->...]
	for _, th := range tips.GetHashes() {
		block := tipsBlocks[th]
		if block == nil {
			continue
		}
		for _, ancestor := range tips.GetHashes() {
			if block.Hash() == ancestor {
				continue
			}
			isAncestor, err := c.bc.IsAncestorByTips(block.Header(), ancestor)
			if err != nil {
				return nil, err
			}
			if isAncestor {
				log.Warn("Creator remove ancestor tips",
					"block", block.Hash().Hex(),
					"ancestor", ancestor.Hex(),
					"tips", tips.Print(),
				)
				tips.Remove(ancestor)
				delete(tipsBlocks, ancestor)
				c.bc.RemoveTips(common.HashArray{ancestor})
			}
		}
	}

	return tipsBlocks, nil
}

func (c *Creator) createNewBlock(coinbase common.Address, creators []common.Address, header *types.Header, wg *sync.WaitGroup) {
	log.Info("Try to create new block", "slot", header.Slot, "coinbase", coinbase.Hex())
	defer wg.Done()

	if coinbase == (common.Address{}) {
		log.Error("Refusing to create without etherbase")
		return
	}
	header.Coinbase = coinbase

	// Fill the block with all available pending transactions.
	pendingTxs := c.getPending(coinbase, creators)

	syncData := validatorsync.GetPendingValidatorSyncData(c.bc)

	//syncData log
	for _, sd := range syncData {
		amt := new(big.Int)
		if sd.Amount != nil {
			amt.Set(sd.Amount)
		}
		log.Info("Creator: validator sync data",
			"OpType", sd.OpType,
			"ProcEpoch", sd.ProcEpoch,
			"Index", sd.Index,
			"Creator", fmt.Sprintf("%#x", sd.Creator),
			"amount", amt.String(),
			"TxHash", fmt.Sprintf("%#x", sd.TxHash),
		)
	}

	log.Info("Block creation: assigned txs", "len(pendingTxs)", len(pendingTxs), "len(syncData)", len(syncData))

	// Short circuit if no pending transactions
	if len(pendingTxs) == 0 && len(syncData) == 0 {
		pendAddr, queAddr, _ := c.eth.TxPool().StatsByAddrs()
		log.Warn("Skipping block creation: no assigned txs (short circuit)", "creator", coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
		return
	}

	txs := types.NewTransactionsByPriceAndNonce(c.current.signer, pendingTxs, header.BaseFee)
	if c.appendTransactions(txs, header) {
		if len(syncData) > 0 && c.isAddressAssigned(coinbase, *c.bc.Config().ValidatorsStateAddress, creators) {
			if err := c.processValidatorTxs(syncData, header); err != nil {
				log.Warn("Skipping block creation: processing validator txs err 0", "creator", coinbase, "err", err)
				return
			}

			c.create(header, true)

			return
		}
		pendAddr, queAddr, _ := c.eth.TxPool().StatsByAddrs()
		log.Warn("Skipping block creation: no assigned txs", "creator", coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
		return
	}

	if len(syncData) > 0 && c.isAddressAssigned(coinbase, *c.bc.Config().ValidatorsStateAddress, creators) {
		if err := c.processValidatorTxs(syncData, header); err != nil {
			log.Warn("Skipping block creation: processing validator txs err 1", "creator", coinbase, "err", err)
			return
		}
	}

	c.create(header, true)
}

func (c *Creator) create(header *types.Header, update bool) {
	block := types.NewStatelessBlock(
		header,
		c.getUnhandledTxs(header.Coinbase),
		trie.NewStackTrie(nil),
	)

	// Short circuit when receiving empty result.
	if block == nil {
		log.Error("Created block is nil")
		return
	}
	// Short circuit when receiving duplicate result caused by resubmitting.
	if c.bc.HasBlock(block.Hash()) {
		log.Error("Created block is already creating")
		return
	}

	// Commit block to database.
	_, err := c.bc.WriteCreatedDagBlock(block)
	if err != nil {
		log.Error("Failed write dag block", "err", err)
		return
	}
	// Broadcast the block and announce bc insertion event
	err = c.mux.Post(core.NewMinedBlockEvent{Block: block})
	if err != nil {
		log.Error("Failed broadcast the block and announce bc insertion event", "error", err)
		return
	}
	// Insert the block into the set of pending ones to resultLoop for confirmations
	log.Info("🔨 created dag block",
		"slot", block.Slot(),
		"epoch", c.bc.GetSlotInfo().SlotToEpoch(block.Slot()),
		"era", block.Era(),
		"height", block.Height(),
		"hash", block.Hash().Hex(),
		"creator", block.Coinbase().Hex(),
		"parents", block.ParentHashes(),
		"CpHash", block.CpHash().Hex(),
		"CpNumber", block.CpNumber(),
	)

	if update {
		c.updateSnapshot(header)
	}
	return
}

// isSyncing returns tru while sync pocess
func (c *Creator) isSyncing() bool {
	if badTips := c.bc.GetUnsynchronizedTipsHashes(); len(badTips) > 0 {
		return true
	}
	if c.eth.Downloader().Synchronising() {
		return true
	}
	return false
}

func (c *Creator) getUnhandledTxs(coinbase common.Address) types.Transactions {
	c.current.txsMu.Lock()
	defer c.current.txsMu.Unlock()
	return c.current.txs[coinbase].txs
}

// makeCurrent creates a new environment for the current cycle.
func (c *Creator) makeCurrent() error {
	// Retrieve the stable state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit

	env := &environment{
		signer: types.MakeSigner(c.bc.Config()),
		txs:    make(map[common.Address]*txsWithCumulativeGas),
		txsMu:  new(sync.Mutex),
	}

	c.current = env
	return nil
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (c *Creator) updateSnapshot(header *types.Header) {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()

	txs := c.getUnhandledTxs(header.Coinbase)

	c.snapshotBlock = types.NewBlock(
		header,
		txs,
		nil,
		trie.NewStackTrie(nil),
	)
}

func (c *Creator) appendTransaction(tx *types.Transaction, header *types.Header, isValidatorOp bool) error {
	if isValidatorOp {
		c.current.txs[header.Coinbase].txs = append(c.current.txs[header.Coinbase].txs, tx)
		return nil
	}

	gas, err := c.bc.TxEstimateGas(tx, header)
	if err != nil {
		log.Error("Failed to estimate gas for the transaction", "err", err)
		return err
	}

	expectedGas := c.current.txs[header.Coinbase].cumulativeGas + gas
	if expectedGas <= header.GasLimit {
		c.current.txs[header.Coinbase].txs = append(c.current.txs[header.Coinbase].txs, tx)
	}
	c.current.txs[header.Coinbase].cumulativeGas = expectedGas
	return nil
}

func (c *Creator) appendTransactions(txs *types.TransactionsByPriceAndNonce, header *types.Header) bool {
	c.current.txsMu.Lock()
	defer c.current.txsMu.Unlock()

	// Short circuit if current is nil
	if c.current == nil {
		log.Warn("Skipping block creation: no current environment", "env", c.current)
		return true
	}

	gasLimit := header.GasLimit
	if c.current.gasPool == nil {
		c.current.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	defer func(tStart time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(tStart)),
			"func:", "appendTransactions",
			"txs", len(c.current.txs[header.Coinbase].txs),
		)
	}(time.Now())

	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if c.current.gasPool.Gas() < params.TxGas || c.current.txs[header.Coinbase].cumulativeGas > header.GasLimit {
			log.Warn("Not enough gas for further transactions",
				"have", c.current.gasPool,
				"want", params.TxGas,
				"cumulativeGas", header.GasLimit,
				"GasLimit", gasLimit,
			)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			log.Info("Creator: adding txs to block end", "coinbase", header.Coinbase, "txs", len(c.current.txs[header.Coinbase].txs))
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(c.current.signer, tx)

		err := c.appendTransaction(tx, header, false)

		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Error("Gas limit exceeded for current block while create", "sender", from, "hash", tx.Hash().Hex())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, shift in the next transaction from the same account
			log.Debug("Tx added",
				"hash", tx.Hash().Hex(),
				"gasLimit", gasLimit,
				"cumulativeGas", c.current.txs[header.Coinbase].cumulativeGas,
			)
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Info("Tx failed, account skipped while create", "hash", tx.Hash().Hex(), "err", err)
			txs.Shift()
		}
	}

	if len(c.current.txs[header.Coinbase].txs) == 0 {
		log.Warn("Skipping block creation: no txs", "count", len(c.current.txs[header.Coinbase].txs))
		return true
	}

	if !c.IsRunning() && len(coalescedLogs) > 0 {
		// We don't push the pendingLogsEvent while we are creating. The reason is that
		// when we are creating, the Creator will regenerate a created block every 3 seconds.
		// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.

		// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
		// logs by filling in the block hash when the block was mined by the local miner. This can
		// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		c.pendingLogsFeed.Send(cpy)
	}
	return false
}

// isCreatorActive returns true if creator is assigned to create blocks in current slot.
func (c *Creator) isCreatorActive(coinbase common.Address, assigned *Assignment) bool {
	for _, creator := range assigned.Creators {
		if creator == coinbase {
			return true
		}
	}
	return false
}

// getPending returns all pending transactions for current miner
func (c *Creator) getPending(coinbase common.Address, creators []common.Address) map[common.Address]types.Transactions {
	pending := c.eth.TxPool().Pending(true)

	for address := range pending {
		for _, creator := range creators {
			if address == creator && creator != coinbase {
				delete(pending, address)
			}
		}
	}

	for fromAdr, txs := range pending {
		_txs := types.Transactions{}
		if c.isAddressAssigned(coinbase, fromAdr, creators) || fromAdr == coinbase {
			_txs = txs
		}
		if len(_txs) > 0 {
			pending[fromAdr] = _txs
		} else {
			delete(pending, fromAdr)
		}
	}
	return pending
}

// isAddressAssigned checks if miner is allowed to add transaction from that address
func (c *Creator) isAddressAssigned(coinbase common.Address, address common.Address, creators []common.Address) bool {
	var creatorNr = int64(-1)

	if len(creators) == 0 {
		return false
	}
	for i, creator := range creators {
		if creator == coinbase {
			creatorNr = int64(i)
			break
		}
	}

	if creatorNr < 0 {
		return false
	}

	return core.IsAddressAssigned(address, creators, creatorNr)
}

func (c *Creator) processValidatorTxs(syncData map[[28]byte]*types.ValidatorSync, header *types.Header) error {
	nonce := c.eth.TxPool().Nonce(header.Coinbase)
	for _, validatorSync := range syncData {
		if validatorSync.ProcEpoch <= c.bc.GetSlotInfo().SlotToEpoch(c.bc.GetSlotInfo().CurrentSlot()) {
			valSyncTx, err := validatorsync.CreateValidatorSyncTx(c.eth, header.CpHash, header.Coinbase, validatorSync, nonce)
			if err != nil {
				log.Error("failed to create validator sync tx", "error", err)
				continue
			}
			c.current.txsMu.Lock()
			err = c.appendTransaction(valSyncTx, header, true)
			if err != nil {
				log.Error("can`t create validator sync tx", "error", err)
				return err
			}
			c.current.txsMu.Unlock()
			nonce++
		}
	}

	return nil
}
