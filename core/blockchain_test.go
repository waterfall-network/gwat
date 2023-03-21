package core

import (
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

// This test checks that if there are no blocks in the epoch,
// we are looking for blocks from the previous epoch.
// If there were no blocks at all, the genesis hash is taken.
func TestFirstEpochBlockHash(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	numberCache, err := lru.New(10)
	testutils.AssertNoError(t, err)

	blockCache, err := lru.New(10)
	testutils.AssertNoError(t, err)

	bc := &BlockChain{db: db, slotInfo: &types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: 4,
		SlotsPerEpoch:  32,
	}, hc: &HeaderChain{chainDb: db, numberCache: numberCache}, blockCache: blockCache}

	firstEpoch := uint64(0)
	thirdEpoch := firstEpoch + 2
	fifthEpoch := thirdEpoch + 2

	blockNumber := uint64(0)
	firstBlock := types.NewBlock(&types.Header{
		Number:   &blockNumber,
		Slot:     0,
		GasLimit: 21000,
	}, nil, nil, nil)

	rawdb.WriteFinalizedHashNumber(db, firstBlock.Hash(), *firstBlock.Number())
	bc.hc.numberCache.Add(firstBlock.Hash(), blockNumber)
	bc.blockCache.Add(firstBlock.Hash(), firstBlock)

	// Write first block hash to the db.
	bc.WriteFirstEpochBlockHash(firstBlock)
	if !bc.ExistFirstEpochBlockHash(firstEpoch) {
		t.Fatal()
	}

	// Read hash from db for the first epoch.
	hashFromDB := bc.ReadFirstEpochBlockHash(firstEpoch)
	testutils.AssertEqual(t, firstBlock.Hash(), hashFromDB)

	// Read hash from db for the third epoch.
	// At this case result must be the first epoch first block hash,
	// because there are no blocks after first block.
	hashFromDB = bc.ReadFirstEpochBlockHash(thirdEpoch)
	testutils.AssertEqual(t, firstBlock.Hash(), hashFromDB)

	// Read hash from db for the fifth epoch.
	// At this case result must be the first epoch first block hash,
	// because there are no blocks after first block.
	hashFromDB = bc.ReadFirstEpochBlockHash(fifthEpoch)
	testutils.AssertEqual(t, firstBlock.Hash(), hashFromDB)

	// Create another block with another hash
	blockNumber = uint64(8)
	newBlock := types.NewBlock(&types.Header{
		Number:   &blockNumber,
		Slot:     75,
		GasLimit: 21000,
	}, nil, nil, nil)

	rawdb.WriteFinalizedHashNumber(db, newBlock.Hash(), *newBlock.Number())
	bc.hc.numberCache.Add(newBlock.Hash(), blockNumber)
	bc.blockCache.Add(newBlock.Hash(), newBlock)

	// Write the third block hash to the db.
	bc.WriteFirstEpochBlockHash(newBlock)
	if !bc.ExistFirstEpochBlockHash(thirdEpoch) {
		t.Fatal()
	}

	// Read hash from db for the fifth epoch.
	// At this case result must be the third epoch first block hash.
	hashFromDB = bc.ReadFirstEpochBlockHash(fifthEpoch)
	testutils.AssertEqual(t, newBlock.Hash(), hashFromDB)

	// Remove the first epoch block hash from the db and check that it is not there
	bc.DeleteFirstEpochBlockHash(firstEpoch)
	if bc.ExistFirstEpochBlockHash(firstEpoch) {
		t.Fatal()
	}

	// Remove the third epoch block hash from the db and check that it is not there
	bc.DeleteFirstEpochBlockHash(thirdEpoch)
	if bc.ExistFirstEpochBlockHash(thirdEpoch) {
		t.Fatal()
	}
}

func TestFindEra(t *testing.T) {
	// Create a mock database
	db := rawdb.NewMemoryDatabase()
	// Create a new blockchain instance
	bc := &BlockChain{db: db}

	// Test case 0: There are no eras in the database
	lastEra := bc.findEra(5)
	if lastEra != nil {
		t.Errorf("Expected nil, got %v", lastEra)
	}

	// Create some eras and add them to the database
	era0 := era.NewEra(0, 0, 10, common.Hash{})
	err := rawdb.WriteEra(db, era0.Number, *era0)
	if err != nil {
		t.Errorf("Failed to write era 0: %v", err)
	}

	era1 := era.NewEra(1, 11, 20, common.Hash{})
	err = rawdb.WriteEra(db, era1.Number, *era1)
	if err != nil {
		t.Errorf("Failed to write era 1: %v", err)
	}

	era2 := era.NewEra(2, 21, 30, common.Hash{})
	err = rawdb.WriteEra(db, era2.Number, *era2)
	if err != nil {
		t.Errorf("Failed to write era 2: %v", err)
	}

	// Test case 1: Return exact era
	lastEra = bc.findEra(1)
	if lastEra == nil {
		t.Error("Expected an era, got nil")
	} else if lastEra.Number != 1 {
		t.Errorf("Expected era 2, got era %d", lastEra.Number)
	}

	// Test case 2: The last era exists
	lastEra = bc.findEra(5)
	if lastEra == nil {
		t.Error("Expected an era, got nil")
	} else if lastEra.Number != 2 {
		t.Errorf("Expected era 2, got era %d", lastEra.Number)
	}
}
