package operation

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// AllowanceOperation contains attributes for an allowance operation
type AllowanceOperation interface {
	Operation
	addresser
	Owner() common.Address
	Spender() common.Address
}

// ApproveOperation contains attributes for a token approve operation
type ApproveOperation interface {
	Operation
	Spender() common.Address
	Value() *big.Int
}

// BalanceOfOperation contains attrubutes for token balance of call
type BalanceOfOperation interface {
	Operation
	addresser
	Owner() common.Address
}

// BurnOperation contains attributes for a burn token operation
type BurnOperation interface {
	Operation
	TokenId() *big.Int
}

// CreateOperation contatins all attributes for creating WRC-20 or WRC-721 token
// Methods for getting optional attributes also return boolean values which indicates if the attribute was set
type CreateOperation interface {
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

// IsApprovedForAllOperation contains attributes for WRC-721 is approved for all operation
type IsApprovedForAllOperation interface {
	Operation
	addresser
	Owner() common.Address
	Operator() common.Address
}

// MintOperation contains attributes for a mint operation
type MintOperation interface {
	Operation
	To() common.Address
	TokenId() *big.Int
	Metadata() ([]byte, bool)
}

// PropertiesOperation contatins attributes for a token properties call
type PropertiesOperation interface {
	Operation
	addresser
	TokenId() (*big.Int, bool)
}

// SafeTransferFromOperation contains attributes for WRC-721 safe transfer from operation
type SafeTransferFromOperation interface {
	TransferFromOperation
	Data() ([]byte, bool)
}

// SetApprovalForAllOperation contains attributes for set approval for all operation
type SetApprovalForAllOperation interface {
	Operation
	Operator() common.Address
	IsApproved() bool
}

// TokenOfOwnerByIndexOperation contatins attributes for WRC-721 token of owner by index operation
type TokenOfOwnerByIndexOperation interface {
	Operation
	addresser
	Owner() common.Address
	Index() *big.Int
}

// TransferFromOperation contains attrubute for WRC-20 or WRC-721 transfer from operation
type TransferFromOperation interface {
	TransferOperation
	From() common.Address
}

// TransferOperation contains attributes for token transfer call
type TransferOperation interface {
	Operation
	To() common.Address
	Value() *big.Int
}
