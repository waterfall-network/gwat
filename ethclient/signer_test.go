package ethclient

import (
	"context"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/ethconfig"
	"gitlab.waterfall.network/waterfall/protocol/gwat/node"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rpc"
)

var genesis = &core.Genesis{
	Config:    params.AllEthashProtocolChanges,
	Alloc:     core.GenesisAlloc{testAddr: {Balance: testBalance}},
	ExtraData: []byte("test genesis"),
	Timestamp: 9000,
	BaseFee:   big.NewInt(params.InitialBaseFee),
}

var testTx1 = types.MustSignNewTx(testKey, types.LatestSigner(genesis.Config), &types.LegacyTx{
	Nonce:    0,
	Value:    big.NewInt(12),
	GasPrice: big.NewInt(params.InitialBaseFee * 6),
	Gas:      params.TxGas,
	To:       &common.Address{2},
})

var testTx2 = types.MustSignNewTx(testKey, types.LatestSigner(genesis.Config), &types.LegacyTx{
	Nonce:    1,
	Value:    big.NewInt(8),
	GasPrice: big.NewInt(params.InitialBaseFee * 6),
	Gas:      params.TxGas,
	To:       &common.Address{2},
})

func TestEthClientSigner(t *testing.T) {
	backend, _ := newSignerTestBackend(t)
	client, _ := backend.Attach()
	defer backend.Close()
	defer client.Close()

	tests := map[string]struct {
		test func(t *testing.T)
	}{
		"TransactionSender": {
			func(t *testing.T) { testTransactionSender(t, client) },
		},
	}

	for name, tt := range tests {
		t.Run(name, tt.test)
	}
}

func testTransactionSender(t *testing.T, client *rpc.Client) {
	ec := NewClient(client)
	ctx := context.Background()

	// Retrieve testTx1 via RPC.
	block2, err := ec.HeaderByNumber(ctx, big.NewInt(2))
	if err != nil {
		t.Fatal("can't get block 1:", err)
	}
	tx1, err := ec.TransactionInBlock(ctx, block2.Hash(), 0)
	if err != nil {
		t.Fatal("can't get tx:", err)
	}
	if tx1.Hash() != testTx1.Hash() {
		t.Fatalf("wrong tx hash %v, want %v", tx1.Hash(), testTx1.Hash())
	}

	// The sender address is cached in tx1, so no additional RPC should be required in
	// TransactionSender. Ensure the server is not asked by canceling the context here.
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	sender1, err := ec.TransactionSender(canceledCtx, tx1, block2.Hash(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if sender1 != testAddr {
		t.Fatal("wrong sender:", sender1)
	}

	// Now try to get the sender of testTx2, which was not fetched through RPC.
	// TransactionSender should query the server here.
	sender2, err := ec.TransactionSender(ctx, testTx2, block2.Hash(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sender2 != testAddr {
		t.Fatal("wrong sender:", sender2)
	}
}

func newSignerTestBackend(t *testing.T) (*node.Node, []*types.Block) {
	// Generate test chain.
	genesis, blocks := generateSignerTestChain()
	// Create node
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis}
	ethservice, err := eth.New(n, config)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	// Import the test chain.
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().SyncInsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}
	ethservice.BlockChain().SetIsSynced(true)
	return n, blocks
}

func generateSignerTestChain() (*core.Genesis, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:       config,
		Alloc:        core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData:    []byte("test genesis"),
		Timestamp:    9000,
		BaseFee:      big.NewInt(params.InitialBaseFee),
		ParentHashes: []common.Hash{},
		Validators:   core.GenDepositData(64),
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
		if i == 1 {
			// Test transactions are included in block #2.
			g.AddTx(testTx1)
			g.AddTx(testTx2)
		}
	}
	gblock := genesis.ToBlock(db)
	genesis.MustCommit(db)

	blocks, _ := core.GenerateChain(config, gblock, db, 2, generate)
	blocks = append([]*types.Block{gblock}, blocks...)
	for _, bl := range blocks {
		nr := bl.Height()
		bl.SetNumber(&nr)
	}
	return genesis, blocks
}
