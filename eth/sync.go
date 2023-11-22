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
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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

// TODO: deprecated
// loop runs in its own goroutine and launches the sync when necessary.
func (cs *chainSyncer) loop() {
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
		si := cs.handler.chain.GetSlotInfo()
		if si != nil {
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

			case <-cs.handler.quitSync:
				log.Warn("sync: quit")
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
