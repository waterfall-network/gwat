package storage

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrNilKey          = errors.New("key is nil")
	ErrBadMapId        = errors.New("bad map id")
	ErrKeySizeIsZero   = errors.New("key size is 0")
	ErrValueSizeIsZero = errors.New("value size is 0")
	ErrBadKeySize      = errors.New("expected key size and real do not match")
	ErrBadValueSize    = errors.New("expected value size and real do not match")
)

type ByteMap struct {
	uniqPrefix []byte
	stream     *StorageStream

	// bytes count
	keySize, valueSize uint64
}

func newByteMap(mapId []byte, stream *StorageStream, keySize, valueSize uint64) (*ByteMap, error) {
	if len(mapId) == 0 {
		return nil, ErrBadMapId
	}

	if keySize == 0 {
		return nil, ErrKeySizeIsZero
	}

	if valueSize == 0 {
		return nil, ErrValueSizeIsZero
	}

	return &ByteMap{
		uniqPrefix: mapId,
		stream:     stream,
		keySize:    keySize,
		valueSize:  valueSize,
	}, nil
}

func (m *ByteMap) Put(key, value []byte) error {
	if key == nil {
		return ErrNilKey
	}

	if uint64(len(key)) != m.keySize {
		return ErrBadKeySize
	}

	if uint64(len(value)) > m.valueSize {
		return ErrBadValueSize
	}

	off := calculateOffset(m.uniqPrefix, key)
	_, err := m.stream.WriteAt(value, off)
	return err
}

func (m *ByteMap) Get(key []byte) ([]byte, error) {
	if uint64(len(key)) != m.keySize {
		return nil, ErrBadKeySize
	}

	buf := make([]byte, m.valueSize)
	off := calculateOffset(m.uniqPrefix, key)

	_, err := m.stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func calculateOffset(uniqPrefix, key []byte) *big.Int {
	h := new(big.Int).SetBytes(crypto.Keccak256(uniqPrefix, key))
	return h.Mul(h, big.NewInt(int64(len(Slot{}))))
}
