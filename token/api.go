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

type wrc20Properties struct {
	Name        *hexutil.Bytes `json:"name"`
	Symbol      *hexutil.Bytes `json:"symbol"`
	Decimals    *hexutil.Uint  `json:"decimals"`
	TotalSupply *hexutil.Big   `json:"totalSupply"`
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

func (s *PublicTokenAPI) Wrc20Props(ctx context.Context, tokenAddr common.Address) (*wrc20Properties, error) {
	log.Info("WRC-20 token properties", "tokenAddr", tokenAddr)
	return nil, nil
}

func (s *PublicTokenAPI) Wrc20BalanceOf(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address) (*hexutil.Big, error) {
	log.Info("WRC-20 balance of", "tokenAddr", tokenAddr, "ownerAddr", ownerAddr)
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
