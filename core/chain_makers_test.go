package core

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

func TestAddBlocksToChain(t *testing.T) {
	bc, blocks := getTestBlockchainAndBlocks()
	if err := addBlocksToChainTest(bc, blocks); err != nil {
		t.Fatal(err)
	}
}

func TestAddFinalizationBlocks(t *testing.T) {
	bc, blocks := getTestBlockchainAndBlocks()
	if err := addBlocksToDag(bc, blocks); err != nil {
		t.Fatal(err)
	}
}

func addBlocksToChainTest(bc *BlockChain, blocks []*types.Block) error {
	if len(blocks) == 0 {
		return errors.New("blocks slice is empty")
	}
	err := AddBlocksToFinalized(bc, blocks)
	if err != nil {
		return err
	}
	for i, block := range blocks {
		bc := bc.GetBlockByNumber(uint64(i + 1))
		if bc.Hash() != block.Hash() {
			return errors.New("blocks aren't added")
		}
	}

	return nil
}

func addBlocksToDag(bc *BlockChain, blocks []*types.Block) error {
	if len(blocks) == 0 {
		return errors.New("blocks slice is empty")
	}
	AddBlocksToDag(bc, blocks)
	tips := bc.GetTips()
	if len(tips.GetHashes()) != 1 {
		return errors.New("wrong tips struct")
	}
	tipDag := tips.Get(tips.GetHashes()[0])
	tipBlk := blocks[len(blocks)-1]
	if tipDag.Hash != tipBlk.Hash() {
		return errors.New("Tips BlockDag: bad hash")
	}
	if tipDag.Height != tipBlk.Height() {
		return errors.New("Tips BlockDag: bad height")
	}
	ctx := context.Background()
	for _, block := range blocks {
		blc := bc.GetBlock(ctx, block.Hash())
		if blc == nil {
			return errors.New("block is not saved")
		}
		if blc.Number() != nil {
			return errors.New("block is finalised")
		}
		bd := bc.GetBlockDag(block.Hash())
		if bd == nil || bd.Hash != block.Hash() || bd.Height != block.Height() {
			return errors.New("BlockDag does not response block")
		}
	}
	return nil
}

func getTestBlockchainAndBlocks() (*BlockChain, []*types.Block) {
	db := rawdb.NewMemoryDatabase()
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	addr := crypto.PubkeyToAddress(key.PublicKey)
	depositData := make(DepositData, 0)
	for i := 0; i < 64; i++ {
		valData := &ValidatorData{
			Pubkey:            common.BytesToBlsPubKey(testutils.RandomData(96)).String(),
			CreatorAddress:    common.BytesToAddress(testutils.RandomData(20)).String(),
			WithdrawalAddress: common.BytesToAddress(testutils.RandomData(20)).String(),
			Amount:            3200,
		}

		depositData = append(depositData, valData)
	}
	genspec := Genesis{
		Config:     params.AllEthashProtocolChanges,
		GasLimit:   1000000000000000000,
		Alloc:      map[common.Address]GenesisAccount{addr: {Balance: big.NewInt(1000000000000000000)}},
		Validators: depositData,
	}

	genesisBlock := genspec.MustCommit(db)

	// Use genesis hash as seed for first and second epochs
	genesisCp := &types.Checkpoint{
		Epoch:    0,
		FinEpoch: 0,
		Root:     common.Hash{},
		Spine:    genesisBlock.Hash(),
	}
	rawdb.WriteLastCoordinatedCheckpoint(db, genesisCp)
	rawdb.WriteCoordinatedCheckpoint(db, genesisCp)
	rawdb.WriteEpoch(db, 0, genesisCp.Spine)

	genesisEraLength := era.EstimateEraLength(genspec.Config, uint64(len(genspec.Validators)))
	genesisEra := era.Era{0, 0, genesisEraLength - 1, genesisBlock.Root()}
	rawdb.WriteEra(db, genesisEra.Number, genesisEra)
	rawdb.WriteCurrentEra(db, genesisEra.Number)

	bc, _ := NewBlockChain(db, nil, params.TestChainConfig, vm.Config{}, nil)
	blocks, _ := GenerateChain(params.AllCliqueProtocolChanges, bc.genesisBlock, bc.db, 3, nil)

	return bc, blocks
}
