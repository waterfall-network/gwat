//go:build ((linux && amd64) || (linux && arm64) || (darwin && amd64) || (darwin && arm64) || (windows && amd64)) && !blst_disabled
// +build linux,amd64 linux,arm64 darwin,amd64 darwin,arm64 windows,amd64
// +build !blst_disabled

package bls_sig

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	p *blstPublicKey
}

// PublicKeyFromBytes creates a BLS public key from a  BigEndian byte slice.
func PublicKeyFromBytes(pubKey []byte) (*PublicKey, error) {
	if len(pubKey) != common.BlsPubKeyLength {
		return nil, fmt.Errorf("public key must be %d bytes", common.BlsPubKeyLength)
	}
	// Subgroup check NOT done when decompressing pubkey.
	p := new(blstPublicKey).Uncompress(pubKey)
	if p == nil {
		return nil, errors.New("could not unmarshal bytes into public key")
	}
	// Subgroup and infinity check
	if !p.KeyValidate() {
		// NOTE: the error is not quite accurate since it includes group check
		return nil, errors.New("ErrInfinitePubKey")
	}
	pubKeyObj := &PublicKey{p: p}
	return pubKeyObj, nil
}

// Marshal a public key into a LittleEndian byte slice.
func (p *PublicKey) Marshal() []byte {
	return p.p.Compress()
}

// Copy the public key to a new pointer reference.
func (p *PublicKey) Copy() *PublicKey {
	np := *p.p
	return &PublicKey{p: &np}
}

// IsInfinite checks if the public key is infinite.
func (p *PublicKey) IsInfinite() bool {
	zeroKey := new(blstPublicKey)
	return p.p.Equals(zeroKey)
}
