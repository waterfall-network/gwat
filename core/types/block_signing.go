package types

import (
	"crypto/ecdsa"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
)

func SignBlock(block *Block, prv *ecdsa.PrivateKey) (*Block, error) {
	h := block.UnsignedHash()

	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}

	return block.withSignature(sig), nil
}

func BlockSigner(block *Block) (common.Address, error) {
	v, r, s := block.header.rawSignatureValues()

	addr, err := recoverPlain(block.UnsignedHash(), r, s, v, true)

	return addr, err
}
