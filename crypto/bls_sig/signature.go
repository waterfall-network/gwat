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

//go:build ((linux && amd64) || (linux && arm64) || (darwin && amd64) || (darwin && arm64) || (windows && amd64)) && !blst_disabled
// +build linux,amd64 linux,arm64 darwin,amd64 darwin,arm64 windows,amd64
// +build !blst_disabled

package bls_sig

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

// Signature used in the BLS signature scheme.
type Signature struct {
	s *blstSignature
}

// SignatureFromBytes creates a BLS signature from a LittleEndian byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	if len(sig) != common.BlsSigLength {
		return nil, fmt.Errorf("signature must be %d bytes", common.BlsSigLength)
	}
	signature := new(blstSignature).Uncompress(sig)
	if signature == nil {
		return nil, errors.New("could not unmarshal bytes into signature")
	}
	// Group check signature. Do not check for infinity since an aggregated signature
	// could be infinite.
	if !signature.SigValidate(false) {
		return nil, errors.New("signature not in group")
	}
	return &Signature{s: signature}, nil
}

// Verify a bls signature given a public key, a message.
func (s *Signature) Verify(pubKey PublicKey, msg []byte) bool {
	return s.s.Verify(false, pubKey.p, false, msg, dst)
}

// Marshal a signature into a LittleEndian byte slice.
func (s *Signature) Marshal() []byte {
	return s.s.Compress()
}

// Copy returns a full deep copy of a signature.
func (s *Signature) Copy() *Signature {
	sign := *s.s
	return &Signature{s: &sign}
}

// VerifyCompressed verifies that the compressed signature and pubkey
// are valid from the message provided.
func VerifyCompressed(signature, pub, msg []byte) bool {
	// Validate signature and PKs since we will uncompress them here
	return new(blstSignature).VerifyCompressed(signature, true, pub, true, msg, dst)
}
