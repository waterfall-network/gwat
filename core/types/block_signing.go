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
