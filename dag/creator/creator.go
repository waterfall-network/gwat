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

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainSideChanSize is the size of channel listening to ChainSideEvent.
	chainSideChanSize = 10

	// miningLogAtDepth is the number of confirmations before logging successful block creation.
	miningLogAtDepth = 7
)

// Backend wraps all methods required for block creation.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
	Etherbase() (eb common.Address, err error)
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

	tcount        int           // tx count in cycle
	cumutativeGas uint64        // tx count in cycle
	gasPool       *core.GasPool // available gas used to pack transactions

	header *types.Header
	txs    []*types.Transaction
}

// task contains all information for consensus engine sealing and result submitting.
type task struct {
	tips      *types.Tips
	block     *types.Block
	createdAt time.Time
}

// newWorkReq represents a request for new sealing work submitting with relative interrupt notifier.
type newWorkReq struct {
	tips      types.Tips
	timestamp int64
}

// Creator is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type Creator struct {
	config      *Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Subscriptions
	mux    *event.TypeMux
	txsCh  chan core.NewTxsEvent
	txsSub event.Subscription

	chainSideCh  chan core.ChainSideEvent
	chainSideSub event.Subscription

	// Channels
	newWorkCh    chan *newWorkReq
	resultCh     chan *task
	finishWorkCh chan *types.Block
	errWorkCh    chan *error
	exitCh       chan struct{}

	current *environment // An environment for current running cycle.

	unconfirmed *unconfirmedBlocks // A set of locally mined blocks pending canonicalness confirmations.

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.
	newTxs  int32 // New arrival transaction count since last sealing work submitting.

	// Test hooks
	skipSealHook func(*task) bool                   // Method to decide whether skipping the sealing.
	fullTaskHook func()                             // Method to call before pushing the full sealing task.
	resubmitHook func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.

	cacheAssignment *Assignment

	canStart    bool
	shouldStart bool

	checkpoint *types.Checkpoint
}

// New creates new Creator instance
func New(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux) *Creator {
	creator := &Creator{
		config:      config,
		chainConfig: chainConfig,
		engine:      engine,
		eth:         eth,
		mux:         mux,
		chain:       eth.BlockChain(),

		unconfirmed:  newUnconfirmedBlocks(eth.BlockChain(), miningLogAtDepth),
		txsCh:        make(chan core.NewTxsEvent, txChanSize),
		chainSideCh:  make(chan core.ChainSideEvent, chainSideChanSize),
		newWorkCh:    make(chan *newWorkReq),
		resultCh:     make(chan *task),
		finishWorkCh: make(chan *types.Block),
		errWorkCh:    make(chan *error),
		exitCh:       make(chan struct{}),

		cacheAssignment: nil,
		canStart:        true,
		shouldStart:     false,
	}
	// Subscribe NewTxsEvent for tx pool
	creator.txsSub = eth.TxPool().SubscribeNewTxsEvent(creator.txsCh)

	// Subscribe events for blockchain
	creator.chainSideSub = eth.BlockChain().SubscribeChainSideEvent(creator.chainSideCh)

	go creator.mainLoop()
	go creator.resultLoop()

	return creator
}

// API

// Start prepare to create new blocks
// sets the running status as 1
func (c *Creator) Start(coinbase common.Address) {
	c.SetEtherbase(coinbase)
	if c.canStart {
		atomic.StoreInt32(&c.running, 1)
	}
	c.shouldStart = true
}

// Stop sets the running status as 0.
func (c *Creator) Stop() {
	atomic.StoreInt32(&c.running, 0)
	c.shouldStart = false
	c.cacheAssignment = nil
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

	block := c.chain.GetLastFinalizedBlock()
	state, err := c.chain.StateAt(block.Root())
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

// SetEtherbase sets the etherbase used to initialize the block coinbase field.
func (c *Creator) SetEtherbase(addr common.Address) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coinbase = addr
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

func (c *Creator) GetCheckpoint() *types.Checkpoint {
	return c.checkpoint
}

func (c *Creator) SaveCheckpoint(cp *types.Checkpoint) {
	c.checkpoint = cp
}

func (c *Creator) ResetCheckpoint() {
	c.checkpoint = nil
}

// isSyncing returns tru while sync pocess
func (c *Creator) isSyncing() bool {
	//check tips
	//depracated
	//if tips := c.chain.GetTips(); len(tips) == 0 {
	//	return true
	//}
	if badTips := c.chain.GetUnsynchronizedTipsHashes(); len(badTips) > 0 {
		return true
	}
	if c.eth.Downloader().Synchronising() {
		return true
	}
	return false
}

// isSlotLocked compare incoming epoch/slot with the latest epoch/slot of chain.
func (c *Creator) isSlotLocked(info *Assignment) bool {
	// check epoch/slot info of tips and lastFinalized block
	// if epoch/slot >= in chain
	// - rewind to correct position
	assig := c.getAssignment()
	if assig.Slot > info.Slot {
		return true
	}
	if assig.Slot == info.Slot {
		tips := c.chain.GetTips()
		blTips := c.chain.GetBlocksByHashes(tips.GetHashes())
		//  if current creator have created block in current slot
		for _, bl := range blTips {
			if bl.Slot() == assig.Slot && bl.Coinbase() == c.coinbase {
				return true
			}
		}
	}
	return false
}

// CreateBlock starts process of block creation
func (c *Creator) CreateBlock(assigned *Assignment, tips *types.Tips) (*types.Block, error) {
	if !c.IsRunning() {
		log.Warn("Creator stopped")
		return nil, ErrCreatorStopped
	}

	if c.isSlotLocked(assigned) {
		log.Warn("Creator skipping due to slot locked", "Slot", assigned.Slot, "lastSlot", c.getAssignment().Slot)
		return nil, ErrSlotLocked
	}

	if c.isSyncing() {
		log.Warn("Creator skipping due to synchronization")
		return nil, ErrSynchronization
	}

	if c.canStart {
		c.setAssignment(assigned)
		if !c.isCreatorActive(assigned) {
			log.Info("Creator skipping due to not active")
			return nil, ErrCreatorNotActive
		}
		c.canStart = false
		c.newWorkCh <- &newWorkReq{tips: tips.Copy(), timestamp: time.Now().Unix()}
		for {
			select {
			case block := <-c.finishWorkCh:
				c.canStart = true
				return block, nil
			case err := <-c.errWorkCh:
				c.canStart = true
				return nil, *err
			case <-c.exitCh:
				c.canStart = true
				return nil, ErrCreatorStopped
			}
		}
	}
	return nil, ErrCreatorStopped
}

// close terminates all background threads maintained by the Creator.
// Note the Creator does not support being closed multiple times.
func (c *Creator) close() {
	atomic.StoreInt32(&c.running, 0)
	close(c.exitCh)
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (c *Creator) mainLoop() {
	defer c.txsSub.Unsubscribe()
	defer c.chainSideSub.Unsubscribe()

	for {
		select {
		case req := <-c.newWorkCh:
			c.commitNewWork(req.tips, req.timestamp)
		case <-c.chainSideCh:
		case <-c.txsCh:
			// Apply transactions to the pending state if we're not creator.
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current creating block. These transactions will
			// be automatically eliminated.
			continue

		// System stopped
		case <-c.exitCh:
			return
		case err := <-c.txsSub.Err():
			c.errWorkCh <- &err
			return
		case err := <-c.chainSideSub.Err():
			c.errWorkCh <- &err
			return
		}
	}
}

// resultLoop is a standalone goroutine to handle sealing result submitting
// and flush relative data to the database.
func (c *Creator) resultLoop() {
	for {
		select {
		case task := <-c.resultCh:
			c.resultHandler(task)
			c.finishWorkCh <- task.block
		case <-c.exitCh:
			return
		}
	}
}

func (c *Creator) resultHandler(task *task) {
	// Short circuit when receiving empty result.
	if task.block == nil {
		return
	}
	// Short circuit when receiving duplicate result caused by resubmitting.
	if c.chain.HasBlock(task.block.Hash()) {
		return
	}
	var (
		hash = task.block.Hash()
	)
	// Commit block to database.
	_, err := c.chain.WriteCreatedDagBlock(task.block)
	if err != nil {
		log.Error("Failed writing block to chain (creator)", "err", err)
		return
	}
	// Broadcast the block and announce chain insertion event
	c.mux.Post(core.NewMinedBlockEvent{Block: task.block})

	// Insert the block into the set of pending ones to resultLoop for confirmations
	log.Info("🔨 created dag block",
		"slot", task.block.Slot(),
		"epoch", c.chain.GetSlotInfo().SlotToEpoch(task.block.Slot()),
		"era", task.block.Era(),
		"height", task.block.Height(),
		"hash", hash.Hex(),
		"parents", task.block.ParentHashes(),
		"CpHash", task.block.CpHash().Hex(),
		"CpNumber", task.block.CpNumber(),
	)

	//TODO depracated code memo
	//// Commit block to database.
	//_, err := c.chain.WriteMinedBlock(task.block)
	//if err != nil {
	//	log.Error("Failed writing block to chain (creator)", "err", err)
	//	return
	//}
	//
	////update state of tips
	//bc := c.chain
	////1. remove stale tips
	//bc.RemoveTips(task.block.ParentHashes())
	//
	////create new blockDag
	//cpHeader := bc.GetHeader(task.block.CpHash())
	//tips := task.tips.Copy()
	//dagChainHashes, err := bc.CollectDagChainHashesByTips(tips, cpHeader.Hash())
	//if err != nil {
	//	log.Error("Creator failed", "err", err)
	//	return
	//}
	//newBlockDag := &types.BlockDAG{
	//	Hash:           task.block.Hash(),
	//	Height:         task.block.Height(),
	//	Slot:           task.block.Slot(),
	//	CpHash:         task.block.CpHash(),
	//	CpHeight:       cpHeader.Height,
	//	DagChainHashes: dagChainHashes,
	//}
	//c.chain.AddTips(newBlockDag)
	//c.chain.WriteCurrentTips()
	//
	//log.Info("Creator: end tips",
	//	"tipsHashes", c.chain.GetTips().GetHashes(),
	//	//"tips", c.chain.GetTips().Print(),
	//)
	//
	//c.chain.MoveTxsToProcessing(types.Blocks{task.block})
	//
	//// Broadcast the block and announce chain insertion event
	//c.mux.Post(core.NewMinedBlockEvent{Block: task.block})
	//
	//// Insert the block into the set of pending ones to resultLoop for confirmations
	//log.Info("🔨 created dag block",
	//	"slot", task.block.Slot(),
	//	"epoch", c.chain.GetSlotInfo().SlotToEpoch(task.block.Slot()),
	//	"era", task.block.Era(),
	//	"height", task.block.Height(),
	//	"hash", hash.Hex(),
	//	"parents", task.block.ParentHashes(),
	//	"CpHash", task.block.CpHash().Hex(),
	//	"CpNumber", task.block.CpNumber(),
	//)

}

func (c *Creator) getUnhandledTxs() []*types.Transaction {
	return c.current.txs
}

// makeCurrent creates a new environment for the current cycle.
func (c *Creator) makeCurrent(header *types.Header) error {
	// Retrieve the stable state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit

	env := &environment{
		signer: types.MakeSigner(c.chainConfig),
		header: header,
		txs:    []*types.Transaction{},
	}

	// Keep track of transactions which return errors so they can be removed
	env.tcount = 0

	c.current = env
	return nil
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (c *Creator) updateSnapshot() {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()

	txs := c.getUnhandledTxs()
	//receipts := c.getUnhandledReceipts()

	c.snapshotBlock = types.NewBlock(
		c.current.header,
		txs,
		nil,
		trie.NewStackTrie(nil),
	)
}

func (c *Creator) appendTransaction(tx *types.Transaction, lfNumber *uint64, isValidatorOp bool) error {
	if isValidatorOp {
		c.current.txs = append(c.current.txs, tx)
		return nil
	}

	gas, err := c.chain.TxEstimateGas(tx, lfNumber)
	if err != nil {
		log.Error("Failed to estimate gas for the transaction", "err", err)
		return err
	}

	expectedGas := c.current.cumutativeGas + gas
	if expectedGas <= c.current.header.GasLimit {
		c.current.txs = append(c.current.txs, tx)
	}
	c.current.cumutativeGas = expectedGas
	return nil
}

func (c *Creator) appendTransactions(txs *types.TransactionsByPriceAndNonce, coinbase common.Address, lfNumber *uint64) bool {
	// Short circuit if current is nil
	if c.current == nil {
		log.Warn("Skipping block creation: no current environment", "env", c.current)
		return true
	}

	gasLimit := c.current.header.GasLimit
	if c.current.gasPool == nil {
		c.current.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	defer func(tStart time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(tStart)),
			"func:", "appendTransactions",
			"txs", c.current.tcount,
		)
	}(time.Now())

	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if c.current.gasPool.Gas() < params.TxGas || c.current.cumutativeGas > c.current.header.GasLimit {
			log.Warn("Not enough gas for further transactions",
				"have", c.current.gasPool,
				"want", params.TxGas,
				"cumutativeGas", c.current.cumutativeGas,
				"GasLimit", c.current.header.GasLimit,
			)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			log.Info("Creator: adding txs to block end", "txs", c.current.tcount)
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(c.current.signer, tx)

		err := c.appendTransaction(tx, lfNumber, false)

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
				"cumutativeGas", c.current.cumutativeGas,
			)
			c.current.tcount++
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Info("Tx failed, account skipped while create", "hash", tx.Hash().Hex(), "err", err)
			txs.Shift()
		}
	}

	if c.current.tcount == 0 {
		log.Warn("Skipping block creation: no txs", "count", c.current.tcount)
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

// commitNewWork generates several new sealing tasks based on the parent block.
func (c *Creator) commitNewWork(tips types.Tips, timestamp int64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bc := c.chain
	genesis := bc.Genesis().Hash()
	tstart := time.Now()

	slotInfo := c.getAssignment()
	tipsBlocks := bc.GetBlocksByHashes(tips.GetHashes())
	blocks := bc.GetBlocksByHashes(tipsBlocks.Hashes())
	for _, bl := range blocks {
		if bl.Slot() >= slotInfo.Slot {
			for _, ph := range bl.ParentHashes() {
				_dag := bc.GetBlockDag(ph)
				if _dag == nil {
					parentBlock := bc.GetHeader(ph)
					cpHeader := bc.GetHeader(parentBlock.CpHash)
					if parentBlock == nil {
						log.Warn("Creator reorg tips failed: bad parent in dag", "slot", bl.Slot(), "height", bl.Height(), "hash", bl.Hash().Hex(), "parent", ph.Hex())
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
					log.Warn("Creator reorg tips: active BlockDag not found", "parent", ph.Hex(), "parent.slot", parentBlock.Slot, "parent.height", parentBlock.Height, "slot", bl.Slot(), "height", bl.Height(), "hash", bl.Hash().Hex())
					isCpAncestor, ancestors, unl, err = bc.CollectAncestorsAftCpByParents(bl.ParentHashes(), bl.CpHash())
					if err != nil {
						c.errWorkCh <- &err
						return
					}
					if len(unl) > 0 {
						log.Error("Creator reorg tips: should never happen",
							"err", core.ErrInsertUncompletedDag,
							"parent", ph.Hex(),
							"parent.slot", parentBlock.Slot,
							"parent.height", parentBlock.Height,
							"slot", bl.Slot(),
							"height", bl.Height(),
							"hash", bl.Hash().Hex(),
						)
						c.errWorkCh <- &core.ErrInsertUncompletedDag
						return
					}
					if !isCpAncestor {
						log.Error("Creator reorg tips: should never happen",
							"err", core.ErrCpIsnotAncestor,
							"parent", ph.Hex(),
							"parent.slot", parentBlock.Slot,
							"parent.height", parentBlock.Height,
							"slot", bl.Slot(),
							"height", bl.Height(),
							"hash", bl.Hash().Hex(),
						)
						c.errWorkCh <- &core.ErrCpIsnotAncestor
						return
					}
					delete(ancestors, cpHeader.Hash())
					dagChainHashes = ancestors.Hashes()
					_dag = &types.BlockDAG{
						Hash:           ph,
						Height:         parentBlock.Height,
						Slot:           parentBlock.Slot,
						CpHash:         parentBlock.CpHash,
						CpHeight:       cpHeader.Height,
						DagChainHashes: dagChainHashes,
					}
				}
				_dag.DagChainHashes = _dag.DagChainHashes.Difference(common.HashArray{genesis})
				tips.Add(_dag)
			}
			delete(tips, bl.Hash())
			log.Info("Creator reorg tips", "blSlot", bl.Slot(), "blHeight", bl.Height(), "blHash", bl.Hash().Hex(), "tips", tips.Print())
		}
	}
	tipsBlocks = bc.GetBlocksByHashes(tips.GetHashes())

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
			//isAncestor, err := bc.IsAncestorRecursive(block.Header(), ancestor)
			isAncestor, err := bc.IsAncestorByTips(block.Header(), ancestor)
			if err != nil {
				c.errWorkCh <- &err
				return
			}
			if isAncestor {
				log.Warn("Creator remove ancestor tips",
					"block", block.Hash().Hex(),
					"ancestor", ancestor.Hex(),
					"tips", tips.Print(),
				)
				tips.Remove(ancestor)
				delete(tipsBlocks, ancestor)
				bc.RemoveTips(common.HashArray{ancestor})
			}
		}
	}

	// if max slot of parents is less or equal to last finalized block slot
	// - add last finalized block to parents
	lastFinBlock := bc.GetLastFinalizedBlock()
	maxParentSlot := uint64(0)
	for _, blk := range tipsBlocks {
		if blk.Slot() > maxParentSlot {
			maxParentSlot = blk.Slot()
		}
	}
	if maxParentSlot <= lastFinBlock.Slot() {
		tipsBlocks[lastFinBlock.Hash()] = lastFinBlock
	}

	log.Info("Creator: start tips",
		"tipsHashes", tipsBlocks.Hashes(),
		//"tips", tips.Print(),
	)

	parentHashes := tipsBlocks.Hashes().Sort()

	// Use checkpoint spine as CpBlock
	checkpoint := c.GetCheckpoint()
	cpHeader := bc.GetHeader(checkpoint.Spine)
	//newHeight, err := bc.CalcBlockHeightByParents(parentHashes, cpHeader.Hash())
	newHeight, err := bc.CalcBlockHeightByTips(tips, cpHeader.Hash())
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		c.errWorkCh <- &err
		return
	}

	si := bc.GetSlotInfo()

	log.Info("Creator calculate block height", "newHeight", newHeight)
	log.Info("########## CREATOR slot epoch era",
		"blHeight", newHeight,
		"blEpoch", si.SlotToEpoch(slotInfo.Slot),
		"blSlot", slotInfo.Slot,
		"currSlot", si.CurrentSlot(),
		"currEpoch", si.SlotToEpoch(si.CurrentSlot()),
		"eraNum", bc.GetEraInfo().Number(),
		"from", bc.GetEraInfo().FromEpoch(),
		"to", bc.GetEraInfo().ToEpoch(),
	)

	era := bc.GetEraInfo().Number()
	if si.SlotToEpoch(si.CurrentSlot()) >= bc.GetEraInfo().NextEraFirstEpoch() {
		era++
	}
	header := &types.Header{
		ParentHashes: parentHashes,
		Slot:         slotInfo.Slot,
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
	creatorsPerSlotCount := c.chainConfig.ValidatorsPerSlot
	if creatorsPerSlot, err := bc.ValidatorStorage().GetCreatorsBySlot(bc, header.Slot); err == nil {
		creatorsPerSlotCount = uint64(len(creatorsPerSlot))
	}
	validators, _ := bc.ValidatorStorage().GetValidators(bc, header.Slot, true, false, "commitNewWork")
	header.BaseFee = misc.CalcSlotBaseFee(c.chainConfig, header, uint64(len(validators)), bc.Genesis().GasLimit(), params.BurnMultiplier, creatorsPerSlotCount)

	// Only set the coinbase if our consensus engine is running (avoid spurious block rewards)
	if c.IsRunning() {
		if c.coinbase == (common.Address{}) {
			log.Error("Refusing to create without etherbase")
			err := errors.New("refusing to create without etherbase")
			c.errWorkCh <- &err
			return
		}
		header.Coinbase = c.coinbase
	}

	//todo fix c.engine.Prepare
	if err = c.engine.Prepare(bc, header); err != nil {
		log.Error("Failed to prepare header for creating block", "err", err)
		c.errWorkCh <- &err
		return
	}

	// Could potentially happen if starting to mine in an odd state.
	err = c.makeCurrent(header)
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		c.errWorkCh <- &err
		return
	}

	// Fill the block with all available pending transactions.
	pendingTxs := c.getPending()

	syncData := validatorsync.GetPendingValidatorSyncData(bc)

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
		log.Warn("Skipping block creation: no assigned txs (short circuit)", "creator", c.coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
		c.errWorkCh <- &ErrNoTxs

		return
	}

	txs := types.NewTransactionsByPriceAndNonce(c.current.signer, pendingTxs, header.BaseFee)
	if c.appendTransactions(txs, c.coinbase, &header.CpNumber) {
		if len(syncData) > 0 && c.isAddressAssigned(*c.chainConfig.ValidatorsStateAddress) {
			if err := c.processValidatorTxs(header.CpHash, syncData, header.CpNumber); err != nil {
				log.Warn("Skipping block creation: processing validator txs err 0", "creator", c.coinbase, "err", err)
				return
			}

			c.commit(tips, c.fullTaskHook, true, tstart)

			return
		}
		pendAddr, queAddr, _ := c.eth.TxPool().StatsByAddrs()
		log.Warn("Skipping block creation: no assigned txs", "creator", c.coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
		c.errWorkCh <- &ErrNoTxs

		return
	}

	if len(syncData) > 0 && c.isAddressAssigned(*c.chainConfig.ValidatorsStateAddress) {
		if err := c.processValidatorTxs(header.CpHash, syncData, header.CpNumber); err != nil {
			log.Warn("Skipping block creation: processing validator txs err 1", "creator", c.coinbase, "err", err)
			return
		}
	}

	c.commit(tips, c.fullTaskHook, true, tstart)
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (c *Creator) commit(tips types.Tips, interval func(), update bool, start time.Time) error {

	block := types.NewStatelessBlock(
		c.current.header,
		c.getUnhandledTxs(),
		trie.NewStackTrie(nil),
	)

	task := &task{
		block:     block,
		tips:      &tips,
		createdAt: time.Now(),
	}

	if c.IsRunning() {
		if interval != nil {
			interval()
		}
		select {

		case c.resultCh <- task:
			log.Info("Commit new block creation work",
				"txs", c.current.tcount,
				"gas", block.GasUsed(), "fees", c.current.cumutativeGas,
				"tips", tips.GetHashes(),
				"elapsed", common.PrettyDuration(time.Since(start)),
			)
			log.Info("^^^^^^^^^^^^ TIME",
				"elapsed", common.PrettyDuration(time.Since(start)),
				"func:", "CreateBlock",
			)

		case <-c.exitCh:
			log.Info("Worker has exited")
		}
	}
	if update {
		c.updateSnapshot()
	}
	return nil
}

// isCreatorActive returns true if creator is assigned to create blocks in current slot.
func (c *Creator) isCreatorActive(assigned *Assignment) bool {
	if assigned == nil {
		return false
	}
	var (
		currMiner = c.coinbase
		creators  = assigned.Creators
	)
	for _, m := range creators {
		if m == currMiner {
			return true
		}
	}
	return false
}

// getPending returns all pending transactions for current miner
func (c *Creator) getPending() map[common.Address]types.Transactions {
	pending := c.eth.TxPool().Pending(true)

	// to correct handling validators' sync transactions
	// each creators handle own transactions while assigned slot
	// here removing all txs of other creators
	currCreators := c.getAssignment().Creators
	for address := range pending {
		for _, creator := range currCreators {
			if address == creator && creator != c.coinbase {
				delete(pending, address)
			}
		}
	}

	for fromAdr, txs := range pending {
		_txs := types.Transactions{}
		if c.isAddressAssigned(fromAdr) || fromAdr == c.coinbase {
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
func (c *Creator) isAddressAssigned(address common.Address) bool {
	var (
		currMiner    = c.coinbase
		creators     = c.getAssignment().Creators
		creatorCount = len(creators)
		creatorNr    = int64(-1)
	)
	if creatorCount == 0 {
		return false
	}
	for i, m := range creators {
		if m == currMiner {
			creatorNr = int64(i)
			break
		}
	}
	return core.IsAddressAssigned(address, creators, creatorNr)
}

// getAssignment returns list of creators and slot
func (c *Creator) getAssignment() Assignment {
	if c.cacheAssignment != nil {
		return *c.cacheAssignment
	}
	var (
		lfb     = c.eth.BlockChain().GetLastFinalizedBlock()
		maxSlot = lfb.Slot()
	)
	tips := c.eth.BlockChain().GetTips()
	if len(tips) > 0 {
		tipsBlocks := c.eth.BlockChain().GetBlocksByHashes(tips.GetHashes())
		for _, bl := range tipsBlocks {
			if bl.Coinbase() != c.coinbase {
				continue
			}
			if maxSlot > bl.Slot() {
				maxSlot = bl.Slot()
				continue
			}
		}
	}
	c.setAssignment(&Assignment{
		Slot:     maxSlot,
		Creators: nil,
	})
	return *c.cacheAssignment
}

// setAssignment
func (c *Creator) setAssignment(assigned *Assignment) {
	c.cacheAssignment = assigned
}

func (c *Creator) processValidatorTxs(blockHash common.Hash, syncData map[[28]byte]*types.ValidatorSync, lfNumber uint64) error {
	nonce := c.eth.TxPool().Nonce(c.coinbase)
	for _, validatorSync := range syncData {
		if validatorSync.ProcEpoch <= c.chain.GetSlotInfo().SlotToEpoch(c.chain.GetSlotInfo().CurrentSlot()) {
			valSyncTx, err := validatorsync.CreateValidatorSyncTx(c.eth, blockHash, c.coinbase, validatorSync, nonce)
			if err != nil {
				log.Error("failed to create validator sync tx", "error", err)
				continue
			}

			err = c.appendTransaction(valSyncTx, &lfNumber, true)
			if err != nil {
				log.Error("can`t commit validator sync tx", "error", err)
				return err
			}
			nonce++
		}
	}

	return nil
}
