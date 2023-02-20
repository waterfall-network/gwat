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
	"reflect"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDefaultGenesisBlock(t *testing.T) {
	block := DefaultGenesisBlock().ToBlock(nil)
	if block.Hash() != params.MainnetGenesisHash {
		t.Errorf("wrong mainnet genesis hash, got %v, want %v", block.Hash(), params.MainnetGenesisHash)
	}

	block = DefaultDevNetGenesisBlock().ToBlock(nil)
	if block.Hash() != params.DevNetGenesisHash {
		t.Errorf("wrong wf test net genesis hash, got %v, want %v", block.Hash(), params.DevNetGenesisHash)
	}
}

func TestInvalidCliqueConfig(t *testing.T) {
	block := DefaultDevNetGenesisBlock()
	block.ExtraData = []byte{}
	if _, err := block.Commit(nil); err == nil {
		t.Fatal("Expected error on invalid clique config")
	}
}

func TestSetupGenesis(t *testing.T) {
	var (
		customghash = common.HexToHash("0x7a07542cc52961f64bef9335a4d48f6ab4ea686aa67f1cd6b319209b83e384a9")
		customg     = Genesis{
			Config: &params.ChainConfig{},
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	oldcustomg.Config = &params.ChainConfig{}
	tests := []struct {
		name       string
		fn         func(ethdb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, new(Genesis))
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: params.AllEthashProtocolChanges,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "mainnet block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				DefaultGenesisBlock().MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "custom block in DB, genesis == devnet",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultDevNetGenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.DevNetGenesisHash},
			wantHash:   params.DevNetGenesisHash,
			wantConfig: params.DevNetChainConfig,
		},
	}

	for _, test := range tests {
		db := rawdb.NewMemoryDatabase()
		config, hash, err := test.fn(db)
		// Check the return values.
		if !reflect.DeepEqual(err, test.wantErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Errorf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := rawdb.ReadBlock(db, test.wantHash)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
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

	genesis := Genesis{
		Config: &params.ChainConfig{
			ChainID:                big.NewInt(333777222),
			SecondsPerSlot:         4,
			SlotsPerEpoch:          32,
			ForkSlotSubNet1:        1000,
			ValidatorsStateAddress: nil,
		},
		Timestamp:    0,
		ExtraData:    nil,
		GasLimit:     0,
		Coinbase:     common.Address{},
		Alloc:        nil,
		Validators:   validators,
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

	buf := make([]byte, len(genesis.Validators)*common.AddressLength)
	for i, validator := range genesis.Validators {
		beginning := i * common.AddressLength
		end := beginning + common.AddressLength
		copy(buf[beginning:end], validator[:])
	}

	validatorsStateAddress := crypto.Keccak256Address(buf)

	testutils.AssertEqual(t, *config.ValidatorsStateAddress, validatorsStateAddress)
}
