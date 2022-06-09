package token

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"testing"
)

var (
	operator     = common.HexToAddress("13e4acefe6a6700604929946e70e6443e4e73447")
	address      = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	spender      = common.HexToAddress("2cccf5e0538493c235d1c5ef6580f77d99e91396")
	owner        = common.HexToAddress("2e068e8bd9e38e801a592fc61118e66d29d1124c")
	to           = common.HexToAddress("7dc9c9730689ff0b0fd506c67db815f12d90a448")
	from         = common.HexToAddress("7986bad81f4cbd9317f5a46861437dae58d69113")
	value        = big.NewInt(2)
	id           = big.NewInt(12345)
	totalSupply  = big.NewInt(9999)
	index        = big.NewInt(20)
	decimals     = uint8(5)
	name         = []byte("Test Tokken")
	symbol       = []byte("TT")
	baseURI      = []byte("test.token.com")
	data         = []byte{243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5}
	addressToken = common.HexToAddress("4aed511BDB3b9577369DDa6F24961eBb6210fe0e")
	caller       = vm.AccountRef(owner)
)

var (
	stateDb       *state.StateDB
	p             *Processor
	wrc20Address  = common.Address{}
	wrc721Address = common.Address{}
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	ctx := vm.BlockContext{
		CanTransfer: nil,
		Transfer:    nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
	}

	p = NewProcessor(ctx, stateDb)
}

func TestProcessorCreateOperation(t *testing.T) {

	opCreateWrc721, err := NewWrc721CreateOperation(name, symbol, baseURI)
	if err != nil {
		t.Fatal(err)
	}

	opCreateWrc20, err := NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	if err != nil {
		t.Fatal(err)
	}

	wrc20Bytes, err := p.Call(caller, common.Address{}, opCreateWrc20)
	if err != nil {
		t.Fatal(err)
	}
	wrc20Address = common.BytesToAddress(wrc20Bytes)

	b, err := p.Call(caller, common.Address{}, opCreateWrc721)
	if err != nil {
		t.Fatal(err)
	}
	wrc721Address = common.BytesToAddress(b)
}
