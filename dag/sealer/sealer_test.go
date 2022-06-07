package sealer

import (
	"math/big"
	"testing"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/rawdb"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/core/vm"
	"github.com/waterfall-foundation/gwat/crypto"
	"github.com/waterfall-foundation/gwat/params"
)

// This test case is a repro of an annoying bug that took us forever to catch.
// In Sealer PoA networks (Rinkeby, Görli, etc), consecutive blocks might have
// the same state root (no block subsidy, empty block). If a node crashes, the
// chain ends up losing the recent state and needs to regenerate it from blocks
// already in the database. The bug was that processing the block *prior* to an
// empty one **also completes** the empty one, ending up in a known-block error.
func TestReimportMirroredState(t *testing.T) {
	// Initialize a Sealer chain with a single signer
	var (
		db     = rawdb.NewMemoryDatabase()
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr   = crypto.PubkeyToAddress(key.PublicKey)
		//engine = New(params.AllCliqueProtocolChanges.Clique, db)
		engine = New(db)
		signer = new(types.HomesteadSigner)
	)
	genspec := &core.Genesis{
		ExtraData: make([]byte, extraVanity+common.AddressLength+extraSeal),
		Alloc: map[common.Address]core.GenesisAccount{
			addr: {Balance: big.NewInt(10000000000000000)},
		},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	copy(genspec.ExtraData[extraVanity:], addr[:])
	genesis := genspec.MustCommit(db)

	// Generate a batch of blocks, each properly signed
	chain, _ := core.NewBlockChain(db, nil, params.AllCliqueProtocolChanges, engine, vm.Config{}, nil)
	defer chain.Stop()

	blocks, _ := core.GenerateChain(params.AllCliqueProtocolChanges, genesis, engine, db, 3, func(i int, block *core.BlockGen) {
		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		if i != 1 {
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(addr), common.Address{0x00}, new(big.Int), params.TxGas, block.BaseFee(), nil), signer, key)
			if err != nil {
				panic(err)
			}
			block.AddTxWithChain(chain, tx)
		}
	})
	for i, block := range blocks {
		header := block.Header()
		if i > 0 {
			header.ParentHashes[0] = blocks[i-1].Hash()
		}
		header.Extra = make([]byte, extraVanity+extraSeal)

		sig, _ := crypto.Sign(SealHash(header).Bytes(), key)
		copy(header.Extra[len(header.Extra)-extraSeal:], sig)
		blocks[i] = block.WithSeal(header)
		nr := blocks[i].Height()
		blocks[i].SetNumber(&nr)
	}
	// Insert the first two blocks and make sure the chain is valid
	db = rawdb.NewMemoryDatabase()
	genspec.MustCommit(db)

	chain, _ = core.NewBlockChain(db, nil, params.AllCliqueProtocolChanges, engine, vm.Config{}, nil)
	defer chain.Stop()

	if _, err := chain.SyncInsertChain(blocks[:2]); err != nil {
		t.Fatalf("failed to insert initial blocks: %v", err)
	}
	if head := chain.GetLastFinalizedNumber(); head != 2 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 2)
	}

	// Simulate a crash by creating a new chain on top of the database, without
	// flushing the dirty states out. Insert the last block, triggering a sidechain
	// reimport.
	chain, _ = core.NewBlockChain(db, nil, params.AllCliqueProtocolChanges, engine, vm.Config{}, nil)
	defer chain.Stop()

	if _, err := chain.SyncInsertChain(blocks[2:]); err != nil {
		t.Fatalf("failed to insert final block: %v", err)
	}
	if head := chain.GetLastFinalizedNumber(); head != 3 {
		t.Fatalf("chain head mismatch: have %d, want %d", head, 3)
	}
}

func TestSealHash(t *testing.T) {
	have := SealHash(&types.Header{
		Number:  new(uint64),
		Extra:   make([]byte, 32+65),
		BaseFee: new(big.Int),
	})
	want := common.HexToHash("4b90024d2ddcbed94ae978c233f908aee094ce6cd20494d77187df05ca2ddd72")
	if have != want {
		t.Errorf("have %x, want %x", have, want)
	}
}
