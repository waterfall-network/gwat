package storage

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrNilKey              = errors.New("key is nil")
	ErrKeySizeIsZero       = errors.New("key size is 0")
	ErrValueSizeIsZero     = errors.New("value size is 0")
	ErrMismatchedKeySize   = errors.New("expected key size and real do not match")
	ErrMismatchedValueSize = errors.New("expected value size and real do not match")
)

type Encoder func(interface{}) ([]byte, error)
type Decoder func([]byte, interface{}) error

var DefaultEncoder = scalarToBytes
var DefaultDecoder = bytesToScalar

type ByteMap struct {
	uniqPrefix []byte
	stream     *StorageStream

	// bytes count
	keySize, valueSize uint64

	keyEncoder, valueEncoder Encoder
}

func newByteMap(
	mapId []byte,
	stream *StorageStream,
	keyEncoder, valueEncoder Encoder,
	keySize, valueSize uint64,
) (*ByteMap, error) {
	if keySize == 0 {
		return nil, ErrKeySizeIsZero
	}
	if valueSize == 0 {
		return nil, ErrValueSizeIsZero
	}

	pref := make([]byte, uint64Size*2+len(mapId))
	binary.BigEndian.PutUint64(pref[:uint64Size], keySize)
	binary.BigEndian.PutUint64(pref[uint64Size:uint64Size*2], valueSize)
	copy(pref[uint64Size*2:], mapId)

	return &ByteMap{
		stream:       stream,
		keySize:      keySize,
		valueSize:    valueSize,
		uniqPrefix:   pref,
		keyEncoder:   keyEncoder,
		valueEncoder: valueEncoder,
	}, nil
}

func (m *ByteMap) Put(key, value interface{}) error {
	if key == nil {
		return ErrNilKey
	}

	keyB, err := m.keyEncoder(key)
	if err != nil {
		return err
	}

	if uint64(len(keyB)) < m.keySize {
		return ErrMismatchedKeySize
	}

	valueB, err := m.valueEncoder(value)
	if err != nil {
		return err
	}

	if uint64(len(valueB)) < m.valueSize {
		return ErrMismatchedValueSize
	}

	off := calculateOffset(m.uniqPrefix, keyB)
	_, err = m.stream.WriteAt(valueB, off)
	if err != nil {
		return err
	}

	return nil
}

func (m *ByteMap) Get(key interface{}) (interface{}, error) {
	keyB, err := m.keyEncoder(key)
	if err != nil {
		return nil, err
	}

	if uint64(len(keyB)) < m.keySize {
		return nil, ErrMismatchedKeySize
	}

	buf := make([]byte, m.valueSize)
	off := calculateOffset(m.uniqPrefix, keyB)

	_, err = m.stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func calculateOffset(uniqPrefix, key []byte) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(uniqPrefix, key))
}
