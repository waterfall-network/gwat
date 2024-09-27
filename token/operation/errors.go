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
	"errors"
)

var (
	ErrNoTokenSupply    = errors.New("token supply is required")
	ErrNoName           = errors.New("token name is required")
	ErrNoSymbol         = errors.New("token symbol is required")
	ErrNoBaseURI        = errors.New("token baseURI is required")
	ErrNoAddress        = errors.New("token address is required")
	ErrNoOwner          = errors.New("token owner address is required")
	ErrNoValue          = errors.New("value is required")
	ErrNegativeCost     = errors.New("cost is negative")
	ErrNoTo             = errors.New("to address is required")
	ErrNoFrom           = errors.New("from address is required")
	ErrNoSpender        = errors.New("spender address is required")
	ErrNoOperator       = errors.New("operator address is required")
	ErrNoTokenId        = errors.New("token id is required")
	ErrNoIndex          = errors.New("token index is required")
	ErrStandardNotValid = errors.New("not valid value for token standard")
	ErrPrefixNotValid   = errors.New("not valid value for prefix")
	ErrRawDataShort     = errors.New("binary data for token operation is short")
	ErrOpNotValid       = errors.New("not valid op code for token operation")
)
