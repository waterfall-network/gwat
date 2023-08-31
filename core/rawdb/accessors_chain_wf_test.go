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
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

// deprecated
// Tests that head headers and head blocks can be assigned, individually.
func TestHeadStorageWf(t *testing.T) {
	db := NewMemoryDatabase()

	blockHead := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block header")})
	blockFull := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block full")})
	blockFast := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block fast")})

	// Check that no head entries are in a pristine database
	if entry := ReadTipsHashes(db); !entry.IsEqualTo(common.HashArray{}) {
		t.Fatalf("Non head header entry returned: %v", entry)
	}
	if entry := ReadLastCanonicalHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non head block entry returned: %v", entry)
	}
	if entry := ReadHeadFastBlockHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non fast head block entry returned: %v", entry)
	}

	// Assign separate entries for the head header and block
	WriteTipsHashes(db, common.HashArray{blockHead.Hash()})
	WriteLastCanonicalHash(db, blockFull.Hash())
	WriteHeadFastBlockHash(db, blockFast.Hash())

	// Check that both heads are present, and different (i.e. two heads maintained)
	if entry := ReadTipsHashes(db); !entry.IsEqualTo(common.HashArray{blockHead.Hash()}) {
		t.Fatalf("Head header hash mismatch: have %v, want %v", entry, common.HashArray{blockHead.Hash()})
	}
	if entry := ReadLastCanonicalHash(db); entry != blockFull.Hash() {
		t.Fatalf("Head block hash mismatch: have %v, want %v", entry, blockFull.Hash())
	}
	if entry := ReadHeadFastBlockHash(db); entry != blockFast.Hash() {
		t.Fatalf("Fast head block hash mismatch: have %v, want %v", entry.Hex(), blockFast.Hash())
	}

	WriteHeader(db, blockHead.Header())
	WriteBlock(db, blockHead)
}

// Tests that head headers and head blocks can be assigned, individually.
func TestLastFinalizedBlockWf(t *testing.T) {
	db := NewMemoryDatabase()

	// Check FinalizedHeightByHash
	if entry := ReadFinalizedNumberByHash(db, common.Hash{}); entry != nil {
		t.Fatalf("Non empty hash: %v", entry)
	}
	finHeight := uint64(111111111)
	finBlock := types.NewBlockWithHeader(&types.Header{Extra: []byte("test FinBlock")})
	_writeFinalizedNumberByHash(db, finBlock.Hash(), finHeight)
	if entry := ReadFinalizedNumberByHash(db, finBlock.Hash()); *entry != finHeight {
		t.Fatalf("finBlock height mismatch: have %d, want %d", entry, finHeight)
	}

	// Check FinalizedHashByHeight
	if entry := ReadFinalizedNumberByHash(db, common.Hash{}); entry != nil {
		t.Fatalf("Non empty hash: %v", entry)
	}
	finHeight1 := uint64(252222222222)
	finBlock1 := types.NewBlockWithHeader(&types.Header{Extra: []byte("test FinBlock")})
	_writeFinalizedHashByNumber(db, finHeight1, finBlock1.Hash())
	if entry := ReadFinalizedHashByNumber(db, finHeight1); entry != finBlock1.Hash() {
		t.Fatalf("finBlock hash mismatch: have %v, want %v", entry, finBlock1.Hash())
	}

	// Check CpHash
	if entry := ReadLastFinalizedHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non empty hash: %v", entry)
	}
	lastFinBlock := types.NewBlockWithHeader(&types.Header{Extra: []byte("test lastFinBlock")})
	WriteLastFinalizedHash(db, lastFinBlock.Hash())
	if entry := ReadLastFinalizedHash(db); entry != lastFinBlock.Hash() {
		t.Fatalf("lastFinBlock hash mismatch: have %v, want %v", entry, lastFinBlock.Hash())
	}

	// Check CpHeight WriteFinalizedHashNumber
	if entry := ReadLastFinalizedNumber(db); entry != uint64(0) {
		t.Fatalf("Non empty hash: %v", entry)
	}
	lastFinHeight1 := uint64(33333333)
	lastFinBlock1 := types.NewBlockWithHeader(&types.Header{Extra: []byte("test lastFinBlock1")})
	WriteFinalizedHashNumber(db, lastFinBlock1.Hash(), lastFinHeight1)
	WriteLastFinalizedHash(db, lastFinBlock1.Hash())
	if entry := ReadLastFinalizedNumber(db); entry != lastFinHeight1 {
		t.Fatalf("lastFinBlock1 hash mismatch: have %d, want %d", entry, lastFinHeight1)
	}

}

// Tests that head headers and head blocks can be assigned, individually.
func TestBlockDAGWf(t *testing.T) {
	db := NewMemoryDatabase()

	// Check FinalizedHeightByHash
	if entry := ReadBlockDag(db, common.Hash{}); entry != nil {
		t.Fatalf("Non empty hash: %v", entry)
	}

	finBlock := types.NewBlockWithHeader(&types.Header{Extra: []byte("test FinBlock")})

	blockDag := &types.BlockDAG{
		Hash:                   finBlock.Hash(),
		Height:                 finBlock.Height(),
		CpHash:                 finBlock.Hash(),
		CpHeight:               1455646545646,
		OrderedAncestorsHashes: common.HashArray{common.Hash{}, finBlock.Hash(), common.Hash{}},
	}

	WriteBlockDag(db, blockDag)
	if entry := ReadBlockDag(db, finBlock.Hash()); fmt.Sprintf("%v", entry) != fmt.Sprintf("%v", blockDag) {
		t.Fatalf("BlockDag W-R failed:  %#v != %#v", entry, blockDag)
	}

	DeleteBlockDag(db, blockDag.Hash)
	if entry := ReadBlockDag(db, finBlock.Hash()); entry != nil {
		t.Fatalf("BlockDag D-R failed:  %#v != nil", entry)
	}
}

func TestValidatorSyncWf_Ok(t *testing.T) {
	db := NewMemoryDatabase()

	src_1 := &types.ValidatorSync{
		OpType:     2,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		TxHash:     &common.Hash{7, 8, 9},
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_1.Amount.SetString("32789456000000", 10)

	WriteValidatorSync(db, src_1)
	entry := ReadValidatorSync(db, src_1.InitTxHash)
	if fmt.Sprintf("%v", entry) != fmt.Sprintf("%v", src_1) {
		t.Fatalf("ValidatorSync W-R failed:  %#v != %#v", entry, src_1)
	}

	DeleteValidatorSync(db, src_1.InitTxHash)
	if entry := ReadValidatorSync(db, src_1.InitTxHash); entry != nil {
		t.Fatalf("ValidatorSync D-R failed:  %#v != nil", entry)
	}
}

func TestValidatorSyncWf_Ok_noTxHash(t *testing.T) {
	db := NewMemoryDatabase()

	src_1 := &types.ValidatorSync{
		OpType:    2,
		ProcEpoch: 45645,
		Index:     45645,
		Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:    new(big.Int),
		//TxHash:     &common.Hash{7, 8, 9},
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_1.Amount.SetString("32789456000000", 10)

	WriteValidatorSync(db, src_1)
	entry := ReadValidatorSync(db, src_1.InitTxHash)
	if fmt.Sprintf("%v", entry) != fmt.Sprintf("%v", src_1) {
		t.Fatalf("ValidatorSync W-R failed:  %#v != %#v", entry, src_1)
	}

	DeleteValidatorSync(db, src_1.InitTxHash)
	if entry := ReadValidatorSync(db, src_1.InitTxHash); entry != nil {
		t.Fatalf("ValidatorSync D-R failed:  %#v != nil", entry)
	}
}

func TestValidatorSyncWf_Ok_noAmount(t *testing.T) {
	db := NewMemoryDatabase()

	src_1 := &types.ValidatorSync{
		OpType:    1,
		ProcEpoch: 45645,
		Index:     45645,
		Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		//Amount:     new(big.Int),
		TxHash:     &common.Hash{7, 8, 9},
		InitTxHash: common.Hash{1, 2, 3},
	}
	//src_1.Amount.SetString("32789456000000", 10)

	WriteValidatorSync(db, src_1)
	entry := ReadValidatorSync(db, src_1.InitTxHash)
	if fmt.Sprintf("%v", entry) != fmt.Sprintf("%v", src_1) {
		t.Fatalf("ValidatorSync W-R failed:  %#v != %#v", entry, src_1)
	}

	DeleteValidatorSync(db, src_1.InitTxHash)
	if entry := ReadValidatorSync(db, src_1.InitTxHash); entry != nil {
		t.Fatalf("ValidatorSync D-R failed:  %#v != nil", entry)
	}
}

func TestNotProcessedValidatorSyncWf(t *testing.T) {
	db := NewMemoryDatabase()

	src_1 := &types.ValidatorSync{
		OpType:     types.Activate,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0x11, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		TxHash:     nil,
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_2 := &types.ValidatorSync{
		OpType:     types.Deactivate,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0x22, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		TxHash:     nil,
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_3 := &types.ValidatorSync{
		OpType:     types.UpdateBalance,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0x33, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		TxHash:     nil,
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_3.Amount.SetString("32789456000000", 10)

	valSyncOps := []*types.ValidatorSync{src_1, src_2, src_3}

	WriteNotProcessedValidatorSyncOps(db, valSyncOps)
	if entry := ReadNotProcessedValidatorSyncOps(db); reflect.DeepEqual(entry, valSyncOps) {
		t.Fatalf("ValidatorSync W-R failed:  %#v != %#v", entry, valSyncOps)
	}
}
