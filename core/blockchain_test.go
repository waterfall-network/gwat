package core

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/shuffle"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/token/testutils"
	"math/big"
	"testing"
)

func TestBreakByCreatorsPerSlotCount(t *testing.T) {
	bc := &BlockChain{}
	// Test input with 5 creators and 2 creators per slot
	creators := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5")}
	bc.slotInfo = &types.SlotInfo{SlotsPerEpoch: 2}
	expected := [][]common.Address{
		{common.HexToAddress("0x1"), common.HexToAddress("0x2")},
		{common.HexToAddress("0x3"), common.HexToAddress("0x4")},
	}

	result := bc.breakByValidatorsBySlotCount(creators, 2)

	testutils.AssertEqual(t, expected, result)
}

func TestBlockChain_seed(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	genesis := GenesisBlockForTesting(db, common.HexToAddress("1"), big.NewInt(int64(1e18)))
	bc := BlockChain{db: db, genesisBlock: genesis}

	// Test case 1: Test that seed exist for first epoch.
	expectedSeed := crypto.Keccak256(append(genesis.Hash().Bytes(), shuffle.Bytes8(0)...))

	seed, err := bc.seed(0)
	testutils.AssertNoError(t, err)

	if seed != common.BytesToHash(expectedSeed) {
		t.Fatalf("unexpected seed, want %x, got %x", expectedSeed, seed)
	}

	// Test case 2: Test that seed exist for second epoch.
	expectedSeed = crypto.Keccak256(append(genesis.Hash().Bytes(), shuffle.Bytes8(1)...))

	seed, err = bc.seed(1)
	testutils.AssertNoError(t, err)
	if seed != common.BytesToHash(expectedSeed) {
		t.Fatalf("unexpected seed, want %x, got %x", expectedSeed, seed)
	}

	// Test case 3: Test that seed does not exist for next epoch
	_, err = bc.seed(2)
	if err != errNoEpochSeed {
		t.Fatalf("unexpected error, want %v, got %v", errNoEpochSeed, err)
	}
}
