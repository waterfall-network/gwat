package token

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Backend interface{}

// PublicTokenAPI provides an API to access native token functions.
type PublicTokenAPI struct {
	b Backend
}

// NewPublicTokenAPI creates a new native token API.
func NewPublicTokenAPI(b Backend) *PublicTokenAPI {
	return &PublicTokenAPI{b}
}

type wrc721Properties struct {
	Name   *hexutil.Bytes `json:"name"`
	Symbol *hexutil.Bytes `json:"symbol"`
}

// wrc20Properties stores results of the following view functions of EIP-20: name, symbol, decimals, totalSupply.
type wrc20Properties struct {
	wrc721Properties
	Decimals    *hexutil.Uint `json:"decimals"`
	TotalSupply *hexutil.Big  `json:"totalSupply"`
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
	ByTokenId *wrc721ByTokenIdProperties `json:"byTokenId",omitempty`
}

type TokenArgs struct {
	// WRC-20 properties
	wrc20Properties
	// WRC-721 properties
	BaseURI *hexutil.Bytes `json:"baseURI"`
}

// TokenCreate creates a collection of tokens for a caller. Can be used for creating both WRC-20 and WRC-721 tokens.
//
// Will create a WRC-721 token if BaseURI field is given in the args. Returns a raw data with token attributes.
// Use the raw data in the Data field when sending a transaction to create the token.
func (s *PublicTokenAPI) TokenCreate(ctx context.Context, args TokenArgs) (hexutil.Bytes, error) {
	name := ""
	if args.Name != nil {
		name = string(*args.Name)
	}
	symbol := ""
	if args.Name != nil {
		symbol = string(*args.Symbol)
	}
	decimals := uint(0)
	if args.Decimals != nil {
		decimals = uint(*args.Decimals)
	}
	totalSupply := big.NewInt(0)
	if args.TotalSupply != nil {
		totalSupply = args.TotalSupply.ToInt()
	}
	log.Info("Create WRC-20 token", "name", name, "symbol", symbol, "decimals", decimals, "totalSupply", totalSupply)

	return nil, nil
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
func (s *PublicTokenAPI) TokenProperties(ctx context.Context, tokenAddr common.Address, tokenId *hexutil.Big) (interface{}, error) {
	name := "Test token"
	nameBytes := hexutil.Bytes(name)
	symbol := "TST"
	symbolBytes := hexutil.Bytes(symbol)

	if tokenId != nil {
		tokenURI := "https://waterfall.foundation/testtoken/1.json"
		tokenURIBytes := hexutil.Bytes(tokenURI)
		ownerOf := "0x3552fea0d44cb11d56210ca8a8f04a69c67ebf48"
		ownerOfBytes := common.HexToAddress(ownerOf)
		getApproved := "0x43b5339ea30687e43c39336d96c5f0db278debf6"
		getApprovedBytes := common.HexToAddress(getApproved)
		metadata := "{ name: \"Test token\" }"
		metadataBytes := hexutil.Bytes(metadata)

		log.Info("Return WRC-721 properties by token id", "name", name, "symbol", symbol, "tokenURI", tokenURI, "ownerOf", ownerOf, "getApproved", getApproved, "metadata", metadata)
		return &wrc721TokenProperties{
			wrc721Properties{
				Name:   &nameBytes,
				Symbol: &symbolBytes,
			},
			&wrc721ByTokenIdProperties{
				TokenURI:    &tokenURIBytes,
				OwnerOf:     &ownerOfBytes,
				GetApproved: &getApprovedBytes,
				Metadata:    &metadataBytes,
			},
		}, nil
	}

	decimals := hexutil.Uint(3)
	totalSupply := (*hexutil.Big)(big.NewInt(1000))

	log.Info("Return WRC-20 properties", "name", name, "symbol", symbol, "decimals", decimals, "totalSupply", totalSupply)
	return &wrc20Properties{
		wrc721Properties{
			Name:   &nameBytes,
			Symbol: &symbolBytes,
		},
		&decimals,
		totalSupply,
	}, nil
}

// TokenBalanceOf returns the balance of another account with owner address for a WRC-20 token.
// For WRC-721 token returns the number of NFTs assigned to an owner, possibly zero.
func (s *PublicTokenAPI) TokenBalanceOf(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address) (*hexutil.Big, error) {
	log.Info("Token balance of", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr)
	return nil, nil
}

// Wrc20Transfer transfers `value` amount of WRC-20 tokens of a caller to address `to`.
//
// Returns success of the operation.
func (s *PublicTokenAPI) Wrc20Transfer(ctx context.Context, tokenAddr common.Address, to common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 transfer", "tokenAddr", tokenAddr, "to", to, "value", value.ToInt())
	return false, nil
}

// Wrc20TransferFrom transfers `value` amount of WRC-20 tokens from address `from` to address `to`.
//
// Returns success of the operation.
func (s *PublicTokenAPI) Wrc20TransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "value", value.ToInt())
	return false, nil
}

// Wrc20Approve allows spender to withdraw WRC-20 tokens from your account multiple times, up to the value amount.
// If this function is called again it overwrites the current allowance with value.
//
// Returns success of the operation.
func (s *PublicTokenAPI) Wrc20Approve(ctx context.Context, tokenAddr common.Address, spenderAddr common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 approve", "tokenAddr", tokenAddr, "spenderAddr", spenderAddr, "value", value.ToInt())
	return false, nil
}

// Wrc20Allowance returns the amount of WRC-20 tokens which spender is still allowed to withdraw from owner.
func (s *PublicTokenAPI) Wrc20Allowance(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, spenderAddr common.Address) (*hexutil.Big, error) {
	log.Info("WRC-20 allowance", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "spenderAddr", spenderAddr)
	return nil, nil
}

// Wrc721IsApprovedForAll returns true if an operator is the approved operator of WRC-721 tokens for an owner, false otherwise.
// The operator can manage all NFTs of the owner.
func (s *PublicTokenAPI) Wrc721IsApprovedForAll(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, operatorAddr common.Address) (bool, error) {
	log.Info("WRC-721 is approved for all", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "operatorAddr", operatorAddr)
	return false, nil
}

// Wrc721SafeTransferFrom transfers the ownership of a WRC-721 token with given tokenId from one address to another address.
// Throws unless a caller is the current owner, an authorized operator, or the approved address for this NFT.
// Throws if `from` is  not the current owner. Throws if `to` is the zero address.
// Throws if `tokenId` is not a valid NFT.
// Parameter `data` contains additional data with no specified format, sent in call to `to`
//
// Returns hash of the transfer transaction. If the function throws you can check a status in receipts of the transaction.
func (s *PublicTokenAPI) Wrc721SafeTransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, tokenId hexutil.Big, data *hexutil.Bytes) (common.Hash, error) {
	log.Info("WRC-721 safe transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "tokenId", tokenId, "data", data)
	return common.Hash{}, nil
}

// Wrc721TransferFrom transfers ownership of an NFT -- THE CALLER IS RESPONSIBLE TO CONFIRM THAT `to` IS CAPABLE OF RECEIVING NFTS OR ELSE
// THEY MAY BE PERMANENTLY LOST.
// Throws unless a caller is the current owner, an authorized operator, or the approved address for this NFT.
// Throws if `from` is not the current owner.
// Throws if `to` is the zero address.
// Throws if `tokenId` is not a valid NFT.
//
// Returns hash of the transfer transaction. If the function throws you can check a status in receipts of the transaction.
func (s *PublicTokenAPI) Wrc721TransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, tokenId hexutil.Big) (common.Hash, error) {
	log.Info("WRC-721 transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "tokenId", tokenId)
	return common.Hash{}, nil
}

// Wrc721SetApprovalForAll enables or disables approval for a third party ("operator") to manage all of caller's assets.
//
// Returns hash of the approval transaction.
func (s *PublicTokenAPI) Wrc721SetApprovalForAll(ctx context.Context, tokenAddr common.Address, operatorAddr common.Address, isApproved bool) (common.Hash, error) {
	log.Info("WRC-721 set approval for all", "tokenAddr", tokenAddr, "operatorAddr", operatorAddr, "isApproved", isApproved)
	return common.Hash{}, nil
}

// Wrc721Mint mints a new token. Reverts if the given token ID already exists.
// Metadata should be given in JSON format and will be stored natively in the blockchain.
//
// Returns hash of the mint transaction. If the function reverts you can check a status in receipts of the transaction.
func (s *PublicTokenAPI) Wrc721Mint(ctx context.Context, tokenAddr common.Address, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (bool, error) {
	log.Info("WRC-721 mint", "tokenAddr", tokenAddr, "to", to, "tokenId", tokenId, "metadata", metadata)
	return false, nil
}

// Wrc721SafeMint safely mints a new token. Reverts if the given token ID already exists.
// If the target address is a contract, it must implement onERC721Received, which is called upon a safe transfer, and return the magic value
// `bytes4(keccak256("onERC721Received(address,address,uint256,bytes)"));` otherwise, the transfer is reverted.
//
// Returns hash of the mint transaction. If the function reverts you can check a status in receipts of the transaction.
func (s *PublicTokenAPI) Wrc721SafeMint(ctx context.Context, tokenAddr common.Address, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (bool, error) {
	log.Info("WRC-721 safe mint", "tokenAddr", tokenAddr, "to", to, "tokenId", tokenId, "metadata", metadata)
	return false, nil
}

// Wrc721Burn burns a specific token. Reverts if the token does not exist.
//
// Returns hash of the burn transaction. If the function reverts you can check a status in receipts of the transaction.
func (s *PublicTokenAPI) Wrc721Burn(ctx context.Context, tokenAddr common.Address, tokenId hexutil.Big) (common.Hash, error) {
	log.Info("WRC-721 burn", "tokenAddr", tokenAddr, "tokenId", tokenId)
	return common.Hash{}, nil
}

// Wrc721TokenOfOwnerByIndex enumerates NFTs assigned to an owner.
// Throws if `index` >= `balanceOf(ownerAddr)` or if `ownerAddr` is the zero address, representing invalid NFTs.
//
// Returns the token identifier for the `index`th NFT assigned to `ownerAddr`.
func (s *PublicTokenAPI) Wrc721TokenOfOwnerByIndex(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, index hexutil.Big) (*hexutil.Big, error) {
	log.Info("WRC-721 token of owner by index", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "index", index)
	return nil, nil
}

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
