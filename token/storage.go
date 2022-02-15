package token

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const SlotSize = 32

type Storage struct {
	pos   int
	slots []common.Hash

	readSlot func(slot int) common.Hash
}

func NewStorage(readSlot func(int) common.Hash) Storage {
	return Storage{
		slots:    make([]common.Hash, 1),
		readSlot: readSlot,
	}
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
		copy(bSlice, slotSlice)
	}, func(slot int) common.Hash {
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
		copy(slotSlice, bSlice)
	}, s.readSlot)
}

func (s *Storage) do(b []byte, action func(slotSlice, bSlice []byte), getSlot func(int) common.Hash) {
	if s.pos >= SlotSize {
		s.slots = append(s.slots, getSlot(len(s.slots)))
		s.pos = 0
	}

	if s.pos+len(b) > SlotSize {
		size := SlotSize - s.pos
		defer s.do(b[size:], action, getSlot)
		action(s.slots[len(s.slots)-1][s.pos:], b[:size])
		s.pos += size
		return
	}

	action(s.slots[len(s.slots)-1][s.pos:s.pos+len(b)], b)
	s.pos += len(b)
}

func (s *Storage) Flush() {
}
