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

package ethtest

import (
	"strconv"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/p2p"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

// TestEthProtocolNegotiation tests whether the test suite
// can negotiate the highest eth protocol in a status message exchange
func TestEthProtocolNegotiation(t *testing.T) {
	var tests = []struct {
		conn     *Conn
		caps     []p2p.Cap
		expected uint32
	}{
		{
			conn: &Conn{
				ourHighestProtoVersion: 65,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 65,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 65,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 64,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			expected: 64,
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 65,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 0},
				{Name: "eth", Version: 89},
				{Name: "eth", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 64,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "wrongProto", Version: 65},
			},
			expected: uint32(64),
		},
		{
			conn: &Conn{
				ourHighestProtoVersion: 65,
			},
			caps: []p2p.Cap{
				{Name: "eth", Version: 63},
				{Name: "eth", Version: 64},
				{Name: "wrongProto", Version: 65},
			},
			expected: uint32(64),
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tt.conn.negotiateEthProtocol(tt.caps)
			testutils.AssertEqual(t, tt.expected, uint32(tt.conn.negotiatedProtoVersion))
		})
	}
}

//
//// TestChain_GetHeaders tests whether the test suite can correctly
//// respond to a GetBlockHeaders request from a node.
//func TestChain_GetHeaders(t *testing.T) {
//	_, blocks := getTestBlockchainAndBlocks()
//
//	chain := NewTestChain(blocks, params.AllEthashProtocolChanges)
//
//	//chain, err := loadChain(chainFile, genesisFile)
//	//if err != nil {
//	//	t.Fatal(err)
//	//}
//
//	var tests = []struct {
//		req      GetBlockHeaders
//		expected BlockHeaders
//	}{
//		{
//			req: GetBlockHeaders{
//				Origin: &eth.HashOrNumber{
//					Number: uint64(2),
//				},
//				Amount:  uint64(1),
//				Skip:    1,
//				Reverse: false,
//			},
//			expected: BlockHeaders{
//				chain.blocks[2].Header(),
//				//chain.blocks[4].Header(),
//				//chain.blocks[6].Header(),
//				//chain.blocks[8].Header(),
//				//chain.blocks[10].Header(),
//			},
//		},
//		//{
//		//	req: GetBlockHeaders{
//		//		Origin: &eth.HashOrNumber{
//		//			Number: uint64(chain.Len() - 1),
//		//		},
//		//		Amount:  uint64(3),
//		//		Skip:    0,
//		//		Reverse: true,
//		//	},
//		//	expected: BlockHeaders{
//		//		chain.blocks[chain.Len()-1].Header(),
//		//		chain.blocks[chain.Len()-2].Header(),
//		//		chain.blocks[chain.Len()-3].Header(),
//		//	},
//		//},
//		//{
//		//	req: GetBlockHeaders{
//		//		Origin: &eth.HashOrNumber{
//		//			Hash: chain.Head().Hash(),
//		//		},
//		//		Amount:  uint64(1),
//		//		Skip:    0,
//		//		Reverse: false,
//		//	},
//		//	expected: BlockHeaders{
//		//		chain.Head().Header(),
//		//	},
//		//},
//	}
//
//	for i, tt := range tests {
//		t.Run(strconv.Itoa(i), func(t *testing.T) {
//			headers, err := chain.GetHeaders(tt.req)
//			if err != nil {
//				t.Fatal(err)
//			}
//			testutils.AssertEqual(t, headers, tt.expected)
//		})
//	}
//}
//
//func getTestBlockchainAndBlocks() (*core.BlockChain, []*types.Block) {
//	db := rawdb.NewMemoryDatabase()
//	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
//	addr := crypto.PubkeyToAddress(key.PublicKey)
//	depositData := make(core.DepositData, 0)
//	for i := 0; i < 64; i++ {
//		valData := &core.ValidatorData{
//			Pubkey:            common.BytesToBlsPubKey(testutils.RandomData(96)).String(),
//			CreatorAddress:    common.BytesToAddress(testutils.RandomData(20)).String(),
//			WithdrawalAddress: common.BytesToAddress(testutils.RandomData(20)).String(),
//			Amount:            3200,
//		}
//
//		depositData = append(depositData, valData)
//	}
//	genspec := core.Genesis{
//		Config:     params.AllEthashProtocolChanges,
//		GasLimit:   1000000000000000000,
//		Alloc:      map[common.Address]core.GenesisAccount{addr: {Balance: big.NewInt(1000000000000000000)}},
//		Validators: depositData,
//	}
//
//	genesisBlock := genspec.MustCommit(db)
//
//	// Use genesis hash as seed for first and second epochs
//	genesisCp := &types.Checkpoint{
//		Epoch:    0,
//		FinEpoch: 0,
//		Root:     common.Hash{},
//		Spine:    genesisBlock.Hash(),
//	}
//	rawdb.WriteLastCoordinatedCheckpoint(db, genesisCp)
//	rawdb.WriteCoordinatedCheckpoint(db, genesisCp)
//	rawdb.WriteEpoch(db, 0, genesisCp.Spine)
//
//	genesisEraLength := era.EstimateEraLength(genspec.Config, uint64(len(genspec.Validators)))
//	genesisEra := era.Era{0, 0, genesisEraLength - 1, genesisBlock.Root()}
//	rawdb.WriteEra(db, genesisEra.Number, genesisEra)
//	rawdb.WriteCurrentEra(db, genesisEra.Number)
//
//	bc, _ := core.NewBlockChain(db, nil, params.TestChainConfig, vm.Config{}, nil)
//	blocks, _ := core.GenerateChain(params.AllCliqueProtocolChanges, genesisBlock, db, 3, nil)
//
//	return bc, blocks
//}
//
//type TestChain struct {
//	blocks      []*types.Block
//	chainConfig *params.ChainConfig
//}
//
//func NewTestChain(blocks []*types.Block, chainConfig *params.ChainConfig) *TestChain {
//	return &TestChain{blocks: blocks, chainConfig: chainConfig}
//}
//
//// Len returns the length of the chain.
//func (c *TestChain) Len() int {
//	return len(c.blocks)
//}
//
//// ForkID gets the fork id of the chain.
//func (c *TestChain) ForkID() forkid.ID {
//	return forkid.NewID(c.chainConfig, c.blocks[0].Hash(), uint64(c.Len()))
//}
//
//// Shorten returns a copy chain of a desired height from the imported
//func (c *TestChain) Shorten(height int) *Chain {
//	blocks := make([]*types.Block, height)
//	copy(blocks, c.blocks[:height])
//
//	config := *c.chainConfig
//	return &Chain{
//		blocks:      blocks,
//		chainConfig: &config,
//	}
//}
//
//// Head returns the chain head.
//func (c *TestChain) Head() *types.Block {
//	return c.blocks[c.Len()-1]
//}
//
//func (c *TestChain) GetHeaders(req GetBlockHeaders) (BlockHeaders, error) {
//	if req.Amount < 1 {
//		return nil, fmt.Errorf("no block headers requested")
//	}
//
//	headers := make(BlockHeaders, req.Amount)
//	var blockNumber uint64
//
//	// range over blocks to check if our chain has the requested header
//	for _, block := range c.blocks {
//		if block.Hash() == req.Origin.Hash || block.Nr() == req.Origin.Number {
//			headers[0] = block.Header()
//			blockNumber = block.Nr()
//		}
//	}
//	if headers[0] == nil {
//		return nil, fmt.Errorf("no headers found for given origin number %v, hash %v", req.Origin.Number, req.Origin.Hash)
//	}
//
//	if req.Reverse {
//		for i := 1; i < int(req.Amount); i++ {
//			blockNumber -= (1 - req.Skip)
//			headers[i] = c.blocks[blockNumber].Header()
//		}
//		return headers, nil
//	}
//
//	for i := 1; i < int(req.Amount); i++ {
//		blockNumber += (1 + req.Skip)
//		headers[i] = c.blocks[blockNumber].Header()
//	}
//
//	return headers, nil
//}
