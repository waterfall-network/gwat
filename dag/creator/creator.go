package creator

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/common/hexutil"
	"github.com/waterfall-foundation/gwat/consensus"
	"github.com/waterfall-foundation/gwat/consensus/misc"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/eth/downloader"
	"github.com/waterfall-foundation/gwat/event"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/params"
	"github.com/waterfall-foundation/gwat/trie"
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

	state *state.StateDB // apply state changes here

	tcount  int           // tx count in cycle
	gasPool *core.GasPool // available gas used to pack transactions

	header         *types.Header
	txs            []*types.Transaction
	receipts       []*types.Receipt
	recommitBlocks []*types.Block
}

// task contains all information for consensus engine sealing and result submitting.
type task struct {
	receipts  []*types.Receipt
	tips      *types.Tips
	state     *state.StateDB
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
	taskCh       chan *task
	resultCh     chan *types.Block
	finishWorkCh chan *types.Block
	errWorkCh    chan *error
	exitCh       chan struct{}

	current *environment // An environment for current running cycle.

	unconfirmed *unconfirmedBlocks // A set of locally mined blocks pending canonicalness confirmations.

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	pendingMu    sync.RWMutex
	pendingTasks map[common.Hash]*task

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.
	newTxs  int32 // New arrival transaction count since last sealing work submitting.

	// Test hooks
	newTaskHook  func(*task)                        // Method to call upon receiving a new sealing task.
	skipSealHook func(*task) bool                   // Method to decide whether skipping the sealing.
	fullTaskHook func()                             // Method to call before pushing the full sealing task.
	resubmitHook func(time.Duration, time.Duration) // Method to call upon updating resubmitting interval.

	cacheAssignment *Assignment

	canStart    bool
	shouldStart bool
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
		pendingTasks: make(map[common.Hash]*task),
		txsCh:        make(chan core.NewTxsEvent, txChanSize),
		chainSideCh:  make(chan core.ChainSideEvent, chainSideChanSize),
		newWorkCh:    make(chan *newWorkReq),
		taskCh:       make(chan *task),
		resultCh:     make(chan *types.Block),
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
	go creator.taskLoop()

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
	if c.snapshotState == nil {
		return nil, nil
	}
	return c.snapshotBlock, c.snapshotState.Copy()
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

// isSyncing returns tru while sync pocess
func (c *Creator) isSyncing() bool {
	//check tips
	if tips := c.chain.GetTips(); len(tips) == 0 {
		return true
	}
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
	if c.current != nil && c.current.state != nil {
		c.current.state.StopPrefetcher()
	}
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

// taskLoop is a standalone goroutine to fetch sealing task from the generator and
// push them to consensus engine.
func (c *Creator) taskLoop() {
	var (
		stopCh chan struct{}
		prev   common.Hash
	)

	// interrupt aborts the in-flight sealing task.
	interrupt := func() {
		if stopCh != nil {
			close(stopCh)
			stopCh = nil
		}
	}
	for {
		select {
		case task := <-c.taskCh:
			log.Info("Creator start", "c.newTaskHook", c.newTaskHook != nil, "c.skipSealHook", c.skipSealHook != nil)
			if c.newTaskHook != nil {
				c.newTaskHook(task)
			}
			// Reject duplicate sealing work due to resubmitting.
			sealHash := c.engine.SealHash(task.block.Header())

			if sealHash == prev {
				continue
			}

			// Interrupt previous sealing operation
			interrupt()
			stopCh, prev = make(chan struct{}), sealHash

			if c.skipSealHook != nil && c.skipSealHook(task) {
				continue
			}

			c.pendingMu.Lock()
			c.pendingTasks[sealHash] = task
			c.pendingMu.Unlock()
			if err := c.engine.Seal(c.chain, task.block, c.resultCh, stopCh); err != nil {
				log.Warn("Block sealing failed", "err", err)
				c.errWorkCh <- &err
			}

		case <-c.exitCh:
			interrupt()
			return
		}
	}
}

// resultLoop is a standalone goroutine to handle sealing result submitting
// and flush relative data to the database.
func (c *Creator) resultLoop() {
	for {
		select {
		case block := <-c.resultCh:
			c.resultHandler(block)
			c.finishWorkCh <- block
		case <-c.exitCh:
			return
		}
	}
}

func (c *Creator) resultHandler(block *types.Block) {
	// Short circuit when receiving empty result.
	if block == nil {
		return
	}
	// Short circuit when receiving duplicate result caused by resubmitting.
	if c.chain.HasBlock(block.Hash()) {
		return
	}
	var (
		sealhash = c.engine.SealHash(block.Header())
		hash     = block.Hash()
	)
	c.pendingMu.RLock()
	task, exist := c.pendingTasks[sealhash]
	c.pendingMu.RUnlock()
	if !exist {
		log.Error("Block found but no relative pending task", "Hash", block.Hash(), "sealhash", sealhash, "hash", hash)
		return
	}
	// Different block could share same sealhash, deep copy here to prevent write-write conflict.
	var (
		receipts = make([]*types.Receipt, len(task.receipts))
		logs     []*types.Log
	)
	for i, receipt := range task.receipts {
		// add block location fields
		receipt.BlockHash = hash
		receipt.TransactionIndex = uint(i)

		receipts[i] = new(types.Receipt)
		*receipts[i] = *receipt
		// Update the block hash in all logs since it is now available and not when the
		// receipt/log of individual transactions were created.
		for _, log := range receipt.Logs {
			log.BlockHash = hash
		}
		logs = append(logs, receipt.Logs...)
	}

	// Commit block and state to database.
	_, err := c.chain.WriteMinedBlock(block, receipts, logs, task.state)
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return
	}
	//update state of tips
	//1. remove stale tips
	c.chain.RemoveTips(block.ParentHashes())
	//2. create for new blockDag
	finDag := task.tips.GetFinalizingDag()
	tmpFinalityPoints := finDag.FinalityPoints
	tmpDagChainHashes := task.tips.GetOrderedDagChainHashes()
	if finDag.Hash != c.chain.Genesis().Hash() {
		tmpFinalityPoints = append(tmpFinalityPoints, finDag.Hash)
	} else {
		tmpDagChainHashes = tmpFinalityPoints.Difference(common.HashArray{c.chain.Genesis().Hash()})
	}

	newBlockDag := &types.BlockDAG{
		Hash:                block.Hash(),
		Height:              block.Height(),
		LastFinalizedHash:   c.chain.GetLastFinalizedBlock().Hash(),
		LastFinalizedHeight: c.chain.GetLastFinalizedNumber(),
		DagChainHashes:      tmpDagChainHashes,
		FinalityPoints:      tmpFinalityPoints,
	}
	c.chain.AddTips(newBlockDag)
	c.chain.ReviseTips()

	c.chain.MoveTxsToProcessing(types.Blocks{block})

	log.Info("Successfully sealed new block", "height", block.Height(), "hash", block.Hash().Hex(), "sealhash", sealhash, "elapsed", common.PrettyDuration(time.Since(task.createdAt)))

	// Broadcast the block and announce chain insertion event
	c.mux.Post(core.NewMinedBlockEvent{Block: block})

	// Insert the block into the set of pending ones to resultLoop for confirmations
	log.Info("🔨 created dag block", "height", block.Height(), "hash", hash.Hex(), "parents", block.ParentHashes())
}

func (c *Creator) getUnhandledTxs() []*types.Transaction {
	return c.current.txs
}
func (c *Creator) getUnhandledReceipts() []*types.Receipt {
	return c.current.receipts
}

// makeCurrent creates a new environment for the current cycle.
func (c *Creator) makeCurrent(tips types.Tips, header *types.Header) error {
	// Retrieve the stable state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit

	state, _, recommitBlocks, _, stateErr := c.chain.CollectStateDataByParents(tips.GetHashes())
	if stateErr != nil {
		return stateErr
	}

	state.StartPrefetcher("miner")

	env := &environment{
		signer:         types.MakeSigner(c.chainConfig),
		state:          state,
		header:         header,
		recommitBlocks: recommitBlocks,
		txs:            []*types.Transaction{},
		receipts:       []*types.Receipt{},
	}

	// Keep track of transactions which return errors so they can be removed
	env.tcount = 0

	// Swap out the old work with the new one, terminating any leftover prefetcher
	// processes in the mean time and starting a new one.
	if c.current != nil && c.current.state != nil {
		c.current.state.StopPrefetcher()
	}
	c.current = env
	return nil
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (c *Creator) updateSnapshot() {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()

	txs := c.getUnhandledTxs()
	receipts := c.getUnhandledReceipts()

	c.snapshotBlock = types.NewBlock(
		c.current.header,
		txs,
		receipts,
		trie.NewStackTrie(nil),
	)
	c.snapshotReceipts = copyReceipts(receipts)
	c.snapshotState = c.current.state.Copy()
}

func (c *Creator) commitTransaction(tx *types.Transaction, coinbase common.Address) ([]*types.Log, error) {
	snap := c.current.state.Snapshot()

	receipt, err := core.ApplyTransaction(c.chainConfig, c.chain, &coinbase, c.current.gasPool, c.current.state, c.current.header, tx, &c.current.header.GasUsed, *c.chain.GetVMConfig())
	if err != nil {
		c.current.state.RevertToSnapshot(snap)
		return nil, err
	}
	c.current.txs = append(c.current.txs, tx)
	c.current.receipts = append(c.current.receipts, receipt)

	return receipt.Logs, nil
}

func (c *Creator) commitTransactions(txs *types.TransactionsByPriceAndNonce, coinbase common.Address) bool {
	// Short circuit if current is nil
	if c.current == nil {
		return true
	}

	gasLimit := c.current.header.GasLimit
	if c.current.gasPool == nil {
		c.current.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if c.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", c.current.gasPool, "want", params.TxGas)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}
		// Error may be ignored here. The error has already been checked
		// during transaction acceptance is the transaction pool.
		//
		// We use the eip155 signer regardless of the current hf.
		from, _ := types.Sender(c.current.signer, tx)

		// Start executing the transaction
		c.current.state.Prepare(tx.Hash(), c.current.tcount)

		logs, err := c.commitTransaction(tx, coinbase)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Error("Gas limit exceeded for current block while create", "sender", from, "hash", tx.Hash().Hex())
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping tx with low nonce while create", "sender", from, "nonce", tx.Nonce(), "hash", tx.Hash().Hex())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Error("Skipping account with hight nonce while create", "sender", from, "nonce", tx.Nonce(), "hash", tx.Hash().Hex())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			c.current.tcount++
			txs.Shift()

		case errors.Is(err, core.ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported tx type while create", "sender", from, "type", tx.Type(), "hash", tx.Hash().Hex())
			txs.Pop()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Info("Tx failed, account skipped while create", "hash", tx.Hash().Hex(), "err", err)
			txs.Shift()
		}
	}

	if c.current.tcount == 0 {
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

	genesis := c.eth.BlockChain().Genesis().Hash()
	tstart := time.Now()

	slotInfo := c.getAssignment()
	tipsBlocks := c.chain.GetBlocksByHashes(tips.GetHashes())
	blocks := c.eth.BlockChain().GetBlocksByHashes(tipsBlocks.Hashes())
	for _, bl := range blocks {
		if bl.Slot() >= slotInfo.Slot {
			tip := tips.Get(bl.Hash())
			for _, ph := range bl.ParentHashes() {
				_, _, _, graph, _, _ := c.eth.BlockChain().ExploreChainRecursive(bl.Hash())
				_dag := c.eth.BlockChain().ReadBockDag(ph)
				if _dag == nil {
					parentBlock := c.eth.BlockChain().GetBlock(ph)
					if parentBlock == nil {
						log.Warn("Creator reorg tips failed: bad parent in dag", "height", bl.Height(), "hash", bl.Hash().Hex(), "parent", ph.Hex())
						continue
					}
					_dag = &types.BlockDAG{
						Hash:   ph,
						Height: parentBlock.Height(),
					}
				}
				_dag.LastFinalizedHash = tip.LastFinalizedHash
				_dag.LastFinalizedHeight = tip.LastFinalizedHeight
				if dch := graph.GetDagChainHashes(); dch != nil {
					_dag.DagChainHashes = *dch
				}
				if fp := graph.GetFinalityPoints(); fp != nil {
					_dag.FinalityPoints = *fp
				}
				_dag.DagChainHashes = _dag.DagChainHashes.Difference(common.HashArray{genesis})
				_dag.FinalityPoints = _dag.FinalityPoints.Difference(common.HashArray{genesis})
				tips.Add(_dag)
			}
			delete(tips, bl.Hash())
			log.Info("Creator reorg tips", "blHeight", bl.Height(), "blHash", bl.Hash().Hex(), "tips", tips.Print())
		}
	}
	tipsBlocks = c.chain.GetBlocksByHashes(tips.GetHashes())

	// check tips in ancestors other tips [a->b->c , c->...]
	for _, th := range tips.GetHashes() {
		block := tipsBlocks[th]
		for _, ancestor := range tips.GetHashes() {
			if block.Hash() == ancestor {
				continue
			}
			if c.chain.IsAncestorRecursive(block, ancestor) {
				log.Warn("Creator remove ancestor tips",
					"block", block.Hash().Hex(),
					"ancestor", ancestor.Hex(),
					"tips", tips.Print(),
				)
				delete(tips, ancestor)
			}
		}
	}

	if maxTipsTs := tipsBlocks.GetMaxTime(); maxTipsTs >= uint64(timestamp) {
		timestamp = int64(maxTipsTs + 1)
	}

	finDag := tips.GetFinalizingDag()
	if finDag == nil {
		log.Error("Tips empty, skipping block creation", "Initial", c.chain.GetTips().Print(), "uncompleted", c.chain.GetUnsynchronizedTipsHashes())
		err := errors.New("tips empty, skipping block creation")
		c.errWorkCh <- &err
		return
	}
	tmpFinalityPoints := finDag.FinalityPoints.Uniq()
	tmpDagChainHashes := tips.GetOrderedDagChainHashes()

	// after reorg tips can content hashes of finalized blocks
	finHashes := common.HashArray{}
	for i, h := range tmpDagChainHashes.Copy() {
		block := c.eth.BlockChain().GetBlock(h)
		finNr := block.Number()
		if finNr != nil {
			finHashes = append(finHashes, h)
			log.Info("Calc new block Height", "i", i, "Nr", block.Nr(), "Height", block.Height(), "Hash", block.Hash().Hex())
		}
	}
	if len(finHashes) > 0 {
		tmpFinalityPoints = tmpFinalityPoints.Difference(finHashes)
		tmpDagChainHashes = tmpDagChainHashes.Difference(finHashes)
	}

	if finDag.Hash != c.chain.Genesis().Hash() {
		tmpFinalityPoints = append(tmpFinalityPoints, finDag.Hash)
	} else {
		tmpDagChainHashes = tmpDagChainHashes.Difference(common.HashArray{c.chain.Genesis().Hash()})
	}

	lastBlueHash := finDag.Hash
	lastBlueBlock := c.chain.GetBlockByHash(lastBlueHash)
	lastBlueHeight := lastBlueBlock.Height()
	lastBlueIx := tmpDagChainHashes.IndexOf(lastBlueHash)
	redCount := len(tmpDagChainHashes) - lastBlueIx
	newHeight := lastBlueHeight + uint64(redCount)

	log.Info("Creator calculate block height", "newHeight", newHeight,
		"lastBlueHeight", lastBlueHeight,
		"lastBlueIx", lastBlueIx,
		"len(tmpDagChainHashes)", len(tmpDagChainHashes),
		"redCount", redCount,
		"lastBlueHash", lastBlueHash.Hex(),
		"tmpDagChainHashes", tmpDagChainHashes,
	)

	log.Info("Creator data", "tips", tips.Print())

	header := &types.Header{
		ParentHashes: tipsBlocks.Hashes(),
		Slot:         slotInfo.Slot,
		Height:       newHeight,
		GasLimit:     core.CalcGasLimit(tipsBlocks.AvgGasLimit(), c.config.GasCeil),
		Extra:        c.extra,
		Time:         uint64(timestamp),
	}

	// Set baseFee and GasLimit
	lastFinBlock := c.chain.GetBlockByHash(finDag.LastFinalizedHash)
	header.BaseFee = misc.CalcBaseFee(c.chainConfig, lastFinBlock.Header())

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
	if err := c.engine.Prepare(c.chain, header); err != nil {
		log.Error("Failed to prepare header for creating block", "err", err)
		c.errWorkCh <- &err
		return
	}

	// Could potentially happen if starting to mine in an odd state.
	err := c.makeCurrent(tips, header)
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		c.errWorkCh <- &err
		return
	}

	//recommit transactions of dag chain
	cacheChain := common.HashArray{}
	for _, bl := range c.current.recommitBlocks {
		statedb := c.chain.RecommitBlockTransactions(bl, c.current.state)
		//cache state
		cacheChain = append(cacheChain, bl.Hash())
		c.chain.SetCashedRecommit(cacheChain, statedb, nil)
	}

	// Fill the block with all available pending transactions.
	pending := c.getPending()

	// Short circuit if no pending transactions
	if len(pending) == 0 {
		log.Warn("Skipping block creation: no assigned txs")
		c.errWorkCh <- &ErrNoTxs
		return
	}

	if len(pending) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(c.current.signer, pending, header.BaseFee)
		if c.commitTransactions(txs, c.coinbase) {
			log.Warn("Skipping block creation: no assigned txs")
			c.errWorkCh <- &ErrNoTxs
			return
		}
	}
	c.commit(tips, c.fullTaskHook, true, tstart)
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (c *Creator) commit(tips types.Tips, interval func(), update bool, start time.Time) error {
	// Deep copy receipts here to avoid interaction between different tasks.
	receipts := copyReceipts(c.getUnhandledReceipts())
	s := c.current.state.Copy()
	block, err := c.engine.FinalizeAndAssemble(c.chain, c.current.header, s, c.getUnhandledTxs(), receipts)
	if err != nil {
		c.errWorkCh <- &err
		return err
	}
	if c.IsRunning() {
		if interval != nil {
			interval()
		}
		select {

		case c.taskCh <- &task{
			receipts:  receipts,
			state:     s,
			block:     block,
			tips:      &tips,
			createdAt: time.Now(),
		}:
			log.Info("Commit new block creation work", "sealhash", c.engine.SealHash(block.Header()),
				"txs", c.current.tcount,
				"gas", block.GasUsed(), "fees", totalFees(block, receipts),
				"tips", tips.GetHashes(),
				"elapsed", common.PrettyDuration(time.Since(start)),
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

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

// postSideBlock fires a side chain event, only use it for testing.
func (c *Creator) postSideBlock(event core.ChainSideEvent) {
	select {
	case c.chainSideCh <- event:
	case <-c.exitCh:
	}
}

// totalFees computes total consumed miner fees in ETH. Block transactions and receipts have to have the same order.
func totalFees(block *types.Block, receipts []*types.Receipt) *big.Float {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Ether)))
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
	for k, txs := range pending {
		_txs := types.Transactions{}
		if c.isAddressAssigned(k) {
			_txs = txs
		}
		if len(_txs) > 0 {
			pending[k] = _txs
		} else {
			delete(pending, k)
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
		countVal     = big.NewInt(int64(creatorCount))
		creatorNr    = int64(-1)
		val          = address.Hash().Big()
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
	pos := new(big.Int).Mod(val, countVal).Int64()
	return pos == creatorNr
}

// getAssignment returns list of creators, slot and epoch
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
