// Copyright 2017 The go-ethereum Authors
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

package core

import (
	"math/big"
	"strconv"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDefaultGenesisBlock(t *testing.T) {
	gen := DefaultGenesisBlock()
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
	gen.Validators = depositData
	genBlock := gen.ToBlock(nil)
	if genBlock.Hash() != params.MainnetGenesisHash {
		t.Errorf("wrong mainnet genesis hash, got %v, want %v", genBlock.Hash(), params.MainnetGenesisHash)
	}

	gen = DefaultDevNetGenesisBlock()
	gen.Validators = depositData
	genBlock = gen.ToBlock(nil)
	if genBlock.Hash() != params.DevNetGenesisHash {
		t.Errorf("wrong wf test net genesis hash, got %v, want %v", genBlock.Hash(), params.DevNetGenesisHash)
	}
}

func TestSetupGenesis(t *testing.T) {
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

	var (
		customg = Genesis{
			Config:     params.AllEthashProtocolChanges,
			GasLimit:   1000000000000000000,
			Validators: depositData,
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
	)
	var wantHash = common.HexToHash("0x5ecbb10ee28cdd9ac2c280a5f1e57651fbc9e9e4c802c2e2aa00681e86a7681f")
	db := rawdb.NewMemoryDatabase()
	config, hash, err := SetupGenesisBlock(db, &customg)
	// Check the return values.
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, config, params.AllEthashProtocolChanges)
	testutils.AssertEqual(t, hash, wantHash)

	// Check database content.wantHashwantHash
	stored := rawdb.ReadBlock(db, wantHash)
	if stored.Hash() != wantHash {
		t.Errorf("block in DB has hash %s, want %s", stored.Hash(), wantHash)
	}

}

// TestGenesisHashes checks the congruity of default genesis data to corresponding hardcoded genesis hash values.
func TestGenesisHashes(t *testing.T) {
	cases := []struct {
		genesis *Genesis
		hash    common.Hash
	}{
		{
			genesis: DefaultGenesisBlock(),
			hash:    params.MainnetGenesisHash,
		},
		{
			genesis: DefaultDevNetGenesisBlock(),
			hash:    params.DevNetGenesisHash,
		},
	}
	for i, c := range cases {
		b := c.genesis.MustCommit(rawdb.NewMemoryDatabase())
		if got := b.Hash(); got != c.hash {
			t.Errorf("case: %d, want: %s, got: %s", i, c.hash.Hex(), got.Hex())
		}
	}
}

func TestSetupGenesisWithValidators(t *testing.T) {
	expectedHash := common.HexToHash("0x0dbc388cfc8dd97f1f505c911935fdbb0ad2c3e860836ccbbbcc5fe59ec29fd4")

	validators := make([]common.Address, 50)

	for i := 0; i < 50; i++ {
		validators[i] = common.HexToAddress(strconv.Itoa(i))
	}

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

	genesis := Genesis{
		Config: &params.ChainConfig{
			ChainID:                big.NewInt(333777222),
			SecondsPerSlot:         4,
			SlotsPerEpoch:          32,
			ForkSlotSubNet1:        1000,
			ValidatorsStateAddress: nil,
			EffectiveBalance:       big.NewInt(3200),
		},
		Timestamp:    0,
		ExtraData:    nil,
		GasLimit:     0,
		Coinbase:     common.Address{},
		Alloc:        nil,
		Validators:   depositData,
		GasUsed:      0,
		ParentHashes: nil,
		Slot:         0,
		Height:       0,
		BaseFee:      nil,
	}

	db := rawdb.NewMemoryDatabase()
	config, hash, err := SetupGenesisBlock(db, &genesis)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, hash, expectedHash)

	buf := make([]byte, len(validators)*common.AddressLength)
	for i, validator := range validators {
		beginning := i * common.AddressLength
		end := beginning + common.AddressLength
		copy(buf[beginning:end], validator[:])
	}

	validatorsStateAddress := crypto.Keccak256Address(buf)

	testutils.AssertEqual(t, *config.ValidatorsStateAddress, validatorsStateAddress)
}
