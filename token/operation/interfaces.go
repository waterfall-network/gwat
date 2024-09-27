// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operation

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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

// Create contains all attributes for creating WRC-20 or WRC-721 token
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
	PercentFee() uint8
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

// TransferFrom contains attribute for WRC-20 or WRC-721 transfer from operation
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

// SetPrice contains attributes for price token call
type SetPrice interface {
	Operation
	TokenId() (*big.Int, bool)
	Value() *big.Int
}

// Buy contains attributes for buy token call
type Buy interface {
	Operation
	TokenId() (*big.Int, bool)
	NewCost() (*big.Int, bool)
}

// Cost contains attributes for cost token call
type Cost interface {
	Operation
	addresser
	TokenId() (*big.Int, bool)
}
