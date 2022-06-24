package storage

import (
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type storageStream interface {
	ReadAt(b []byte, off int) (int, error)
	WriteAt(b []byte, off int) (int, error)
	Flush()
}

type Slot common.Hash

type StorageStream struct {
	stateDb       vm.StateDB
	tokenAddress  common.Address
	bufferedSlots map[common.Hash]*Slot
	buf           []byte
}

func NewStorageStream(tokenAddr common.Address, statedb vm.StateDB) *StorageStream {
	return &StorageStream{
		stateDb:       statedb,
		tokenAddress:  tokenAddr,
		bufferedSlots: make(map[common.Hash]*Slot),
	}
}

func (s *StorageStream) WriteAt(b []byte, off int) (int, error) {

	return s.do(b, off, func(streamBuf, b []byte) int {
		return copy(streamBuf, b)
	})
}

func (s *StorageStream) ReadAt(b []byte, off int) (int, error) {
	return s.do(b, off, func(streamBuf, b []byte) int {
		return copy(b, streamBuf)
	})
}

func (s *StorageStream) Flush() {
	for k, v := range s.bufferedSlots {
		s.stateDb.SetState(s.tokenAddress, k, common.Hash(*v))
	}
}

func (s *StorageStream) do(b []byte, off int, action func(streamBuf, b []byte) int) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	slotPos, err := position(off)
	if err != nil {
		return 0, err
	}

	var res int
	for res = 0; res < len(b); {
		err = s.getSlot(off + res)
		if err != nil {
			return 0, err
		}

		wb := action(s.buf[slotPos:], b[res:])
		res += wb
		slotPos = 0
	}

	return res, nil
}

func (s *StorageStream) getSlot(off int) error {
	slotIndex, err := slot(off)
	if err != nil {
		return err
	}
	slotKey := common.Hash{}
	binary.BigEndian.PutUint64(slotKey[:], slotIndex)

	slot, ok := s.bufferedSlots[slotKey]
	if !ok {
		tmp := Slot(s.stateDb.GetState(s.tokenAddress, slotKey))
		slot = &tmp
		s.bufferedSlots[slotKey] = slot
	}

	s.buf = slot[:]

	return nil
}

func slot(shift int) (uint64, error) {
	if shift < 0 {
		return 0, errors.New("negative shift")
	}

	return uint64(shift / 32), nil
}

func position(shift int) (int, error) {
	if shift < 0 {
		return 0, errors.New("negative shift")
	}

	if shift == 0 {
		return 0, nil
	}

	return shift % 32, nil
}
