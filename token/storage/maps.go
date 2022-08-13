package storage

import (
	"encoding/binary"
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

	// bytes count
	keySize uint64
}

func newByteMap(mapId []byte, keySize uint64) (*ByteMap, error) {
	if len(mapId) == 0 {
		return nil, ErrBadMapId
	}

	if keySize == 0 {
		return nil, ErrKeySizeIsZero
	}

	return &ByteMap{
		uniqPrefix: mapId,
		keySize:    keySize,
	}, nil
}

func (m *ByteMap) Put(stream *StorageStream, key, value []byte) error {
	if key == nil {
		return ErrNilKey
	}

	if uint64(len(key)) > m.keySize {
		return ErrBadKeySize
	}

	valueSize := uint16(len(value))
	valueWithSize := make([]byte, Uint16Size+valueSize)
	binary.BigEndian.PutUint16(valueWithSize[:Uint16Size], valueSize)
	copy(valueWithSize[Uint16Size:], value)

	off := calculateOffset(m.uniqPrefix, key)
	_, err := stream.WriteAt(valueWithSize, off)
	return err
}

func (m *ByteMap) Get(stream *StorageStream, key []byte) ([]byte, error) {
	if uint64(len(key)) > m.keySize {
		return nil, ErrBadKeySize
	}

	buf := make([]byte, Uint16Size)
	off := calculateOffset(m.uniqPrefix, key)

	// read value size
	_, err := stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	valSize := binary.BigEndian.Uint16(buf)
	off.Add(off, big.NewInt(Uint16Size))
	buf = make([]byte, valSize)

	// read value
	_, err = stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func calculateOffset(uniqPrefix, key []byte) *big.Int {
	h := new(big.Int).SetBytes(crypto.Keccak256(uniqPrefix, key))
	return h.Mul(h, big.NewInt(int64(len(Slot{}))))
}
