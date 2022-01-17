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

type PublicTokenAPI struct {
	b Backend
}

func NewPublicTokenAPI(b Backend) *PublicTokenAPI {
	return &PublicTokenAPI{b}
}

type wrc721Properties struct {
	Name   *hexutil.Bytes `json:"name"`
	Symbol *hexutil.Bytes `json:"symbol"`
}

type wrc20Properties struct {
	wrc721Properties
	Decimals    *hexutil.Uint `json:"decimals"`
	TotalSupply *hexutil.Big  `json:"totalSupply"`
}

type wrc721ByTokenIdProperties struct {
	TokenURI    *hexutil.Bytes  `json:"tokenURI"`
	OwnerOf     *common.Address `json:"ownerOf"`
	GetApproved *common.Address `json:"getApproved"`
	Metadata    *hexutil.Bytes  `json:"metadata"`
}

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

func (s *PublicTokenAPI) TokenBalanceOf(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address) (*hexutil.Big, error) {
	log.Info("Token balance of", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr)
	return nil, nil
}

func (s *PublicTokenAPI) Wrc20Transfer(ctx context.Context, tokenAddr common.Address, to common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 transfer", "tokenAddr", tokenAddr, "to", to, "value", value.ToInt())
	return false, nil
}

func (s *PublicTokenAPI) Wrc20TransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "value", value.ToInt())
	return false, nil
}

func (s *PublicTokenAPI) Wrc20Approve(ctx context.Context, tokenAddr common.Address, spenderAddr common.Address, value hexutil.Big) (bool, error) {
	log.Info("WRC-20 approve", "tokenAddr", tokenAddr, "spenderAddr", spenderAddr, "value", value.ToInt())
	return false, nil
}

func (s *PublicTokenAPI) Wrc20Allowance(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, spenderAddr common.Address) (*hexutil.Big, error) {
	log.Info("WRC-20 allowance", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "spenderAddr", spenderAddr)
	return nil, nil
}

func (s *PublicTokenAPI) Wrc721IsApprovedForAll(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, operatorAddr common.Address) (bool, error) {
	log.Info("WRC-721 is approved for all", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr, "operatorAddr", operatorAddr)
	return false, nil
}

func (s *PublicTokenAPI) Wrc721SafeTransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, tokenId hexutil.Big, data *hexutil.Bytes) (common.Hash, error) {
	log.Info("WRC-721 safe transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "tokenId", tokenId, "data", data)
	return common.Hash{}, nil
}

func (s *PublicTokenAPI) Wrc721TransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, tokenId hexutil.Big) (common.Hash, error) {
	log.Info("WRC-721 transfer from", "tokenAddr", tokenAddr, "from", from, "to", to, "tokenId", tokenId)
	return common.Hash{}, nil
}

func (s *PublicTokenAPI) Wrc721SetApprovalForAll(ctx context.Context, tokenAddr common.Address, operatorAddr common.Address, isApproved bool) (common.Hash, error) {
	log.Info("WRC-721 set approval for all", "tokenAddr", tokenAddr, "operatorAddr", operatorAddr, "isApproved", isApproved)
	return common.Hash{}, nil
}

func (s *PublicTokenAPI) Wrc721Mint(ctx context.Context, tokenAddr common.Address, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (bool, error) {
	log.Info("WRC-721 mint", "tokenAddr", tokenAddr, "to", to, "tokenId", tokenId, "metadata", metadata)
	return false, nil
}

func (s *PublicTokenAPI) Wrc721SafeMint(ctx context.Context, tokenAddr common.Address, to common.Address, tokenId hexutil.Big, metadata *hexutil.Bytes) (bool, error) {
	log.Info("WRC-721 safe mint", "tokenAddr", tokenAddr, "to", to, "tokenId", tokenId, "metadata", metadata)
	return false, nil
}

func (s *PublicTokenAPI) Wrc721Burn(ctx context.Context, tokenAddr common.Address, tokenId hexutil.Big) (common.Hash, error) {
	log.Info("WRC-721 burn", "tokenAddr", tokenAddr, "tokenId", tokenId)
	return common.Hash{}, nil
}

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
