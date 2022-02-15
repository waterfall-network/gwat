package token

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrTokenAlreadyExists = errors.New("token address already exists")
	ErrTokenNotExists     = errors.New("token doesn't exist")
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
