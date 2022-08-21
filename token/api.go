package token

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/token/operation"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	ErrNotEnoughArgs = errors.New("not enough arguments for token create operation")
)

type Backend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	GetTP(ctx context.Context, state *state.StateDB, header *types.Header) (*Processor, func() error, error)
}

// PublicTokenAPI provides an API to access native token functions.
type PublicTokenAPI struct {
	b Backend
}

// NewPublicTokenAPI creates a new native token API.
func NewPublicTokenAPI(b Backend) *PublicTokenAPI {
	return &PublicTokenAPI{b}
}

type wrc721Properties struct {
	Name       *hexutil.Bytes `json:"name"`
	Symbol     *hexutil.Bytes `json:"symbol"`
	PercentFee *hexutil.Uint8 `json:"percentFee,omitempty"`
	Cost       *hexutil.Big   `json:"cost,omitempty"`
}

// wrc20Properties stores results of the following view functions of EIP-20: name, symbol, decimals, totalSupply.
type wrc20Properties struct {
	wrc721Properties
	Decimals    *hexutil.Uint8 `json:"decimals,omitempty"`
	TotalSupply *hexutil.Big   `json:"totalSupply,omitempty"`
}

// wrc721ByTokenIdProperties contains Metadata field which is custom field of WRC-721 token.
type wrc721ByTokenIdProperties struct {
	TokenURI    *hexutil.Bytes  `json:"tokenURI"`
	OwnerOf     *common.Address `json:"ownerOf"`
	GetApproved *common.Address `json:"getApproved"`
	// Metadata contains JSON metadata for an NTF. The data will be stored in the blockchain natively.
	Metadata *hexutil.Bytes `json:"metadata"`
}

// wrc721Properties stores results of the following view functions of EIP-721: name, symbol, tokenURI, ownerOf, getApproved.
//
// Properties in the ByTokenId field will not be returned if tokenId isn't given.
type wrc721TokenProperties struct {
	wrc721Properties
	BaseURI   *hexutil.Bytes             `json:"baseURI,omitempty"`
	ByTokenId *wrc721ByTokenIdProperties `json:"byTokenId,omitempty"`
}

type TokenArgs struct {
	// WRC-20 properties
	wrc20Properties
	// WRC-721 properties
	BaseURI *hexutil.Bytes `json:"baseURI,omitempty"`
}

// TokenCreate creates a collection of tokens for a caller. Can be used for creating both WRC-20 and WRC-721 tokens.
//
// Will create a WRC-721 token if BaseURI field is given in the args. Returns a raw data with token attributes.
// Use the raw data in the Data field when sending a transaction to create the token.
func (s *PublicTokenAPI) TokenCreate(ctx context.Context, args TokenArgs) (hexutil.Bytes, error) {
	if args.Name == nil {
		return nil, operation.ErrNoName
	}
	if args.Symbol == nil {
		return nil, operation.ErrNoSymbol
	}
	name := []byte(*args.Name)
	symbol := []byte(*args.Symbol)

	var (
		op  operation.Operation
		err error
	)

	switch {
	case args.TotalSupply != nil:
		decimals := (*uint8)(args.Decimals)
		totalSupply := args.TotalSupply.ToInt()

		if op, err = operation.NewWrc20CreateOperation(name, symbol, decimals, totalSupply); err != nil {
			return nil, err
		}
	case args.BaseURI != nil:
		percentFee := (*uint8)(args.PercentFee)
		baseURI := []byte(*args.BaseURI)

		if op, err = operation.NewWrc721CreateOperation(name, symbol, baseURI, percentFee); err != nil {
			return nil, err
		}
	default:
		return nil, ErrNotEnoughArgs
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode token create operation", "err", err)
		return nil, err
	}
	return b, nil
}

func (s *PublicTokenAPI) newTokenProcessor(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (tp *Processor, cancel context.CancelFunc, tpError func() error, err error) {
	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, nil, nil, err
	}

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	timeout := s.b.RPCEVMTimeout()
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	tp, tpError, err = s.b.GetTP(ctx, state, header)
	if err != nil {
		return nil, nil, nil, err
	}

	return
}

// TokenProperties returns properties of the token. Returns different structures for WRC-20 and WRC-721 tokens.
//
// For a WRC-20 token returns wrc20Properties structure. For a WRC-721 token returns wrc721Properties structure.
//
// TokenProperties implements the following view functions of EIP-20: name, symbol, decimals, totalSupply.
//
// It also implements view functions of EIP-721: name, symbol, tokenURI, ownerOf, getApproved.
// TokenURI, ownerOf and getApproved are only returned if tokenId parameter is given. Also with tokenId given
// TokenProperties returns custom metadata field for WRC-721 tokens in the result structure.
func (s *PublicTokenAPI) TokenProperties(ctx context.Context, tokenAddr common.Address, blockNrOrHash rpc.BlockNumberOrHash, tokenId *hexutil.Big) (ret interface{}, err error) {
	tp, cancel, tpError, err := s.newTokenProcessor(ctx, blockNrOrHash)
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	if err != nil {
		return nil, err
	}

	op, err := operation.NewPropertiesOperation(tokenAddr, tokenId.ToInt())
	if err != nil {
		return nil, err
	}

	res, err := tp.Properties(op)
	if err != nil {
		return nil, err
	}
	if err := tpError(); err != nil {
		return nil, err
	}

	switch v := res.(type) {
	case *WRC20PropertiesResult:
		nameBytes := hexutil.Bytes(v.Name)
		symbolBytes := hexutil.Bytes(v.Symbol)
		decimals := hexutil.Uint8(v.Decimals)
		totalSupply := (*hexutil.Big)(v.TotalSupply)

		ret = &wrc20Properties{
			wrc721Properties{
				Name:   &nameBytes,
				Symbol: &symbolBytes,
			},
			&decimals,
			totalSupply,
		}
	case *WRC721PropertiesResult:
		nameBytes := hexutil.Bytes(v.Name)
		symbolBytes := hexutil.Bytes(v.Symbol)
		percentFee := hexutil.Uint8(v.PercentFee)
		cost := (*hexutil.Big)(v.Cost)

		props := &wrc721TokenProperties{
			wrc721Properties: wrc721Properties{
				Name:       &nameBytes,
				Symbol:     &symbolBytes,
				PercentFee: &percentFee,
				Cost:       cost,
			},
		}
		if len(v.BaseURI) > 0 {
			baseURIBytes := hexutil.Bytes(v.BaseURI)
			props.BaseURI = &baseURIBytes
		}

		if tokenId != nil {
			tokenURIBytes := hexutil.Bytes(v.TokenURI)
			metadataBytes := hexutil.Bytes(v.Metadata)

			props.ByTokenId = &wrc721ByTokenIdProperties{
				TokenURI:    &tokenURIBytes,
				OwnerOf:     &v.OwnerOf,
				GetApproved: &v.GetApproved,
				Metadata:    &metadataBytes,
			}
		}

		ret = props
	}

	return ret, nil
}

// TokenBalanceOf returns the balance of another account with owner address for a WRC-20 token.
// For WRC-721 token returns the number of NFTs assigned to an owner, possibly zero.
func (s *PublicTokenAPI) TokenBalanceOf(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	tp, cancel, tpError, err := s.newTokenProcessor(ctx, blockNrOrHash)
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	if err != nil {
		return nil, err
	}

	op, err := operation.NewBalanceOfOperation(tokenAddr, ownerAddr)
	if err != nil {
		return nil, err
	}

	res, err := tp.BalanceOf(op)
	if err != nil {
		return nil, err
	}
	if err := tpError(); err != nil {
		return nil, err
	}

	return (*hexutil.Big)(res), nil
}

// Wrc20Transfer transfers `value` amount of WRC-20 tokens of a caller to address `to`.
//
// Returns a raw data with transfer operation attributes.
// Use the raw data in the Data field when sending a transaction to transfer a token.
func (s *PublicTokenAPI) Wrc20Transfer(ctx context.Context, to common.Address, value hexutil.Big) (hexutil.Bytes, error) {
	v := value.ToInt()
	op, err := operation.NewTransferOperation(to, v)
	if err != nil {
		log.Error("Can't create a transfer operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token transfer operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc20TransferFrom transfers `value` amount of WRC-20 tokens from address `from` to address `to`.
//
// Returns a raw data with transfer operation attributes.
// Use the raw data in the Data field when sending a transaction to transfer a token.
func (s *PublicTokenAPI) Wrc20TransferFrom(ctx context.Context, from common.Address, to common.Address, value hexutil.Big) (hexutil.Bytes, error) {
	v := value.ToInt()
	op, err := operation.NewTransferFromOperation(operation.StdWRC20, from, to, v)
	if err != nil {
		log.Error("Can't create a transfer from operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token transfer from operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc20Approve allows spender to withdraw WRC-20 tokens from your account multiple times, up to the value amount.
// If this function is called again it overwrites the current allowance with value.
//
// Returns a raw data with approve operation attributes.
// Use the raw data in the Data field when sending a transaction to allow spender to withdraw a token.
func (s *PublicTokenAPI) Wrc20Approve(ctx context.Context, spenderAddr common.Address, value hexutil.Big) (hexutil.Bytes, error) {
	v := value.ToInt()
	op, err := operation.NewApproveOperation(operation.StdWRC20, spenderAddr, v)
	if err != nil {
		log.Error("Can't create an approve operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode an approve operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc20Allowance returns the amount of WRC-20 tokens which spender is still allowed to withdraw from owner.
func (s *PublicTokenAPI) Wrc20Allowance(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, spenderAddr common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	tp, cancel, tpError, err := s.newTokenProcessor(ctx, blockNrOrHash)
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	if err != nil {
		return nil, err
	}

	op, err := operation.NewAllowanceOperation(tokenAddr, ownerAddr, spenderAddr)
	if err != nil {
		return nil, err
	}

	res, err := tp.Allowance(op)
	if err != nil {
		return nil, err
	}
	if err := tpError(); err != nil {
		return nil, err
	}

	return (*hexutil.Big)(res), nil
}

// Wrc721IsApprovedForAll returns true if an operator is the approved operator of WRC-721 tokens for an owner, false otherwise.
// The operator can manage all NFTs of the owner.
func (s *PublicTokenAPI) Wrc721IsApprovedForAll(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, operatorAddr common.Address, blockNrOrHash rpc.BlockNumberOrHash) (bool, error) {
	tp, cancel, tpError, err := s.newTokenProcessor(ctx, blockNrOrHash)
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()
	if err != nil {
		return false, err
	}

	op, err := operation.NewIsApprovedForAllOperation(tokenAddr, ownerAddr, operatorAddr)
	if err != nil {
		return false, err
	}

	res, err := tp.IsApprovedForAll(op)
	if err != nil {
		return false, err
	}
	if err := tpError(); err != nil {
		return false, err
	}

	return res, nil
}

// Wrc721Approve changes or reaffirmes the approved address for an NFT.
// The zero address indicates there is no approved address.
// Throws unless the transaction sender is the current NFT owner, or an authorized
// operator of the current owner.
//
// Returns a raw data with mint operation attributes.
// Use the raw data in the Data field when sending a transaction to approve address for the NFT.
func (s *PublicTokenAPI) Wrc721Approve(ctx context.Context, approved common.Address, tokenId hexutil.Big) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	op, err := operation.NewApproveOperation(operation.StdWRC721, approved, id)
	if err != nil {
		log.Error("Can't create a NFT approve operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a NFT approve operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc721SafeTransferFrom transfers the ownership of a WRC-721 token with given tokenId from one address to another address.
// Throws unless a caller is the current owner, an authorized operator, or the approved address for this NFT.
// Throws if `from` is  not the current owner. Throws if `to` is the zero address.
// Throws if `tokenId` is not a valid NFT.
// Parameter `data` contains additional data with no specified format, sent in call to `to`
//
// Returns a raw data with safe transfer operation attributes.
// Use the raw data in the Data field when sending a transaction to safe transfer an NFT.
//
// TODO: Implement safeTransferFrom for NFTs.
/* func (s *PublicTokenAPI) Wrc721SafeTransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, tokenId hexutil.Big, data *hexutil.Bytes) (hexutil.Bytes, error) {
	log.Info("WRC-721 safe transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "tokenId", tokenId, "data", data)
	return nil, nil
}*/

// Wrc721TransferFrom transfers ownership of an NFT -- THE CALLER IS RESPONSIBLE TO CONFIRM THAT `to` IS CAPABLE OF RECEIVING NFTS OR ELSE
// THEY MAY BE PERMANENTLY LOST.
// Throws unless a caller is the current owner, an authorized operator, or the approved address for this NFT.
// Throws if `from` is not the current owner.
// Throws if `to` is the zero address.
// Throws if `tokenId` is not a valid NFT.
//
// Returns a raw data with transfer operation attributes.
// Use the raw data in the Data field when sending a transaction to transfer an NFT.
func (s *PublicTokenAPI) Wrc721TransferFrom(ctx context.Context, from common.Address, to common.Address, tokenId hexutil.Big) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	op, err := operation.NewTransferFromOperation(operation.StdWRC721, from, to, id)
	if err != nil {
		log.Error("Can't create a transfer NFT from operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a transfer NFT from operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc721SetApprovalForAll enables or disables approval for a third party ("operator") to manage all of caller's assets.
//
// Returns a raw data with approval operation attributes.
// Use the raw data in the Data field when sending a transaction to enable or disable approval to manage an NFT.
func (s *PublicTokenAPI) Wrc721SetApprovalForAll(ctx context.Context, operatorAddr common.Address, isApproved bool) (hexutil.Bytes, error) {
	op, err := operation.NewSetApprovalForAllOperation(operatorAddr, isApproved)
	if err != nil {
		log.Error("Can't create a set approval for all operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a set approval for all operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc721Mint mints a new token. Reverts if the given token ID already exists.
// Metadata should be given in JSON format and will be stored natively in the blockchain.
//
// Returns a raw data with mint operation attributes.
// Use the raw data in the Data field when sending a transaction to mint an NFT.
func (s *PublicTokenAPI) Wrc721Mint(ctx context.Context, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	var tokenMeta []byte = nil
	if metadata != nil {
		tokenMeta = *metadata
	}

	op, err := operation.NewMintOperation(to, id, tokenMeta)
	if err != nil {
		log.Error("Can't create a token mint operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token mint operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Wrc721SafeMint safely mints a new token. Reverts if the given token ID already exists.
// If the target address is a contract, it must implement onERC721Received, which is called upon a safe transfer, and return the magic value
// `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"));` otherwise, the transfer is reverted.
//
// Returns hash of the mint transaction. If the function reverts you can check a status in receipts of the transaction.
//
// TODO: Implement safe minting of NFTs.
/* func (s *PublicTokenAPI) Wrc721SafeMint(ctx context.Context, tokenAddr common.Address, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (bool, error) {
	log.Info("WRC-721 safe mint", "tokenAddr", tokenAddr, "to", to, "tokenId", tokenId, "metadata", metadata)
	return false, nil
}
*/

// Wrc721Burn burns a specific token. Reverts if the token does not exist.
//
// Returns a raw data with mint operation attributes.
// Use the raw data in the Data field when sending a transaction to burn an NFT.
func (s *PublicTokenAPI) Wrc721Burn(ctx context.Context, tokenId hexutil.Big) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	op, err := operation.NewBurnOperation(id)
	if err != nil {
		log.Error("Can't create a token mint operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token mint operation", "err", err)
		return nil, err
	}
	return b, nil
}

// TokenCost returns the cost a token. `tokeAddr` is required.
// For WRC-20 the `tokenId` is NOT required.
// For WRC-721 the `tokenId` is not required.
func (s *PublicTokenAPI) TokenCost(ctx context.Context, tokenAddr common.Address, tokenId hexutil.Big, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	id := tokenId.ToInt()

	tp, cancel, tpError, err := s.newTokenProcessor(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	defer cancel()

	op, err := operation.NewCostOperation(tokenAddr, id)
	if err != nil {
		return nil, err
	}

	res, err := tp.Cost(op)
	if err != nil {
		return nil, err
	}
	if err := tpError(); err != nil {
		return nil, err
	}

	return (*hexutil.Big)(res), nil
}

// SetPrice sets a price for a token. 'value' is required
// For WRC-20 the `tokenId` is NOT required.
// For WRC-721 sets price for a token by `tokenId`.
func (s *PublicTokenAPI) SetPrice(ctx context.Context, tokenId, value hexutil.Big) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	val := value.ToInt()

	op, err := operation.NewSetPriceOperation(id, val)
	if err != nil {
		log.Error("Cannot create a token set price operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token set price operation", "err", err)
		return nil, err
	}

	return b, nil
}

// Buy buys a token.
// Tx's `value` is total spent count of wei, the rest will be returned.
// For WRC-20 `tokenId` and `newValue` NOT required.
// For WRC-721 `tokenId` and `newValue` are required.
func (s *PublicTokenAPI) Buy(_ context.Context, tokenId, newValue hexutil.Big) (hexutil.Bytes, error) {
	id := tokenId.ToInt()
	newVal := newValue.ToInt()

	op, err := operation.NewBuyOperation(id, newVal)
	if err != nil {
		log.Error("Cannot create a token buy operation", "err", err)
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Error("Failed to encode a token buy operation", "err", err)
		return nil, err
	}

	return b, nil
}

// Wrc721TokenOfOwnerByIndex enumerates NFTs assigned to an owner.
// Throws if `index` >= `balanceOf(ownerAddr)` or if `ownerAddr` is the zero address, representing invalid NFTs.
//
// Returns the token identifier for the `index`th NFT assigned to `ownerAddr`.
//
// TODO: Implement tokenOfOwnerByIndex for NFTs.
/* func (s *PublicTokenAPI) Wrc721TokenOfOwnerByIndex(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, index hexutil.Big) (*hexutil.Big, error) {
	log.Info("WRC-721 token of owner by index", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "index", index)
	return nil, nil
}*/

func GetAPIs(apiBackend Backend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "wat",
			Version:   "1.0",
			Service:   NewPublicTokenAPI(apiBackend),
			Public:    true,
		},
	}
}
