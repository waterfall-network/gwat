package operation

import (
	ssz "github.com/waterfall-network/fastssz"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto/bls_sig"
)

type DepositMessage struct {
	PublicKey             []byte
	CreatorAddress        []byte
	WithdrawalCredentials []byte
	Amount                uint64
}

func (d *DepositMessage) GetTree() (*ssz.Node, error) {
	//TODO implement me
	panic("implement me")
}

// SizeSSZ returns the ssz encoded size in bytes for the DepositMessage object
func (d *DepositMessage) SizeSSZ() (size int) {
	size = 96
	return
}

// HashTreeRoot ssz hashes the DepositMessage object
func (d *DepositMessage) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(d)
}

// HashTreeRootWith ssz hashes the DepositMessage object with a hasher
func (d *DepositMessage) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'PublicKey'
	if len(d.PublicKey) != 48 {
		err = ssz.ErrBytesLength
		return
	}
	hh.PutBytes(d.PublicKey)

	// Field (1) 'CreatorAddress'
	if len(d.CreatorAddress) != 20 {
		err = ssz.ErrBytesLength
		return
	}
	hh.PutBytes(d.CreatorAddress)

	// Field (2) 'WithdrawalCredentials'
	if len(d.WithdrawalCredentials) != 20 {
		err = ssz.ErrBytesLength
		return
	}
	hh.PutBytes(d.WithdrawalCredentials)

	// Field (3) 'Amount'
	hh.PutUint64(d.Amount)

	hh.Merkleize(indx)
	return
}

type SigningData struct {
	ObjectRoot []byte
	Domain     []byte
}

func (s *SigningData) GetTree() (*ssz.Node, error) {
	//TODO implement me
	panic("implement me")
}

// SizeSSZ returns the ssz encoded size in bytes for the SigningData object
func (s *SigningData) SizeSSZ() (size int) {
	size = 64
	return
}

// HashTreeRoot ssz hashes the SigningData object
func (s *SigningData) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(s)
}

// HashTreeRootWith ssz hashes the SigningData object with a hasher
func (s *SigningData) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'ObjectRoot'
	if len(s.ObjectRoot) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	hh.PutBytes(s.ObjectRoot)

	// Field (1) 'Domain'
	if len(s.Domain) != 32 {
		err = ssz.ErrBytesLength
		return
	}
	hh.PutBytes(s.Domain)

	hh.Merkleize(indx)
	return
}

func VerifyDepositSig(
	sig common.BlsSignature,
	pk common.BlsPubKey,
	creatorAddr common.Address,
	withdrawalCred common.Address,
) error {
	sigData := &DepositMessage{
		PublicKey:             pk.Bytes(),
		CreatorAddress:        creatorAddr.Bytes(),
		WithdrawalCredentials: withdrawalCred.Bytes(),
		Amount:                0,
	}
	sigDataRoot, err := sigData.HashTreeRoot()
	if err != nil {
		return err
	}
	root, err := (&SigningData{ObjectRoot: sigDataRoot[:], Domain: depositDomain()}).HashTreeRoot()
	if err != nil {
		return err
	}
	isValid := bls_sig.VerifyCompressed(sig[:], pk[:], root[:])
	if !isValid {
		return ErrInvalidDepositSig
	}
	return nil
}

func depositDomain() []byte {
	//res of `func ComputeDomain(domainType [DomainByteLength]byte, forkVersion, genesisValidatorsRoot []byte) ([]byte, error)`
	//for deposit on coordinator
	return []byte{
		0x3, 0x0, 0x0, 0x0, 0xF5, 0xA5, 0xFD, 0x42, 0xD1, 0x6A, 0x20, 0x30, 0x27, 0x98, 0xEF, 0x6E,
		0xD3, 0x9, 0x97, 0x9B, 0x43, 0x0, 0x3D, 0x23, 0x20, 0xD9, 0xF0, 0xE8, 0xEA, 0x98, 0x31, 0xA9,
	}
}
