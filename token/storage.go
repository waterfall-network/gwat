package token

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

// Storage is sequential stream that writes data to slots of Ethereum words (256 bit or 32 bytes) under the hood.
// An account storage uses slots of the same size so the stream can be flushed to it using Flush method.
//
// Every method of the Storage do operation from the current position in the stream.
// The only exception is value of map type. Map is written using the following algorithm:
// 1. Write an empty slot referenced as map slot.
// 2. Using a position (index) of the map slot and key calculate hash used as a slot index for the value using Keccak256.
// 3. Write the value in the calculated slot
type Storage struct {
	pos     int
	slots   map[common.Hash]*common.Hash
	statedb vm.StateDB
	addr    common.Address

	slot       *common.Hash
	slotNumber uint64
}

// NewStorage creates new storage for a token with specific address
// Storage uses StateDB under the hood so its instance should be passed to the factory function.
func NewStorage(tokenAddr common.Address, statedb vm.StateDB) *Storage {
	return &Storage{
		slots:      make(map[common.Hash]*common.Hash),
		statedb:    statedb,
		addr:       tokenAddr,
		slotNumber: 0,
	}
}

func (s *Storage) addSlot(getSlot func(common.Hash) common.Hash) (common.Hash, *common.Hash) {
	hash := common.Hash{}
	binary.BigEndian.PutUint64(hash[:], s.slotNumber)
	slot := getSlot(hash)
	s.slots[hash] = &slot
	s.pos = 0

	s.slotNumber += 1
	return hash, &slot
}

// SkipBytes skips byte slice
// It just moves the position in the stream
func (s *Storage) SkipBytes() {
	l := s.ReadUint64()
	s.skip(int(l))
}

// SkitUint8 skips uint8 value
// It just moves the position in the stream
func (s *Storage) SkipUint8() {
	s.skip(1)
}

// SkitUint8 skips uint256 value
// It just moves the position in the stream
func (s *Storage) SkipUint256() {
	s.skip(32)
}

// SkitUint8 skips common.Address value
// It just moves the position in the stream
func (s *Storage) SkipAddress() {
	s.skip(20)
}

func (s *Storage) skip(l int) {
	s.do(nil, l, func([]byte, []byte) {}, func() *common.Hash {
		_, ret := s.addSlot(func(hash common.Hash) common.Hash {
			return s.statedb.GetState(s.addr, hash)
		})
		return ret
	})
}

// ReadMapSlot reads a map slot index
// It can be used for reading or skipping of the map slot
func (s *Storage) ReadMapSlot() common.Hash {
	hash, _ := s.addSlot(func(hash common.Hash) common.Hash {
		return common.Hash{}
	})
	s.pos = common.HashLength
	return hash
}

// WriteBoolToMap writes a boolean value to the map using a map slot index
func (s *Storage) WriteBoolToMap(mapSlot common.Hash, key []byte, value bool) {
	buf := []byte{0}
	if value {
		buf[0] = 1
	}
	s.writeToMap(mapSlot, key, buf)
}

// WriteUint256ToMap writes a uint256 value to the map using a map slot index
func (s *Storage) WriteUint256ToMap(mapSlot common.Hash, key []byte, value *big.Int) {
	buf := value.FillBytes(make([]byte, 32))
	s.writeToMap(mapSlot, key, buf)
}

// WriteAddressToMap writes a common.Address value to the map using a map slot index
func (s *Storage) WriteAddressToMap(mapSlot common.Hash, key []byte, address common.Address) {
	s.writeToMap(mapSlot, key, address[:])
}

func (s *Storage) writeToMap(mapSlot common.Hash, key []byte, value []byte) {
	prevPos := s.pos
	s.writeToMapWithoutPosReset(mapSlot, key, value, s.makeSlotGetterForWriter(mapSlot, key))
	s.pos = prevPos
}

// WriteBytesToMap writes a byte slice to the map using a map slot index
func (s *Storage) WriteBytesToMap(mapSlot common.Hash, key []byte, value []byte) {
	prevPos := s.pos
	getSlot := s.makeSlotGetterForWriter(mapSlot, key)

	// Write length of byte array
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(len(value)))
	s.writeToMapWithoutPosReset(mapSlot, key, buf, getSlot)

	s.writeToMapWithoutPosReset(mapSlot, key, value, getSlot)
	s.pos = prevPos
}

func (s *Storage) writeToMapWithoutPosReset(mapSlot common.Hash, key []byte, value []byte, getSlot func() *common.Hash) {
	s.do(value, len(value), func(slotSlice, bSlice []byte) {
		copy(slotSlice, bSlice)
	}, getSlot)
}

func (s *Storage) makeSlotGetterForWriter(mapSlot common.Hash, key []byte) func() *common.Hash {
	return s.makeIthSlotGetter(mapSlot, key, func(hash common.Hash) *common.Hash {
		return &common.Hash{}
	})
}

// ReadBoolFromMap reads a boolean value from the map using a map slot index
func (s *Storage) ReadBoolFromMap(mapSlot common.Hash, key []byte) bool {
	buf := make([]byte, 1)
	s.readFromMap(mapSlot, key, buf)
	r := false
	if buf[0] > 0 {
		r = true
	}
	return r
}

// ReadUint256FromMap reads a uint256 value from the map using a map slot index
func (s *Storage) ReadUint256FromMap(mapSlot common.Hash, key []byte) *big.Int {
	buf := make([]byte, 32)
	s.readFromMap(mapSlot, key, buf)
	v := new(big.Int)
	return v.SetBytes(buf)
}

// ReadAddressFromMap reads a common.Address value from the map using a map slot index
func (s *Storage) ReadAddressFromMap(mapSlot common.Hash, key []byte) common.Address {
	buf := common.Address{}
	s.readFromMap(mapSlot, key, buf[:])
	return buf
}

// ReadBytesFromMap reads a byte slice from the map using a map slot index
func (s *Storage) ReadBytesFromMap(mapSlot common.Hash, key []byte) []byte {
	prevPos := s.pos
	getSlot := s.makeSlotGetterForReader(mapSlot, key)

	// Get length of byte array
	buf := make([]byte, 8)
	s.readFromMapWithoutPosReset(mapSlot, key, buf, getSlot)
	l := binary.BigEndian.Uint64(buf)

	buf = make([]byte, l)
	s.readFromMapWithoutPosReset(mapSlot, key, buf, getSlot)
	s.pos = prevPos

	return buf
}

func (s *Storage) readFromMap(mapSlot common.Hash, key []byte, value []byte) {
	prevPos := s.pos
	s.readFromMapWithoutPosReset(mapSlot, key, value, s.makeSlotGetterForReader(mapSlot, key))
	s.pos = prevPos
}

func (s *Storage) readFromMapWithoutPosReset(mapSlot common.Hash, key []byte, value []byte, getSlot func() *common.Hash) {
	s.do(value, len(value), func(slotSlice, bSlice []byte) {
		copy(bSlice, slotSlice)
	}, getSlot)
}

func (s *Storage) makeSlotGetterForReader(mapSlot common.Hash, key []byte) func() *common.Hash {
	return s.makeIthSlotGetter(mapSlot, key, func(hash common.Hash) *common.Hash {
		slot := s.statedb.GetState(s.addr, hash)
		return &slot
	})
}

func (s *Storage) makeIthSlotGetter(mapSlot common.Hash, key []byte, getSlot func(common.Hash) *common.Hash) func() *common.Hash {
	var i uint64 = 0
	getIthSlot := func() *common.Hash {
		indexBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(indexBuf, i)
		hash := common.BytesToHash(crypto.Keccak256(mapSlot[:], key, indexBuf))
		s.slots[hash] = getSlot(hash)
		s.pos = 0

		i += 1
		return s.slots[hash]
	}
	return getIthSlot
}

// WriteUint8 writes a uint8 value to the stream
func (s *Storage) WriteUint8(v uint8) {
	s.write([]byte{v})
}

// WriteUint16 writes a uint16 value to the stream
func (s *Storage) WriteUint16(v uint16) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	s.write(buf)
}

// WriteUint64 writes a uint64 value to the stream
func (s *Storage) WriteUint64(v uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	s.write(buf)
}

// WriteUint256 writes a uint256 value to the stream
func (s *Storage) WriteUint256(v *big.Int) {
	buf := v.FillBytes(make([]byte, 32))
	s.write(buf)
}

// WriteAddress writes a common.Address value to the stream
func (s *Storage) WriteAddress(a common.Address) {
	s.write(a[:])
}

// Write writes a byte slice to the stream
// Implements io.Writer.
func (s *Storage) Write(b []byte) (int, error) {
	s.WriteUint64(uint64(len(b)))
	s.write(b)
	return len(b) + 8, nil
}

func (s *Storage) write(b []byte) {
	s.do(b, len(b), func(slotSlice, bSlice []byte) {
		copy(slotSlice, bSlice)
	}, func() *common.Hash {
		_, ret := s.addSlot(func(common.Hash) common.Hash {
			return common.Hash{}
		})
		return ret
	})
}

// ReadUint8 reads a uint8 value from the stream
func (s *Storage) ReadUint8() uint8 {
	buf := make([]byte, 1)
	s.read(buf)
	return buf[0]
}

// ReadUint16 reads a uint16 value from the stream
func (s *Storage) ReadUint16() uint16 {
	buf := make([]byte, 2)
	s.read(buf)
	return binary.BigEndian.Uint16(buf)
}

// ReadUint64 reads a uint64 value from the stream
func (s *Storage) ReadUint64() uint64 {
	buf := make([]byte, 8)
	s.read(buf)
	return binary.BigEndian.Uint64(buf)
}

// ReadUint256 reads a uint256 value from the stream
func (s *Storage) ReadUint256() *big.Int {
	buf := make([]byte, 32)
	s.read(buf)
	v := new(big.Int)
	return v.SetBytes(buf)
}

// ReadBytes reads a byte slice from the stream
func (s *Storage) ReadBytes() []byte {
	l := s.ReadUint64()
	b := make([]byte, l)
	s.read(b)
	return b
}

// ReadAddress reads a common.Address value from the stream
func (s *Storage) ReadAddress() common.Address {
	a := common.Address{}
	s.read(a[:])
	return a
}

func (s *Storage) read(b []byte) {
	s.do(b, len(b), func(slotSlice, bSlice []byte) {
		copy(bSlice, slotSlice)
	}, func() *common.Hash {
		_, ret := s.addSlot(func(hash common.Hash) common.Hash {
			return s.statedb.GetState(s.addr, hash)
		})
		return ret
	})
}

func (s *Storage) do(b []byte, l int, action func(slotSlice, bSlice []byte), addSlot func() *common.Hash) {
	if s.pos >= common.HashLength || len(s.slots) == 0 {
		s.slot = addSlot()
	}

	if s.pos+l > common.HashLength {
		size := common.HashLength - s.pos
		if b != nil {
			defer s.do(b[size:], len(b[size:]), action, addSlot)
			action(s.slot[s.pos:], b[:size])
		} else {
			defer s.do(nil, l-size, action, addSlot)
			action(s.slot[s.pos:], nil)
		}
		s.pos += size
		return
	}

	action(s.slot[s.pos:s.pos+l], b)
	s.pos += l
}

// Flush flushes the slots of the token storage to StateDB
func (s *Storage) Flush() {
	for k, v := range s.slots {
		s.statedb.SetState(s.addr, k, *v)
	}
}
