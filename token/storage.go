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
}

func NewStorage() Storage {
	return Storage{
		slots: make([]common.Hash, 1),
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
	if s.pos >= SlotSize {
		s.slots = append(s.slots, common.Hash{})
		s.pos = 0
	}

	if s.pos+len(b) > SlotSize {
		size := SlotSize - s.pos
		defer s.write(b[size:])
		copy(s.slots[len(s.slots)-1][s.pos:], b[:size])
		s.pos += size
		return
	}

	copy(s.slots[len(s.slots)-1][s.pos:s.pos+len(b)], b)
	s.pos += len(b)
}
