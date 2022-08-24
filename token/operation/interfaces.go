package operation

import (
	"math/big"

	"github.com/waterfall-foundation/gwat/common"
)

// Allowance contains attributes for an allowance operation
type Allowance interface {
	Operation
	addresser
	Owner() common.Address
	Spender() common.Address
}

// Approve contains attributes for a token approve operation
type Approve interface {
	Operation
	Spender() common.Address
	Value() *big.Int
}

// BalanceOf contains attrubutes for token balance of call
type BalanceOf interface {
	Operation
	addresser
	Owner() common.Address
}

// Burn contains attributes for a burn token operation
type Burn interface {
	Operation
	TokenId() *big.Int
}

// Create contatins all attributes for creating WRC-20 or WRC-721 token
// Methods for getting optional attributes also return boolean values which indicates if the attribute was set
type Create interface {
	Operation
	Name() []byte
	Symbol() []byte
	// That's not required arguments
	// WRC-20 arguments
	Decimals() uint8
	TotalSupply() (*big.Int, bool)
	// WRC-721 arguments
	BaseURI() ([]byte, bool)
}

// IsApprovedForAll contains attributes for WRC-721 is approved for all operation
type IsApprovedForAll interface {
	Operation
	addresser
	Owner() common.Address
	Operator() common.Address
}

// Mint contains attributes for a mint operation
type Mint interface {
	Operation
	To() common.Address
	TokenId() *big.Int
	Metadata() ([]byte, bool)
}

// Properties contatins attributes for a token properties call
type Properties interface {
	Operation
	addresser
	TokenId() (*big.Int, bool)
}

// SafeTransferFrom contains attributes for WRC-721 safe transfer from operation
type SafeTransferFrom interface {
	TransferFrom
	Data() ([]byte, bool)
}

// SetApprovalForAll contains attributes for set approval for all operation
type SetApprovalForAll interface {
	Operation
	Operator() common.Address
	IsApproved() bool
}

// TokenOfOwnerByIndex contatins attributes for WRC-721 token of owner by index operation
type TokenOfOwnerByIndex interface {
	Operation
	addresser
	Owner() common.Address
	Index() *big.Int
}

// TransferFrom contains attrubute for WRC-20 or WRC-721 transfer from operation
type TransferFrom interface {
	Transfer
	From() common.Address
}

// Transfer contains attributes for token transfer call
type Transfer interface {
	Operation
	To() common.Address
	Value() *big.Int
}
