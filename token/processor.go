package token

import (
	"errors"
	"math/big"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core/vm"
	"github.com/waterfall-foundation/gwat/crypto"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/token/operation"
	tokenStorage "github.com/waterfall-foundation/gwat/token/storage"

	"github.com/holiman/uint256"
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
	ErrIncorrectTransferFrom   = errors.New("transfer from incorrect owner")
	ErrWrongCaller             = errors.New("caller is not owner nor approved")
	ErrWrongMinter             = errors.New("caller can't mint or burn NFTs")
	ErrNotNilTo                = errors.New("address to is not nil")
	ErrTokenIsNotForSale       = errors.New("token is not for sale")
	ErrTooSmallTxValue         = errors.New("too small transaction's value")
	ErrTokenIdIsNotSet         = errors.New("token id is not set")
	ErrNewValueIsNotSet        = errors.New("new value is not set")
	ErrUint256Overflow         = errors.New("value overflow")
	ErrTokenAddressCollision   = errors.New("token address collision")
	ErrMetadataExceedsMaxSize  = errors.New("metadata exceeds max size")
)

const (
	// 1024 bytes
	MetadataMaxSize = 1 << 10

	// Common fields
	// NameField is []byte
	NameField = "Name"
	// StandardField is []byte
	StandardField = "Standard"
	// SymbolField is []byte
	SymbolField = "Symbol"
	// BalancesField is AddressUint256Map
	BalancesField = "Balances"

	// WRC20
	// CreatorField is common.Address
	CreatorField = "Creator"
	// TotalSupplyField is Uint256
	TotalSupplyField = "TotalSupply"
	// DecimalsField is Uint8
	DecimalsField = "Decimals"
	// AllowancesField is AddressUint256Map
	AllowancesField = "Allowances"
	// CostField is Uint256
	CostField = "Cost"

	// WRC721
	// MinterField is common.Address
	MinterField = "Minter"
	// BaseUriField is []byte
	BaseUriField = "BaseUri"
	// OwnersField is AddressAddressMap
	OwnersField = "Owners"
	// MetadataField is AddressByteArrayMap
	MetadataField = "Metadata"
	// OperatorApprovalsField is KeccakBoolMap
	OperatorApprovalsField = "OperatorApprovals"
	// TokenApprovalsField is KeccakAddressMap
	TokenApprovalsField = "TokenApprovals"
	// CostMapField is Uint256Uint256Map
	CostMapField = "CostMap"
	// PercentFeeField is Uint8
	PercentFeeField = "PercentFee"
)

// Ref represents caller of the token processor
type Ref interface {
	Address() common.Address
}

// Processor is a processor of all token related operations.
// All transaction related operations that mutates state of the token are called using Call method.
// Methods of the operation name are used for getting state of the token.
type Processor struct {
	state vm.StateDB
	ctx   vm.BlockContext
}

// NewProcessor creates new token processor
func NewProcessor(blockCtx vm.BlockContext, stateDb vm.StateDB) *Processor {
	return &Processor{
		ctx:   blockCtx,
		state: stateDb,
	}
}

// Call performs all transaction related operations that mutates state of the token
//
// The only following operations can be performed using the method:
//  * token creation of WRC-20 or WRC-721 tokens
//  * transfer from
//  * transfer
//  * approve
//  * mint
//  * burn
//  * set approval for all
//	* buy
//	* setPrice
//
// It returns byte representation of the return value of an operation.
func (p *Processor) Call(caller Ref, token common.Address, value *big.Int, op operation.Operation) (ret []byte, err error) {
	if _, ok := op.(operation.Create); ok && token != (common.Address{}) {
		return nil, ErrNotNilTo
	}

	nonce := p.state.GetNonce(caller.Address())
	p.state.SetNonce(caller.Address(), nonce+1)

	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case operation.Create:
		var addr common.Address
		addr, err = p.tokenCreate(caller, v)
		if err == nil {
			ret = addr.Bytes()
		}
	case operation.TransferFrom:
		ret, err = p.transferFrom(caller, token, v)
	case operation.Transfer:
		ret, err = p.transfer(caller, token, v)
	case operation.Approve:
		ret, err = p.approve(caller, token, v)
	case operation.Mint:
		ret, err = p.mint(caller, token, v)
	case operation.Buy:
		ret, err = p.buy(caller, value, token, v)
	case operation.SetPrice:
		ret, err = p.setPrice(caller, token, v)
	case operation.Burn:
		ret, err = p.burn(caller, token, v)
	case operation.SetApprovalForAll:
		ret, err = p.setApprovalForAll(caller, token, v)
	}

	if err != nil {
		p.state.RevertToSnapshot(snapshot)
	}

	return ret, err
}

func (p *Processor) tokenCreate(caller Ref, op operation.Create) (tokenAddr common.Address, err error) {
	tokenAddr = crypto.CreateAddress(caller.Address(), p.state.GetNonce(caller.Address()))
	if p.state.Exist(tokenAddr) {
		return common.Address{}, ErrTokenAlreadyExists
	}

	if p.state.GetNonce(tokenAddr) != 0 {
		return common.Address{}, ErrTokenAddressCollision
	}

	p.state.CreateAccount(tokenAddr)
	p.state.SetNonce(tokenAddr, 1)

	fieldsDescriptors, err := newFieldsDescriptors(op)
	if err != nil {
		return common.Address{}, err
	}

	storage, err := tokenStorage.NewStorage(tokenStorage.NewStorageStream(tokenAddr, p.state), fieldsDescriptors)
	if err != nil {
		return common.Address{}, err
	}

	stdB, err := op.Standard().MarshalBinary()
	if err != nil {
		return common.Address{}, err
	}

	err = storage.WriteField(StandardField, stdB)
	if err != nil {
		return common.Address{}, err
	}

	err = storage.WriteField(NameField, op.Name())
	if err != nil {
		return common.Address{}, err
	}

	err = storage.WriteField(SymbolField, op.Symbol())
	if err != nil {
		return common.Address{}, err
	}

	switch op.Standard() {
	case operation.StdWRC20:
		err = storage.WriteField(CreatorField, caller.Address())
		if err != nil {
			return common.Address{}, err
		}

		err = storage.WriteField(DecimalsField, op.Decimals())
		if err != nil {
			return common.Address{}, err
		}

		v, _ := op.TotalSupply()
		val, ok := uint256.FromBig(v)
		if ok {
			return common.Address{}, ErrUint256Overflow
		}

		err = storage.WriteField(TotalSupplyField, val)
		if err != nil {
			return common.Address{}, err
		}

		addr := caller.Address()
		err = writeToMap(storage, BalancesField, addr[:], val)
		if err != nil {
			return common.Address{}, err
		}
	case operation.StdWRC721:
		err = storage.WriteField(MinterField, caller.Address())
		if err != nil {
			return common.Address{}, err
		}

		v, ok := op.BaseURI()
		if !ok {
			v = []byte{}
		}

		err = storage.WriteField(BaseUriField, v)
		if err != nil {
			return common.Address{}, err
		}

		err = storage.WriteField(PercentFeeField, op.PercentFee())
		if err != nil {
			return common.Address{}, err
		}
	default:
		return common.Address{}, operation.ErrStandardNotValid
	}

	log.Info("Create token", "address", tokenAddr)
	storage.Flush()

	return tokenAddr, nil
}

// WRC20PropertiesResult stores result of the properties operation for WRC-20 tokens
type WRC20PropertiesResult struct {
	Std         operation.Std
	Name        []byte
	Symbol      []byte
	Decimals    uint8
	TotalSupply *big.Int
}

// WRC721PropertiesResult stores result of the properties operation for WRC-721 tokens
type WRC721PropertiesResult struct {
	Std         operation.Std
	Name        []byte
	Symbol      []byte
	BaseURI     []byte
	TokenURI    []byte
	OwnerOf     common.Address
	GetApproved common.Address
	Metadata    []byte
	PercentFee  uint8
	Cost        *big.Int
}

// Properties performs the token properties operation
// It returns WRC20PropertiesResult or WRC721PropertiesResult according to the token type.
func (p *Processor) Properties(op operation.Properties) (interface{}, error) {
	log.Info("Token properties", "address", op.Address())

	storage, standard, err := p.newStorageWithoutStdCheck(op.Address())
	if err != nil {
		return nil, err
	}

	var name, symbol []byte
	err = storage.ReadField(NameField, &name)
	if err != nil {
		return nil, err
	}

	err = storage.ReadField(SymbolField, &symbol)
	if err != nil {
		return nil, err
	}

	var r interface{}
	switch standard {
	case operation.StdWRC20:
		var decimals uint8
		err = storage.ReadField(DecimalsField, &decimals)
		if err != nil {
			return nil, err
		}

		var totalSupply uint256.Int
		err = storage.ReadField(TotalSupplyField, &totalSupply)
		if err != nil {
			return nil, err
		}

		r = &WRC20PropertiesResult{
			Std:         operation.StdWRC20,
			Name:        name,
			Symbol:      symbol,
			Decimals:    decimals,
			TotalSupply: totalSupply.ToBig(),
		}
	case operation.StdWRC721:
		var baseURI []byte
		err = storage.ReadField(BaseUriField, &baseURI)
		if err != nil {
			return nil, err
		}

		var percentFee uint8
		err = storage.ReadField(PercentFeeField, &percentFee)
		if err != nil {
			return nil, err
		}

		props := &WRC721PropertiesResult{
			Std:        operation.StdWRC721,
			Name:       name,
			Symbol:     symbol,
			BaseURI:    baseURI,
			PercentFee: percentFee,
		}
		if id, ok := op.TokenId(); ok {
			props.TokenURI = concatTokenURI(baseURI, id)

			props.OwnerOf, err = readAddressFromMap(storage, OwnersField, id.Bytes())
			if err != nil {
				return nil, err
			}

			if props.OwnerOf == (common.Address{}) {
				return nil, ErrNotMinted
			}

			err = readFromMap(storage, MetadataField, id.Bytes(), &props.Metadata)
			if err != nil {
				return nil, err
			}

			props.GetApproved, err = readAddressFromMap(storage, TokenApprovalsField, id.Bytes())
			if err != nil {
				return nil, err
			}
		}

		r = props
	default:
		return nil, operation.ErrStandardNotValid
	}

	return r, nil
}

func concatTokenURI(baseURI []byte, tokenId *big.Int) []byte {
	delim := byte('/')
	b := append(baseURI, delim)
	id := []byte(tokenId.String())
	return append(b, id...)
}

func (p *Processor) transfer(caller Ref, token common.Address, op operation.Transfer) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, operation.ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	value := op.Value()
	switch op.Standard() {
	case operation.StdWRC20:
		err = transfer(storage, caller.Address(), op.To(), op.Value())
		if err != nil {
			return nil, err
		}
	}

	log.Info("Transfer token", "address", token, "to", op.To(), "value", op.Value())
	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc20SpendAllowance(storage tokenStorage.Storage, owner common.Address, spender common.Address, amount *big.Int) error {
	var currentAllowance uint256.Int
	key := crypto.Keccak256(owner[:], spender[:])
	err := readFromMap(storage, AllowancesField, key, &currentAllowance)
	if err != nil {
		return err
	}

	ca := currentAllowance.ToBig()
	if ca.Cmp(amount) < 0 {
		return ErrInsufficientAllowance
	}

	allowance, _ := uint256.FromBig(new(big.Int).Sub(ca, amount))
	return writeToMap(storage, AllowancesField, key, allowance)
}

func (p *Processor) transferFrom(caller Ref, token common.Address, op operation.TransferFrom) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, operation.ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	value := op.Value()
	switch op.Standard() {
	case operation.StdWRC20:
		err = p.wrc20SpendAllowance(storage, op.From(), caller.Address(), op.Value())
		if err != nil {
			return nil, err
		}

		err := transfer(storage, op.From(), op.To(), op.Value())
		if err != nil {
			return nil, err
		}

		log.Info("Transfer token", "address", token, "from", op.From(), "to", op.To(), "value", value)
	case operation.StdWRC721:
		if err := p.wrc721TransferFrom(storage, caller, op); err != nil {
			return nil, err
		}
		log.Info("Transfer token", "address", token, "from", op.From(), "to", op.To(), "tokenId", value)
	}

	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc721TransferFrom(storage tokenStorage.Storage, caller Ref, op operation.TransferFrom) error {
	address := caller.Address()
	tokenId := op.Value()

	owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
	if err != nil {
		return err
	}
	if owner == (common.Address{}) {
		return ErrNotMinted
	}

	tokenApprovals, err := readAddressFromMap(storage, TokenApprovalsField, tokenId.Bytes())
	if err != nil {
		return err
	}

	isApprovedForAll := false
	err = readFromMap(storage, OperatorApprovalsField, crypto.Keccak256(owner[:], address[:]), &isApprovedForAll)
	if err != nil {
		return err
	}

	if owner != address && tokenApprovals != address && !isApprovedForAll {
		return ErrWrongCaller
	}

	to := op.To()
	if to == (common.Address{}) {
		return ErrIncorrectTo
	}

	from := op.From()
	if from != owner {
		return ErrIncorrectTransferFrom
	}

	// Clear approvals from the previous owner
	err = writeToMap(storage, TokenApprovalsField, tokenId.Bytes(), common.Address{})
	if err != nil {
		return err
	}

	err = transfer(storage, from, to, big.NewInt(1))
	if err != nil {
		return err
	}

	return writeToMap(storage, OwnersField, tokenId.Bytes(), to)
}

func (p *Processor) approve(caller Ref, token common.Address, op operation.Approve) ([]byte, error) {
	if token == (common.Address{}) {
		return nil, operation.ErrNoAddress
	}

	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	owner := caller.Address()
	spender := op.Spender()
	value := op.Value()
	switch op.Standard() {
	case operation.StdWRC20:
		key := crypto.Keccak256(owner[:], spender[:])
		v, ok := uint256.FromBig(op.Value())
		if ok {
			return nil, ErrUint256Overflow
		}

		err = writeToMap(storage, AllowancesField, key, v)
		if err != nil {
			return nil, err
		}

		log.Info("Approve to spend a token", "owner", owner, "spender", spender, "value", value)
	case operation.StdWRC721:
		if err := p.wrc721Approve(storage, caller, op); err != nil {
			return nil, err
		}
		log.Info("Approve to spend an NFT", "owner", owner, "spender", spender, "tokenId", value)
	}

	storage.Flush()

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) wrc721Approve(storage tokenStorage.Storage, caller Ref, op operation.Approve) error {
	address := caller.Address()
	tokenId := op.Value()

	owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
	if err != nil {
		return err
	}
	if owner == (common.Address{}) {
		return ErrNotMinted
	}

	approvedForAll := false
	err = readFromMap(storage, OperatorApprovalsField, crypto.Keccak256(owner[:], address[:]), &approvedForAll)
	if err != nil {
		return err
	}

	if owner != address && !approvedForAll {
		return ErrWrongCaller
	}

	return writeToMap(storage, TokenApprovalsField, tokenId.Bytes(), op.Spender())
}

// BalanceOf performs the token balance of operations
// It returns uint256 value with the token balance of number of NFTs.
func (p *Processor) BalanceOf(op operation.BalanceOf) (*big.Int, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(op.Address())
	if err != nil {
		return nil, err
	}

	owner := op.Owner()
	switch standard {
	case operation.StdWRC20:
		fallthrough
	case operation.StdWRC721:
		var balance uint256.Int
		err = readFromMap(storage, BalancesField, owner[:], &balance)
		if err != nil {
			return nil, err
		}

		return balance.ToBig(), nil
	default:
		return nil, ErrTokenOpStandardNotValid
	}
}

// Allowance performs the token allowance operation
// It returns uint256 value with allowed count of the token to spend.
// The method only works for WRC-20 tokens.
func (p *Processor) Allowance(op operation.Allowance) (*big.Int, error) {
	storage, err := p.newStorage(op.Address(), op)
	if err != nil {
		return nil, err
	}

	switch op.Standard() {
	case operation.StdWRC20:
		owner := op.Owner()
		spender := op.Spender()

		var allowance uint256.Int
		key := crypto.Keccak256(owner[:], spender[:])
		err = readFromMap(storage, AllowancesField, key, &allowance)
		if err != nil {
			return nil, err
		}

		return allowance.ToBig(), nil
	}

	return nil, ErrTokenOpStandardNotValid
}

// Cost performs the token cost of operations
// It returns uint256 value with the token cost in wei.
func (p *Processor) Cost(op operation.Cost) (*big.Int, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(op.Address())
	if err != nil {
		return nil, err
	}

	cost := new(uint256.Int)
	switch standard {
	case operation.StdWRC20:
		err = storage.ReadField(CostField, cost)
		if err != nil {
			return nil, err
		}
	case operation.StdWRC721:
		tokenId, ok := op.TokenId()
		if !ok {
			return nil, ErrTokenIdIsNotSet
		}

		owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
		if err != nil {
			return nil, err
		}
		if owner == (common.Address{}) {
			return nil, ErrNotMinted
		}

		id, ok := uint256.FromBig(tokenId)
		if ok {
			return nil, ErrUint256Overflow
		}

		err = readFromMap(storage, CostMapField, id, cost)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrTokenOpStandardNotValid
	}

	if cost.Eq(uint256.NewInt(0)) {
		return nil, ErrTokenIsNotForSale
	}

	return cost.ToBig(), nil
}

func (p *Processor) mint(caller Ref, token common.Address, op operation.Mint) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	minter, err := readAddress(storage, MinterField)
	if err != nil {
		return nil, err
	}

	if caller.Address() != minter {
		return nil, ErrWrongMinter
	}

	tokenId := op.TokenId()
	owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
	if err != nil {
		return nil, err
	}
	if owner != (common.Address{}) {
		return nil, ErrAlreadyMinted
	}

	to := op.To()
	var balance uint256.Int
	err = readFromMap(storage, BalancesField, to[:], &balance)
	if err != nil {
		return nil, err
	}

	newBalance, ok := uint256.FromBig(new(big.Int).Add(balance.ToBig(), big.NewInt(1)))
	if ok {
		return nil, ErrUint256Overflow
	}

	err = writeToMap(storage, BalancesField, to[:], newBalance)
	if err != nil {
		return nil, err
	}

	err = writeToMap(storage, OwnersField, tokenId.Bytes(), to[:])
	if err != nil {
		return nil, err
	}

	if tokenMeta, ok := op.Metadata(); ok {
		if len(tokenMeta) > MetadataMaxSize {
			return nil, ErrMetadataExceedsMaxSize
		}

		err = writeToMap(storage, MetadataField, tokenId.Bytes(), tokenMeta[:])
		if err != nil {
			return nil, err
		}
	}

	log.Info("Token mint", "address", token, "to", to, "tokenId", tokenId)
	storage.Flush()

	return tokenId.Bytes(), nil
}

func (p *Processor) burn(caller Ref, token common.Address, op operation.Burn) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	address := caller.Address()
	tokenId := op.TokenId()

	minter, err := readAddress(storage, MinterField)
	if err != nil {
		return nil, err
	}

	if address != minter {
		return nil, ErrWrongMinter
	}

	owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
	if err != nil {
		return nil, err
	}

	if owner != address {
		return nil, ErrIncorrectOwner
	}

	var balance uint256.Int
	err = readFromMap(storage, BalancesField, owner[:], &balance)
	if err != nil {
		return nil, err
	}

	newBalance, ok := uint256.FromBig(new(big.Int).Sub(balance.ToBig(), big.NewInt(1)))
	if ok {
		return nil, ErrUint256Overflow
	}

	err = writeToMap(storage, BalancesField, owner[:], newBalance)
	if err != nil {
		return nil, err
	}

	// Empty value for the owner
	err = writeToMap(storage, OwnersField, tokenId.Bytes(), []byte{})
	if err != nil {
		return nil, err
	}

	// Empty metadata for the token
	err = writeToMap(storage, MetadataField, tokenId.Bytes(), []byte{})
	if err != nil {
		return nil, err
	}

	log.Info("Token burn", "address", token, "tokenId", tokenId)
	storage.Flush()

	return tokenId.Bytes(), nil
}

func (p *Processor) setApprovalForAll(caller Ref, token common.Address, op operation.SetApprovalForAll) ([]byte, error) {
	storage, err := p.newStorage(token, op)
	if err != nil {
		return nil, err
	}

	owner := caller.Address()
	operator := op.Operator()

	err = writeToMap(storage, OperatorApprovalsField, crypto.Keccak256(owner[:], operator[:]), op.IsApproved())
	if err != nil {
		return nil, err
	}

	log.Info("Set approval for all WRC-721 tokens", "address", token, "owner", owner, "operator", operator)
	storage.Flush()

	return operator[:], nil
}

func (p *Processor) setPrice(caller Ref, token common.Address, op operation.SetPrice) ([]byte, error) {
	storage, std, err := p.newStorageWithoutStdCheck(token)
	if err != nil {
		return nil, err
	}

	newCost, ok := uint256.FromBig(op.Value())
	if ok {
		return nil, ErrUint256Overflow
	}

	switch std {
	case operation.StdWRC20:
		creator, err := readAddress(storage, CreatorField)
		if err != nil {
			return nil, err
		}

		if caller.Address() != creator {
			return nil, ErrIncorrectOwner
		}

		err = storage.WriteField(CostField, newCost)
		if err != nil {
			return nil, err
		}
	case operation.StdWRC721:
		tokenId, ok := op.TokenId()
		if !ok {
			return nil, ErrTokenIdIsNotSet
		}

		owner, err := readAddressFromMap(storage, OwnersField, tokenId.Bytes())
		if err != nil {
			return nil, err
		}

		if owner == (common.Address{}) {
			return nil, ErrNotMinted
		}

		if owner != caller.Address() {
			return nil, ErrIncorrectOwner
		}

		id, ok := uint256.FromBig(tokenId)
		if ok {
			return nil, ErrUint256Overflow
		}

		err = writeToMap(storage, CostMapField, id, newCost)
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrTokenOpStandardNotValid
	}

	storage.Flush()
	return token.Bytes(), nil
}

func (p *Processor) buy(caller Ref, value *big.Int, token common.Address, op operation.Buy) ([]byte, error) {
	storage, std, err := p.newStorageWithoutStdCheck(token)
	if err != nil {
		return nil, err
	}

	percentFee := uint8(0)
	transferFrom := common.Address{}
	transferTo := caller.Address()
	transferValue, paymentValue := big.NewInt(0), big.NewInt(0)
	switch std {
	case operation.StdWRC20:
		transferFrom, err = readAddress(storage, CreatorField)
		if err != nil {
			return nil, err
		}

		cost := new(uint256.Int)
		err = storage.ReadField(CostField, cost)
		if err != nil {
			return nil, err
		}

		if cost.Eq(uint256.NewInt(0)) {
			return nil, ErrTokenIsNotForSale
		}

		if cost.ToBig().Cmp(value) > 0 {
			return nil, ErrTooSmallTxValue
		}

		reminder := big.NewInt(0).Mod(value, cost.ToBig())
		paymentValue.Sub(value, reminder)
		transferValue.Div(paymentValue, cost.ToBig())
	case operation.StdWRC721:
		tokenId, ok := op.TokenId()
		if !ok {
			return nil, ErrTokenIdIsNotSet
		}

		id, ok := uint256.FromBig(tokenId)
		if ok {
			return nil, ErrUint256Overflow
		}

		newCost, ok := op.NewCost()
		if !ok {
			return nil, ErrNewValueIsNotSet
		}

		// check if token exist
		var err error
		transferFrom, err = readAddressFromMap(storage, OwnersField, tokenId.Bytes())
		if err != nil {
			return nil, err
		}
		if transferFrom == (common.Address{}) {
			return nil, ErrNotMinted
		}

		// check cost
		cost := new(uint256.Int)
		err = readFromMap(storage, CostMapField, id, cost)
		if err != nil {
			return nil, err
		}

		if cost.Eq(uint256.NewInt(0)) {
			return nil, ErrTokenIsNotForSale
		}

		if cost.ToBig().Cmp(value) == 1 {
			return nil, ErrTooSmallTxValue
		}

		err = writeToMap(storage, OwnersField, tokenId.Bytes(), transferTo)
		if err != nil {
			return nil, err
		}

		// clear approvals from the previous owner
		err = writeToMap(storage, TokenApprovalsField, tokenId.Bytes(), common.Address{})
		if err != nil {
			return nil, err
		}

		// set new cost
		newCostUint, ok := uint256.FromBig(newCost)
		if ok {
			return nil, ErrUint256Overflow
		}

		err = writeToMap(storage, CostMapField, id, newCostUint)
		if err != nil {
			return nil, err
		}

		// get percentFee
		err = storage.ReadField(PercentFeeField, &percentFee)
		if err != nil {
			return nil, err
		}

		transferValue.SetInt64(1)
		paymentValue.Set(cost.ToBig())
	default:
		return nil, ErrTokenOpStandardNotValid
	}

	err = transfer(storage, transferFrom, transferTo, transferValue)
	if err != nil {
		return nil, err
	}

	err = p.makePayment(storage, transferTo, transferFrom, paymentValue, percentFee)
	if err != nil {
		return nil, err
	}

	storage.Flush()
	return token.Bytes(), nil
}

func (p *Processor) makePayment(storage tokenStorage.Storage, caller, owner common.Address, value *big.Int, percentFee uint8) error {
	// check balance
	callerBalance := p.state.GetBalance(caller)
	if callerBalance.Cmp(value) < 0 {
		return ErrNotEnoughBalance
	}

	if value.Cmp(big.NewInt(0)) == 0 {
		return nil
	}

	// only for WRC-721
	if percentFee > 0 {
		minter, err := readAddress(storage, MinterField)
		if err != nil {
			return err
		}

		if owner != minter {
			fee := big.NewInt(int64(percentFee))
			fee = fee.Mul(fee, value)
			fee = fee.Div(fee, big.NewInt(100))

			// take fee
			p.state.SubBalance(caller, fee)
			p.state.AddBalance(minter, fee)

			value = value.Sub(value, fee)
		}
	}

	// take payment
	p.state.SubBalance(caller, value)
	p.state.AddBalance(owner, value)

	return nil
}

// IsApprovedForAll performs the is approved for all operation for WRC-721 tokens
// Returns boolean value that indicates whether the operator can perform any operation on the token.
func (p *Processor) IsApprovedForAll(op operation.IsApprovedForAll) (bool, error) {
	storage, err := p.newStorage(op.Address(), op)
	if err != nil {
		return false, err
	}

	owner := op.Owner()
	operator := op.Operator()

	isApprovedForAll := false
	return isApprovedForAll, readFromMap(storage, OperatorApprovalsField, crypto.Keccak256(owner[:], operator[:]), &isApprovedForAll)
}

func (p *Processor) newStorageWithoutStdCheck(token common.Address) (tokenStorage.Storage, operation.Std, error) {
	if !p.state.Exist(token) {
		log.Error("Token doesn't exist", "address", token)
		return nil, operation.Std(0), ErrTokenNotExists
	}

	storage, err := tokenStorage.ReadStorage(tokenStorage.NewStorageStream(token, p.state))
	if err != nil {
		return nil, operation.Std(0), err
	}

	var stdB []byte
	err = storage.ReadField(StandardField, &stdB)
	if err != nil {
		return nil, operation.Std(0), err
	}

	var std operation.Std
	if err = std.UnmarshalBinary(stdB); err != nil {
		return nil, operation.Std(0), err
	}

	if std == operation.Std(0) {
		log.Error("Token doesn't exist", "address", token, "std", std)
		return nil, operation.Std(0), ErrTokenNotExists
	}

	return storage, std, nil
}

func (p *Processor) newStorage(token common.Address, op operation.Operation) (tokenStorage.Storage, error) {
	storage, standard, err := p.newStorageWithoutStdCheck(token)
	if err != nil {
		return nil, err
	}

	if standard != op.Standard() {
		log.Error("Token standard isn't valid for the operation", "address", token, "standard", standard, "opStandard", op.Standard())
		return nil, ErrTokenOpStandardNotValid
	}

	return storage, nil
}

func newFieldsDescriptors(op operation.Create) ([]tokenStorage.FieldDescriptor, error) {
	fieldDescriptors := make([]tokenStorage.FieldDescriptor, 0, 10)

	// Standard
	stdFd, err := newByteArrayDescriptor(StandardField, tokenStorage.Uint16Size)
	if err != nil {
		return nil, err
	}
	fieldDescriptors = append(fieldDescriptors, stdFd)

	// Name
	nameFd, err := newByteArrayDescriptor(NameField, uint64(len(op.Name())))
	if err != nil {
		return nil, err
	}
	fieldDescriptors = append(fieldDescriptors, nameFd)

	// Symbol
	symbolFd, err := newByteArrayDescriptor(SymbolField, uint64(len(op.Symbol())))
	if err != nil {
		return nil, err
	}
	fieldDescriptors = append(fieldDescriptors, symbolFd)

	// Balances
	balancesFd, err := newByteArrayScalarMapDescriptor(BalancesField, common.AddressLength, tokenStorage.Uint256Type)
	if err != nil {
		return nil, err
	}
	fieldDescriptors = append(fieldDescriptors, balancesFd)

	switch op.Standard() {
	case operation.StdWRC20:
		// Creator
		creatorFd, err := newByteArrayDescriptor(CreatorField, common.AddressLength)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, creatorFd)

		// Decimals
		decimalsFd, err := newScalarField(DecimalsField, tokenStorage.Uint8Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, decimalsFd)

		// TotalSupply
		totalSupplyFd, err := newScalarField(TotalSupplyField, tokenStorage.Uint256Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, totalSupplyFd)

		// Allowances
		allowancesFd, err := newByteArrayScalarMapDescriptor(AllowancesField, common.HashLength, tokenStorage.Uint256Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, allowancesFd)

		// Cost
		costFd, err := newScalarField(CostField, tokenStorage.Uint256Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, costFd)
	case operation.StdWRC721:
		// Minter
		minterFd, err := newByteArrayDescriptor(MinterField, common.AddressLength)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, minterFd)

		// BaseUri
		baseURI, _ := op.BaseURI()
		baseUriFd, err := newByteArrayDescriptor(BaseUriField, uint64(len(baseURI)))
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, baseUriFd)

		// Owners
		ownersFd, err := newByteArrayByteArrayMapDescriptor(OwnersField, common.AddressLength, common.AddressLength)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, ownersFd)

		// Metadata
		metadataFd, err := newByteArrayByteSliceMapDescriptor(MetadataField, common.AddressLength)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, metadataFd)

		// TokenApprovals
		tokenApprovalsFd, err := newByteArrayByteArrayMapDescriptor(TokenApprovalsField, common.AddressLength, common.AddressLength)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, tokenApprovalsFd)

		// OperatorApprovals
		operatorApprovalsFd, err := newByteArrayScalarMapDescriptor(OperatorApprovalsField, common.HashLength, tokenStorage.Uint8Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, operatorApprovalsFd)

		// PercentFee
		percentFeeFd, err := newScalarField(PercentFeeField, tokenStorage.Uint8Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, percentFeeFd)

		// CostMap
		costFd, err := newScalarScalarMapDescriptor(CostMapField, tokenStorage.Uint256Type, tokenStorage.Uint256Type)
		if err != nil {
			return nil, err
		}
		fieldDescriptors = append(fieldDescriptors, costFd)
	}

	return fieldDescriptors, nil
}

func newByteArrayDescriptor(name string, l uint64) (tokenStorage.FieldDescriptor, error) {
	sc, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), tokenStorage.NewArrayProperties(sc, l))
}

func newByteArrayByteArrayMapDescriptor(name string, keySize, valueSize uint64) (tokenStorage.FieldDescriptor, error) {
	keyScalar, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	valueScalar, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	mp, err := tokenStorage.NewMapProperties(
		tokenStorage.NewArrayProperties(keyScalar, keySize),
		tokenStorage.NewArrayProperties(valueScalar, valueSize),
	)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), mp)
}

func newByteArrayByteSliceMapDescriptor(name string, keySize uint64) (tokenStorage.FieldDescriptor, error) {
	keyScalar, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	valueScalar, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	mp, err := tokenStorage.NewMapProperties(
		tokenStorage.NewArrayProperties(keyScalar, keySize),
		tokenStorage.NewSliceProperties(valueScalar),
	)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), mp)
}

func newByteArrayScalarMapDescriptor(name string, keySize uint64, valType tokenStorage.Type) (tokenStorage.FieldDescriptor, error) {
	keyScalar, err := tokenStorage.NewScalarProperties(tokenStorage.Uint8Type)
	if err != nil {
		return nil, err
	}

	valueScalar, err := tokenStorage.NewScalarProperties(valType)
	if err != nil {
		return nil, err
	}

	mp, err := tokenStorage.NewMapProperties(tokenStorage.NewArrayProperties(keyScalar, keySize), valueScalar)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), mp)
}

func newScalarScalarMapDescriptor(name string, keyType, valType tokenStorage.Type) (tokenStorage.FieldDescriptor, error) {
	keyScalar, err := tokenStorage.NewScalarProperties(keyType)
	if err != nil {
		return nil, err
	}

	valueScalar, err := tokenStorage.NewScalarProperties(valType)
	if err != nil {
		return nil, err
	}

	mp, err := tokenStorage.NewMapProperties(keyScalar, valueScalar)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), mp)
}

func readAddressFromMap(storage tokenStorage.Storage, name string, key interface{}) (common.Address, error) {
	var addrB []byte
	err := readFromMap(storage, name, key, &addrB)
	if err != nil {
		return common.Address{}, err
	}

	return common.BytesToAddress(addrB), nil
}

func writeToMap(storage tokenStorage.Storage, name string, key, value interface{}) error {
	kv := tokenStorage.NewKeyValuePair(key, value)
	return storage.WriteField(name, kv)
}

func readFromMap(storage tokenStorage.Storage, name string, key interface{}, refToRes interface{}) error {
	kv := tokenStorage.NewKeyValuePair(key, refToRes)
	return storage.ReadField(name, kv)
}

func newScalarField(name string, tp tokenStorage.Type) (tokenStorage.FieldDescriptor, error) {
	sc, err := tokenStorage.NewScalarProperties(tp)
	if err != nil {
		return nil, err
	}

	return tokenStorage.NewFieldDescriptor([]byte(name), sc)
}

func transfer(storage tokenStorage.Storage, from, to common.Address, swapValue *big.Int) error {
	var fromBalance uint256.Int
	err := readFromMap(storage, BalancesField, from.Bytes(), &fromBalance)
	if err != nil {
		return err
	}

	if fromBalance.ToBig().Cmp(swapValue) < 0 {
		return ErrNotEnoughBalance
	}

	newFromBalance, ok := uint256.FromBig(new(big.Int).Sub(fromBalance.ToBig(), swapValue))
	if ok {
		return ErrUint256Overflow
	}

	err = writeToMap(storage, BalancesField, from.Bytes(), newFromBalance)
	if err != nil {
		return err
	}

	var toBalance uint256.Int
	err = readFromMap(storage, BalancesField, to.Bytes(), &toBalance)
	if err != nil {
		return err
	}

	newToBalance, ok := uint256.FromBig(new(big.Int).Add(toBalance.ToBig(), swapValue))
	if ok {
		return ErrUint256Overflow
	}

	return writeToMap(storage, BalancesField, to.Bytes(), newToBalance)
}

func readAddress(storage tokenStorage.Storage, field string) (common.Address, error) {
	var addr []byte
	err := storage.ReadField(field, &addr)
	if err != nil {
		return common.Address{}, err
	}

	return common.BytesToAddress(addr), nil
}
