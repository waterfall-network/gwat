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

const badBlockToKeep = 10

type badBlock struct {
	Header *types.Header
	Body   *types.Body
}

// badBlockList implements the sort interface to allow sorting a list of
// bad blocks by their number in the reverse order.
type badBlockList []*badBlock

func (s badBlockList) Len() int { return len(s) }
func (s badBlockList) Less(i, j int) bool {
	return s[i].Header.Nr() < s[j].Header.Nr()
}
func (s badBlockList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// ReadBadBlock retrieves the bad block with the corresponding block hash.
func ReadBadBlock(db ethdb.Reader, hash common.Hash) *types.Block {
	blob, err := db.Get(badBlockKey)
	if err != nil {
		return nil
	}
	var badBlocks badBlockList
	if err := rlp.DecodeBytes(blob, &badBlocks); err != nil {
		return nil
	}
	for _, bad := range badBlocks {
		if bad.Header.Hash() == hash {
			return types.NewBlockWithHeader(bad.Header).WithBody(bad.Body.Transactions)
		}
	}
	return nil
}

// ReadAllBadBlocks retrieves all the bad blocks in the database.
// All returned blocks are sorted in reverse order by number.
func ReadAllBadBlocks(db ethdb.Reader) []*types.Block {
	blob, err := db.Get(badBlockKey)
	if err != nil {
		return nil
	}
	var badBlocks badBlockList
	if err := rlp.DecodeBytes(blob, &badBlocks); err != nil {
		return nil
	}
	var blocks []*types.Block
	for _, bad := range badBlocks {
		blocks = append(blocks, types.NewBlockWithHeader(bad.Header).WithBody(bad.Body.Transactions))
	}
	return blocks
}

// WriteBadBlock serializes the bad block into the database. If the cumulated
// bad blocks exceeds the limitation, the oldest will be dropped.
func WriteBadBlock(db ethdb.KeyValueStore, block *types.Block) {
	blob, err := db.Get(badBlockKey)
	if err != nil {
		log.Warn("Failed to load old bad blocks", "error", err)
	}
	var badBlocks badBlockList
	if len(blob) > 0 {
		if err := rlp.DecodeBytes(blob, &badBlocks); err != nil {
			log.Crit("Failed to decode old bad blocks", "error", err)
		}
	}
	for _, b := range badBlocks {
		if b.Header.Nr() == block.Nr() && b.Header.Hash() == block.Hash() {
			log.Info("Skip duplicated bad block", "number", block.Nr(), "hash", block.Hash().Hex())
			return
		}
	}
	badBlocks = append(badBlocks, &badBlock{
		Header: block.Header(),
		Body:   block.Body(),
	})
	if len(badBlocks) > badBlockToKeep {
		badBlocks = badBlocks[:badBlockToKeep]
	}
	data, err := rlp.EncodeToBytes(badBlocks)
	if err != nil {
		log.Crit("Failed to encode bad blocks", "err", err)
	}
	if err := db.Put(badBlockKey, data); err != nil {
		log.Crit("Failed to write bad blocks", "err", err)
	}
}

// DeleteBadBlocks deletes all the bad blocks from the database
func DeleteBadBlocks(db ethdb.KeyValueWriter) {
	if err := db.Delete(badBlockKey); err != nil {
		log.Crit("Failed to delete bad blocks", "err", err)
	}
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
	height := ReadFinalizedNumberByHash(db, hash)
	if height == nil {
		return uint64(0)
	}
	return *height
}

// ReadLastCoordinatedHash retrieves the hash of the last Coordinated block.
func ReadLastCoordinatedHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(lastCoordHashKey)
	return common.BytesToHash(data)
}

// WriteLastCoordinatedHash stores the hash of the last Coordinated block
func WriteLastCoordinatedHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(lastCoordHashKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store the Last Coordinated Hash", "err", err)
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

func WriteFirstEpochBlockHash(db ethdb.KeyValueWriter, epoch uint64, hash common.Hash) {
	key := firstEpochBlockKey(epoch)

	err := db.Put(key, hash.Bytes())
	if err != nil {
		log.Crit("Failed to store epoch seed", "err", err, "epoch", epoch)
	}
}

func ReadFirstEpochBlockHash(db ethdb.KeyValueReader, epoch uint64) common.Hash {
	key := firstEpochBlockKey(epoch)
	buf, err := db.Get(key)
	if err != nil {
		return common.Hash{}
	}

	seed := common.BytesToHash(buf)

	return seed
}

func DeleteFirstEpochBlockHash(db ethdb.KeyValueWriter, epoch uint64) {
	key := firstEpochBlockKey(epoch)
	err := db.Delete(key)
	if err != nil {
		log.Crit("Failed to delete epoch seed", "err", err, "epoch", epoch)
	}
}

func ExistFirstEpochBlockHash(db ethdb.KeyValueReader, epoch uint64) bool {
	key := firstEpochBlockKey(epoch)
	exist, _ := db.Has(key)

	return exist
}

func WriteEra(db ethdb.KeyValueWriter, number uint64, era types.Era) error {
	key := eraKey(number)

	encoded, err := rlp.EncodeToBytes(era)
	if err != nil {
		log.Crit("Failed to encode era", "err", err, "key:", key, "era:", number)
		return err
	}

	return db.Put(key, encoded)
}

func GetEra(db ethdb.KeyValueReader, number uint64) (*types.Era, error) {
	key := eraKey(number)
	encoded, err := db.Get(key)
	if err != nil {
		return nil, err
	}

	var era types.Era
	err = rlp.DecodeBytes(encoded, &era)
	if err != nil {
		return nil, err
	}

	return &era, nil
}

func GetEraByEpoch(db ethdb.KeyValueReader, epoch uint64, lastEraNumber uint64) (*types.Era, error) {
	// The number of eras
	numEras := lastEraNumber

	// Get the last era
	lastEra, err := GetEra(db, lastEraNumber)
	if err != nil {
		return nil, errors.New("last era not found")
	}

	if lastEra.End < epoch {
		return nil, errors.New("epoch is after last era")
	}

	middleEpoch := (lastEra.End) / 2

	// Determine which half of the eras to search
	var startEra uint64
	if epoch < middleEpoch {
		startEra = 0
	} else {
		startEra = lastEraNumber / 2
	}

	// Loop through the relevant eras until we find one that contains the epoch
	var currentEra *types.Era
	for i := startEra; i <= numEras; i++ {
		era, err := GetEra(db, i)
		if err != nil {
			return nil, err
		}

		if era.Begin <= epoch && epoch < era.End {
			currentEra = era
			break
		}
	}

	// If no epoch in eras was found, return an error
	if currentEra == nil {
		return nil, errors.New("epoch not found in any era")
	}

	return currentEra, nil
}