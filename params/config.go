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
	MainnetGenesisHash = common.HexToHash("0x5c46071bf3056d26734af3696fb829ba87dbd24502cbfc3a20c5c9285e637ceb")
	// Testnet8GenesisHash  waterfall test net
	Testnet8GenesisHash = common.HexToHash("0xa7531d17d43684576b864662852e3cbb2dc20df7cdb9fc5405d5a0a253f623eb")
)

var (
	// testnet8AcceptCpRootOnFinEpoch fix sync finalization by hard define cp.finEpoch/cpRoot combo
	testnet8AcceptCpRootOnFinEpoch = map[common.Hash][]uint64{
		common.HexToHash("0xd76dd012c08baefd84750cf9752a6987dbc8ff4451069056cfd110e32250ed4a"): {96176},
		common.HexToHash("0x771d385d42fa93c0303ea34086f09d8adf99f97813f761aa6a437451959a0841"): {126283},
	}
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of
// the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]*TrustedCheckpoint{
	MainnetGenesisHash:  MainnetTrustedCheckpoint,
	Testnet8GenesisHash: TestNet8TrustedCheckpoint,
}

//// CheckpointOracles associates each known checkpoint oracles with the genesis hash of
//// the chain it belongs to.
//var CheckpointOracles = map[common.Hash]*CheckpointOracleConfig{
//	MainnetGenesisHash:  MainnetCheckpointOracle,
//	Testnet8GenesisHash: Testnet8CheckpointOracle,
//}

var (
	validatorsStateAddress = common.HexToAddress("0x329c3A3d65Ab0bE08c6eff6695933391Cfc02cCA")
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:                big.NewInt(181),
		SecondsPerSlot:         6,
		SlotsPerEpoch:          32,
		EpochsPerEra:           16,
		TransitionPeriod:       2,
		ValidatorsStateAddress: &validatorsStateAddress,
		ValidatorsPerSlot:      8,
		EffectiveBalance:       big.NewInt(32000),
		ValidatorOpExpireSlots: 14400,
		ForkSlotSubNet1:        math.MaxUint64,
		ForkSlotDelegate:       0,
		ForkSlotPrefixFin:      0,
		ForkSlotShanghai:       0,
		ForkSlotValOpTracking:  216000,
		ForkSlotReduceBaseFee:  216000,
		StartEpochsPerEra:      0,
	}

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	MainnetTrustedCheckpoint = &TrustedCheckpoint{
		//SectionIndex: 395,
		//SectionHead:  common.HexToHash("0xbfca95b8c1de014e252288e9c32029825fadbff58285f5b54556525e480dbb5b"),
		//CHTRoot:      common.HexToHash("0x2ccf3dbb58eb6375e037fdd981ca5778359e4b8fa0270c2878b14361e64161e7"),
		//BloomRoot:    common.HexToHash("0x2d46ec65a6941a2dc1e682f8f81f3d24192021f492fdf6ef0fdd51acb0f4ba0f"),
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

	// Testnet8ChainConfig contains the chain parameters to run a node on the Testnet8.
	Testnet8ChainConfig = &ChainConfig{
		ChainID:                big.NewInt(8601152),
		SecondsPerSlot:         4,
		SlotsPerEpoch:          32,
		EpochsPerEra:           16,
		TransitionPeriod:       2,
		ValidatorsStateAddress: nil,
		ValidatorsPerSlot:      5,
		EffectiveBalance:       big.NewInt(3200),
		ValidatorOpExpireSlots: 21600,
		ForkSlotSubNet1:        math.MaxUint64,
		ForkSlotDelegate:       2729920,
		ForkSlotPrefixFin:      4058240,
		ForkSlotShanghai:       math.MaxUint64,
		ForkSlotValOpTracking:  math.MaxUint64,
		ForkSlotReduceBaseFee:  math.MaxUint64,
		StartEpochsPerEra:      math.MaxUint64,
		AcceptCpRootOnFinEpoch: testnet8AcceptCpRootOnFinEpoch,
	}

	// TestNet8TrustedCheckpoint contains the light client trusted checkpoint for the Testnet8.
	TestNet8TrustedCheckpoint = &TrustedCheckpoint{
		//SectionIndex: 329,
		//SectionHead:  common.HexToHash("0xe66f7038333a01fb95dc9ea03e5a2bdaf4b833cdcb9e393b9127e013bd64d39b"),
		//CHTRoot:      common.HexToHash("0x1b0c883338ac0d032122800c155a2e73105fbfebfaa50436893282bc2d9feec5"),
		//BloomRoot:    common.HexToHash("0x3cc98c88d283bf002378246f22c653007655cbcea6ed89f98d739f73bd341a01"),
	}

	//// Testnet8CheckpointOracle contains a set of configs for the Testnet8 oracle.
	//Testnet8CheckpointOracle = &CheckpointOracleConfig{
	//	//Address: common.HexToAddress("0xEF79475013f154E6A65b54cB2742867791bf0B84"),
	//	//Signers: []common.Address{
	//	//	common.HexToAddress("0x32162F3581E88a5f62e8A61892B42C46E2c18f7b"), // Peter
	//	//	common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
	//	//	common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
	//	//	common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
	//	//	common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
	//	//},
	//	//Threshold: 2,
	//}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{
		ChainID:                big.NewInt(1337),
		SecondsPerSlot:         4,
		SlotsPerEpoch:          32,
		EpochsPerEra:           8,
		TransitionPeriod:       2,
		ValidatorsStateAddress: nil,
		ValidatorsPerSlot:      6,
		EffectiveBalance:       big.NewInt(3200),
		ValidatorOpExpireSlots: 14400,
		ForkSlotSubNet1:        math.MaxUint64,
		ForkSlotDelegate:       0,
		ForkSlotPrefixFin:      0,
		ForkSlotShanghai:       0,
		ForkSlotValOpTracking:  0,
		ForkSlotReduceBaseFee:  0,
		StartEpochsPerEra:      0,
	}

	TestChainConfig = &ChainConfig{
		ChainID:                big.NewInt(1337),
		SecondsPerSlot:         4,
		SlotsPerEpoch:          32,
		EpochsPerEra:           8,
		TransitionPeriod:       2,
		ValidatorsStateAddress: nil,
		ValidatorsPerSlot:      6,
		EffectiveBalance:       big.NewInt(3200),
		ValidatorOpExpireSlots: 14400,
		ForkSlotSubNet1:        math.MaxUint64,
		ForkSlotDelegate:       0,
		ForkSlotPrefixFin:      0,
		ForkSlotShanghai:       0,
		ForkSlotValOpTracking:  0,
		ForkSlotReduceBaseFee:  0,
		StartEpochsPerEra:      0,
	}
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
	SecondsPerSlot         uint64 `json:"secondsPerSlot"`
	SlotsPerEpoch          uint64 `json:"slotsPerEpoch"`
	EpochsPerEra           uint64 `json:"epochsPerEra"`
	TransitionPeriod       uint64 `json:"transitionPeriod"` // The number of epochs before new era starts
	ValidatorsStateAddress *common.Address
	ValidatorsPerSlot      uint64   `json:"validatorsPerSlot"`
	EffectiveBalance       *big.Int `json:"effectiveBalance"`
	ValidatorOpExpireSlots uint64   `json:"validatorOpExpireSlots"`
	// Fork slots
	ForkSlotSubNet1       uint64 `json:"forkSlotSubNet1,omitempty"`
	ForkSlotDelegate      uint64 `json:"forkSlotDelegate,omitempty"`
	ForkSlotPrefixFin     uint64 `json:"forkSlotPrefixFin,omitempty"`
	ForkSlotShanghai      uint64 `json:"forkSlotShanghai,omitempty"`
	ForkSlotValOpTracking uint64 `json:"forkSlotValOpTracking,omitempty"`
	ForkSlotReduceBaseFee uint64 `json:"forkSlotReduceBaseFee,omitempty"`
	// Fork eras
	StartEpochsPerEra uint64 `json:"startEpochsPerEra"`

	// fix sync finalization by hard define cp.finEpoch/cpRoot combo
	AcceptCpRootOnFinEpoch map[common.Hash][]uint64 `json:"acceptCpRootOnFinEpoch"`
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
	return fmt.Sprintf("{ChainID: %v, SecondsPerSlot: %v, SlotsPerEpoch: %v, EpochsPerEra: %v, TransitionPeriod: %v, "+
		"ValidatorsPerSlot %v, ValidatorsStateAddress %v, EffectiveBalance: %v, ValidatorOpExpireSlots: %v, ForkSlotSubNet1: %v, ForkSlotDelegate: %v, "+
		"ForkSlotPrefixFin: %v, ForkSlotShanghai: %v, ForkSlotValOpTracking: %v, ForkSlotReduceBaseFee: %v, StartEpochsPerEra: %v, AcceptCpRootOnFinEpoch: %v}",
		c.ChainID,
		c.SecondsPerSlot,
		c.SlotsPerEpoch,
		c.EpochsPerEra,
		c.TransitionPeriod,
		c.ValidatorsPerSlot,
		c.ValidatorsStateAddress,
		c.EffectiveBalance,
		c.ValidatorOpExpireSlots,
		c.ForkSlotSubNet1,
		c.ForkSlotDelegate,
		c.ForkSlotPrefixFin,
		c.ForkSlotShanghai,
		c.ForkSlotValOpTracking,
		c.ForkSlotReduceBaseFee,
		c.StartEpochsPerEra,
		c.AcceptCpRootOnFinEpoch,
	)
}

// IsForkSlotSubNet1 returns true if provided slot greater or equal of the fork slot ForkSlotSubNet1.
func (c *ChainConfig) IsForkSlotSubNet1(slot uint64) bool {
	return slot >= c.ForkSlotSubNet1
}

// IsForkSlotDelegate returns true if provided slot greater or equal of the fork slot ForkSlotDelegate.
func (c *ChainConfig) IsForkSlotDelegate(slot uint64) bool {
	return slot >= c.ForkSlotDelegate
}

// IsForkSlotPrefixFin returns true if provided slot greater or equal of the fork slot ForkSlotPrefixFin.
func (c *ChainConfig) IsForkSlotPrefixFin(slot uint64) bool {
	return slot >= c.ForkSlotPrefixFin
}

// IsForkSlotValOpTracking returns true if provided slot greater or equal of the fork slot ForkSlotValOpTracking.
func (c *ChainConfig) IsForkSlotValOpTracking(slot uint64) bool {
	return slot >= c.ForkSlotValOpTracking
}

// IsForkSlotReduceBaseFee returns true if provided slot greater or equal of the fork slot ForkSlotReduceBaseFee.
func (c *ChainConfig) IsForkSlotReduceBaseFee(slot uint64) bool {
	return slot >= c.ForkSlotReduceBaseFee
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	type fork struct {
		name     string
		slot     *big.Int
		optional bool // if true, the fork may be nil and next fork is still allowed
	}
	var lastFork fork
	for _, cur := range []fork{
		//{name: "forkSlotDelegate", slot: new(big.Int).SetUint64(c.ForkSlotDelegate)},
		//{name: "forkSlotPrefixFin", slot: new(big.Int).SetUint64(c.ForkSlotPrefixFin)},
	} {
		if lastFork.name != "" {
			// Next one must be higher number
			if lastFork.slot == nil && cur.slot != nil {
				return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %v",
					lastFork.name, cur.name, cur.slot)
			}
			if lastFork.slot != nil && cur.slot != nil {
				if lastFork.slot.Cmp(cur.slot) > 0 {
					return fmt.Errorf("unsupported fork ordering: %v enabled at %v, but %v enabled at %v",
						lastFork.name, lastFork.slot, cur.name, cur.slot)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || cur.slot != nil {
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

	// TODO: add ForkSlotSubnet parameter checking for subnet support

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
	IsBerlin, IsLondon, IsShanghai                          bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(slot uint64) Rules {
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
		IsShanghai:       c.ForkSlotShanghai <= slot,
	}
}

func (c *ChainConfig) HibernationSpinesThreshold() uint64 {
	return 128
}

// OverrideTestnet5 given conf while run a node with --testconf param.
func OverrideTestnet5(conf *ChainConfig) *ChainConfig {
	conf.ChainID = new(big.Int).SetInt64(1501865)
	//conf.SecondsPerSlot = 0
	//conf.SlotsPerEpoch = 0
	//conf.EpochsPerEra = 0
	//conf.TransitionPeriod = 0
	//conf.ValidatorsStateAddress = nil
	//conf.ValidatorsPerSlot = 0
	//conf.EffectiveBalance = nil
	conf.ValidatorOpExpireSlots = 100
	conf.ForkSlotSubNet1 = math.MaxUint64
	//conf.ForkSlotDelegate = 0
	//conf.ForkSlotPrefixFin = 0
	//conf.ForkSlotShanghai = 0
	conf.ForkSlotValOpTracking = 0
	conf.ForkSlotReduceBaseFee = 0
	//conf.StartEpochsPerEra = 0
	//conf.AcceptCpRootOnFinEpoch = nil

	return conf
}

// OverrideTestnet9 given conf while run a node with --testconf param.
func OverrideTestnet9(conf *ChainConfig) *ChainConfig {
	valsStateAddr := common.HexToAddress("0xc3653bd746859b94839c3ba0a8020febec009714")

	conf.ChainID = new(big.Int).SetInt64(1501869)
	conf.SecondsPerSlot = 6
	conf.SlotsPerEpoch = 32
	conf.EpochsPerEra = 4
	conf.TransitionPeriod = 2
	conf.ValidatorsStateAddress = &valsStateAddr
	conf.ValidatorsPerSlot = 5
	conf.EffectiveBalance = new(big.Int).SetInt64(32000)
	conf.ForkSlotSubNet1 = math.MaxUint64
	//conf.ForkSlotDelegate = 0
	//conf.ForkSlotPrefixFin = 0
	//conf.ForkSlotShanghai = 0
	conf.ValidatorOpExpireSlots = 320
	conf.ForkSlotValOpTracking = math.MaxUint64
	conf.ForkSlotReduceBaseFee = math.MaxUint64
	conf.StartEpochsPerEra = 0
	//conf.AcceptCpRootOnFinEpoch = nil

	return conf
}
