package token

import (
	"context"

	"github.com/ethereum/common/hexutil"
)

type Backend interface{}

type PublicTokenAPI struct {
	b Backend
}

func NewPublicTokenAPI(b Backend) *PublicTokenAPI {
	return &PublicTokenAPI{b}
}

type TokenArgs struct {
	// WRC-20 properties
	Name        *hexutil.Bytes `json:"name"`
	Symbol      *hexutil.Bytes `json:"symbol"`
	Decimals    *hexutil.Uint  `json:"decimals"`
	TotalSupply *hexutil.Big   `json:"totalSupply"`
	// WRC-721 properties
	BaseURI *hexutil.Bytes `json:"baseURI"`
}

func (s *PublicTokenAPI) TokenCreate(ctx context.Context, args TokenArgs) (hexutil.Bytes, error) {}
