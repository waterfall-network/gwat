// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"bytes"
	"context"
	"errors"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/rawdb"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/rlp"
)

// errNonCanonicalHash is returned if the requested chain data doesn't belong
// to the canonical chain. ODR can only retrieve the canonical chain data covered
// by the CHT or Bloom trie for verification.
var errNonCanonicalHash = errors.New("hash is not currently canonical")

// GetHeaderByNumber retrieves the canonical block header corresponding to the
// given number. The returned header is proven by local CHT.
func GetHeaderByNumber(ctx context.Context, odr OdrBackend, number uint64) (*types.Header, error) {
	// Try to find it in the local database first.
	db := odr.Database()
	hash := rawdb.ReadFinalizedHashByNumber(db, number)

	// If there is a canonical hash, there should have a header too.
	// But if it's pruned, re-fetch from network again.
	if (hash != common.Hash{}) {
		if header := rawdb.ReadHeader(db, hash); header != nil {
			header.Number = &number
			return header, nil
		}
	}
	// Retrieve the header via ODR, ensure the requested header is covered
	// by local trusted CHT.
	chts, _, chtHead := odr.ChtIndexer().Sections()
	if number >= chts*odr.IndexerConfig().ChtSize {
		return nil, errNoTrustedCht
	}
	r := &ChtRequest{
		ChtRoot: GetChtRoot(db, chts-1, chtHead),
		ChtNum:  chts - 1,
		Config:  odr.IndexerConfig(),
	}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	}
	r.Header.Number = &number
	return r.Header, nil
}

func GetHeader(ctx context.Context, odr OdrBackend, hash common.Hash) (*types.Header, error) {
	// Try to find it in the local database first.
	db := odr.Database()
	// If there is a canonical hash, there should have a header too.
	// But if it's pruned, re-fetch from network again.
	if (hash != common.Hash{}) {
		if header := rawdb.ReadHeader(db, hash); header != nil {
			return header, nil
		}
	}
	// Retrieve the header via ODR, ensure the requested header is covered
	// by local trusted CHT.
	chts, _, chtHead := odr.ChtIndexer().Sections()
	//if number >= chts*odr.IndexerConfig().ChtSize {
	//	return nil, errNoTrustedCht
	//}
	r := &ChtRequest{
		ChtRoot: GetChtRoot(db, chts-1, chtHead),
		ChtNum:  chts - 1,
		//BlockNum: number,
		Config: odr.IndexerConfig(),
	}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	}
	return r.Header, nil
}

// GetCanonicalHash retrieves the canonical block hash corresponding to the number.
func GetCanonicalHash(ctx context.Context, odr OdrBackend, number uint64) (common.Hash, error) {
	hash := rawdb.ReadFinalizedHashByNumber(odr.Database(), number)
	if hash != (common.Hash{}) {
		return hash, nil
	}
	header, err := GetHeader(ctx, odr, hash)
	if err != nil {
		return common.Hash{}, err
	}
	// number -> canonical mapping already be stored in db, get it.
	return header.Hash(), nil
}

// GetBodyRLP retrieves the block body (transactions and uncles) in RLP encoding.
func GetBodyRLP(ctx context.Context, odr OdrBackend, hash common.Hash) (rlp.RawValue, error) {
	if data := rawdb.ReadBodyRLP(odr.Database(), hash); data != nil {
		return data, nil
	}
	// Retrieve the block header first and pass it for verification.
	header, err := GetHeader(ctx, odr, hash)
	if err != nil {
		return nil, errNoHeader
	}
	if header.Hash() != hash {
		return nil, errNonCanonicalHash
	}
	r := &BlockRequest{
		Hash:   hash,
		Header: header,
	}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	}
	return r.Rlp, nil
}

// GetBody retrieves the block body (transactions, uncles) corresponding to the
// hash.
func GetBody(ctx context.Context, odr OdrBackend, hash common.Hash) (*types.Body, error) {
	data, err := GetBodyRLP(ctx, odr, hash)
	if err != nil {
		return nil, err
	}
	body := new(types.Body)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		return nil, err
	}
	return body, nil
}

// GetBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body.
func GetBlock(ctx context.Context, odr OdrBackend, hash common.Hash) (*types.Block, error) {
	// Retrieve the block header and body contents
	header, err := GetHeader(ctx, odr, hash)
	if err != nil {
		return nil, errNoHeader
	}
	body, err := GetBody(ctx, odr, hash)
	if err != nil {
		return nil, err
	}
	// Reassemble the block and return
	return types.NewBlockWithHeader(header).WithBody(body.Transactions), nil
}

// GetBlockReceipts retrieves the receipts generated by the transactions included
// in a block given by its hash.
func GetBlockReceipts(ctx context.Context, odr OdrBackend, hash common.Hash) (types.Receipts, error) {
	// Assume receipts are already stored locally and attempt to retrieve.
	receipts := rawdb.ReadRawReceipts(odr.Database(), hash)
	if receipts == nil {
		header, err := GetHeader(ctx, odr, hash)
		if err != nil {
			return nil, errNoHeader
		}
		if header.Hash() != hash {
			return nil, errNonCanonicalHash
		}
		r := &ReceiptsRequest{Hash: hash, Header: header}
		if err := odr.Retrieve(ctx, r); err != nil {
			return nil, err
		}
		receipts = r.Receipts
	}
	// If the receipts are incomplete, fill the derived fields
	if len(receipts) > 0 && receipts[0].TxHash == (common.Hash{}) {
		block, err := GetBlock(ctx, odr, hash)
		if err != nil {
			return nil, err
		}
		genesis := rawdb.ReadFinalizedHashByNumber(odr.Database(), 0)
		config := rawdb.ReadChainConfig(odr.Database(), genesis)

		if err := receipts.DeriveFields(config, block.Hash(), block.Nr(), block.Transactions()); err != nil {
			return nil, err
		}
		rawdb.WriteReceipts(odr.Database(), hash, receipts)
	}
	return receipts, nil
}

// GetBlockLogs retrieves the logs generated by the transactions included in a
// block given by its hash.
func GetBlockLogs(ctx context.Context, odr OdrBackend, hash common.Hash) ([][]*types.Log, error) {
	// Retrieve the potentially incomplete receipts from disk or network
	receipts, err := GetBlockReceipts(ctx, odr, hash)
	if err != nil {
		return nil, err
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

// GetUntrustedBlockLogs retrieves the logs generated by the transactions included in a
// block. The retrieved logs are regarded as untrusted and will not be stored in the
// database. This function should only be used in light client checkpoint syncing.
func GetUntrustedBlockLogs(ctx context.Context, odr OdrBackend, header *types.Header) ([][]*types.Log, error) {
	// Retrieve the potentially incomplete receipts from disk or network
	hash, number := header.Hash(), header.Nr()
	receipts := rawdb.ReadRawReceipts(odr.Database(), hash)
	if receipts == nil {
		r := &ReceiptsRequest{Hash: hash, Number: number, Header: header, Untrusted: true}
		if err := odr.Retrieve(ctx, r); err != nil {
			return nil, err
		}
		receipts = r.Receipts
		// Untrusted receipts won't be stored in the database. Therefore
		// derived fields computation is unnecessary.
	}
	// Return the logs without deriving any computed fields on the receipts
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

// GetBloomBits retrieves a batch of compressed bloomBits vectors belonging to
// the given bit index and section indexes.
func GetBloomBits(ctx context.Context, odr OdrBackend, bit uint, sections []uint64) ([][]byte, error) {
	var (
		reqIndex    []int
		reqSections []uint64
		db          = odr.Database()
		result      = make([][]byte, len(sections))
	)
	blooms, _, sectionHead := odr.BloomTrieIndexer().Sections()
	for i, section := range sections {
		sectionHead := rawdb.ReadFinalizedHashByNumber(db, (section+1)*odr.IndexerConfig().BloomSize-1)
		// If we don't have the canonical hash stored for this section head number,
		// we'll still look for an entry with a zero sectionHead (we store it with
		// zero section head too if we don't know it at the time of the retrieval)
		if bloomBits, _ := rawdb.ReadBloomBits(db, bit, section, sectionHead); len(bloomBits) != 0 {
			result[i] = bloomBits
			continue
		}
		// TODO(rjl493456442) Convert sectionIndex to BloomTrie relative index
		if section >= blooms {
			return nil, errNoTrustedBloomTrie
		}
		reqSections = append(reqSections, section)
		reqIndex = append(reqIndex, i)
	}
	// Find all bloombits in database, nothing to query via odr, return.
	if reqSections == nil {
		return result, nil
	}
	// Send odr request to retrieve missing bloombits.
	r := &BloomRequest{
		BloomTrieRoot:    GetBloomTrieRoot(db, blooms-1, sectionHead),
		BloomTrieNum:     blooms - 1,
		BitIdx:           bit,
		SectionIndexList: reqSections,
		Config:           odr.IndexerConfig(),
	}
	if err := odr.Retrieve(ctx, r); err != nil {
		return nil, err
	}
	for i, idx := range reqIndex {
		result[idx] = r.BloomBits[i]
	}
	return result, nil
}

// GetTransaction retrieves a canonical transaction by hash and also returns
// its position in the chain. There is no guarantee in the LES protocol that
// the mined transaction will be retrieved back for sure because of different
// reasons(the transaction is unindexed, the malicous server doesn't reply it
// deliberately, etc). Therefore, unretrieved transactions will receive a certain
// number of retrys, thus giving a weak guarantee.
func GetTransaction(ctx context.Context, odr OdrBackend, txHash common.Hash) (*types.Transaction, common.Hash, uint64, error) {
	r := &TxStatusRequest{Hashes: []common.Hash{txHash}}
	if err := odr.RetrieveTxStatus(ctx, r); err != nil || r.Status[0].Status != core.TxStatusIncluded {
		return nil, common.Hash{}, 0, err
	}
	pos := r.Status[0].Lookup
	// first ensure that we have the header, otherwise block body retrieval will fail
	// also verify if this is a canonical block by getting the header by number and checking its hash
	if header, err := GetHeader(ctx, odr, pos.BlockHash); err != nil || header.Hash() != pos.BlockHash {
		return nil, common.Hash{}, 0, err
	}
	body, err := GetBody(ctx, odr, pos.BlockHash)
	if err != nil || uint64(len(body.Transactions)) <= pos.Index || body.Transactions[pos.Index].Hash() != txHash {
		return nil, common.Hash{}, 0, err
	}
	return body.Transactions[pos.Index], pos.BlockHash, pos.Index, nil
}
