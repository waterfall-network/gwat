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
	ErrBadDataLen = errors.New("bad data length")

	ErrNoPubKey            = errors.New("pubkey is required")
	ErrNoInitTxHash        = errors.New("initTxHash is required")
	ErrNoCreatorAddress    = errors.New("creator_address is required")
	ErrNoWithdrawalAddress = errors.New("withdrawal_address is required")
	ErrNoSignature         = errors.New("signature is required")
	ErrNoAmount            = errors.New("amount is required")
	ErrNoBalance           = errors.New("balance is required")
	ErrNoRules             = errors.New("rules is required")
	ErrRawDataShort        = errors.New("binary data for validator operation is short")

	ErrPrefixNotValid = errors.New("not valid value for prefix")
	ErrOpNotValid     = errors.New("op code is not valid")
	ErrOpBadVersion   = errors.New("op version is not valid")

	ErrNoExitRoles         = errors.New("no exit roles")
	ErrNoWithdrawalRoles   = errors.New("no withdrawal roles")
	ErrBadProfitShare      = errors.New("profit share totally must be 100%")
	ErrBadStakeShare       = errors.New("stake share totally must be 100%")
	ErrDelegateForkRequire = errors.New("can not process transaction before fork of delegating stake")

	ErrInvalidDepositSig = errors.New("invalid deposit signature")
)
