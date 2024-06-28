package creator

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts/keystore"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
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
	Etherbase   common.Address `toml:",omitempty"` // Public address for block creation rewards (default = first account)
	PasswordDir string         // Keystore password directory
	Notify      []string       `toml:",omitempty"` // HTTP URL list to be notified of new work packages (only useful in ethash).
	NotifyFull  bool           `toml:",omitempty"` // Notify with pending block headers instead of work packages
	ExtraData   hexutil.Bytes  `toml:",omitempty"` // Block extra data set by the miner
	GasFloor    uint64         // Target gas floor for mined blocks.
	GasCeil     uint64         // Target gas ceiling for mined blocks.
	GasPrice    *big.Int       // Minimum gas price for mining a transaction
	Recommit    time.Duration  // The time interval for creator to re-create block creation work.
	Noverify    bool           // Disable remote block creation solution verification(only useful in ethash).
}

// environment is the Creator's current environment and holds all of the current state information.
type environment struct {
	signer   types.Signer
	keystore *keystore.KeyStore

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
	config  *Config
	backend Backend
	bc      *core.BlockChain

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

	nodeCreators map[common.Address]struct{}
	creatorsMu   sync.RWMutex
}

// New creates new Creator instance
func New(config *Config, backend Backend, mux *event.TypeMux) *Creator {
	creator := &Creator{
		config:       config,
		backend:      backend,
		mux:          mux,
		bc:           backend.BlockChain(),
		nodeCreators: make(map[common.Address]struct{}),
	}

	creator.SetNodeCreators(backend.AccountManager().Accounts())

	accCh := make(chan accounts.WalletEvent)
	am := backend.AccountManager()
	am.Subscribe(accCh)
	go creator.accountsWatcherLoop(accCh)

	return creator
}

func (c *Creator) accountsWatcherLoop(eventCh chan accounts.WalletEvent) {
	for event := range eventCh {
		c.SetNodeCreators(c.backend.AccountManager().Accounts())
		log.Info("Creator: accounts evt", "kind", event.Kind, "url", event.Wallet.URL(), "accs", c.backend.AccountManager().Accounts())
	}
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
func (c *Creator) RunBlockCreation(slot uint64,
	slotCreators []common.Address,
	tips types.Tips,
	checkpoint *types.Checkpoint,
) error {
	defer func(start time.Time) {
		log.Info("BLOCK CREATION TIME - TOTAL",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "RunBlockCreation",
			"slot", slot,
		)
	}(time.Now())

	if c.isSyncing() {
		log.Warn("Creator skipping due to synchronization")
		return errors.New("synchronization")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	assigned := &Assignment{
		Slot:     slot,
		Creators: slotCreators,
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

	//check hibernate mode
	isHibernateMode, err := c.bc.IsHibernateSlot(header)
	if err != nil {
		log.Error("Creator failed to check is hibernate mode", "err", err)
		return err
	}
	if isHibernateMode {
		log.Info("Creator: run in hibernate mode",
			"isHibernateMode", isHibernateMode,
			"slot", header.Slot,
		)
	}

	wg := new(sync.WaitGroup)
	for _, account := range assigned.Creators {
		var needEmptyBlock bool
		if c.IsCreatorActive(account) {
			if account == assigned.Creators[0] {
				needEmptyBlock, err = c.needEmptyBlock(slot)
				if err != nil {
					return err
				}
			}
			wg.Add(1)
			c.current.txsMu.Lock()
			c.current.txs[account] = &txsWithCumulativeGas{}
			c.current.txsMu.Unlock()
			go c.createNewBlock(account, assigned.Creators, types.CopyHeader(header), wg, needEmptyBlock, isHibernateMode)
		}
	}
	wg.Wait()
	return nil
}

func (c *Creator) needEmptyBlock(slot uint64) (bool, error) {
	slotEpoch := c.bc.GetSlotInfo().SlotToEpoch(slot)

	currentEpochStartSlot, err := c.bc.GetSlotInfo().SlotOfEpochStart(slotEpoch)
	if err != nil {
		log.Error("error while calculating epoch start slot", "error", err)
		return false, err
	}

	if slot == currentEpochStartSlot {
		have, err := c.bc.HaveEpochBlocks(slotEpoch - 1)
		if err != nil {
			log.Error("error while check previous epoch blocks", "error", err)
			return false, err
		}

		if !have {
			return true, nil
		}
	}

	return false, nil
}

func (c *Creator) prepareBlockHeader(assigned *Assignment, tipsBlocks types.BlockMap, tips types.Tips, checkpoint *types.Checkpoint) (*types.Header, error) {
	// if max slot of parents is less or equal to last finalized block slot
	// - add last finalized block to parents
	lastFinBlock := c.bc.GetLastFinalizedBlock()
	lfHash := lastFinBlock.Hash()

	var parentTips types.Tips
	if c.bc.Config().IsForkSlotPrefixFin(assigned.Slot) {
		parentTips = tips.Copy()
		isNotLfInPast := true
		for _, t := range parentTips {
			if t.OrderedAncestorsHashes.Has(lfHash) || t.CpHash == lfHash {
				isNotLfInPast = false
				break
			}
		}
		if isNotLfInPast {
			//add last finalized to parents
			tipsBlocks[lfHash] = lastFinBlock
			// check last finalized blockDag exists
			lfBlDag := c.bc.GetBlockDag(lfHash)
			if lfBlDag == nil {
				cpHash := lastFinBlock.CpHash()
				if lastFinBlock.Height() == 0 {
					cpHash = lastFinBlock.Hash()
				}
				_, anc, _, err := c.bc.CollectAncestorsAftCpByParents(lastFinBlock.ParentHashes(), cpHash)
				if err != nil {
					return nil, err
				}
				lfBlDag = &types.BlockDAG{
					Hash:                   lastFinBlock.Hash(),
					Height:                 lastFinBlock.Height(),
					Slot:                   lastFinBlock.Slot(),
					CpHash:                 cpHash,
					CpHeight:               c.bc.GetHeader(cpHash).Height,
					OrderedAncestorsHashes: anc.Hashes(),
				}
				c.bc.SaveBlockDag(lfBlDag)
			}
			parentTips.Add(lfBlDag)
		}
	} else {
		parentTips = tips
		maxParentSlot := uint64(0)
		for _, blk := range tipsBlocks {
			if blk.Slot() > maxParentSlot {
				maxParentSlot = blk.Slot()
			}
		}
		if maxParentSlot <= lastFinBlock.Slot() {
			tipsBlocks[lfHash] = lastFinBlock
		}
	}

	parentHashes := tipsBlocks.Hashes().Sort()

	cpHeader := c.bc.GetHeader(checkpoint.Spine)
	newHeight, err := c.bc.CalcBlockHeightByTips(parentTips, cpHeader.Hash())
	if err != nil {
		log.Error("Creator calculate block height failed", "err", err)
		return nil, err
	}

	log.Info("Creator calculate block height",
		"newHeight", newHeight,
		"cpHash", checkpoint.Spine.Hex(),
		"parentHashes", parentHashes,
	)

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
	validatorsCount, err := c.bc.ValidatorStorage().GetActiveValidatorsCount(c.bc, header.Slot)
	if err != nil {
		return nil, err
	}
	header.BaseFee = misc.CalcSlotBaseFee(c.bc.Config(), creatorsPerSlotCount, validatorsCount, c.bc.Genesis().GasLimit())

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
					if parentBlock == nil {
						log.Warn("Creator reorg tips failed: bad parent in dag", "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "parent", hash.Hex())
						continue
					}
					cpHeader := c.bc.GetHeader(parentBlock.CpHash)
					//if block not finalized
					log.Warn("Creator reorg tips: active BlockDag not found", "parent", hash.Hex(), "parent.slot", parentBlock.Slot, "parent.height", parentBlock.Height, "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex())
					isCpAncestor, ancestors, unloaded, err := c.bc.CollectAncestorsAftCpByParents(block.ParentHashes(), block.CpHash())
					if err != nil {
						return nil, err
					}
					if len(unloaded) > 0 {
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
					dagBlock = &types.BlockDAG{
						Hash:                   hash,
						Height:                 parentBlock.Height,
						Slot:                   parentBlock.Slot,
						CpHash:                 parentBlock.CpHash,
						CpHeight:               cpHeader.Height,
						OrderedAncestorsHashes: ancestors.Hashes(),
					}
				}
				dagBlock.OrderedAncestorsHashes = dagBlock.OrderedAncestorsHashes.Difference(common.HashArray{genesis})
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

func (c *Creator) createNewBlock(coinbase common.Address, creators []common.Address, header *types.Header, wg *sync.WaitGroup, needEmptyBlock, isHibernateMode bool) {
	start := time.Now()

	log.Info("Try to create new block", "slot", header.Slot, "coinbase", coinbase.Hex())
	defer wg.Done()

	if coinbase == (common.Address{}) {
		log.Error("Refusing to create without etherbase")
		return
	}
	header.Coinbase = coinbase

	// Fill the block with all available pending transactions.
	pendingTxs := make(map[common.Address]types.Transactions)
	// Skip while hibernate mode.
	if !isHibernateMode {
		pendingTxs = c.getPending(coinbase, creators)
	}

	syncData := validatorsync.GetPendingValidatorSyncData(c.bc)
	if len(syncData) > 0 || len(pendingTxs) > 0 || needEmptyBlock {
		startTime := time.Now()
		ks, err := c.getKeystore(c.backend.AccountManager())
		if err != nil {
			log.Error("Failed to fetch keystore", "error", err)
			return
		}
		log.Info("BLOCK CREATION TIME",
			"elapsed", common.PrettyDuration(time.Since(startTime)),
			"func:", "getKeyStore",
			"slot", header.Slot,
		)

		acc := accounts.Account{Address: coinbase}
		start = time.Now()
		ok := ks.IsUnlocked(acc)
		log.Info("BLOCK CREATION TIME",
			"elapsed", common.PrettyDuration(time.Since(startTime)),
			"func:", "IsUnlocked",
			"account", acc.Address.Hex(),
			"slot", header.Slot,
		)
		if !ok {
			startTime = time.Now()
			if err := c.unlockAccount(ks, acc.Address.String()); err != nil {
				log.Warn("Creator: unlock account failed",
					"error", err,
					"elapsed", common.PrettyDuration(time.Since(startTime)),
					"slot", header.Slot,
					"addr", acc.Address.String())
				return
			}

			log.Info("BLOCK CREATION TIME",
				"elapsed", common.PrettyDuration(time.Since(startTime)),
				"func:", "unlockAccount",
				"address", acc.Address.Hex(),
				"slot", header.Slot,
			)
		}
	}

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
			"amount", sd.Amount.String(),
			"TxHash", fmt.Sprintf("%#x", sd.TxHash),
			"curCoinbase", fmt.Sprintf("%#x", coinbase),
			"creators", creators,
		)
	}

	log.Info("Block creation: assigned op for", "senders", len(pendingTxs), "validators", len(syncData))

	// Short circuit if no pending transactions
	if len(pendingTxs) == 0 && len(syncData) == 0 {
		if needEmptyBlock {
			c.create(header, true)
			log.Info("Create empty block", "creator", header.Coinbase.Hex(), "slot", header.Slot)

			log.Info("BLOCK CREATION TIME",
				"elapsed", common.PrettyDuration(time.Since(start)),
				"func:", "createNewBlock",
				"coinbase", coinbase.Hex(),
				"slot", header.Slot,
			)
			return
		}
		pendAddr, queAddr, _ := c.backend.TxPool().StatsByAddrs()
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

			log.Info("BLOCK CREATION TIME",
				"elapsed", common.PrettyDuration(time.Since(start)),
				"func:", "createNewBlock",
				"coinbase", coinbase.Hex(),
				"slot", header.Slot,
			)
			return
		}
		pendAddr, queAddr, _ := c.backend.TxPool().StatsByAddrs()
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

	log.Info("BLOCK CREATION TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "createNewBlock",
		"coinbase", coinbase.Hex(),
		"slot", header.Slot,
	)
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

	start := time.Now()
	signedHeader, err := c.signBlockHeader(block.Header())
	if err != nil {
		log.Error("Failed to sign block", "coinbase", header.Coinbase.Hex(), "blockHash", block.Hash().Hex(), "err", err)
		return
	}

	block.SetHeader(signedHeader)

	log.Info("BLOCK CREATION TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "SignBlock",
		"signer", block.Coinbase().Hex(),
		"slot", header.Slot,
	)

	// Short circuit when receiving duplicate result caused by resubmitting.
	if c.bc.HasBlock(block.Hash()) {
		log.Error("Created block is already creating")
		return
	}

	// Commit block to database.
	_, err = c.bc.WriteCreatedDagBlock(block)
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
		c.updateSnapshot(block)
	}
}

// isSyncing returns tru while sync pocess
func (c *Creator) isSyncing() bool {
	if badTips := c.bc.GetUnsynchronizedTipsHashes(); len(badTips) > 0 {
		return true
	}
	if c.backend.Downloader().Synchronising() {
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
func (c *Creator) updateSnapshot(block *types.Block) {
	c.snapshotMu.Lock()
	defer c.snapshotMu.Unlock()
	c.snapshotBlock = block
}

func (c *Creator) appendTransaction(tx *types.Transaction, header *types.Header, isValidatorOp bool) error {
	if isValidatorOp {
		c.current.txs[header.Coinbase].txs = append(c.current.txs[header.Coinbase].txs, tx)
		return nil
	}

	expectedGas := c.current.txs[header.Coinbase].cumulativeGas + tx.Gas()
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

	//var coalescedLogs []*types.Log

	for {
		// If we don't have enough gas for any further transactions then we're done
		if c.current.gasPool.Gas() < params.TxGas || c.current.txs[header.Coinbase].cumulativeGas > header.GasLimit {
			log.Warn("Not enough gas for further transactions",
				"have", c.current.gasPool,
				"want", params.TxGas,
				"cumulativeGas", header.GasLimit,
				"GasLimit", gasLimit,
				"gasPool<TxGas", c.current.gasPool.Gas() < params.TxGas,
				"cumulativeGas>GasLimit", c.current.txs[header.Coinbase].cumulativeGas > header.GasLimit,
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

	//if !c.IsRunning() && len(coalescedLogs) > 0 {
	//	// We don't push the pendingLogsEvent while we are creating. The reason is that
	//	// when we are creating, the Creator will regenerate a created block every 3 seconds.
	//	// In order to avoid pushing the repeated pendingLog, we disable the pending log pushing.
	//
	//	// make a copy, the state caches the logs and these logs get "upgraded" from pending to mined
	//	// logs by filling in the block hash when the block was mined by the local miner. This can
	//	// cause a race condition if a log was "upgraded" before the PendingLogsEvent is processed.
	//	cpy := make([]*types.Log, len(coalescedLogs))
	//	for i, l := range coalescedLogs {
	//		cpy[i] = new(types.Log)
	//		*cpy[i] = *l
	//	}
	//	c.pendingLogsFeed.Send(cpy)
	//}
	return false
}

// isCreatorActive returns true if creator is assigned to create blocks in current slot.
func (c *Creator) IsCreatorActive(coinbase common.Address) bool {
	c.creatorsMu.Lock()
	defer c.creatorsMu.Unlock()

	_, ok := c.nodeCreators[coinbase]
	return ok
}

// getPending returns all pending transactions for current miner
func (c *Creator) getPending(coinbase common.Address, creators []common.Address) map[common.Address]types.Transactions {
	pending := c.backend.TxPool().Pending(true)

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

func (c *Creator) processValidatorTxs(syncData map[common.Hash]*types.ValidatorSync, header *types.Header) error {
	nonce := c.backend.TxPool().Nonce(header.Coinbase)
	for _, validatorSync := range syncData {
		if validatorSync.ProcEpoch <= c.bc.GetSlotInfo().SlotToEpoch(c.bc.GetSlotInfo().CurrentSlot()) {
			valSyncTx, err := validatorsync.CreateValidatorSyncTx(c.backend, header.CpHash, header.Coinbase, header.Slot, validatorSync, nonce, c.current.keystore)
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

func (c *Creator) SetNodeCreators(accounts []common.Address) {
	c.creatorsMu.Lock()
	defer c.creatorsMu.Unlock()

	if c.nodeCreators == nil {
		c.nodeCreators = make(map[common.Address]struct{})
	}

	for _, account := range accounts {
		c.nodeCreators[account] = struct{}{}
	}
}

// unlockAccount unlocks a specified account.
func (c *Creator) unlockAccount(ks *keystore.KeyStore, targetAddress string) error {
	passwords, err := c.getPasswords()
	if err != nil {
		return err
	}
	keystoreAccounts := ks.Accounts()

	// Find the position of the target account.
	position := findAccountPosition(keystoreAccounts, targetAddress)

	// Unlock the account.log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	return unlockAccount(ks, targetAddress, position, passwords)
}

// getPasswords returns a list of passwords from the password directory.
func (c *Creator) getPasswords() ([]string, error) {
	return makePasswordList(c.config.PasswordDir)
}

// getKeystore retrieves and set cache the encrypted keystore from the account manager.
func (c *Creator) getKeystore(am *accounts.Manager) (*keystore.KeyStore, error) {
	if c.current.keystore != nil {
		return c.current.keystore, nil
	}
	if ks := am.Backends(keystore.KeyStoreType); len(ks) > 0 {
		c.current.keystore = ks[0].(*keystore.KeyStore)
		return ks[0].(*keystore.KeyStore), nil
	}

	return nil, errors.New("local keystore not used")
}

func (c *Creator) signBlockHeader(h *types.Header) (*types.Header, error) {
	key, err := c.current.keystore.GetKey(h.Coinbase)
	if err != nil {
		return nil, err
	}

	return types.SignBlockHeader(h, key)
}
