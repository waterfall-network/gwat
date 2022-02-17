package token

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrTokenAlreadyExists      = errors.New("token address already exists")
	ErrTokenNotExists          = errors.New("token doesn't exist")
	ErrTokenOpStandardNotValid = errors.New("token standard isn't valid for the operation")
)

type Ref interface {
	Address() common.Address
}

type Processor struct {
	state vm.StateDB
	ctx   vm.BlockContext
}

func NewProcessor(blockCtx vm.BlockContext, statedb vm.StateDB) *Processor {
	return &Processor{
		ctx:   blockCtx,
		state: statedb,
	}
}

func (p *Processor) Call(caller Ref, op Operation) (ret []byte, err error) {
	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case CreateOperation:
		if addr, err := p.tokenCreate(caller, v); err == nil {
			ret = addr.Bytes()
		}
	}

	if err != nil {
		p.state.RevertToSnapshot(snapshot)
	}
	return ret, err
}

func (p *Processor) tokenCreate(caller Ref, op CreateOperation) (tokenAddr common.Address, err error) {
	tokenAddr = crypto.CreateAddress(caller.Address(), p.state.GetNonce(caller.Address()))
	if p.state.Exist(tokenAddr) {
		return common.Address{}, ErrTokenAlreadyExists
	}
	p.state.CreateAccount(tokenAddr)
	p.state.SetNonce(tokenAddr, 1)

	storage := NewStorage(tokenAddr, p.state)
	storage.WriteUint16(uint16(op.Standard()))
	storage.Write(op.Name())
	storage.Write(op.Symbol())

	switch op.Standard() {
	case StdWRC20:
		storage.WriteUint8(op.Decimals())
		if v, ok := op.TotalSupply(); ok {
			storage.WriteUint256(v)
		}
	case StdWRC721:
		if v, ok := op.BaseURI(); ok {
			storage.Write(v)
		}
	default:
		return common.Address{}, ErrStandardNotValid
	}

	log.Info("Create token", "address", tokenAddr)
	storage.Flush()

	return tokenAddr, nil
}

type WRC20PropertiesResult struct {
	Name        []byte
	Symbol      []byte
	Decimals    uint8
	TotalSupply *big.Int
}

type WRC721PropertiesResult struct {
	Name    []byte
	Symbol  []byte
	BaseURI []byte
}

func (p *Processor) Properties(op PropertiesOperation) (interface{}, error) {
	log.Info("Token properties", "address", op.Address())
	storage, err := p.newStorage(op)
	if err != nil {
		return nil, err
	}

	name := storage.ReadBytes()
	symbol := storage.ReadBytes()

	var r interface{}
	switch op.Standard() {
	case StdWRC20:
		decimals := storage.ReadUint8()
		totalSupply := storage.ReadUint256()

		r = &WRC20PropertiesResult{
			Name:        name,
			Symbol:      symbol,
			Decimals:    decimals,
			TotalSupply: totalSupply,
		}
	case StdWRC721:
		baseURI := storage.ReadBytes()

		r = &WRC721PropertiesResult{
			Name:    name,
			Symbol:  symbol,
			BaseURI: baseURI,
		}
	default:
		return nil, ErrStandardNotValid
	}

	return r, nil
}

func (p *Processor) transfer(op TransferOperation) ([]byte, error) {
	log.Info("Transfer token", "address", op.Address(), "to", op.To(), "value", op.Value())
	_, err := p.newStorage(op)
	if err != nil {
		return nil, err
	}

	switch op.Standard() {
	case StdWRC20:
	}

	return nil, err
}

func (p *Processor) newStorage(op Operation) (*Storage, error) {
	if !p.state.Exist(op.Address()) {
		log.Error("Token doesn't exist", "address", op.Address())
		return nil, ErrTokenNotExists
	}

	storage := NewStorage(op.Address(), p.state)
	std := storage.ReadUint16()
	if std == 0 {
		log.Error("Token doesn't exist", "address", op.Address(), "std", std)
		return nil, ErrTokenNotExists
	}
	standard := Std(std)
	if standard != op.Standard() {
		log.Error("Token standard isn't valid for the operation", "address", op.Address(), "standard", standard, "opStandard", op.Standard())
		return nil, ErrTokenOpStandardNotValid
	}

	return storage, nil
}
