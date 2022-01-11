package token

import (
	"context"

	"github.com/ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common"
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

func (s *PublicTokenAPI) TokenCreate(ctx context.Context, args TokenArgs) (hexutil.Bytes, error) {}

func (s *PublicTokenAPI) Wrc20Props(ctx context.Context, tokenAddr common.Address) (*wrc20Properties, error) {
}

func (s *PublicTokenAPI) Wrc20BalanceOf(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address) (*hexutil.Big, error) {
}

func (s *PublicTokenAPI) Wrc20Transfer(ctx context.Context, tokenAddr common.Address, to common.Address, value hexutil.Big) (bool, error) {
}

func (s *PublicTokenAPI) Wrc20TransferFrom(ctx context.Context, tokenAddr common.Address, from common.Address, to common.Address, value hexutil.Big) (bool, error) {
}

func (s *PublicTokenAPI) Wrc20Approve(ctx context.Context, tokenAddr common.Address, spenderAddr common.Address, value hexutil.Big) (bool, error) {
}

func (s *PublicTokenAPI) Wrc20Allowance(ctx context.Context, tokenAddr common.Address, ownerAddr common.Address, spenderAddress common.Address) (*hexutil.Big, error) {
}
