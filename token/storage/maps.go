package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type ByteMap struct {
	mapIndex    []byte
	stream      *StorageStream
	totalLength int
	valueSize   int
}

func newByteMap(stream *StorageStream, mapIndex uint8, totalLength, valueSize int) (*ByteMap, error) {
	m := ByteMap{
		mapIndex:    []byte{mapIndex},
		stream:      stream,
		totalLength: totalLength,
		valueSize:   valueSize,
	}

	return &m, nil
}

func (m *ByteMap) Put(value []byte, key ...[]byte) error {
	keys := createKeys(key...)

	off := mapPosition(m.mapIndex, keys, m.totalLength)

	if len(value) < m.valueSize {
		return ErrWrongBuf
	}

	_, err := m.stream.WriteAt(value[:m.valueSize], off)
	if err != nil {
		return err
	}

	return nil
}

func (m *ByteMap) Get(key ...[]byte) ([]byte, error) {
	keys := createKeys(key...)

	buf := make([]byte, m.valueSize)
	off := mapPosition(m.mapIndex, keys, m.totalLength)

	_, err := m.stream.ReadAt(buf, off)
	if err != nil {
		return nil, err
	}

	return buf, err
}

func mapPosition(mapIndex, key []byte, totalLength int) *big.Int {
	slotLen := len(Slot{})
	total := (totalLength/slotLen + 1) * slotLen

	off := new(big.Int).SetBytes(crypto.Keccak256(mapIndex, key))
	off.Mul(off, big.NewInt(int64(slotLen)))
	off.Add(off, big.NewInt(int64(total)))

	return off
}

type addressReadWriter struct {
	byteMap *ByteMap
	key     [][]byte
}

func NewAddressReadWriter(byteMap *ByteMap, key *big.Int) AddressReadWriter {
	return &addressReadWriter{
		byteMap: byteMap,
		key: [][]byte{
			key.Bytes(),
		},
	}
}

func (r *addressReadWriter) Read() (common.Address, error) {
	buf, err := r.byteMap.Get(r.key...)
	if err != nil {
		return [20]byte{}, err
	}

	return common.BytesToAddress(buf), nil
}

func (r *addressReadWriter) Write(address common.Address) error {
	return r.byteMap.Put(address.Bytes(), r.key...)
}

type boolReadWriter struct {
	byteMap *ByteMap
	key     [][]byte
}

func NewBoolReadWriter(byteMap *ByteMap, owner, operator common.Address) BoolReadWriter {
	return &boolReadWriter{
		byteMap: byteMap,
		key: [][]byte{
			owner.Bytes(),
			operator.Bytes(),
		},
	}
}

func (r *boolReadWriter) Read() (bool, error) {
	buf, err := r.byteMap.Get(r.key...)
	if err != nil {
		return false, err
	}

	if buf[0] > 0 {
		return true, nil
	}

	return false, nil
}

func (r *boolReadWriter) Write(b bool) error {
	buf := []byte{0}
	if b {
		buf[0] = 1
	}
	return r.byteMap.Put(buf, r.key...)
}

type uint256ReadWriter struct {
	byteMap *ByteMap
	key     [][]byte
}

func NewUint256ReadWriter(byteMap *ByteMap, address ...common.Address) Uint256ReadWriter {
	readWriter := uint256ReadWriter{
		byteMap: byteMap,
		key:     nil,
	}

	for i, addr := range address {
		readWriter.key[i] = addr.Bytes()
	}

	return &readWriter
}

func (r *uint256ReadWriter) Read() (*big.Int, error) {
	res := new(big.Int)
	buf, err := r.byteMap.Get(r.key...)
	if err != nil {
		return nil, err
	}

	return res.SetBytes(buf), nil
}

func (r *uint256ReadWriter) Write(value *big.Int) error {
	return r.byteMap.Put(value.Bytes(), r.key...)
}

func createKeys(key ...[]byte) []byte {
	keys := make([]byte, len(key))
	for _, bytes := range key {
		keys = append(keys, bytes...)
	}

	return keys
}
