// Copyright 2014 The go-ethereum Authors
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
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/math"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus/misc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var (
	//go:embed genesis_json/mainnet/genesis.json
	mainnetGenesisJson []byte
	//go:embed genesis_json/mainnet/deposit_data.json
	mainnetDepositDataJson []byte
	//go:embed genesis_json/testnet8/genesis.json
	testnet8GenesisJson []byte
	//go:embed genesis_json/testnet8/deposit_data.json
	testnet8DepositDataJson []byte
)

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

type ValidatorData struct {
	Pubkey            string `json:"pubkey"`
	CreatorAddress    string `json:"creator_address"`
	WithdrawalAddress string `json:"withdrawal_address"`
	Amount            uint64 `json:"amount"`
}

type DepositData []*ValidatorData

func (d DepositData) Addresses() []common.Address {
	addresses := make([]common.Address, 0)
	uniqAddresses := make(map[common.Address]struct{})
	for _, data := range d {
		address := common.HexToAddress(data.CreatorAddress)
		if _, ok := uniqAddresses[address]; !ok {
			addresses = append(addresses, address)
			uniqAddresses[address] = struct{}{}
		}
	}

	return addresses
}

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`
	Validators DepositData

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	GasUsed      uint64        `json:"gasUsed"`
	ParentHashes []common.Hash `json:"parentHashes"`
	Slot         uint64        `json:"slot"`
	Height       uint64        `json:"height"`
	BaseFee      *big.Int      `json:"baseFeePerGas"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Timestamp    math.HexOrDecimal64
	ExtraData    hexutil.Bytes
	GasLimit     math.HexOrDecimal64
	GasUsed      math.HexOrDecimal64
	BaseFee      *math.HexOrDecimal256
	Alloc        map[common.UnprefixedAddress]GenesisAccount
	Validators   []common.Address
	ParentHashes []common.Hash
	Slot         math.HexOrDecimal64
	Height       math.HexOrDecimal64
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible genesis (have %x, new %x)", e.Stored, e.New)
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//	                     genesis == nil       genesis != nil
//	                  +------------------------------------------
//	db has no genesis |  main-net default  |  genesis
//	db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db ethdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	return SetupGenesisBlockWithOverride(db, genesis, nil, nil)
}

func SetupGenesisBlockWithOverride(db ethdb.Database, genesis *Genesis, delegatingStakeSlot, prefixFinSlot *uint64) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}
	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadFinalizedHashByNumber(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		if delegatingStakeSlot != nil {
			genesis.Config.ForkSlotDelegate = *delegatingStakeSlot
		}
		if prefixFinSlot != nil {
			genesis.Config.ForkSlotPrefixFin = *prefixFinSlot
		}
		block, err := genesis.Commit(db)
		log.Info("Writing custom genesis block", "hash", block.Hash().Hex(), "root", block.Root().Hex())
		if err != nil {
			return genesis.Config, common.Hash{}, err
		}
		return genesis.Config, block.Hash(), nil
	}
	// We have the genesis block in database(perhaps in ancient database)
	// but the corresponding state is missing.
	header := rawdb.ReadHeader(db, stored)
	if header == nil && stored != (common.Hash{}) {
		log.Crit("Incompatible database structure")
	}

	if _, err := state.New(header.Root, state.NewDatabaseWithConfig(db, nil), nil); err != nil {
		if genesis == nil {
			genesis = DefaultGenesisBlock()
		}
		// Ensure the stored genesis matches with the given one.
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, hash, err
		}
		return genesis.Config, block.Hash(), nil
	}
	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}
	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	if delegatingStakeSlot != nil {
		newcfg.ForkSlotDelegate = *delegatingStakeSlot
	}
	if prefixFinSlot != nil {
		newcfg.ForkSlotPrefixFin = *prefixFinSlot
	}
	if err := newcfg.CheckConfigForkOrder(); err != nil {
		return newcfg, common.Hash{}, err
	}
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		if storedcfg.ForkSlotDelegate == 0 {
			storedcfg.ForkSlotDelegate = newcfg.ForkSlotDelegate
		}
		if delegatingStakeSlot != nil {
			storedcfg.ForkSlotDelegate = *delegatingStakeSlot
		}
		if storedcfg.ForkSlotPrefixFin == 0 {
			storedcfg.ForkSlotPrefixFin = newcfg.ForkSlotPrefixFin
		}
		if prefixFinSlot != nil {
			storedcfg.ForkSlotPrefixFin = *prefixFinSlot
		}
		if storedcfg.ForkSlotSubNet1 == 0 {
			storedcfg.ForkSlotSubNet1 = newcfg.ForkSlotSubNet1
		}
		storedcfg.AcceptCpRootOnFinEpoch = newcfg.AcceptCpRootOnFinEpoch
		return storedcfg, stored, nil
	}
	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.

	lastFinHash := rawdb.ReadLastFinalizedHash(db)
	if lastFinHash == (common.Hash{}) {
		return newcfg, stored, fmt.Errorf("missing last finalized blocks for head")
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.Testnet8GenesisHash:
		return params.Testnet8ChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db ethdb.Database) *types.Block {
	if db == nil {
		db = rawdb.NewMemoryDatabase()
	}
	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	if err != nil {
		panic(err)
	}
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	head := &types.Header{
		Time:         g.Timestamp,
		ParentHashes: g.ParentHashes,
		Slot:         g.Slot,
		Height:       g.Height,
		GasLimit:     g.GasLimit,
		GasUsed:      g.GasUsed,
		BaseFee:      g.BaseFee,
		Coinbase:     g.Coinbase,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}

	validatorsStateAddress := g.GenerateValidatorStateAddress()

	if g.Config != nil {
		g.Config.ValidatorsStateAddress = validatorsStateAddress
	} else {
		g.Config = &params.ChainConfig{ValidatorsStateAddress: validatorsStateAddress}
	}

	head.BaseFee = misc.CalcSlotBaseFee(g.Config, g.Config.ValidatorsPerSlot, uint64(len(g.Validators.Addresses())), g.GasLimit, g.Slot)
	validatorStorage := valStore.NewStorage(g.Config)

	validatorStorage.SetValidatorsList(statedb, g.Validators.Addresses())
	for i, val := range g.Validators {
		pubKey := common.HexToBlsPubKey(val.Pubkey)
		address := common.HexToAddress(val.CreatorAddress)
		withdrawalAddress := common.HexToAddress(val.WithdrawalAddress)
		v := valStore.NewValidator(pubKey, address, &withdrawalAddress)
		v.SetIndex(uint64(i))
		v.SetActivationEra(uint64(0))

		err := validatorStorage.SetValidator(statedb, v)
		if err != nil {
			log.Error("can`t set validator from genesis", "error", err)
		}
	}

	root := statedb.IntermediateRoot(false)
	head.Root = root
	head.Era = 0

	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true, nil)

	genesisBlock := types.NewBlock(head, nil, nil, trie.NewStackTrie(nil))

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

	genesisEraLength := era.EstimateEraLength(g.Config, uint64(len(g.Validators)), 0)
	genesisEra := era.Era{
		Number:    0,
		From:      0,
		To:        genesisEraLength - 1,
		Root:      genesisBlock.Root(),
		BlockHash: genesisBlock.Hash(),
	}
	rawdb.WriteEra(db, genesisEra.Number, genesisEra)
	rawdb.WriteCurrentEra(db, genesisEra.Number)

	log.Info("Genesis era",
		"number", genesisEra.Number,
		"begin:", genesisEra.From,
		"end:", genesisEra.To,
		"root", genesisEra.Root.Hex(),
		"blockHash", genesisBlock.Hash().Hex(),
	)

	return genesisBlock
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db ethdb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Height() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		return nil, err
	}
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), nil)
	rawdb.WriteLastCanonicalHash(db, block.Hash())

	log.Info("Save genesis hash", "hash", block.Hash(), "fn", "Commit(genesis)")
	rawdb.WriteFinalizedHashNumber(db, block.Hash(), 0)
	rawdb.WriteLastFinalizedHash(db, block.Hash())
	rawdb.WriteHeadFastBlockHash(db, block.Hash())
	rawdb.AddSlotBlockHash(db, block.Slot(), block.Hash())
	//set genesis blockDag
	genesisDag := &types.BlockDAG{
		Hash:                   block.Hash(),
		Height:                 0,
		Slot:                   0,
		CpHash:                 block.Hash(),
		CpHeight:               0,
		OrderedAncestorsHashes: common.HashArray{},
	}
	rawdb.WriteBlockDag(db, genesisDag)
	rawdb.WriteTipsHashes(db, common.HashArray{genesisDag.Hash})
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db ethdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

func (g *Genesis) GenerateValidatorStateAddress() *common.Address {
	buf := make([]byte, len(g.Validators)*common.AddressLength)
	for i, validator := range g.Validators.Addresses() {
		beginning := i * common.AddressLength
		end := beginning + common.AddressLength
		copy(buf[beginning:end], validator[:])
	}

	validatorsStateAddress := crypto.Keccak256Address(buf)
	log.Info("Validators state address", "address", validatorsStateAddress)

	return &validatorsStateAddress
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db ethdb.Database, addr common.Address, balance *big.Int, depositData DepositData) *types.Block {
	g := Genesis{
		Config:     params.AllEthashProtocolChanges,
		Alloc:      GenesisAlloc{addr: {Balance: balance}},
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Validators: depositData,
		GasLimit:   1000000000000000000,
	}
	return g.MustCommit(db)
}

// DefaultGenesisBlock returns the Ethereum main net genesis block.
func DefaultGenesisBlock() *Genesis {
	genesis := new(Genesis)
	err := json.Unmarshal(mainnetGenesisJson, &genesis)
	if err != nil {
		log.Crit("Failed to unmarshal mainnet genesis data", "err", err)
	}
	depositData := make(DepositData, 0)
	err = json.Unmarshal(mainnetDepositDataJson, &depositData)
	if err != nil {
		log.Crit("Failed to unmarshal mainnet deposit data", "err", err)
	}
	if len(depositData) == 0 {
		log.Crit("Empty mainnet genesis data")
	}
	genesis.Validators = depositData
	if err = genesis.Config.Validate(); err != nil {
		log.Crit("Invalid mainnet genesis config", "err", err)
	}
	return genesis
}

// DefaultTestNet8GenesisBlock returns the tesnet8 network genesis block.
func DefaultTestNet8GenesisBlock() *Genesis {
	genesis := new(Genesis)
	err := json.Unmarshal(testnet8GenesisJson, &genesis)
	if err != nil {
		log.Crit("Failed to unmarshal testnet8 genesis data", "err", err)
	}
	depositData := make(DepositData, 0)
	err = json.Unmarshal(testnet8DepositDataJson, &depositData)
	if err != nil {
		log.Crit("Failed to unmarshal testnet8 deposit data", "err", err)
	}
	if len(depositData) == 0 {
		log.Crit("Empty testnet8 genesis data")
	}
	genesis.Validators = depositData
	if err = genesis.Config.Validate(); err != nil {
		log.Crit("Invalid testnet8 genesis config", "err", err)
	}
	return genesis
}

// DeveloperGenesisBlock returns the 'geth --dev' genesis block.
// Deprecated
func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllEthashProtocolChanges
	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:    &config,
		ExtraData: append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:  11500000,
		BaseFee:   big.NewInt(params.InitialBaseFee),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}
