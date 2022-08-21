package interfaces

import (
	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core/types"
)

type BlockChain interface {
	GetBlockByHash(hash common.Hash) *types.Block
	GetBlocksByHashes(hashes common.HashArray) types.BlockMap
}
