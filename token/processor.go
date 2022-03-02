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
	ErrAlreadyMinted           = errors.New("token has already minted")
	ErrNotMinted               = errors.New("token hasn't been minted")
	ErrIncorrectOwner          = errors.New("token isn't owned by the caller")
	ErrIncorrectTo             = errors.New("transfer to the zero address")
	ErrWrongCaller             = errors.New("caller is not owner nor approved")
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
	if _, ok := op.(CreateOperation); !ok {
		nonce := p.state.GetNonce(caller.Address())
		p.state.SetNonce(caller.Address(), nonce+1)
	}
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
	case MintOperation:
		ret, err = p.mint(caller, token, v)
	case BurnOperation:
		ret, err = p.burn(caller, token, v)
	case SetApprovalForAllOperation:
		ret, err = p.setApprovalForAll(caller, token, v)
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

	nonce := p.state.GetNonce(caller.Address())
	p.state.SetNonce(caller.Address(), nonce+1)

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
		} else {
			storage.Write([]byte{})
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
	Name        []byte
	Symbol      []byte
	BaseURI     []byte
	TokenURI    []byte
	OwnerOf     common.Address
	GetApproved common.Address
	Metadata    []byte
}

func (p *Processor) Properties(op PropertiesOperation) (interface{}, error) {
	log.Info("Token properties", "address", op.Address())
	storage, standard, err := p.newStorageWithoutStdCheck(op.Address(), op)
	if err != nil {
		return nil, err
	}

	name := storage.ReadBytes()
	symbol := storage.ReadBytes()

	var r interface{}
	switch standard {
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

		props := &WRC721PropertiesResult{
			Name:    name,
			Symbol:  symbol,
			BaseURI: baseURI,
		}
		if id, ok := op.TokenId(); ok {
			props.TokenURI = concatTokenURI(baseURI, id)
			owners := storage.ReadMapSlot()
			props.OwnerOf = storage.ReadAddressFromMap(owners, id.Bytes())
			if props.OwnerOf == (common.Address{}) {
				return nil, ErrNotMinted
			}

			// Skip balances
			storage.ReadMapSlot()
			metadata := storage.ReadMapSlot()
			props.Metadata = storage.ReadBytesFromMap(metadata, id.Bytes())

			approvals := storage.ReadMapSlot()
			props.GetApproved = storage.ReadAddressFromMap(approvals, id.Bytes())
		}

		r = props
	default:
		return nil, ErrStandardNotValid
	}

	return r, nil
}

func concatTokenURI(baseURI []byte, tokenId *big.Int) []byte {
	delim := byte('/')
	b := append(baseURI, delim)
	return append(b, tokenId.Bytes()...)
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
		log.Info("Transfer token", "address", token, "from", op.From(), "to", op.To(), "value", value)
	case StdWRC721:
		if err := p.wrc721TransferFrom(storage, caller, op); err != nil {
			return nil, err
		}
		log.Info("Transfer token", "address", token, "from", op.From(), "to", op.To(), "tokenId", value)
	}

	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc721TransferFrom(storage *Storage, caller Ref, op TransferFromOperation) error {
	owners, balances := p.prepareNftStorage(storage)

	address := caller.Address()
	tokenId := op.Value()
	owner := storage.ReadAddressFromMap(owners, tokenId.Bytes())

	if owner == (common.Address{}) {
		return ErrNotMinted
	}

	// metadata
	storage.ReadMapSlot()
	tokenApprovals := storage.ReadMapSlot()
	approved := storage.ReadAddressFromMap(tokenApprovals, tokenId.Bytes())

	operatorApprovals := storage.ReadMapSlot()
	key := crypto.Keccak256(owner[:], address[:])
	isApprovedForAll := storage.ReadBoolFromMap(operatorApprovals, key)
	if owner != address && approved != address && !isApprovedForAll {
		return ErrWrongCaller
	}

	to := op.To()
	if to == (common.Address{}) {
		return ErrIncorrectTo
	}

	// Clear approvals from the previous owner
	storage.WriteAddressToMap(tokenApprovals, tokenId.Bytes(), common.Address{})

	from := op.From()
	fromBalance := storage.ReadUint256FromMap(balances, from[:])
	newFromBalance := new(big.Int).Sub(fromBalance, big.NewInt(1))
	storage.WriteUint256ToMap(balances, from[:], newFromBalance)

	toBalance := storage.ReadUint256FromMap(balances, to[:])
	newToBalance := new(big.Int).Add(toBalance, big.NewInt(1))
	storage.WriteUint256ToMap(balances, to[:], newToBalance)

	storage.WriteAddressToMap(owners, tokenId.Bytes(), to)

	return nil
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

		log.Info("Approve to spend a token", "owner", owner, "spender", spender, "value", value)
	case StdWRC721:
		if err := p.wrc721Approve(storage, caller, op); err != nil {
			return nil, err
		}
		log.Info("Approve to spend an NFT", "owner", owner, "spender", spender, "tokenId", value)
	}

	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc721Approve(storage *Storage, caller Ref, op ApproveOperation) error {
	owners, _ := p.prepareNftStorage(storage)

	address := caller.Address()
	tokenId := op.Value()
	owner := storage.ReadAddressFromMap(owners, tokenId.Bytes())

	// metadata
	storage.ReadMapSlot()
	tokenApprovals := storage.ReadMapSlot()
	operatorApprovals := storage.ReadMapSlot()

	if owner == (common.Address{}) {
		return ErrNotMinted
	}

	key := crypto.Keccak256(owner[:], address[:])
	approvedForAll := storage.ReadBoolFromMap(operatorApprovals, key)
	if owner != address && !approvedForAll {
		return ErrWrongCaller
	}

	storage.WriteAddressToMap(tokenApprovals, tokenId.Bytes(), op.Spender())

	return nil
}

func (p *Processor) BalanceOf(op BalanceOfOperation) (*big.Int, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(op.Address(), op)
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	owner := op.Owner()
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
		balance = storage.ReadUint256FromMap(mapSlot, owner[:])
	case StdWRC721:
		_, balances := p.prepareNftStorage(storage)
		balance = storage.ReadUint256FromMap(balances, owner[:])
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

func (p *Processor) mint(caller Ref, token common.Address, op MintOperation) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}
	owners, balances := p.prepareNftStorage(storage)

	tokenId := op.TokenId()
	owner := storage.ReadAddressFromMap(owners, tokenId.Bytes())
	if owner != (common.Address{}) {
		return nil, ErrAlreadyMinted
	}

	to := op.To()
	balance := storage.ReadUint256FromMap(balances, to[:])
	newBalance := new(big.Int).Add(balance, big.NewInt(1))
	storage.WriteUint256ToMap(balances, to[:], newBalance)

	storage.WriteAddressToMap(owners, tokenId.Bytes(), to)

	if tokenMeta, ok := op.Metadata(); ok {
		metadata := storage.ReadMapSlot()
		storage.WriteBytesToMap(metadata, tokenId.Bytes(), tokenMeta[:])
	}

	log.Info("Token mint", "address", token, "to", to, "tokenId", tokenId)
	storage.Flush()

	return tokenId.Bytes(), nil
}

func (p *Processor) burn(caller Ref, token common.Address, op BurnOperation) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}
	owners, balances := p.prepareNftStorage(storage)

	address := caller.Address()
	tokenId := op.TokenId()
	owner := storage.ReadAddressFromMap(owners, tokenId.Bytes())
	if owner != address {
		return nil, ErrIncorrectOwner
	}

	balance := storage.ReadUint256FromMap(balances, address[:])
	newBalance := new(big.Int).Sub(balance, big.NewInt(1))
	storage.WriteUint256ToMap(balances, address[:], newBalance)

	// Empty value for the owner
	storage.WriteAddressToMap(owners, tokenId.Bytes(), common.Address{})
	// Empty metadata for the token
	metadata := storage.ReadMapSlot()
	storage.WriteBytesToMap(metadata, tokenId.Bytes(), []byte{})

	log.Info("Token burn", "address", token, "tokenId", tokenId)
	storage.Flush()

	return tokenId.Bytes(), nil
}

func (p *Processor) setApprovalForAll(caller Ref, token common.Address, op SetApprovalForAllOperation) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}
	p.prepareNftStorage(storage)
	owner := caller.Address()
	operator := op.Operator()

	// metadata
	storage.ReadMapSlot()
	// tokenApprovals
	storage.ReadMapSlot()
	operatorApprovals := storage.ReadMapSlot()

	key := crypto.Keccak256(owner[:], operator[:])
	storage.WriteBoolToMap(operatorApprovals, key, op.IsApproved())

	log.Info("Set approval for all WRC-721 tokens", "address", token, "owner", owner, "operator", operator)
	storage.Flush()

	return operator[:], nil
}

func (p *Processor) IsApprovedForAll(op IsApprovedForAllOperation) (bool, error) {
	storage, err := p.newStorage(op.Address(), op)
	if err != nil {
		return false, err
	}
	p.prepareNftStorage(storage)
	owner := op.Owner()
	operator := op.Operator()

	// metadata
	storage.ReadMapSlot()
	// tokenApprovals
	storage.ReadMapSlot()
	operatorApprovals := storage.ReadMapSlot()

	key := crypto.Keccak256(owner[:], operator[:])
	return storage.ReadBoolFromMap(operatorApprovals, key), nil
}

func (p *Processor) prepareNftStorage(storage *Storage) (owners common.Hash, balances common.Hash) {
	// name
	storage.SkipBytes()
	// symbol
	storage.SkipBytes()
	// baseURI
	storage.SkipBytes()

	owners = storage.ReadMapSlot()
	balances = storage.ReadMapSlot()
	return
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
