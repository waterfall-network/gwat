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

package types

import (
	"crypto/ecdsa"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
)

func SignBlockHeader(header *Header, prv *ecdsa.PrivateKey) (*Header, error) {
	h := header.UnsignedHash()

	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}

	header.setSignature(sig)

	return header, nil
}

func BlockHeaderSigner(header *Header) (common.Address, error) {
	v, r, s := header.rawSignatureValues()

	if v == nil && r == nil && s == nil {
		return common.Address{}, nil
	}

	addr, err := recoverPlain(header.UnsignedHash(), r, s, v, true)

	return addr, err
}
