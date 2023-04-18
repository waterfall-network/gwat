// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/protocols/eth"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	defaultMinSyncPeers = 1                // Amount of peers desired to start syncing
	dagSlotsLimit       = 32
)

type peerEvtKind int
type peerEvt struct{ kind peerEvtKind }

const (
	evtDefault peerEvtKind = iota
	evtBroadcast
	evtPeerRun
)

// syncTransactions starts sending all currently pending transactions to the given peer.
func (h *handler) syncTransactions(p *eth.Peer) {
	// Assemble the set of transaction to broadcast or announce to the remote
	// peer. Fun fact, this is quite an expensive operation as it needs to sort
	// the transactions if the sorting is not cached yet. However, with a random
	// order, insertions could overflow the non-executable queues and get dropped.
	//
	// TODO(karalabe): Figure out if we could get away with random order somehow
	var txs types.Transactions
	pending := h.txpool.Pending(false)
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	if len(txs) == 0 {
		return
	}
	// The eth/65 protocol introduces proper transaction announcements, so instead
	// of dripping transactions across multiple peers, just send the entire list as
	// an announcement and let the remote side decide what they need (likely nothing).
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	p.AsyncSendPooledTransactionHashes(hashes)
}

// chainSyncer coordinates blockchain sync components.
type chainSyncer struct {
	handler     *handler
	force       *time.Timer
	forced      bool // true when force timer fired
	peerEventCh chan peerEvt
	doneCh      chan error // non-nil when sync is running
}

// chainSyncOp is a scheduled sync operation.
type chainSyncOp struct {
	mode      downloader.SyncMode
	peer      *eth.Peer
	lastFinNr uint64
	dag       common.HashArray
	dagOnly   bool
}

// newChainSyncer creates a chainSyncer.
func newChainSyncer(handler *handler) *chainSyncer {
	return &chainSyncer{
		handler:     handler,
		peerEventCh: make(chan peerEvt),
	}
}

// handlePeerEvent notifies the syncer about a change in the peer set.
// This is called for new peers and every time a peer announces a new chain block.
func (cs *chainSyncer) handlePeerEvent(peer *eth.Peer, kind peerEvtKind) bool {
	select {
	case cs.peerEventCh <- peerEvt{kind}:
		return true
	case <-cs.handler.quitSync:
		return false
	}
}

// loop runs in its own goroutine and launches the sync when necessary.
func (cs *chainSyncer) loop() {
	startTicker := time.NewTicker(10000 * time.Millisecond)
	defer cs.handler.wg.Done()

	cs.handler.blockFetcher.Start()
	cs.handler.txFetcher.Start()
	defer cs.handler.blockFetcher.Stop()
	defer cs.handler.txFetcher.Stop()
	defer cs.handler.downloader.Terminate()

	// The force timer lowers the peer count threshold down to one when it fires.
	// This ensures we'll always start sync even if there aren't enough peers.
	cs.force = time.NewTimer(forceSyncCycle)
	defer cs.force.Stop()
	var pevt peerEvt
	for {
		//deprecated := true
		si := cs.handler.chain.GetSlotInfo()
		if si != nil {
			//var op *chainSyncOp
			if pevt.kind == evtBroadcast {
			}
			//else {
			//	op = cs.explicitSyncOp()
			//	log.Warn("111111111111 SYNC LOOP start", "op", op)
			//	// check sync is busy
			//	if op != nil && !op.dagOnly && cs.handler.downloader.HeadSynchronising() || cs.handler.downloader.DagSynchronising() {
			//		log.Warn("Synchronization canceled (process busy)", "op", op)
			//		op = nil
			//	}
			//}
			//if op != nil {
			//	log.Warn("Synchronization start", "op", op)
			//	cs.startSync(op)
			//}
			pevt.kind = evtDefault
			select {
			case pevt = <-cs.peerEventCh:
				log.Debug("sync: peer evt", "kind", pevt.kind)
				// Peer information changed, recheck.
			case <-cs.doneCh:
				log.Debug("sync: done ch")
				cs.doneCh = nil
				cs.force.Reset(forceSyncCycle)
				cs.forced = false
			case <-cs.force.C:
				log.Debug("sync: force timer")
				cs.forced = true

			case <-startTicker.C:

				log.Info(" ########### case <-startTicker.C:", "peers.len()", cs.handler.peers.len())

				//if len(cs.handler.chain.GetSyncHashes()) != 0 {
				//	var op *chainSyncOp
				//	op = cs.explicitSyncOp()
				//	log.Warn("111111111111 SYNC LOOP start", "op", op)
				//	if op != nil {
				//		log.Warn("Synchronization start", "op", op)
				//		cs.startSync(op)
				//	}
				//	log.Warn("Synchronization start", "op", op)
				//	pevt.kind = evtDefault
				//}

			case <-cs.handler.quitSync:
				log.Debug("sync: quit")
				// Disable all insertion on the blockchain. This needs to happen before
				// terminating the downloader because the downloader waits for blockchain
				// inserts, and these can take a long time to finish.
				cs.handler.chain.StopInsert()
				cs.handler.downloader.Terminate()
				if cs.doneCh != nil {
					<-cs.doneCh
				}
				return
			}
		}
	}
}

// getResyncOp determines whether resync is required.
func (cs *chainSyncer) getResyncOp() *chainSyncOp {
	if cs.doneCh != nil {
		return nil // Sync already running.
	}

	// Ensure we're at minimum peer count.
	minPeers := defaultMinSyncPeers
	if cs.forced {
		minPeers = 1
	} else if minPeers > cs.handler.maxPeers {
		minPeers = cs.handler.maxPeers
	}
	if cs.handler.peers.len() < minPeers {
		return nil
	}
	// We have enough peers, select peer to sync
	peer := cs.handler.peers.getPeer(false)
	if peer == nil {
		return nil
	}
	mode := cs.modeAndLocalHead()
	if mode == downloader.FastSync && atomic.LoadUint32(&cs.handler.snapSync) == 1 {
		// Fast sync via the snap protocol
		mode = downloader.SnapSync
	}
	op := peerToSyncOp(mode, peer)
	lastFinNr := cs.handler.chain.GetLastFinalizedNumber()

	if lastFinNr >= op.lastFinNr {
		return nil
	}

	if lastFinNr < op.lastFinNr {
		// finalized chain sync required
		return op
	}
	return nil
}

// nextSyncOp determines whether sync is required at this time.
func (cs *chainSyncer) nextSyncOp() *chainSyncOp {
	if cs.doneCh != nil {
		return nil // Sync already running.
	}

	// Ensure we're at minimum peer count.
	minPeers := defaultMinSyncPeers
	if cs.forced {
		minPeers = 1
	} else if minPeers > cs.handler.maxPeers {
		minPeers = cs.handler.maxPeers
	}
	if cs.handler.peers.len() < minPeers {
		log.Info("Count of active peers is less then minPeers")
		return nil
	}
	// We have enough peers, select peer to sync
	peer := cs.handler.peers.getPeer(true)
	if peer == nil {
		return nil
	}
	mode := cs.modeAndLocalHead()
	if mode == downloader.FastSync && atomic.LoadUint32(&cs.handler.snapSync) == 1 {
		// Fast sync via the snap protocol
		mode = downloader.SnapSync
	}
	op := peerToSyncOp(mode, peer)
	lastFinNr := cs.handler.chain.GetLastFinalizedNumber()

	if lastFinNr > op.lastFinNr {
		return nil // We're in sync.
	}

	if lastFinNr < op.lastFinNr {
		// finalized chain sync required
		return op
	}

	localTips := cs.handler.chain.GetTips()
	dagHashes := common.HashArray{}
	if dhs := cs.handler.chain.GetDagHashes(); dhs != nil {
		dagHashes = *dhs
	}
	_, dag := peer.GetDagInfo()
	for _, hash := range *dag {
		block := cs.handler.chain.GetBlockByHash(hash)
		if len(localTips) == 0 && block != nil && block.Nr() == lastFinNr {
			// if remote tips set to last finalized block - do same
			cs.handler.chain.ResetTips()
			break
		}
		if block == nil || (!dagHashes.Has(hash) && block.Number() == nil) {
			// dag sync required
			if op.dag == nil {
				op.dag = common.HashArray{}
			}
			op.dag = append(op.dag, hash)
			op.dagOnly = true
		}
	}
	if op.dagOnly {
		return op
	}
	return nil
}

// nextSyncOp determines whether sync is required at this time.
func (cs *chainSyncer) explicitSyncOp() *chainSyncOp {
	log.Warn("  >>>>>> explicitSyncOp Before dine ch sync", "sync hashes", cs.handler.chain.GetSyncHashes())
	if cs.doneCh != nil {
		log.Warn("  >>>>>> explicitSyncOp Sync already running.")
		return nil // Sync already running.
	}

	if len(cs.handler.chain.GetSyncHashes()) == 0 {
		log.Warn("Nothing to sync", "sync hashes", cs.handler.chain.GetSyncHashes())
		return nil // Sync already running.
	}

	for _, hash := range cs.handler.chain.GetSyncHashes() {
		if bl := cs.handler.chain.GetBlock(hash); bl != nil {
			log.Warn("33333333 BLOCK ALREADY SYNCED", "hash", bl.Hash())
			cs.handler.chain.RemoveSyncHash(bl.Hash())
		}
	}
	log.Warn("333333332222222 BLOCK ALREADY SYNCED", "hashes", cs.handler.chain.GetSyncHashes())
	if len(cs.handler.chain.GetSyncHashes()) == 0 {
		log.Warn("Nothing to sync after removal", "sync hashes", cs.handler.chain.GetSyncHashes())
	}

	if len(cs.handler.chain.GetSyncHashes()) == 0 {
		log.Warn("Nothing to sync", "sync hashes", cs.handler.chain.GetSyncHashes())
		return nil // Sync already running.
	}

	// Ensure we're at minimum peer count.
	minPeers := defaultMinSyncPeers
	if cs.forced {
		minPeers = 1
	} else if minPeers > cs.handler.maxPeers {
		minPeers = cs.handler.maxPeers
	}
	if cs.handler.peers.len() < minPeers {
		log.Info("Count of active peers is less then minPeers")
		return nil
	}
	// We have enough peers, select peer to sync
	peer := cs.handler.peers.getPeer(true)
	log.Warn("  >>>>>> explicitSyncOp Before dine ch sync", "sync hashes", peer)
	if peer == nil {
		log.Warn("  >>>>>> NO PEER FOUND <<<<<<<")
		return nil
	}
	op := peerToSyncOp(downloader.FullSync, peer)

	op.dag = cs.handler.chain.GetSyncHashes()
	//if op.dagOnly {
	log.Warn("555555 OP OP OP OP", "op", op)
	return op
	//}
	//return nil
}

func peerToSyncOp(mode downloader.SyncMode, p *eth.Peer) *chainSyncOp {
	lastFinNr, _ := p.GetDagInfo()
	return &chainSyncOp{mode: mode, peer: p, lastFinNr: lastFinNr, dag: common.HashArray{}, dagOnly: false}
}

func (cs *chainSyncer) modeAndLocalHead() downloader.SyncMode {
	// If we're in fast sync mode, return that directly
	if atomic.LoadUint32(&cs.handler.fastSync) == 1 {
		return downloader.FastSync
	}
	// We are probably in full sync, but we might have rewound to before the
	// fast sync pivot, check if we should reenable
	if pivot := rawdb.ReadLastPivotNumber(cs.handler.database); pivot != nil {
		head := cs.handler.chain.GetLastFinalizedBlock()
		height := cs.handler.chain.GetBlockFinalizedNumber(head.Hash())
		if height != nil && *height < *pivot {
			return downloader.FastSync
		}
	}
	// Nope, we're really full syncing
	return downloader.FullSync
}

// startSync launches doSync in a new goroutine.
func (cs *chainSyncer) startSync(op *chainSyncOp) {
	cs.doneCh = make(chan error, 1)
	go func() { cs.doneCh <- cs.handler.doSync(op) }()
}

// doSync synchronizes the local blockchain with a remote peer.
func (h *handler) doSync(op *chainSyncOp) error {
	//if op.mode == downloader.FastSync || op.mode == downloader.SnapSync {
	//	// Before launch the fast sync, we have to ensure user uses the same
	//	// txlookup limit.
	//	// The main concern here is: during the fast sync Geth won't index the
	//	// block(generate tx indices) before the HEAD-limit. But if user changes
	//	// the limit in the next fast sync(e.g. user kill Geth manually and
	//	// restart) then it will be hard for Geth to figure out the oldest block
	//	// has been indexed. So here for the user-experience wise, it's non-optimal
	//	// that user can't change limit during the fast sync. If changed, Geth
	//	// will just blindly use the original one.
	//	limit := h.chain.TxLookupLimit()
	//	if stored := rawdb.ReadFastTxLookupLimit(h.database); stored == nil {
	//		rawdb.WriteFastTxLookupLimit(h.database, limit)
	//	} else if *stored != limit {
	//		h.chain.SetTxLookupLimit(*stored)
	//		log.Warn("Update txLookup limit", "provided", limit, "updated", *stored)
	//	}
	//}
	// Run the sync cycle, and disable fast sync if we're past the pivot block
	err := h.downloader.Synchronise(op.peer.ID(), op.dag, op.lastFinNr, op.mode, op.dagOnly)
	if err != nil {
		return err
	}
	if atomic.LoadUint32(&h.fastSync) == 1 {
		log.Info("Fast sync complete, auto disabling")
		atomic.StoreUint32(&h.fastSync, 0)
	}
	if atomic.LoadUint32(&h.snapSync) == 1 {
		log.Info("Snap sync complete, auto disabling")
		atomic.StoreUint32(&h.snapSync, 0)
	}
	// If we've successfully finished a sync cycle and passed any required checkpoint,
	// enable accepting transactions from the network.
	//lastFinalizedBlock := h.chain.GetLastFinalizedBlock()
	//lastFinalizedNr := h.chain.GetLastFinalizedNumber()

	//log.Info("Sync process",
	//	"lastFinalizedBlock", lastFinalizedBlock,
	//	"lastFinalizedNr", lastFinalizedNr,
	//	"h.checkpointNumber", h.checkpointNumber,
	//	"lastFinalizedNr >= h.checkpointNumber", lastFinalizedNr >= h.checkpointNumber,
	//)

	//if lastFinalizedNr >= h.checkpointNumber {
	//	// Checkpoint passed, sanity check the timestamp to have a fallback mechanism
	//	// for non-checkpointed (number = 0) private networks.
	//	if lastFinalizedBlock.Time() >= uint64(time.Now().AddDate(0, -1, 0).Unix()) {
	//		atomic.StoreUint32(&h.acceptTxs, 1)
	//	}
	//}
	//atomic.StoreUint32(&h.acceptTxs, 1)

	//bTips := h.chain.GetBlocksByHashes(h.chain.GetTips().GetHashes())
	//for _, block := range bTips {
	//	h.BroadcastBlock(block, false)
	//}
	return nil
}
