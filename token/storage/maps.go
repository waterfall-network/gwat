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
	stream     *StorageStream

	// bytes count
	keySize, valueSize uint64

	keyEncoder, valueEncoder Encoder
	valueDecoder             Decoder
}

func newByteMap(
	mapId []byte,
	stream *StorageStream,
	keyEncoder, valueEncoder Encoder,
	valueDecoder Decoder,
	keySize, valueSize uint64,
) (*ByteMap, error) {
	if len(mapId) == 0 {
		return nil, ErrBadMapId
	}

	if keySize == 0 {
		return nil, ErrKeySizeIsZero
	}

	if valueSize == 0 {
		return nil, ErrValueSizeIsZero
	}

	if keyEncoder == nil || valueEncoder == nil {
		return nil, ErrEncoderIsNil
	}

	if valueDecoder == nil {
		return nil, ErrDecoderIsNil
	}

	pref := make([]byte, uint64Size*2+len(mapId))
	binary.BigEndian.PutUint64(pref[:uint64Size], keySize)
	binary.BigEndian.PutUint64(pref[uint64Size:uint64Size*2], valueSize)
	copy(pref[uint64Size*2:], mapId)

	return &ByteMap{
		uniqPrefix:   pref,
		stream:       stream,
		keySize:      keySize,
		valueSize:    valueSize,
		keyEncoder:   keyEncoder,
		valueEncoder: valueEncoder,
		valueDecoder: valueDecoder,
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

	if uint64(len(keyB)) != m.keySize {
		return ErrBadKeySize
	}

	valueB, err := m.valueEncoder(value)
	if err != nil {
		return err
	}

	if uint64(len(valueB)) != m.valueSize {
		return ErrBadValueSize
	}

	off := calculateOffset(m.uniqPrefix, keyB)
	_, err = m.stream.WriteAt(valueB, off)
	return err
}

func (m *ByteMap) Load(key, to interface{}) error {
	keyB, err := m.keyEncoder(key)
	if err != nil {
		return err
	}

	if uint64(len(keyB)) != m.keySize {
		return ErrBadKeySize
	}

	buf := make([]byte, m.valueSize)
	off := calculateOffset(m.uniqPrefix, keyB)

	_, err = m.stream.ReadAt(buf, off)
	if err != nil {
		return err
	}

	return m.valueDecoder(buf, to)
}

func calculateOffset(uniqPrefix, key []byte) *big.Int {
	return new(big.Int).SetBytes(crypto.Keccak256(uniqPrefix, key))
}
