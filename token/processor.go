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
	ErrNotEnoughBalance        = errors.New("transfer amount exceeds token balance")
	ErrInsufficientAllowance   = errors.New("insufficient allowance for token")
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

func (p *Processor) Call(caller Ref, token common.Address, op Operation) (ret []byte, err error) {
	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case CreateOperation:
		if addr, err := p.tokenCreate(caller, v); err == nil {
			ret = addr.Bytes()
		}
	case TransferFromOperation:
		ret, err = p.transferFrom(caller, token, v)
	case TransferOperation:
		ret, err = p.transfer(caller, token, v)
	case ApproveOperation:
		ret, err = p.approve(caller, token, v)
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
		v, _ := op.TotalSupply()
		storage.WriteUint256(v)
		// allowances
		storage.ReadMapSlot()
		// balances
		mapSlot := storage.ReadMapSlot()
		// Set balance for the caller
		addr := caller.Address()
		storage.WriteUint256ToMap(mapSlot, addr[:], v)
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
	storage, err := p.newStorage(op.Address(), op)
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

func (p *Processor) transfer(caller Ref, token common.Address, op TransferOperation) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	value := op.Value()
	switch op.Standard() {
	case StdWRC20:
		// name
		storage.SkipBytes()
		// symbol
		storage.SkipBytes()
		// decimals
		storage.SkipUint8()
		// totalSupply
		storage.SkipUint256()
		// allowances
		storage.ReadMapSlot()

		if err := p.wrc20Transfer(storage, caller.Address(), op.To(), op.Value()); err != nil {
			return nil, err
		}
	}

	log.Info("Transfer token", "address", token, "to", op.To(), "value", op.Value())
	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc20Transfer(storage *Storage, from common.Address, to common.Address, value *big.Int) error {
	mapSlot := storage.ReadMapSlot()

	fromBalance := storage.ReadUint256FromMap(mapSlot, from[:])

	if fromBalance.Cmp(value) >= 0 {
		fromRes := new(big.Int).Sub(fromBalance, value)
		storage.WriteUint256ToMap(mapSlot, from[:], fromRes)
	} else {
		return ErrNotEnoughBalance
	}

	toBalance := storage.ReadUint256FromMap(mapSlot, to[:])
	toRes := new(big.Int).Add(toBalance, value)
	storage.WriteUint256ToMap(mapSlot, to[:], toRes)

	return nil
}

func (p *Processor) wrc20SpendAllowance(storage *Storage, owner common.Address, spender common.Address, amount *big.Int) error {
	mapSlot := storage.ReadMapSlot()

	key := crypto.Keccak256(owner[:], spender[:])
	currentAllowance := storage.ReadUint256FromMap(mapSlot, key)
	if currentAllowance.Cmp(amount) >= 0 {
		allowance := new(big.Int).Sub(currentAllowance, amount)
		storage.WriteUint256ToMap(mapSlot, key, allowance)
	} else {
		return ErrInsufficientAllowance
	}

	return nil
}

func (p *Processor) transferFrom(caller Ref, token common.Address, op TransferFromOperation) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	value := op.Value()
	switch op.Standard() {
	case StdWRC20:
		// name
		storage.SkipBytes()
		// symbol
		storage.SkipBytes()
		// decimals
		storage.SkipUint8()
		// totalSupply
		storage.SkipUint256()

		if err := p.wrc20SpendAllowance(storage, op.From(), caller.Address(), op.Value()); err != nil {
			return nil, err
		}
		if err := p.wrc20Transfer(storage, op.From(), op.To(), op.Value()); err != nil {
			return nil, err
		}
	}

	log.Info("Transfer token", "address", token, "from", op.From(), "to", op.To(), "value", op.Value())
	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) approve(caller Ref, token common.Address, op ApproveOperation) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	owner := caller.Address()
	spender := op.Spender()
	value := op.Value()
	switch op.Standard() {
	case StdWRC20:
		// name
		storage.SkipBytes()
		// symbol
		storage.SkipBytes()
		// decimals
		storage.SkipUint8()
		// totalSupply
		storage.SkipUint256()

		mapSlot := storage.ReadMapSlot()
		key := crypto.Keccak256(owner[:], spender[:])
		storage.WriteUint256ToMap(mapSlot, key, value)
	}

	log.Info("Approve to spend a token", "owner", owner, "spender", spender, "value", value)
	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) BalanceOf(op BalanceOfOperation) (*big.Int, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(op.Address(), op)
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	switch standard {
	case StdWRC20:
		// name
		storage.SkipBytes()
		// symbol
		storage.SkipBytes()
		// decimals
		storage.SkipUint8()
		// totalSupply
		storage.SkipUint256()
		// allowances
		storage.ReadMapSlot()

		mapSlot := storage.ReadMapSlot()
		owner := op.Owner()
		balance = storage.ReadUint256FromMap(mapSlot, owner[:])
	}

	return balance, nil
}

func (p *Processor) Allowance(op AllowanceOperation) (*big.Int, error) {
	storage, err := p.newStorage(op.Address(), op)
	if err != nil {
		return nil, err
	}

	var allowance *big.Int
	switch op.Standard() {
	case StdWRC20:
		// name
		storage.SkipBytes()
		// symbol
		storage.SkipBytes()
		// decimals
		storage.SkipUint8()
		// totalSupply
		storage.SkipUint256()

		mapSlot := storage.ReadMapSlot()
		owner := op.Owner()
		spender := op.Spender()
		key := crypto.Keccak256(owner[:], spender[:])
		allowance = storage.ReadUint256FromMap(mapSlot, key)
	}

	return allowance, nil
}

func (p *Processor) newStorageWithoutStdCheck(token common.Address, op Operation) (*Storage, Std, error) {
	if !p.state.Exist(token) {
		log.Error("Token doesn't exist", "address", token)
		return nil, Std(0), ErrTokenNotExists
	}

	storage := NewStorage(token, p.state)
	std := storage.ReadUint16()
	if std == 0 {
		log.Error("Token doesn't exist", "address", token, "std", std)
		return nil, Std(0), ErrTokenNotExists
	}

	return storage, Std(std), nil
}

func (p *Processor) newStorage(token common.Address, op Operation) (*Storage, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(token, op)
	if err != nil {
		return nil, err
	}

	if standard != op.Standard() {
		log.Error("Token standard isn't valid for the operation", "address", token, "standard", standard, "opStandard", op.Standard())
		return nil, ErrTokenOpStandardNotValid
	}

	return storage, nil
}
