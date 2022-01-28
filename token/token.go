package token

import (
	"github.com/ethereum/go-ethereum/common"
)

// Token standard
type Std uint

const (
	StdWRC20  = 20
	StdWRC721 = 721
)

// Token operation code
type OpCode uint

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	OpCreate            = 0x0C
	OpApprove           = 0x0D
	OpTransfer          = 0x1E
	OpTransferFrom      = 0x1F
	OpProperties        = 0x21
	OpBalanceOf         = 0x22
	OpAllowance         = 0x23
	OpIsApprovedForAll  = 0x24
	OpSetApprovalForAll = 0x25
	OpMint              = 0x26
	OpBurn              = 0x27
)

type Operation interface {
	OpCode() OpCode
	Standard() Std
	Address() common.Address // Token address
}
