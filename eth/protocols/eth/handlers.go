// Copyright 2020 The go-ethereum Authors
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
	"fmt"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
)

// handleGetBlockHeaders66 is the eth/66 version of handleGetBlockHeaders
func handleGetBlockHeaders66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode the complex header query
	var query GetBlockHeadersPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	var response []*types.Header
	if len(*query.Hashes) > 0 {
		response = backend.Chain().GetHeadersByHashes(*query.Hashes).RmEmpty().ToArray()
	} else {
		response = answerGetBlockHeadersQuery(backend, query.GetBlockHeadersPacket, peer)
	}
	return peer.ReplyBlockHeaders(query.RequestId, response)
}

func answerGetBlockHeadersQuery(backend Backend, query *GetBlockHeadersPacket, peer *Peer) []*types.Header {
	hashMode := query.Origin.Hash != (common.Hash{})
	first := true
	maxNonCanonical := uint64(100)

	// Gather headers until the fetch or network limits is reached
	var (
		bytes   common.StorageSize
		headers []*types.Header
		unknown bool
		lookups int
	)
	for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit &&
		len(headers) < maxHeadersServe && lookups < 2*maxHeadersServe {
		lookups++
		// Retrieve the next header satisfying the query
		var origin *types.Header
		if hashMode {
			if first {
				first = false
				origin = backend.Chain().GetHeaderByHash(query.Origin.Hash)
				//if origin != nil {
				//	query.Origin.Number = origin.Number.Uint64()
				//}
			} else {
				//origin = backend.Chain().GetHeader(query.Origin.Hash, query.Origin.Number)
				origin = backend.Chain().GetHeader(query.Origin.Hash)
			}
		} else {
			origin = backend.Chain().GetHeaderByNumber(query.Origin.Number)
		}
		if origin == nil {
			break
		}
		headers = append(headers, origin)
		bytes += estHeaderSize

		// Advance to the next header of the query
		switch {
		case hashMode && query.Reverse:
			// Hash based traversal towards the genesis block
			ancestor := query.Skip + 1
			if ancestor == 0 {
				unknown = true
			} else {
				query.Origin.Hash, query.Origin.Number = backend.Chain().GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
				unknown = (query.Origin.Hash == common.Hash{})
			}
		case hashMode && !query.Reverse:
			// Hash based traversal towards the leaf block
			var (
				//current = origin.Number.Uint64()
				nr             = backend.Chain().ReadFinalizedNumberByHash(origin.Hash())
				current uint64 = 0
				next    uint64 = 0
			)
			if nr != nil {
				current = *nr
				next = current + query.Skip + 1
			}
			if next <= current {
				peer.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", peer.Peer.Info())
				unknown = true
			}
			//} else {
			if header := backend.Chain().GetHeaderByNumber(next); header != nil {
				nextHash := header.Hash()
				expOldHash, _ := backend.Chain().GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
				if expOldHash == query.Origin.Hash {
					query.Origin.Hash, query.Origin.Number = nextHash, next
				} else {
					unknown = true
				}
			} else {
				unknown = true
			}
			//}
		case query.Reverse:
			// Number based traversal towards the genesis block
			if query.Origin.Number >= query.Skip+1 {
				query.Origin.Number -= query.Skip + 1
			} else {
				unknown = true
			}

		case !query.Reverse:
			// Number based traversal towards the leaf block
			query.Origin.Number += query.Skip + 1
		}
	}
	return headers
}

func handleGetBlockBodies66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode the block body retrieval message
	var query GetBlockBodiesPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	response := answerGetBlockBodiesQuery(backend, query.GetBlockBodiesPacket, peer)
	return peer.ReplyBlockBodiesRLP(query.RequestId, response)
}

func answerGetBlockBodiesQuery(backend Backend, query GetBlockBodiesPacket, peer *Peer) []rlp.RawValue {
	// Gather blocks until the fetch or network limits is reached
	var (
		bytes  int
		bodies []rlp.RawValue
	)
	for lookups, hash := range query {
		if bytes >= softResponseLimit || len(bodies) >= maxBodiesServe ||
			lookups >= 2*maxBodiesServe {
			break
		}
		if data := backend.Chain().GetBodyRLP(hash); len(data) != 0 {
			bodies = append(bodies, data)
			bytes += len(data)
		}
	}
	return bodies
}

func handleGetNodeData66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode the trie node data retrieval message
	var query GetNodeDataPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	response := answerGetNodeDataQuery(backend, query.GetNodeDataPacket, peer)
	return peer.ReplyNodeData(query.RequestId, response)
}

func answerGetNodeDataQuery(backend Backend, query GetNodeDataPacket, peer *Peer) [][]byte {
	// Gather state data until the fetch or network limits is reached
	var (
		bytes int
		nodes [][]byte
	)
	for lookups, hash := range query {
		if bytes >= softResponseLimit || len(nodes) >= maxNodeDataServe ||
			lookups >= 2*maxNodeDataServe {
			break
		}
		// Retrieve the requested state entry
		if bloom := backend.StateBloom(); bloom != nil && !bloom.Contains(hash[:]) {
			// Only lookup the trie node if there's chance that we actually have it
			continue
		}
		entry, err := backend.Chain().TrieNode(hash)
		if len(entry) == 0 || err != nil {
			// Read the contract code with prefix only to save unnecessary lookups.
			entry, err = backend.Chain().ContractCodeWithPrefix(hash)
		}
		if err == nil && len(entry) > 0 {
			nodes = append(nodes, entry)
			bytes += len(entry)
		}
	}
	return nodes
}

func handleGetReceipts66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode the block receipts retrieval message
	var query GetReceiptsPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	response := answerGetReceiptsQuery(backend, query.GetReceiptsPacket, peer)
	return peer.ReplyReceiptsRLP(query.RequestId, response)
}

func answerGetReceiptsQuery(backend Backend, query GetReceiptsPacket, peer *Peer) []rlp.RawValue {
	// Gather state data until the fetch or network limits is reached
	var (
		bytes    int
		receipts []rlp.RawValue
	)
	for lookups, hash := range query {
		if bytes >= softResponseLimit || len(receipts) >= maxReceiptsServe ||
			lookups >= 2*maxReceiptsServe {
			break
		}
		// Retrieve the requested block's receipts
		results := backend.Chain().GetReceiptsByHash(hash)
		if results == nil {
			if header := backend.Chain().GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
				continue
			}
		}
		// If known, encode and queue for response packet
		if encoded, err := rlp.EncodeToBytes(results); err != nil {
			log.Error("Failed to encode receipt", "err", err)
		} else {
			receipts = append(receipts, encoded)
			bytes += len(encoded)
		}
	}
	return receipts
}

func handleNewBlockhashes(backend Backend, msg Decoder, peer *Peer) error {
	// A batch of new block announcements just arrived
	ann := new(NewBlockHashesPacket)
	if err := msg.Decode(ann); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Mark the hashes as present at the remote node
	for _, block := range *ann {
		peer.markBlock(block.Hash)
	}
	// Deliver them all to the backend for queuing
	return backend.Handle(peer, ann)
}

func handleNewBlock(backend Backend, msg Decoder, peer *Peer) error {
	// Retrieve and decode the propagated block
	ann := new(NewBlockPacket)
	if err := msg.Decode(ann); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	if err := ann.sanityCheck(); err != nil {
		return err
	}
	if hash := types.DeriveSha(ann.Block.Transactions(), trie.NewStackTrie(nil)); hash != ann.Block.TxHash() {
		log.Warn("Propagated block has invalid body", "have", hash, "exp", ann.Block.TxHash())
		return nil // TODO(karalabe): return error eventually, but wait a few releases
	}
	ann.Block.ReceivedAt = msg.Time()
	ann.Block.ReceivedFrom = peer

	// Mark the peer as owning the block
	peer.markBlock(ann.Block.Hash())

	return backend.Handle(peer, ann)
}

func handleBlockHeaders66(backend Backend, msg Decoder, peer *Peer) error {
	// A batch of headers arrived to one of our previous requests
	res := new(BlockHeadersPacket66)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	requestTracker.Fulfil(peer.id, peer.version, BlockHeadersMsg, res.RequestId)

	return backend.Handle(peer, &res.BlockHeadersPacket)
}

func handleBlockBodies66(backend Backend, msg Decoder, peer *Peer) error {
	// A batch of block bodies arrived to one of our previous requests
	res := new(BlockBodiesPacket66)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	requestTracker.Fulfil(peer.id, peer.version, BlockBodiesMsg, res.RequestId)

	return backend.Handle(peer, &res.BlockBodiesPacket)
}

func handleNodeData66(backend Backend, msg Decoder, peer *Peer) error {
	// A batch of node state data arrived to one of our previous requests
	res := new(NodeDataPacket66)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	requestTracker.Fulfil(peer.id, peer.version, NodeDataMsg, res.RequestId)

	return backend.Handle(peer, &res.NodeDataPacket)
}

func handleReceipts66(backend Backend, msg Decoder, peer *Peer) error {
	// A batch of receipts arrived to one of our previous requests
	res := new(ReceiptsPacket66)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	requestTracker.Fulfil(peer.id, peer.version, ReceiptsMsg, res.RequestId)

	return backend.Handle(peer, &res.ReceiptsPacket)
}

func handleNewPooledTransactionHashes(backend Backend, msg Decoder, peer *Peer) error {
	// New transaction announcement arrived, make sure we have
	// a valid and fresh chain to handle them
	if !backend.AcceptTxs() {
		return nil
	}
	ann := new(NewPooledTransactionHashesPacket)
	if err := msg.Decode(ann); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	// Schedule all the unknown hashes for retrieval
	for _, hash := range *ann {
		peer.markTransaction(hash)
	}
	return backend.Handle(peer, ann)
}

func handleGetPooledTransactions66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode the pooled transactions retrieval message
	var query GetPooledTransactionsPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	hashes, txs := answerGetPooledTransactions(backend, query.GetPooledTransactionsPacket, peer)
	return peer.ReplyPooledTransactionsRLP(query.RequestId, hashes, txs)
}

func answerGetPooledTransactions(backend Backend, query GetPooledTransactionsPacket, peer *Peer) ([]common.Hash, []rlp.RawValue) {
	// Gather transactions until the fetch or network limits is reached
	var (
		bytes  int
		hashes []common.Hash
		txs    []rlp.RawValue
	)
	for _, hash := range query {
		if bytes >= softResponseLimit {
			break
		}
		// Retrieve the requested transaction, skipping if unknown to us
		tx := backend.TxPool().Get(hash)
		if tx == nil {
			continue
		}
		// If known, encode and queue for response packet
		if encoded, err := rlp.EncodeToBytes(tx); err != nil {
			log.Error("Failed to encode transaction", "err", err)
		} else {
			hashes = append(hashes, hash)
			txs = append(txs, encoded)
			bytes += len(encoded)
		}
	}
	return hashes, txs
}

func handleTransactions(backend Backend, msg Decoder, peer *Peer) error {
	// Transactions arrived, make sure we have a valid and fresh chain to handle them
	if !backend.AcceptTxs() || !backend.Chain().IsSynced() {
		log.Debug("skip handle tx: handleTransactions node is syncing", "AcceptTxs", backend.AcceptTxs(), "IsSynced", backend.Chain().IsSynced())
		return nil
	}
	// Transactions can be processed, parse all of them and deliver to the pool
	var txs TransactionsPacket
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	for i, tx := range txs {
		// Validate and mark the remote transaction
		if tx == nil {
			return fmt.Errorf("%w: transaction %d is nil", errDecode, i)
		}
		peer.markTransaction(tx.Hash())
	}
	return backend.Handle(peer, &txs)
}

func handlePooledTransactions66(backend Backend, msg Decoder, peer *Peer) error {
	// Transactions arrived, make sure we have a valid and fresh chain to handle them
	if !backend.AcceptTxs() || !backend.Chain().IsSynced() {
		log.Warn("node is syncing, handlePooledTransactions66 transaction cannot be processed at this moment", "AcceptTxs()", backend.AcceptTxs(), "IsSynced", backend.Chain().IsSynced())
		return nil
	}
	// Transactions can be processed, parse all of them and deliver to the pool
	var txs PooledTransactionsPacket66
	if err := msg.Decode(&txs); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	for i, tx := range txs.PooledTransactionsPacket {
		// Validate and mark the remote transaction
		if tx == nil {
			return fmt.Errorf("%w: transaction %d is nil", errDecode, i)
		}
		peer.markTransaction(tx.Hash())
	}
	requestTracker.Fulfil(peer.id, peer.version, PooledTransactionsMsg, txs.RequestId)

	return backend.Handle(peer, &txs.PooledTransactionsPacket)
}

func handleGetDag66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode retrieval message
	var query GetDagPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	dag, err := answerGetDagQuery(backend, query.GetDagPacket)
	if err != nil {
		return peer.ReplyDagData(query.RequestId, common.HashArray{})
	}
	return peer.ReplyDagData(query.RequestId, dag)
}

func answerGetDagQuery(backend Backend, query GetDagPacket) (common.HashArray, error) {
	dag := common.HashArray{}
	dagHashes := common.HashArray{}
	limitReached := false
	log.Debug("Sync handling: start",
		"baseSpine", query.BaseSpine.Hex(),
		"terminalSpine", query.TerminalSpine.Hex(),
	)
	// baseSpine
	baseHeader := backend.Chain().GetHeaderByHash(query.BaseSpine)
	if baseHeader == nil {
		return dag, fmt.Errorf("%w", errInvalidDag)
	}
	isBaseFinalized := baseHeader.Height > 0 && baseHeader.Nr() > 0 || baseHeader.Hash() == backend.Chain().Genesis().Hash()

	// terminalSpine
	var terminalSpine common.Hash
	if query.TerminalSpine != (common.Hash{}) {
		terminalSpine = query.TerminalSpine
	} else {
		tSlot := uint64(0)
		for h, t := range backend.Chain().GetTips() {
			if t.Slot >= tSlot {
				terminalSpine = h
			}
		}
		if terminalSpine == (common.Hash{}) {
			return dag, fmt.Errorf("%w", errInvalidDag)
		}
	}
	log.Debug("Sync handling: terminal header",
		"baseSpine", query.BaseSpine.Hex(),
		"terminalSpine", query.TerminalSpine.Hex(),
		"terminalHeader", terminalSpine,
	)
	// retrieve terminal header
	terminalHeader := backend.Chain().GetHeaderByHash(terminalSpine)
	if terminalHeader == nil {
		return dag, fmt.Errorf("%w", errInvalidDag)
	}
	isTerminalFinalized := terminalHeader.Height > 0 && terminalHeader.Nr() > 0

	switch {
	case isBaseFinalized && isTerminalFinalized:
		//collect blocks by range of finalized numbers only
		fromNr := baseHeader.Nr()
		toNr := terminalHeader.Nr()
		log.Debug("Sync handling: case 1",
			"baseSpine", query.BaseSpine.Hex(),
			"terminalSpine", query.TerminalSpine.Hex(),
			"isBaseFinalized", isBaseFinalized,
			"isTerminalFinalized", isTerminalFinalized,
			"fromNr", fromNr,
			"toNr", toNr,
		)
		if fromNr > toNr {
			return dag, errBadRequestParam
		}
		dag = getHashesByNumberRange(backend, fromNr, toNr, LimitDagHashes)
		limitReached = len(dag) >= LimitDagHashes

	case isBaseFinalized && !isTerminalFinalized:
		//1. collect finalized part
		fromNr := baseHeader.Nr()
		lfHeader := backend.Chain().GetLastFinalizedHeader()
		toNr := lfHeader.Nr()
		//check data limitation
		if toNr-fromNr > LimitDagHashes {
			toNr = fromNr + LimitDagHashes
		}
		log.Debug("Sync handling: case 2.0",
			"baseSpine", query.BaseSpine.Hex(),
			"terminalSpine", query.TerminalSpine.Hex(),
			"isBaseFinalized", isBaseFinalized,
			"isTerminalFinalized", isTerminalFinalized,
			"fromNr", fromNr,
			"toNr", toNr,
		)
		dag = getHashesByNumberRange(backend, fromNr, toNr, LimitDagHashes)
		limitReached = len(dag) >= LimitDagHashes

		//2. if data less than LimitDagHashes
		// - collect not finalized part (by slots)
		if len(dag) < LimitDagHashes {
			fromSlot := lfHeader.Slot
			toSlot := terminalHeader.Slot
			// expecting that lfHeader.Hash will be received twice
			limit := LimitDagHashes - len(dag) + 1
			log.Debug("Sync handling: case 2.1",
				"baseSpine", query.BaseSpine.Hex(),
				"terminalSpine", query.TerminalSpine.Hex(),
				"isBaseFinalized", isBaseFinalized,
				"isTerminalFinalized", isTerminalFinalized,
				"fromSlot", fromSlot,
				"toSlot", toSlot,
				"limit", limit,
			)
			dagHashes, limitReached = getHashesBySlotRange(backend, fromSlot, toSlot, limit, query.TerminalSpine)
			dag = append(dag, dagHashes...)
			dag.Deduplicate()
		}

	case !isBaseFinalized && !isTerminalFinalized:
		//	collect not finalized part (by slots) only
		fromSlot := baseHeader.Slot
		toSlot := terminalHeader.Slot
		if fromSlot > toSlot {
			return dag, errBadRequestParam
		}
		// expecting that baseHash will be received too
		limit := LimitDagHashes + 1
		log.Debug("Sync handling: case 3",
			"baseSpine", query.BaseSpine.Hex(),
			"terminalSpine", query.TerminalSpine.Hex(),
			"isBaseFinalized", isBaseFinalized,
			"isTerminalFinalized", isTerminalFinalized,
			"fromSlot", fromSlot,
			"toSlot", toSlot,
			"limit", limit,
		)
		dagHashes, limitReached = getHashesBySlotRange(backend, fromSlot, toSlot, limit, query.TerminalSpine)
		if i := dagHashes.IndexOf(baseHeader.Hash()); i >= 0 {
			dag = append(dagHashes[:i], dagHashes[i+1:]...)
		} else {
			dag = dagHashes
		}
	default:
		log.Error("Sync handling: case default",
			"baseSpine", query.BaseSpine.Hex(),
			"terminalSpine", query.TerminalSpine.Hex(),
			"isBaseFinalized", isBaseFinalized,
			"isTerminalFinalized", isTerminalFinalized,
		)
		return dag, errBadRequestParam
	}

	log.Debug("Sync handling: dag retrieved",
		"baseSpine", query.BaseSpine.Hex(),
		"terminalSpine", query.TerminalSpine.Hex(),
		"isBaseFinalized", isBaseFinalized,
		"isTerminalFinalized", isTerminalFinalized,
		"limitReached", limitReached,
		"dag", dag,
	)
	// if requested all not-finalized blocks and limit not reached
	// - add zero hash
	if query.TerminalSpine == (common.Hash{}) && !limitReached {
		dag = append(dag, common.Hash{})
	}
	// if limit is exceeded - cut off right side
	if len(dag) > LimitDagHashes {
		dag = dag[:LimitDagHashes]
	}
	log.Debug("Sync handling: dag response",
		"baseSpine", query.BaseSpine.Hex(),
		"terminalSpine", query.TerminalSpine.Hex(),
		"isBaseFinalized", isBaseFinalized,
		"isTerminalFinalized", isTerminalFinalized,
		"limitReached", limitReached,
		"dag", dag,
	)
	return dag, nil
}

func getHashesByNumberRange(backend Backend, fromNr, toNr uint64, limit int) common.HashArray {
	dag := common.HashArray{}
	if limit <= 0 {
		return dag
	}
	if fromNr > toNr {
		return dag
	}
	//check data limitation
	if toNr-fromNr > uint64(limit) {
		toNr = fromNr + uint64(limit)
	}
	for nr := uint64(1); fromNr+nr <= toNr; nr++ {
		finHash := backend.Chain().ReadFinalizedHashByNumber(fromNr + nr)
		if finHash != (common.Hash{}) {
			dag = append(dag, finHash)
		}
	}
	return dag
}

func getHashesBySlotRange(backend Backend, from, to uint64, limit int, terminalHash common.Hash) (common.HashArray, bool) {
	if limit <= 0 {
		return common.HashArray{}, true
	}
	var limitReached bool
	si := backend.Chain().GetSlotInfo()
	hashes := make(common.HashArray, 0, limit)
	for slot := from; slot < to; slot++ {
		slh := backend.Chain().GetBlockHashesBySlot(slot)
		if len(slh) > 0 {
			// if received terminalHash
			// - add terminalHash only and stop
			if slh.Has(terminalHash) {
				hashes = append(hashes, terminalHash)
				break
			}
			// if limit reached
			// - add the first only and stop
			if len(hashes)+len(slh) >= limit {
				hashes = append(hashes, slh[0])
				limitReached = true
				break
			}
			hashes = append(hashes, slh...)
		}
		if si != nil && slot >= si.CurrentSlot() {
			break
		}
	}
	return hashes, limitReached
}

func handleDag66(backend Backend, msg Decoder, peer *Peer) error {
	// A dag hashes arrived to one of our previous requests
	res := new(DagPacket66)
	if err := msg.Decode(res); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	requestTracker.Fulfil(peer.id, peer.version, DagMsg, res.RequestId)
	return backend.Handle(peer, &res.DagPacket)
}

func handleGetHashesBySlots66(backend Backend, msg Decoder, peer *Peer) error {
	// Decode retrieval message
	var query GetHashesBySlotsPacket66
	if err := msg.Decode(&query); err != nil {
		return fmt.Errorf("%w: message %v: %v", errDecode, msg, err)
	}
	//Gather hashes by slots.
	from, to := query.From, query.To
	maxHashLen := (to - from) * backend.Chain().Config().ValidatorsPerSlot
	// check length limit
	if maxHashLen > LimitDagHashes {
		return fmt.Errorf("%w: %v > %v (get hashes by slots)", errMsgTooLarge, maxHashLen, LimitDagHashes)
	}
	hashes := make(common.HashArray, 0, maxHashLen)
	si := backend.Chain().GetSlotInfo()
	for slot := from; slot < to; slot++ {
		slh := backend.Chain().GetBlockHashesBySlot(slot)
		if len(slh) > 0 {
			hashes = append(hashes, slh...)
		}
		if si != nil && slot >= si.CurrentSlot() {
			break
		}
	}
	return peer.ReplyDagData(query.RequestId, hashes.Uniq())
}
