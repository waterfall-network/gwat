package token

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type Storage struct {
	pos     int
	slots   map[common.Hash]*common.Hash
	statedb vm.StateDB
	addr    common.Address

	slot    *common.Hash
	addSlot func(func(common.Hash) common.Hash) (common.Hash, *common.Hash)
}

func NewStorage(tokenAddr common.Address, statedb vm.StateDB) Storage {
	s := Storage{
		slots:   make(map[common.Hash]*common.Hash),
		statedb: statedb,
		addr:    tokenAddr,
	}

	slotNumber := 0
	addSlot := func(getSlot func(common.Hash) common.Hash) (common.Hash, *common.Hash) {
		hash := common.Hash{}
		binary.BigEndian.PutUint64(hash[:], uint64(slotNumber))
		slot := getSlot(hash)
		s.slots[hash] = &slot
		s.pos = 0

		slotNumber += 1
		return hash, &slot
	}
	s.addSlot = addSlot

	return s
}

func (s *Storage) WriteUint8(v uint8) {
	s.write([]byte{v})
}

func (s *Storage) WriteUint16(v uint16) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	s.write(buf)
}

func (s *Storage) WriteUint64(v uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	s.write(buf)
}

func (s *Storage) WriteUint256(v *big.Int) {
	buf := v.FillBytes(make([]byte, 32))
	s.write(buf)
}

// Implements io.Writer
func (s *Storage) Write(b []byte) (int, error) {
	s.WriteUint64(uint64(len(b)))
	s.write(b)
	return len(b) + 8, nil
}

func (s *Storage) write(b []byte) {
	s.do(b, func(slotSlice, bSlice []byte) {
		copy(slotSlice, bSlice)
	}, func(common.Hash) common.Hash {
		return common.Hash{}
	})
}

func (s *Storage) ReadUint8() uint8 {
	buf := make([]byte, 1)
	s.read(buf)
	return buf[0]
}

func (s *Storage) ReadUint16() uint16 {
	buf := make([]byte, 2)
	s.read(buf)
	return binary.BigEndian.Uint16(buf)
}

func (s *Storage) ReadUint64() uint64 {
	buf := make([]byte, 8)
	s.read(buf)
	return binary.BigEndian.Uint64(buf)
}

func (s *Storage) ReadUint256() *big.Int {
	buf := make([]byte, 32)
	s.read(buf)
	v := new(big.Int)
	return v.SetBytes(buf)
}

func (s *Storage) ReadBytes() []byte {
	l := s.ReadUint64()
	b := make([]byte, l)
	s.read(b)
	return b
}

func (s *Storage) read(b []byte) {
	s.do(b, func(slotSlice, bSlice []byte) {
		copy(bSlice, slotSlice)
	}, func(hash common.Hash) common.Hash {
		return s.statedb.GetState(s.addr, hash)
	})
}

func (s *Storage) do(b []byte, action func(slotSlice, bSlice []byte), getSlot func(common.Hash) common.Hash) {
	if s.pos >= common.HashLength || len(s.slots) == 0 {
		_, s.slot = s.addSlot(getSlot)
	}

	if s.pos+len(b) > common.HashLength {
		size := common.HashLength - s.pos
		defer s.do(b[size:], action, getSlot)
		action(s.slot[s.pos:], b[:size])
		s.pos += size
		return
	}

	action(s.slot[s.pos:s.pos+len(b)], b)
	s.pos += len(b)
}

func (s *Storage) Flush() {
	for k, v := range s.slots {
		s.statedb.SetState(s.addr, k, *v)
	}
}
