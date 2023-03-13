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

package params

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"golang.org/x/crypto/sha3"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0x2d5a825666a4b49059a39c608e5fa1f4949b46ab7793853fc58307a54857743e")
	// DevNetGenesisHash  waterfall test net
	DevNetGenesisHash = common.HexToHash("0x733a3d6e8e0800197423933a03593b235d8a7af19049782af792775c378f25f1")
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of
// the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]*TrustedCheckpoint{
	MainnetGenesisHash: MainnetTrustedCheckpoint,
	DevNetGenesisHash:  DevNetTrustedCheckpoint,
}

// CheckpointOracles associates each known checkpoint oracles with the genesis hash of
// the chain it belongs to.
var CheckpointOracles = map[common.Hash]*CheckpointOracleConfig{
	MainnetGenesisHash: MainnetCheckpointOracle,
	DevNetGenesisHash:  DevNetCheckpointOracle,
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:          big.NewInt(111000111),
		SecondsPerSlot:   4,
		SlotsPerEpoch:    32,
		EpochsPerEra:     8,
		ForkSlotSubNet1:  math.MaxUint64,
		EffectiveBalance: big.NewInt(32000),
	}

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	MainnetTrustedCheckpoint = &TrustedCheckpoint{
		SectionIndex: 395,
		SectionHead:  common.HexToHash("0xbfca95b8c1de014e252288e9c32029825fadbff58285f5b54556525e480dbb5b"),
		CHTRoot:      common.HexToHash("0x2ccf3dbb58eb6375e037fdd981ca5778359e4b8fa0270c2878b14361e64161e7"),
		BloomRoot:    common.HexToHash("0x2d46ec65a6941a2dc1e682f8f81f3d24192021f492fdf6ef0fdd51acb0f4ba0f"),
	}

	// MainnetCheckpointOracle contains a set of configs for the main network oracle.
	MainnetCheckpointOracle = &CheckpointOracleConfig{
		Address: common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"),
		Signers: []common.Address{
			common.HexToAddress("0x1b2C260efc720BE89101890E4Db589b44E950527"), // Peter
			common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
			common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
			common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
			common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
		},
		Threshold: 2,
	}

	// DevNetChainConfig contains the chain parameters to run a node on the DevNet.
	DevNetChainConfig = &ChainConfig{
		ChainID:          big.NewInt(333777555),
		SecondsPerSlot:   4,
		SlotsPerEpoch:    32,
		EpochsPerEra:     8,
		ForkSlotSubNet1:  math.MaxUint64,
		EffectiveBalance: big.NewInt(32000),
	}

	// DevNetTrustedCheckpoint contains the light client trusted checkpoint for the DevNet.
	DevNetTrustedCheckpoint = &TrustedCheckpoint{
		//SectionIndex: 329,
		//SectionHead:  common.HexToHash("0xe66f7038333a01fb95dc9ea03e5a2bdaf4b833cdcb9e393b9127e013bd64d39b"),
		//CHTRoot:      common.HexToHash("0x1b0c883338ac0d032122800c155a2e73105fbfebfaa50436893282bc2d9feec5"),
		//BloomRoot:    common.HexToHash("0x3cc98c88d283bf002378246f22c653007655cbcea6ed89f98d739f73bd341a01"),
	}

	// DevNetCheckpointOracle contains a set of configs for the DevNet oracle.
	DevNetCheckpointOracle = &CheckpointOracleConfig{
		//Address: common.HexToAddress("0xEF79475013f154E6A65b54cB2742867791bf0B84"),
		//Signers: []common.Address{
		//	common.HexToAddress("0x32162F3581E88a5f62e8A61892B42C46E2c18f7b"), // Peter
		//	common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
		//	common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
		//	common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
		//	common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
		//},
		//Threshold: 2,
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{big.NewInt(1337), 4, 32, 8, math.MaxUint64, nil, 6, big.NewInt(32000)}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), 4, 32, 8, math.MaxUint64, nil, 6, big.NewInt(32000)}

	TestChainConfig = &ChainConfig{big.NewInt(1), 4, 32, 8, math.MaxUint64, nil, 6, big.NewInt(32000)}
	TestRules       = TestChainConfig.Rules()
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// HashEqual returns an indicator comparing the itself hash with given one.
func (c *TrustedCheckpoint) HashEqual(hash common.Hash) bool {
	if c.Empty() {
		return hash == common.Hash{}
	}
	return c.Hash() == hash
}

// Hash returns the hash of checkpoint's four key fields(index, sectionHead, chtRoot and bloomTrieRoot).
func (c *TrustedCheckpoint) Hash() common.Hash {
	var sectionIndex [8]byte
	binary.BigEndian.PutUint64(sectionIndex[:], c.SectionIndex)

	w := sha3.NewLegacyKeccak256()
	w.Write(sectionIndex[:])
	w.Write(c.SectionHead[:])
	w.Write(c.CHTRoot[:])
	w.Write(c.BloomRoot[:])

	var h common.Hash
	w.Sum(h[:0])
	return h
}

// Empty returns an indicator whether the checkpoint is regarded as empty.
func (c *TrustedCheckpoint) Empty() bool {
	return c.SectionHead == (common.Hash{}) || c.CHTRoot == (common.Hash{}) || c.BloomRoot == (common.Hash{})
}

// CheckpointOracleConfig represents a set of checkpoint contract(which acts as an oracle)
// config which used for light client checkpoint syncing.
type CheckpointOracleConfig struct {
	Address   common.Address   `json:"address"`
	Signers   []common.Address `json:"signers"`
	Threshold uint64           `json:"threshold"`
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection
	// coordinator slot settings
	SecondsPerSlot uint64 `json:"secondsPerSlot"`
	SlotsPerEpoch  uint64 `json:"slotsPerEpoch"`
	EpochsPerEra   uint64 `json:"epochsPerEra"`

	// Fork slots
	ForkSlotSubNet1 uint64 `json:"forkSlotSubNet1,omitempty"`

	ValidatorsStateAddress *common.Address
	ValidatorsPerSlot      uint64 `json:"validatorsPerSlot"`

	EffectiveBalance *big.Int `json:"effectiveBalance"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	return fmt.Sprintf("{ChainID: %v, SecondsPerSlot: %v, SlotsPerEpoch: %v, EpochsPerEra: %v, ForkSlotSubNet1: %v, "+
		"ValidatorsPerSlot %v, ValidatorsStateAddress %v, EffectiveBalance: %v}",
		c.ChainID,
		c.SecondsPerSlot,
		c.SlotsPerEpoch,
		c.EpochsPerEra,
		c.ForkSlotSubNet1,
		c.ValidatorsPerSlot,
		c.ValidatorsStateAddress,
		c.EffectiveBalance,
	)
}

// IsForkSlotSubNet1 returns true if provided slot greater or equal of the fork slot ForkSlotSubNet1.
func (c *ChainConfig) IsForkSlotSubNet1(slot uint64) bool {
	return slot >= c.ForkSlotSubNet1
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	type fork struct {
		name     string
		block    *big.Int
		optional bool // if true, the fork may be nil and next fork is still allowed
	}
	var lastFork fork
	for _, cur := range []fork{
		//{name: "homesteadBlock", block: c.HomesteadBlock},
		//{name: "daoForkBlock", block: c.DAOForkBlock, optional: true},
		//{name: "eip150Block", block: c.EIP150Block},
		//{name: "eip155Block", block: c.EIP155Block},
		//{name: "eip158Block", block: c.EIP158Block},
		//{name: "byzantiumBlock", block: c.ByzantiumBlock},
		//{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		//{name: "petersburgBlock", block: c.PetersburgBlock},
		//{name: "istanbulBlock", block: c.IstanbulBlock},
		//{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
		//{name: "berlinBlock", block: c.BerlinBlock},
		//{name: "londonBlock", block: c.LondonBlock},
	} {
		if lastFork.name != "" {
			// Next one must be higher number
			if lastFork.block == nil && cur.block != nil {
				return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %v",
					lastFork.name, cur.name, cur.block)
			}
			if lastFork.block != nil && cur.block != nil {
				if lastFork.block.Cmp(cur.block) > 0 {
					return fmt.Errorf("unsupported fork ordering: %v enabled at %v, but %v enabled at %v",
						lastFork.name, lastFork.block, cur.name, cur.block)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || cur.block != nil {
			lastFork = cur
		}
	}
	return nil
}

func (c *ChainConfig) Validate() error {
	if c.SecondsPerSlot == 0 {
		return fmt.Errorf("no seconds per slot parameter")
	}

	if c.SlotsPerEpoch == 0 {
		return fmt.Errorf("no slots per epoch parameter")
	}

	if c.EpochsPerEra == 0 {
		return fmt.Errorf("no epochs per era parameter")
	}

	if c.ForkSlotSubNet1 == 0 {
		return fmt.Errorf("no ForkSlotSubNet parameter")
	}

	if c.ValidatorsPerSlot == 0 {
		return fmt.Errorf("no validators per slot parameter")
	}

	if c.EffectiveBalance == nil || c.EffectiveBalance.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("no effective balance parameter")
	}

	return nil
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                                 *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158               bool
	IsByzantium, IsConstantinople, IsPetersburg, IsIstanbul bool
	IsBerlin, IsLondon                                      bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules() Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      true,
		IsEIP150:         true,
		IsEIP155:         true,
		IsEIP158:         true,
		IsByzantium:      true,
		IsConstantinople: true,
		IsPetersburg:     true,
		IsIstanbul:       true,
		IsBerlin:         true,
		IsLondon:         true,
	}
}
