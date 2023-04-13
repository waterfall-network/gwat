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
	taskCh       chan *task
	resultCh     chan *task
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
		//prev   common.Hash
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
			//sealHash := c.engine.SealHash(task.block.Header())

			//if sealHash == prev {
			//	continue
			//}

			// Interrupt previous sealing operation
			interrupt()
			//stopCh, prev = make(chan struct{}), sealHash

			//if c.skipSealHook != nil && c.skipSealHook(task) {
			//	continue
			//}
			// TODO: resultCh
			c.pendingMu.Lock()
			//c.pendingTasks[sealHash] = task
			c.pendingMu.Unlock()
			//if err := c.engine.Seal(c.chain, task.block, c.resultCh, stopCh); err != nil {
			//	log.Warn("Block sealing failed", "err", err)
			//	c.errWorkCh <- &err
			//}

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

	c.pendingMu.RLock()
	c.pendingMu.RUnlock()

	//// Commit block and state to database.
	_, err := c.chain.WriteMinedBlock(task.block) // TODO: delete delete receipts
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return
	}

	//update state of tips

	//1. remove stale tips
	c.chain.RemoveTips(task.block.ParentHashes())
	//2. create for new blockDag

	tips := task.tips.Copy()
	tmpDagChainHashes := tips.GetOrderedDagChainHashes()
	finDag := tips.GetFinalizingDag()

	// after reorg tips can content hashes of finalized blocks
	finHashes := common.HashArray{}
	for _, h := range tmpDagChainHashes.Copy() {
		blk := c.eth.BlockChain().GetBlock(h)
		finNr := blk.Number()
		if finNr != nil {
			finHashes = append(finHashes, h)
		}
	}
	if len(finHashes) > 0 {
		tmpDagChainHashes = tmpDagChainHashes.Difference(finHashes)
	}
	if finDag.Hash == c.chain.Genesis().Hash() {
		tmpDagChainHashes = tmpDagChainHashes.Difference(common.HashArray{c.chain.Genesis().Hash()})
	}

	newBlockDag := &types.BlockDAG{
		Hash:                task.block.Hash(),
		Height:              task.block.Height(),
		Slot:                task.block.Slot(),
		LastFinalizedHash:   task.block.LFHash(),
		LastFinalizedHeight: task.block.LFNumber(),
		DagChainHashes:      tmpDagChainHashes,
	}
	c.chain.AddTips(newBlockDag)
	c.chain.WriteCurrentTips()

	log.Info("Creator: end tips", "tips", c.chain.GetTips().Print())

	c.chain.MoveTxsToProcessing(types.Blocks{task.block})

	// Broadcast the block and announce chain insertion event
	c.mux.Post(core.NewMinedBlockEvent{Block: task.block})

	// Insert the block into the set of pending ones to resultLoop for confirmations
	log.Info("🔨 created dag block", "slot", task.block.Slot(), "height", task.block.Height(),
		"hash", hash.Hex(), "parents", task.block.ParentHashes(), "LFHash", task.block.LFHash(), "LFNumber", task.block.LFNumber())
}

func (c *Creator) getUnhandledTxs() []*types.Transaction {
	return c.current.txs
}

// makeCurrent creates a new environment for the current cycle.
func (c *Creator) makeCurrent(header *types.Header, recommitBlocks []*types.Block) error {
	// Retrieve the stable state to execute on top and start a prefetcher for
	// the miner to speed block sealing up a bit

	// state.StartPrefetcher("miner") // Merge v0.6-fix-height-calc-validate

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
		c.current.cumutativeGas = expectedGas
		c.current.txs = append(c.current.txs, tx)
	}
	return nil
}

func (c *Creator) appendTransactions(txs *types.TransactionsByPriceAndNonce, coinbase common.Address, lfNumber *uint64) bool {
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
		if c.current.gasPool.Gas() < params.TxGas || c.current.cumutativeGas > c.current.header.GasLimit {
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

		err := c.appendTransaction(tx, lfNumber, false)

		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Error("Gas limit exceeded for current block while create", "sender", from, "hash", tx.Hash().Hex())
			txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, shift in the next transaction from the same account
			log.Info("Tx added", "hash", tx.Hash().Hex(), "err", err)
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
	expCache := core.ExploreResultMap{}
	for _, bl := range blocks {
		if bl.Slot() >= slotInfo.Slot {
			for _, ph := range bl.ParentHashes() {
				_dag := c.eth.BlockChain().ReadBockDag(ph)
				if _dag == nil {
					parentBlock := c.eth.BlockChain().GetBlock(ph)
					if parentBlock == nil {
						log.Warn("Creator reorg tips failed: bad parent in dag", "slot", bl.Slot(), "height", bl.Height(), "hash", bl.Hash().Hex(), "parent", ph.Hex())
						continue
					}
					dagChainHashes := common.HashArray{}
					//if block not finalized
					if parentBlock.Height() > 0 && parentBlock.Nr() == 0 {
						log.Warn("Creator reorg tips: active BlockDag not found", "parent", ph.Hex(), "parent.slot", parentBlock.Slot(), "parent.height", parentBlock.Height(), "slot", bl.Slot(), "height", bl.Height(), "hash", bl.Hash().Hex())
						_, loaded, _, _, exc, _ := c.eth.BlockChain().ExploreChainRecursive(bl.Hash(), expCache)
						expCache = exc
						dagChainHashes = loaded
					}
					_dag = &types.BlockDAG{
						Hash:                ph,
						Height:              parentBlock.Height(),
						Slot:                parentBlock.Slot(),
						LastFinalizedHash:   parentBlock.LFHash(),
						LastFinalizedHeight: parentBlock.LFNumber(),
						DagChainHashes:      dagChainHashes,
					}
				}
				_dag.DagChainHashes = _dag.DagChainHashes.Difference(common.HashArray{genesis})
				tips.Add(_dag)
			}
			delete(tips, bl.Hash())
			log.Info("Creator reorg tips", "blSlot", bl.Slot(), "blHeight", bl.Height(), "blHash", bl.Hash().Hex(), "tips", tips.Print())
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
				delete(tips, ancestor) // TODO check sometimes panic
				delete(tipsBlocks, ancestor)
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
	log.Info("Creator: start tips", "tips", tips.Print())

	//calc new block height
	lastFinBlock := c.chain.GetLastFinalizedBlock()

	// if max slot of parents is less or equal to last finalized block slot
	// - add last finalized block to parents
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

	//calc new block height
	_, stateBlock, recommitBlocks, newHeight, stateErr := c.chain.CollectStateDataByParents(parentHashes)
	if stateErr != nil {
		log.Error("Failed to make block creation context", "err", stateErr)
		c.errWorkCh <- &stateErr
		return
	}

	log.Info("Creator calculate block height", "newHeight", newHeight,
		"recommitsCount", len(recommitBlocks),
		"baseHeight", stateBlock.Height(),
		"baseHash", stateBlock.Hash(),
	)

	// Use checkpoint spine as LFBlock
	checkpoint := c.chain.GetLastCoordinatedCheckpoint()
	checkpointBlock := c.chain.GetBlock(checkpoint.Spine)
	if stateBlock == nil {
		log.Error("Error while get checkpoint spine block", "epoch", checkpoint.Epoch, "root", checkpoint.Root, "spine", checkpoint.Spine)
	}

	header := &types.Header{
		ParentHashes: parentHashes,
		Slot:         slotInfo.Slot,
		Era:          c.chain.GetEraInfo().Number(),
		Height:       newHeight,
		GasLimit:     core.CalcGasLimit(tipsBlocks.AvgGasLimit(), c.config.GasCeil),
		Extra:        c.extra,
		Time:         uint64(timestamp),
		// Checkpoint spine block
		LFHash:        checkpointBlock.Hash(),
		LFNumber:      checkpointBlock.Nr(),
		LFBaseFee:     checkpointBlock.BaseFee(),
		LFBloom:       checkpointBlock.Bloom(),
		LFGasUsed:     checkpointBlock.GasUsed(),
		LFReceiptHash: checkpointBlock.ReceiptHash(),
		LFRoot:        checkpointBlock.Root(),
	}

	// Get active validators number
	creatorsPerSlotCount := c.chainConfig.ValidatorsPerSlot
	if creatorsPerSlot, err := c.chain.ValidatorStorage().GetCreatorsBySlot(c.chain, header.Slot); err == nil {
		creatorsPerSlotCount = uint64(len(creatorsPerSlot))
	}
	validators, _ := c.chain.ValidatorStorage().GetValidators(c.chain, header.Slot, true, false)
	header.BaseFee = misc.CalcSlotBaseFee(c.chainConfig, header, uint64(len(validators)), c.chain.Genesis().GasLimit(), params.BurnMultiplier, creatorsPerSlotCount)

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
	err := c.makeCurrent(header, recommitBlocks)
	if err != nil {
		log.Error("Failed to make block creation context", "err", err)
		c.errWorkCh <- &err
		return
	}

	// Fill the block with all available pending transactions.
	pending := c.getPending()

	// if len(pending) > 0 {
	// 	txs := types.NewTransactionsByPriceAndNonce(c.current.signer, pending, header.BaseFee)
	// 	if c.appendTransactions(txs, c.coinbase, &header.LFNumber) {
	// 		pendAddr, queAddr, _ := c.eth.TxPool().StatsByAddrs()
	// 		log.Warn("Skipping block creation: no assigned txs", "creator", c.coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
	// 		c.errWorkCh <- &ErrNoTxs
	filterPending, err := c.filterPendingTxs(pending)
	if err != nil {
		log.Error("can`t filter pending transactions", "error", err)

		return
	}

	syncData := validatorsync.GetPendingValidatorSyncData(c.chain)

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

	// Short circuit if no pending transactions
	if len(filterPending) == 0 && len(syncData) == 0 {
		pendAddr, queAddr, _ := c.eth.TxPool().StatsByAddrs()
		log.Warn("Skipping block creation: no assigned txs", "creator", c.coinbase, "pendAddr", pendAddr, "queAddr", queAddr)
		c.errWorkCh <- &ErrNoTxs

		return
	}

	txs := types.NewTransactionsByPriceAndNonce(c.current.signer, filterPending, header.BaseFee)
	if c.appendTransactions(txs, c.coinbase, &header.LFNumber) {
		if len(syncData) > 0 && c.isAddressAssigned(*c.chainConfig.ValidatorsStateAddress) {
			if err := c.processValidatorTxs(stateBlock.Hash(), syncData, header.LFNumber); err != nil {
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
		if err := c.processValidatorTxs(stateBlock.Hash(), syncData, header.LFNumber); err != nil {
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
			log.Info("Commit new block creation work", "sealhash", c.engine.SealHash(block.Header()),
				"txs", c.current.tcount,
				"gas", block.GasUsed(), "fees", c.current.cumutativeGas,
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

func (c *Creator) filterPendingTxs(pending map[common.Address]types.Transactions) (map[common.Address]types.Transactions, error) {
	slotCreators, err := c.chain.ValidatorStorage().GetCreatorsBySlot(c.chain, c.chain.GetSlotInfo().CurrentSlot())
	if err != nil {
		return nil, err
	}

	for address := range pending {
		for _, creator := range slotCreators {
			if address == creator && creator != c.coinbase {
				delete(pending, address)
			}
		}
	}

	return pending, nil
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
		}
	}

	return nil
}
