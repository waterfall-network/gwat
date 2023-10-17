// Copyright 2018 The go-ethereum Authors
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

package rawdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

// ReadAllCanonicalHashes retrieves all canonical number and hash mappings at the
// certain chain range. If the accumulated entries reaches the given threshold,
// abort the iteration and return the semi-finish result.
func ReadAllCanonicalHashes(db ethdb.Iteratee, from uint64, to uint64, limit int) ([]uint64, []common.Hash) {
	// Short circuit if the limit is 0.
	if limit == 0 {
		return nil, nil
	}
	var (
		numbers []uint64
		hashes  []common.Hash
	)
	// Construct the key prefix of start point.
	start, end := finHashByNumberKey(from), finHashByNumberKey(to)
	it := db.NewIterator(nil, start)
	defer it.Release()

	for it.Next() {
		if bytes.Compare(it.Key(), end) >= 0 {
			break
		}
		if key := it.Key(); len(key) == len(headerPrefix)+8+1 && bytes.Equal(key[len(key)-1:], headerHashSuffix) {
			numbers = append(numbers, binary.BigEndian.Uint64(key[len(headerPrefix):len(headerPrefix)+8]))
			hashes = append(hashes, common.BytesToHash(it.Value()))
			// If the accumulated entries reaches the limit threshold, return.
			if len(numbers) >= limit {
				break
			}
		}
	}
	return numbers, hashes
}

// ReadLastCanonicalHash retrieves the hash of the current canonical head block.
func ReadLastCanonicalHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(lastCanonicalHashKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteLastCanonicalHash stores the last canonical block's hash.
func WriteLastCanonicalHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(lastCanonicalHashKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last canonical block's hash", "err", err)
	}
}

// ReadHeadFastBlockHash retrieves the hash of the current fast-sync head block.
func ReadHeadFastBlockHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(headFastBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// WriteHeadFastBlockHash stores the hash of the current fast-sync head block.
func WriteHeadFastBlockHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(headFastBlockKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last fast block's hash", "err", err)
	}
}

// ReadLastPivotNumber retrieves the number of the last pivot block. If the node
// full synced, the last pivot will always be nil.
func ReadLastPivotNumber(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(lastPivotKey)
	if len(data) == 0 {
		return nil
	}
	var pivot uint64
	if err := rlp.DecodeBytes(data, &pivot); err != nil {
		log.Error("Invalid pivot block number in database", "err", err)
		return nil
	}
	return &pivot
}

// WriteLastPivotNumber stores the number of the last pivot block.
func WriteLastPivotNumber(db ethdb.KeyValueWriter, pivot uint64) {
	enc, err := rlp.EncodeToBytes(pivot)
	if err != nil {
		log.Crit("Failed to encode pivot block number", "err", err)
	}
	if err := db.Put(lastPivotKey, enc); err != nil {
		log.Crit("Failed to store pivot block number", "err", err)
	}
}

// ReadFastTrieProgress retrieves the number of tries nodes fast synced to allow
// reporting correct numbers across restarts.
func ReadFastTrieProgress(db ethdb.KeyValueReader) uint64 {
	data, _ := db.Get(fastTrieProgressKey)
	if len(data) == 0 {
		return 0
	}
	return new(big.Int).SetBytes(data).Uint64()
}

// WriteFastTrieProgress stores the fast sync trie process counter to support
// retrieving it across restarts.
func WriteFastTrieProgress(db ethdb.KeyValueWriter, count uint64) {
	if err := db.Put(fastTrieProgressKey, new(big.Int).SetUint64(count).Bytes()); err != nil {
		log.Crit("Failed to store fast sync trie progress", "err", err)
	}
}

// ReadTxIndexTail retrieves the number of oldest indexed block
// whose transaction indices has been indexed. If the corresponding entry
// is non-existent in database it means the indexing has been finished.
func ReadTxIndexTail(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(txIndexTailKey)
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteTxIndexTail stores the number of oldest indexed block
// into database.
func WriteTxIndexTail(db ethdb.KeyValueWriter, number uint64) {
	if err := db.Put(txIndexTailKey, encodeBlockNumber(number)); err != nil {
		log.Crit("Failed to store the transaction index tail", "err", err)
	}
}

// ReadFastTxLookupLimit retrieves the tx lookup limit used in fast sync.
func ReadFastTxLookupLimit(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(fastTxLookupLimitKey)
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteFastTxLookupLimit stores the txlookup limit used in fast sync into database.
func WriteFastTxLookupLimit(db ethdb.KeyValueWriter, number uint64) {
	if err := db.Put(fastTxLookupLimitKey, encodeBlockNumber(number)); err != nil {
		log.Crit("Failed to store transaction lookup limit for fast sync", "err", err)
	}
}

// ReadHeaderRLP retrieves a block header in its raw RLP database encoding.
func ReadHeaderRLP(db ethdb.Reader, hash common.Hash) rlp.RawValue {
	var data []byte
	number := ReadFinalizedNumberByHash(db, hash)
	// First try to look up the data in ancient database. Extra hash
	// comparison is necessary since ancient database only maintains
	// the canonical data.
	if number != nil {
		data, _ = db.Ancient(freezerHeaderTable, *number)
		//if len(data) > 0 && crypto.Keccak256Hash(data) == hash {
		if len(data) > 0 {
			return data
		}
	}
	// Then try to look up the data in leveldb.
	data, _ = db.Get(headerKey(hash))
	if len(data) > 0 {
		return data
	}
	//// In the background freezer is moving data from leveldb to flatten files.
	//// So during the first check for ancient db, the data is not yet in there,
	//// but when we reach into leveldb, the data was already moved. That would
	//// result in a not found error.
	if number != nil {
		data, _ = db.Ancient(freezerHeaderTable, *number)
		if len(data) > 0 && crypto.Keccak256Hash(data) == hash {
			return data
		}
	}
	return nil // Can't find the data anywhere.
}

// HasHeader verifies the existence of a block header corresponding to the hash.
func HasHeader(db ethdb.Reader, hash common.Hash) bool {
	number := ReadFinalizedNumberByHash(db, hash)
	if number != nil {
		if has, err := db.Ancient(freezerHashTable, *number); err == nil && common.BytesToHash(has) == hash {
			return true
		}
	}
	if has, err := db.Has(headerKey(hash)); !has || err != nil {
		return false
	}
	return true
}

// ReadHeader retrieves the block header corresponding to the hash.
func ReadHeader(db ethdb.Reader, hash common.Hash) *types.Header {
	data := ReadHeaderRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	if header.Hash() != hash {
		log.Error("Invalid retrieved header's hash", "hash", header.Hash().Hex(), "expected", hash.Hex())
		return nil
	}
	if nr := ReadFinalizedNumberByHash(db, hash); nr != nil {
		header.Number = nr
	}
	return header
}

// WriteHeader stores a block header into the database and also stores the hash-
// to-number mapping.
func WriteHeader(db ethdb.KeyValueWriter, header *types.Header) {
	var (
		hash = header.Hash()
	)
	// Write the hash -> number mapping
	header.Number = nil
	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		log.Crit("Failed to RLP encode header", "err", err)
	}
	key := headerKey(hash)
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
}

// DeleteHeader removes all block header data associated with a hash.
func DeleteHeader(db ethdb.KeyValueWriter, hash common.Hash, finNr *uint64) {
	deleteHeaderWithoutNumber(db, hash)
	if finNr != nil {
		DeleteFinalizedHashNumber(db, hash, *finNr)
	}
}

// deleteHeaderWithoutNumber removes only the block header but does not remove
// the hash to number mapping.
func deleteHeaderWithoutNumber(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(headerKey(hash)); err != nil {
		log.Crit("Failed to delete header", "err", err)
	}
}

// ReadBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func ReadBodyRLP(db ethdb.Reader, hash common.Hash) rlp.RawValue {
	var data []byte
	number := ReadFinalizedNumberByHash(db, hash)
	// First try to look up the data in ancient database. Extra hash
	// comparison is necessary since ancient database only maintains
	// the canonical data.
	if number != nil {
		data, _ = db.Ancient(freezerBodiesTable, *number)
		if len(data) > 0 {
			h, _ := db.Ancient(freezerHashTable, *number)
			if common.BytesToHash(h) == hash {
				return data
			}
		}
	}
	// Then try to look up the data in leveldb.
	data, _ = db.Get(blockBodyKey(hash))
	if len(data) > 0 {
		return data
	}
	// In the background freezer is moving data from leveldb to flatten files.
	// So during the first check for ancient db, the data is not yet in there,
	// but when we reach into leveldb, the data was already moved. That would
	// result in a not found error.
	if number != nil {
		data, _ = db.Ancient(freezerBodiesTable, *number)
		if len(data) > 0 {
			h, _ := db.Ancient(freezerHashTable, *number)
			if common.BytesToHash(h) == hash {
				return data
			}
		}
	}
	return nil // Can't find the data anywhere.
}

// ReadCanonicalBodyRLP retrieves the block body (transactions and uncles) for the canonical
// block at number, in RLP encoding.
func ReadCanonicalBodyRLP(db ethdb.Reader, number uint64) rlp.RawValue {
	// If it's an ancient one, we don't need the canonical hash
	data, _ := db.Ancient(freezerBodiesTable, number)
	if len(data) == 0 {
		// Need to get the hash
		data, _ = db.Get(blockBodyKey(ReadFinalizedHashByNumber(db, number)))
		// In the background freezer is moving data from leveldb to flatten files.
		// So during the first check for ancient db, the data is not yet in there,
		// but when we reach into leveldb, the data was already moved. That would
		// result in a not found error.
		if len(data) == 0 {
			data, _ = db.Ancient(freezerBodiesTable, number)
		}
	}
	return data
}

// WriteBodyRLP stores an RLP encoded block body into the database.
func WriteBodyRLP(db ethdb.KeyValueWriter, hash common.Hash, rlp rlp.RawValue) {
	if err := db.Put(blockBodyKey(hash), rlp); err != nil {
		log.Crit("Failed to store block body", "err", err)
	}
}

// HasBody verifies the existence of a block body corresponding to the hash.
func HasBody(db ethdb.Reader, hash common.Hash) bool {
	number := ReadFinalizedNumberByHash(db, hash)
	if number != nil {
		if has, err := db.Ancient(freezerHashTable, *number); err == nil && common.BytesToHash(has) == hash {
			return true
		}
	}
	if has, err := db.Has(blockBodyKey(hash)); !has || err != nil {
		return false
	}
	return true
}

// ReadBody retrieves the block body corresponding to the hash.
func ReadBody(db ethdb.Reader, hash common.Hash) *types.Body {
	data := ReadBodyRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		log.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

// WriteBody stores a block body into the database.
func WriteBody(db ethdb.KeyValueWriter, hash common.Hash, body *types.Body) {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		log.Crit("Failed to RLP encode body", "err", err)
	}
	WriteBodyRLP(db, hash, data)
}

// DeleteBody removes all block body data associated with a hash.
func DeleteBody(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(blockBodyKey(hash)); err != nil {
		log.Crit("Failed to delete block body", "err", err)
	}
}

// HasReceipts verifies the existence of all the transaction receipts belonging
// to a block.
func HasReceipts(db ethdb.Reader, hash common.Hash) bool {
	number := ReadFinalizedNumberByHash(db, hash)
	if number != nil {
		if has, err := db.Ancient(freezerHashTable, *number); err == nil && common.BytesToHash(has) == hash {
			return true
		}
	}
	if has, err := db.Has(blockReceiptsKey(hash)); !has || err != nil {
		return false
	}
	return true
}

// ReadReceiptsRLP retrieves all the transaction receipts belonging to a block in RLP encoding.
func ReadReceiptsRLP(db ethdb.Reader, hash common.Hash) rlp.RawValue {
	var data []byte
	number := ReadFinalizedNumberByHash(db, hash)
	// First try to look up the data in ancient database. Extra hash
	// comparison is necessary since ancient database only maintains
	// the canonical data.
	if number != nil {
		data, _ = db.Ancient(freezerReceiptTable, *number)
		if len(data) > 0 {
			h, _ := db.Ancient(freezerHashTable, *number)
			if common.BytesToHash(h) == hash {
				return data
			}
		}
	}
	// Then try to look up the data in leveldb.
	data, _ = db.Get(blockReceiptsKey(hash))
	if len(data) > 0 {
		return data
	}
	// In the background freezer is moving data from leveldb to flatten files.
	// So during the first check for ancient db, the data is not yet in there,
	// but when we reach into leveldb, the data was already moved. That would
	// result in a not found error.
	if number != nil {
		data, _ = db.Ancient(freezerReceiptTable, *number)
		if len(data) > 0 {
			h, _ := db.Ancient(freezerHashTable, *number)
			if common.BytesToHash(h) == hash {
				return data
			}
		}
	}
	return nil // Can't find the data anywhere.
}

// ReadRawReceipts retrieves all the transaction receipts belonging to a block.
// The receipt metadata fields are not guaranteed to be populated, so they
// should not be used. Use ReadReceipts instead if the metadata is needed.
func ReadRawReceipts(db ethdb.Reader, hash common.Hash) types.Receipts {
	// Retrieve the flattened receipt slice
	data := ReadReceiptsRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	// Convert the receipts from their storage form to their internal representation
	storageReceipts := []*types.ReceiptForStorage{}
	if err := rlp.DecodeBytes(data, &storageReceipts); err != nil {
		log.Error("Invalid receipt array RLP", "hash", hash, "err", err)
		return nil
	}
	receipts := make(types.Receipts, len(storageReceipts))
	for i, storageReceipt := range storageReceipts {
		receipts[i] = (*types.Receipt)(storageReceipt)
	}
	return receipts
}

// ReadReceipts retrieves all the transaction receipts belonging to a block, including
// its correspoinding metadata fields. If it is unable to populate these metadata
// fields then nil is returned.
//
// The current implementation populates these metadata fields by reading the receipts'
// corresponding block body, so if the block body is not found it will return nil even
// if the receipt itself is stored.
func ReadReceipts(db ethdb.Reader, hash common.Hash, config *params.ChainConfig) types.Receipts {
	// We're deriving many fields from the block body, retrieve beside the receipt
	receipts := ReadRawReceipts(db, hash)
	if receipts == nil {
		return nil
	}
	body := ReadBody(db, hash)
	if body == nil {
		log.Error("Missing body but have receipt", "hash", hash)
		return nil
	}
	number := uint64(0)
	if nr := ReadFinalizedNumberByHash(db, hash); nr != nil {
		number = *nr
	}
	if err := receipts.DeriveFields(config, hash, number, body.Transactions); err != nil {
		log.Error("Failed to derive block receipts fields", "hash", hash, "number", number, "err", err)
		return nil
	}
	return receipts
}

// WriteReceipts stores all the transaction receipts belonging to a block.
func WriteReceipts(db ethdb.KeyValueWriter, hash common.Hash, receipts types.Receipts) {
	// Convert the receipts into their storage form and serialize them
	storageReceipts := make([]*types.ReceiptForStorage, len(receipts))
	for i, receipt := range receipts {
		if receipt == nil {
			receipt = &types.Receipt{}
		}
		storageReceipts[i] = (*types.ReceiptForStorage)(receipt)
	}
	bytes, err := rlp.EncodeToBytes(storageReceipts)
	if err != nil {
		log.Crit("Failed to encode block receipts", "err", err)
	}
	// Store the flattened receipt slice
	if err := db.Put(blockReceiptsKey(hash), bytes); err != nil {
		log.Crit("Failed to store block receipts", "err", err)
	}
}

// DeleteReceipts removes all receipt data associated with a block hash.
func DeleteReceipts(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(blockReceiptsKey(hash)); err != nil {
		log.Crit("Failed to delete block receipts", "err", err)
	}
}

// storedReceiptRLP is the storage encoding of a receipt.
// Re-definition in core/types/receipt.go.
type storedReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Logs              []*types.LogForStorage
}

// ReceiptLogs is a barebone version of ReceiptForStorage which only keeps
// the list of logs. When decoding a stored receipt into this object we
// avoid creating the bloom filter.
type receiptLogs struct {
	Logs []*types.Log
}

// DecodeRLP implements rlp.Decoder.
func (r *receiptLogs) DecodeRLP(s *rlp.Stream) error {
	var stored storedReceiptRLP
	if err := s.Decode(&stored); err != nil {
		return err
	}
	r.Logs = make([]*types.Log, len(stored.Logs))
	for i, log := range stored.Logs {
		r.Logs[i] = (*types.Log)(log)
	}
	return nil
}

// deriveLogFields fills the logs in receiptLogs with information such as block number, txhash, etc.
func deriveLogFields(receipts []*receiptLogs, hash common.Hash, number uint64, txs types.Transactions) error {
	logIndex := uint(0)
	if len(txs) != len(receipts) {
		return errors.New("transaction and receipt count mismatch")
	}
	for i := 0; i < len(receipts); i++ {
		txHash := txs[i].Hash()
		// The derived log fields can simply be set from the block and transaction
		for j := 0; j < len(receipts[i].Logs); j++ {
			receipts[i].Logs[j].BlockNumber = number
			receipts[i].Logs[j].BlockHash = hash
			receipts[i].Logs[j].TxHash = txHash
			receipts[i].Logs[j].TxIndex = uint(i)
			receipts[i].Logs[j].Index = logIndex
			logIndex++
		}
	}
	return nil
}

// ReadLogs retrieves the logs for all transactions in a block. The log fields
// are populated with metadata. In case the receipts or the block body
// are not found, a nil is returned.
func ReadLogs(db ethdb.Reader, hash common.Hash, number uint64) [][]*types.Log {
	// Retrieve the flattened receipt slice
	data := ReadReceiptsRLP(db, hash)
	if len(data) == 0 {
		return nil
	}
	receipts := []*receiptLogs{}
	if err := rlp.DecodeBytes(data, &receipts); err != nil {
		log.Error("Invalid receipt array RLP", "hash", hash, "err", err)
		return nil
	}

	body := ReadBody(db, hash)
	if body == nil {
		log.Error("Missing body but have receipt", "hash", hash, "number", number)
		return nil
	}
	if err := deriveLogFields(receipts, hash, number, body.Transactions); err != nil {
		log.Error("Failed to derive block receipts fields", "hash", hash, "number", number, "err", err)
		return nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs
}

// ReadBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body. If either the header or body could not
// be retrieved nil is returned.
//
// Note, due to concurrent download of header and block body the header and thus
// canonical hash can be stored in the database but the body data not (yet).
func ReadBlock(db ethdb.Reader, hash common.Hash) *types.Block {
	header := ReadHeader(db, hash)
	if header == nil {
		return nil
	}
	body := ReadBody(db, hash)
	if body == nil {
		return nil
	}
	return types.NewBlockWithHeader(header).WithBody(body.Transactions)
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db ethdb.KeyValueWriter, block *types.Block) {
	WriteBody(db, block.Hash(), block.Body())
	WriteHeader(db, block.Header())
}

// WriteAncientBlock writes entire block data into ancient store and returns the total written size.
func WriteAncientBlocks(db ethdb.AncientWriter, blocks []*types.Block, receipts []types.Receipts) (int64, error) {
	var (
		stReceipts []*types.ReceiptForStorage
	)
	return db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i, block := range blocks {
			// Convert receipts to storage format and sum up total difficulty.
			stReceipts = stReceipts[:0]
			for _, receipt := range receipts[i] {
				stReceipts = append(stReceipts, (*types.ReceiptForStorage)(receipt))
			}
			header := block.Header()
			if err := writeAncientBlock(op, block, header, stReceipts); err != nil {
				return err
			}
		}
		return nil
	})
}

func writeAncientBlock(op ethdb.AncientWriteOp, block *types.Block, header *types.Header, receipts []*types.ReceiptForStorage) error {
	if block.Number() == nil {
		return fmt.Errorf("block finalized number is not defined (hash=%v)", block.Hash().Hex())
	}
	num := block.Nr()
	if err := op.AppendRaw(freezerHashTable, num, block.Hash().Bytes()); err != nil {
		return fmt.Errorf("can't add block %d hash: %v", num, err)
	}
	if err := op.Append(freezerHeaderTable, num, header); err != nil {
		return fmt.Errorf("can't append block header %d: %v", num, err)
	}
	if err := op.Append(freezerBodiesTable, num, block.Body()); err != nil {
		return fmt.Errorf("can't append block body %d: %v", num, err)
	}
	if err := op.Append(freezerReceiptTable, num, receipts); err != nil {
		return fmt.Errorf("can't append block %d receipts: %v", num, err)
	}
	return nil
}

// DeleteBlock removes all block data associated with a hash.
func DeleteBlock(db ethdb.KeyValueWriter, hash common.Hash, finNr *uint64) {
	DeleteReceipts(db, hash)
	DeleteHeader(db, hash, finNr)
	DeleteBody(db, hash)
	DeleteChildren(db, hash)
	DeleteBlockDag(db, hash)
}

// DeleteBlockWithoutNumber removes all block data associated with a hash, except
// the hash to number mapping.
func DeleteBlockWithoutNumber(db ethdb.KeyValueWriter, hash common.Hash) {
	DeleteReceipts(db, hash)
	deleteHeaderWithoutNumber(db, hash)
	DeleteBody(db, hash)
	DeleteChildren(db, hash)
	DeleteBlockDag(db, hash)
}

// FindCommonAncestor returns the last common ancestor of two block headers
func FindCommonAncestor(db ethdb.Reader, a, b *types.Header) *types.Header {
	if (a.Nr() == 0 && a.Height > 0) || (b.Nr() == 0 && b.Height > 0) {
		panic("FindCommonAncestor: no implementation for dag part: core/rawdb/accessors_chain.go:782")
	}

	for bn := b.Nr(); a.Nr() > bn; {
		prevHash := ReadFinalizedHashByNumber(db, a.Nr()-1)
		a = ReadHeader(db, prevHash)
		if a == nil {
			return nil
		}
	}
	for an := a.Nr(); an < b.Nr(); {
		prevHash := ReadFinalizedHashByNumber(db, b.Nr()-1)
		b = ReadHeader(db, prevHash)
		if b == nil {
			return nil
		}
	}

	for a.Hash() != b.Hash() {
		prevHashA := ReadFinalizedHashByNumber(db, a.Nr()-1)
		a = ReadHeader(db, prevHashA)
		if a == nil {
			return nil
		}
		prevHashB := ReadFinalizedHashByNumber(db, b.Nr()-1)
		b = ReadHeader(db, prevHashB)
		if b == nil {
			return nil
		}
	}
	return a
}

/**** FINALIZED BLOCK DATA ***/

// ReadFinalizedNumberByHash retrieves block finalization number by hash.
func ReadFinalizedNumberByHash(db ethdb.KeyValueReader, hash common.Hash) *uint64 {
	data, _ := db.Get(finNumberByHashKey(hash))
	if len(data) != 8 {
		return nil
	}
	height := binary.BigEndian.Uint64(data)
	return &height
}

// _writeFinalizedNumberByHash stores the finalised blocks' hash->height mapping.
func _writeFinalizedNumberByHash(db ethdb.KeyValueWriter, hash common.Hash, height uint64) {
	key := finNumberByHashKey(hash)
	enc := encodeBlockNumber(height)
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store hash to height mapping", "err", err)
	}
}

// _deleteFinalizedNumberByHash rm the finalised blocks' hash->height mapping.
func _deleteFinalizedNumberByHash(db ethdb.KeyValueWriter, hash common.Hash) {
	key := finNumberByHashKey(hash)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete Finalized Number By Hash", "err", err)
	}
}

// ReadFinalizedHashByNumber retrieves block finalization hash by number.
func ReadFinalizedHashByNumber(db ethdb.KeyValueReader, number uint64) common.Hash {
	data, _ := db.Get(finHashByNumberKey(number))
	return common.BytesToHash(data)
}

// _writeFinalizedHashByNumber stores the finalised blocks' height->hash mapping.
func _writeFinalizedHashByNumber(db ethdb.KeyValueWriter, number uint64, hash common.Hash) {
	key := finHashByNumberKey(number)
	enc := hash.Bytes()
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store hash to number mapping", "err", err)
	}
}

// _deleteFinalizedHashByNumber removes the finalised blocks' height->hash mapping.
func _deleteFinalizedHashByNumber(db ethdb.KeyValueWriter, number uint64) {
	key := finHashByNumberKey(number)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete Finalized Hash By Number", "err", err)
	}
}

// WriteFinalizedHashNumber writes of block finalization data.
func WriteFinalizedHashNumber(db ethdb.KeyValueWriter, hash common.Hash, number uint64) {
	_writeFinalizedNumberByHash(db, hash, number)
	_writeFinalizedHashByNumber(db, number, hash)
}

// DeleteFinalizedHashNumber removes all block finalization data.
func DeleteFinalizedHashNumber(db ethdb.KeyValueWriter, hash common.Hash, number uint64) {
	_deleteFinalizedNumberByHash(db, hash)
	_deleteFinalizedHashByNumber(db, number)
}

/**** LAST FINALIZED BLOCK ***/

// ReadLastFinalizedHash retrieves the hash of the last finalized block.
func ReadLastFinalizedHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(lastFinalizedHashKey)
	return common.BytesToHash(data)
}

// WriteLastFinalizedHash stores the hash of the last finalized block
func WriteLastFinalizedHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(lastFinalizedHashKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store last Finalized Hash", "err", err)
	}
}

// ReadLastFinalizedNumber retrieves the height of the last finalized block.
func ReadLastFinalizedNumber(db ethdb.KeyValueReader) uint64 {
	hash := ReadLastFinalizedHash(db)
	if hash == (common.Hash{}) {
		return uint64(0)
	}
	nr := ReadFinalizedNumberByHash(db, hash)
	if nr == nil {
		return uint64(0)
	}
	return *nr
}

/**** Coordinated state ***/

// ReadLastCoordinatedCheckpoint retrieves the last Coordinated checkpoint.
func ReadLastCoordinatedCheckpoint(db ethdb.KeyValueReader) *types.Checkpoint {
	data, err := db.Get(lastCoordCpKey)
	if err != nil {
		return nil
	}
	cp, err := types.BytesToCheckpoint(data)
	if err != nil {
		return nil
	}
	return cp
}

// WriteLastCoordinatedCheckpoint stores the last Coordinated checkpoint.
func WriteLastCoordinatedCheckpoint(db ethdb.KeyValueWriter, checkpoint *types.Checkpoint) {
	if checkpoint == nil {
		DeleteLastCoordinatedCheckpoint(db)
	}
	if err := db.Put(lastCoordCpKey, checkpoint.Bytes()); err != nil {
		log.Crit("Failed to store the Last Coordinated Checkpoint", "err", err)
	}
}

// DeleteLastCoordinatedCheckpoint rm the last Coordinated checkpoint.
func DeleteLastCoordinatedCheckpoint(db ethdb.KeyValueWriter) {
	if err := db.Delete(lastCoordCpKey); err != nil {
		log.Warn("Failed to delete Last Coordinated Checkpoint", "err", err)
	}
}

// ReadCoordinatedCheckpoint retrieves the Coordinated checkpoint by checkpoint spine.
func ReadCoordinatedCheckpoint(db ethdb.KeyValueReader, cpSpine common.Hash) *types.Checkpoint {
	key := coordCpKey(cpSpine)
	data, err := db.Get(key)
	if err != nil {
		return nil
	}
	cp, err := types.BytesToCheckpoint(data)
	if err != nil {
		return nil
	}
	return cp
}

// WriteCoordinatedCheckpoint writes a Coordinated Checkpoint to a key-value database.
func WriteCoordinatedCheckpoint(db ethdb.KeyValueWriter, checkpoint *types.Checkpoint) {
	key := coordCpKey(checkpoint.Spine)

	if err := db.Put(key, checkpoint.Bytes()); err != nil {
		log.Crit("Failed to store the coordinated checkpoint", "err", err, "spine", checkpoint.Spine.Hex())
	}
}

// DeleteCoordinatedCheckpoint delete the coordinated checkpoint data.
func DeleteCoordinatedCheckpoint(db ethdb.KeyValueWriter, cpSpine common.Hash) {
	key := coordCpKey(cpSpine)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete coordinated checkpoint", "err", err, "spine", cpSpine.Hex())
	}
}

// ReadEpoch retrieves the checkpoint spine by epoch.
func ReadEpoch(db ethdb.KeyValueReader, epoch uint64) common.Hash {
	data, _ := db.Get(epochCpKey(epoch))
	return common.BytesToHash(data)
}

// WriteEpoch writes an epoch checkpoint spine to a key-value database.
func WriteEpoch(db ethdb.KeyValueWriter, epoch uint64, cpSpine common.Hash) {
	key := epochCpKey(epoch)
	if err := db.Put(key, cpSpine.Bytes()); err != nil {
		log.Crit("Failed to store the epoch spine", "err", err, "epoch", epoch, "cpSpine", cpSpine)
	}
}

// DeleteEpoch removes the epoch's checkpoint spine.
func DeleteEpoch(db ethdb.KeyValueWriter, epoch uint64) {
	key := epochCpKey(epoch)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete the epoch spine", "err", err, "epoch", epoch)
	}
}

/**** ValidatorSync ***/

func parseValidatorSyncKey(validatorSyncKey []byte) (initTxHash common.Hash) {
	start := len(valSyncOpPrefix)
	end := start + common.HashLength
	initTxHash = common.BytesToHash(validatorSyncKey[start:end])
	return initTxHash
}

func decodeValidatorSync(initTxHash common.Hash, data []byte) *types.ValidatorSync {
	//InitTxHash common.Hash - parse from key
	// <OpType><Creator><index><procEpoch><txHash><amountBigInt>`
	minSize := 76
	if len(data) < minSize {
		return nil
	}
	res := &types.ValidatorSync{
		InitTxHash: initTxHash,
		OpType:     types.ValidatorSyncOp(binary.BigEndian.Uint64(data[0:8])),
		Creator:    common.BytesToAddress(data[8:28]),
		Index:      binary.BigEndian.Uint64(data[28:36]),
		ProcEpoch:  binary.BigEndian.Uint64(data[36:44]),
		TxHash:     nil,
		Amount:     nil,
	}
	txh := common.BytesToHash(data[44:76])
	if txh != (common.Hash{}) {
		res.TxHash = &txh
	}
	if len(data[minSize:]) > 0 {
		res.Amount = new(big.Int).SetBytes(data[minSize:])
	}
	return res
}

func encodeValidatorSync(vs types.ValidatorSync) []byte {
	//InitTxHash common.Hash - put in key
	// <OpType><Creator><index><procEpoch><txHash><amountBigInt>`
	var data []byte
	data = append(data, encodeBlockNumber(uint64(vs.OpType))...)
	data = append(data, vs.Creator.Bytes()...)
	data = append(data, encodeBlockNumber(vs.Index)...)
	data = append(data, encodeBlockNumber(vs.ProcEpoch)...)
	txh := vs.TxHash
	if txh == nil {
		txh = &common.Hash{}
	}
	data = append(data, txh.Bytes()...)
	if vs.Amount != nil {
		data = append(data, vs.Amount.Bytes()...)
	}
	return data
}

// ReadValidatorSync retrieves the ValidatorSync data.
func ReadValidatorSync(db ethdb.KeyValueReader, initTxHash common.Hash) *types.ValidatorSync {
	data, _ := db.Get(validatorSyncKey(initTxHash))
	return decodeValidatorSync(initTxHash, data)
}

// WriteValidatorSync stores the ValidatorSync data.
func WriteValidatorSync(db ethdb.KeyValueWriter, vs *types.ValidatorSync) {
	if vs == nil {
		return
	}
	key := validatorSyncKey(vs.InitTxHash)
	enc := encodeValidatorSync(*vs)
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store ValidatorSync data", "err", err)
	}
}

// DeleteValidatorSync delete the ValidatorSync data..
func DeleteValidatorSync(db ethdb.KeyValueWriter, initTxHash common.Hash) {
	key := validatorSyncKey(initTxHash)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete validators sync data", "err", err)
	}
}

// ReadNotProcessedValidatorSyncOps retrieves the not processed validator sync operations.
func ReadNotProcessedValidatorSyncOps(db ethdb.KeyValueReader) []*types.ValidatorSync {
	data, err := db.Get(valSyncNotProcKey)
	if err != nil {
		return nil
	}
	keyLen := len(validatorSyncKey(common.Hash{}))
	if len(data)%keyLen != 0 {
		// alternate return nil
		log.Crit("Failed to read the not processed validator sync operations: bad data length", "err", err)
	}
	resLen := len(data) / keyLen
	res := make([]*types.ValidatorSync, resLen)
	for i := range res {
		start := i * keyLen
		end := start + keyLen
		opKey := data[start:end]
		initTxHash := parseValidatorSyncKey(opKey)
		res[i] = ReadValidatorSync(db, initTxHash)
	}
	return res
}

// WriteNotProcessedValidatorSyncOps stores the not processed validator sync operations.
func WriteNotProcessedValidatorSyncOps(db ethdb.KeyValueWriter, valSyncOps []*types.ValidatorSync) {
	keyLen := len(validatorSyncKey(common.Hash{}))
	dataLen := keyLen * len(valSyncOps)
	data := make([]byte, 0, dataLen)
	for _, vs := range valSyncOps {
		key := validatorSyncKey(vs.InitTxHash)
		WriteValidatorSync(db, vs)
		data = append(data, key...)
	}
	if err := db.Put(valSyncNotProcKey, data); err != nil {
		log.Crit("Failed to store the not processed validator sync operations", "err", err)
	}
}

/**** BlockDag ***/

// ReadBlockDag retrieves the BlockDag structure by hash.
func ReadBlockDag(db ethdb.KeyValueReader, hash common.Hash) *types.BlockDAG {
	data, _ := db.Get(blockDagKey(hash))
	minSize := common.HashLength + common.HashLength + 8
	if len(data) < minSize {
		return nil
	}
	return new(types.BlockDAG).SetBytes(data)
}

// WriteBlockDag stores the BlockDag structure.
func WriteBlockDag(db ethdb.KeyValueWriter, blockDag *types.BlockDAG) {
	key := blockDagKey(blockDag.Hash)
	enc := blockDag.ToBytes()
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store BlockDAG", "err", err)
	}
}

// DeleteBlockDag delete the BlockDAG structure.
func DeleteBlockDag(db ethdb.KeyValueWriter, hash common.Hash) {
	key := blockDagKey(hash)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete BlockDAG", "err", err)
	}
}

// ReadAllBlockDagHashes retrieves all the block dag hashes stored in db
func ReadAllBlockDagHashes(db ethdb.Iteratee) common.HashArray {
	prefix := blockDagPrefix
	hashes := common.HashArray{}
	it := db.NewIterator(prefix, nil)
	defer it.Release()
	for it.Next() {
		if key := it.Key(); len(key) == len(prefix)+32 {
			hashes = append(hashes, common.BytesToHash(key[len(key)-32:]))
		}
	}
	return hashes
}

/**** Tips ***/

// ReadTipsHashes retrieves the hashes of the current tips.
func ReadTipsHashes(db ethdb.KeyValueReader) common.HashArray {
	data, _ := db.Get(tipsHashesKey)
	return common.HashArrayFromBytes(data)
}

// WriteTipsHashes stores the hashes of the current tips.
func WriteTipsHashes(db ethdb.KeyValueWriter, hashes common.HashArray) {
	if err := db.Put(tipsHashesKey, hashes.Uniq().Sort().ToBytes()); err != nil {
		log.Crit("Failed to store tips hashes", "err", err)
	}
}

/**** CHILDREN ***/

// ReadChildren retrieves the hashes of the children
func ReadChildren(db ethdb.KeyValueReader, parent common.Hash) common.HashArray {
	key := childrenKey(parent)
	data, _ := db.Get(key)
	return common.HashArrayFromBytes(data)
}

// WriteChildren stores the hashes of the children
func WriteChildren(db ethdb.KeyValueWriter, parent common.Hash, children common.HashArray) {
	key := childrenKey(parent)
	enc := children.Uniq().Sort().ToBytes()
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store Children", "err", err)
	}
}

// DeleteChildren delete the children
func DeleteChildren(db ethdb.KeyValueWriter, parent common.Hash) {
	key := childrenKey(parent)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete Children", "err", err)
	}
}

// WriteEra writes an era to a key-value database.
func WriteEra(db ethdb.KeyValueWriter, number uint64, era era.Era) {
	key := eraKey(number)

	encoded, err := rlp.EncodeToBytes(era)
	if err != nil {
		log.Warn("Failed to encode era", "err", err, "key:", key, "era:", number)
	}

	db.Put(key, encoded)
}

// ReadEra reads an era from a key-value database.
func ReadEra(db ethdb.KeyValueReader, number uint64) *era.Era {
	key := eraKey(number)
	encoded, err := db.Get(key)
	if err != nil {
		log.Warn("Failed to read era", "err", err, "number", number)
		return nil
	}

	var decoded era.Era
	err = rlp.DecodeBytes(encoded, &decoded)
	if err != nil {
		log.Warn("Failed to decode era", "err", err, "number", number)
		return nil
	}

	return &decoded
}

func DeleteEra(db ethdb.KeyValueWriter, number uint64) {
	key := eraKey(number)
	if err := db.Delete(key); err != nil {
		log.Crit("Failed to delete era", "err", err)
	}
}

// ReadCurrentEra reads the current era number from the database.
func ReadCurrentEra(db ethdb.KeyValueReader) uint64 {
	key := append(currentEraPrefix)
	valueBytes, err := db.Get(key)
	if err != nil {
		log.Warn("Failed to read current era", "err", err)
	}
	return binary.BigEndian.Uint64(valueBytes)
}

// WriteCurrentEra writes the current era number to the database.
func WriteCurrentEra(db ethdb.KeyValueWriter, number uint64) {
	key := append(currentEraPrefix)
	valueBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(valueBytes, number)
	err := db.Put(key, valueBytes)
	if err != nil {
		log.Warn("Failed to write current era", "err", err, "era", number)
	}
}

// DeleteCurrentEra deletes the current era number from the database.
func DeleteCurrentEra(db ethdb.KeyValueWriter) {
	key := append(currentEraPrefix)
	err := db.Delete(key)
	if err != nil {
		log.Warn("Failed to delete current era", "err", err)
	}
}

func FindEra(db ethdb.KeyValueReader, curEra uint64) *era.Era {
	var lastEra *era.Era
	for curEra > 0 {
		log.Info("Try to read era", "number", curEra)
		dbEra := ReadEra(db, curEra)

		if dbEra != nil {
			log.Info("Found era", "number", dbEra.Number, "from", dbEra.From, "to", dbEra.To, "root", dbEra.Root)
			lastEra = dbEra
			return lastEra
		} else {
			log.Info("Era not found", "number", curEra)
			curEra--
			continue
		}
	}
	return lastEra
}

func WriteSlotBlocksHashes(db ethdb.KeyValueWriter, slot uint64, hashes common.HashArray) {
	key := slotBlocksKey(slot)
	err := db.Put(key, hashes.ToBytes())
	if err != nil {
		log.Crit("Failed to store slot blocks", "err", err)
	}
}

func ReadSlotBlocksHashes(db ethdb.KeyValueReader, slot uint64) common.HashArray {
	key := slotBlocksKey(slot)
	buf, _ := db.Get(key)
	if buf != nil {
		return common.HashArrayFromBytes(buf)
	}

	return common.HashArray{}
}

func DeleteSlotBlockHash(db ethdb.Database, slot uint64, hash common.Hash) {
	hashes := ReadSlotBlocksHashes(db, slot)
	for i := 0; i < len(hashes); i++ {
		if hashes[i] == hash {
			hashes = append(hashes[:i], hashes[i+1:]...)
		}
	}

	WriteSlotBlocksHashes(db, slot, hashes)
}

func AddSlotBlockHash(db ethdb.Database, slot uint64, blockHash common.Hash) {
	hashes := ReadSlotBlocksHashes(db, slot)
	for _, hash := range hashes {
		if hash == blockHash {
			return
		}
	}

	hashes = append(hashes, blockHash)
	WriteSlotBlocksHashes(db, slot, hashes)
}
