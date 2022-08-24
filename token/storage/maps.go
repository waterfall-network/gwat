package storage

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/waterfall-foundation/gwat/crypto"
)

var (
	ErrNilKey        = errors.New("key is nil")
	ErrBadMapId      = errors.New("bad map id")
	ErrKeySizeIsZero = errors.New("key size is 0")
	ErrBadKeySize    = errors.New("expected key size and real do not match")
	ErrBadValueSize  = errors.New("expected value size and real do not match")
)

type ByteMap struct {
	uniqPrefix []byte

	// bytes count
	keySize, valueSize uint64
}

// pass 0 as valueSize if a value is a slice
func newByteMap(mapId []byte, keySize, valueSize uint64) (*ByteMap, error) {
	if len(mapId) == 0 {
		return nil, ErrBadMapId
	}

	if keySize == 0 {
		return nil, ErrKeySizeIsZero
	}

	return &ByteMap{
		uniqPrefix: mapId,
		keySize:    keySize,
		valueSize:  valueSize,
	}, nil
}

func (m *ByteMap) Put(stream *StorageStream, key, value []byte) error {
	if key == nil {
		return ErrNilKey
	}

	if uint64(len(key)) > m.keySize {
		return ErrBadKeySize
	}

	if m.valueSize != 0 && uint64(len(value)) > m.valueSize {
		return ErrBadValueSize
	}

	if m.valueSize == 0 {
		// write length of value slice
		valueSize := uint64(len(value))
		valueWithSize := make([]byte, Uint64Size+valueSize)
		binary.BigEndian.PutUint64(valueWithSize[:Uint64Size], valueSize)
		copy(valueWithSize[Uint64Size:], value)
		value = valueWithSize
	}

	off := calculateOffset(m.uniqPrefix, key)
	_, err := stream.WriteAt(value, off)
	return err
}

func (m *ByteMap) Get(stream *StorageStream, key []byte) ([]byte, error) {
	if uint64(len(key)) > m.keySize {
		return nil, ErrBadKeySize
	}

	size := m.valueSize
	off := calculateOffset(m.uniqPrefix, key)
	if size == 0 {
		// read length of value slice
		buf := make([]byte, Uint64Size)
		_, err := stream.ReadAt(buf, off)
		if err != nil {
			return nil, err
		}

		size = binary.BigEndian.Uint64(buf)
		off.Add(off, big.NewInt(Uint64Size))
	}

	buf := make([]byte, size)

	// read value
	_, err := stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func calculateOffset(uniqPrefix, key []byte) *big.Int {
	h := new(big.Int).SetBytes(crypto.Keccak256(uniqPrefix, key))
	return h.Mul(h, big.NewInt(int64(len(Slot{}))))
}
