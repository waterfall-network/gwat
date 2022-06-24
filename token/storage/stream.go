package storage

import (
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"log"
)

type storageStream interface {
	ReadAt(b []byte, off int) (int, error)
	WriteAt(b []byte, off int) (int, error)
	Flush()
}

type Slot common.Hash

type StorageStream struct {
	stateDb      vm.StateDB
	tokenAddress common.Address
	slots        map[common.Hash]*Slot
	slot         *Slot
}

func NewStorageStream(tokenAddr common.Address, statedb vm.StateDB) *StorageStream {
	return &StorageStream{
		stateDb:      statedb,
		tokenAddress: tokenAddr,
		slots:        make(map[common.Hash]*Slot),
	}
}

func (s *StorageStream) WriteAt(b []byte, off int) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	if len(s.slots) == 0 {
		err := s.addSlot(0, func(hash common.Hash) Slot {
			return Slot{}
		})
		if err != nil {
			return 0, err
		}
	}

	var res int
	for res = 0; res < len(b); {
		if off+len(b) > common.HashLength {
			slotPos, err := position(off)
			if err != nil {
				return 0, err
			}
			size := common.HashLength - slotPos
			if len(b)-res < common.HashLength {
				size = len(b) - res
			}
			wb := copy(s.slot[slotPos:], b[:size])
			res += wb
			err = s.addSlot(off+size, func(hash common.Hash) Slot {
				return Slot{}
			})

			if err != nil {
				return 0, err
			}
			off = 0
		} else {
			res = copy(s.slot[off:], b)
		}
	}
	log.Printf("REsult Write--------%v", res)
	return res, nil
}

func (s *StorageStream) ReadAt(b []byte, off int) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}

	var res int
	for res = 0; res < len(b); {
		if off+len(b) > common.HashLength {
			slotPos, err := position(off)
			if err != nil {
				return 0, err
			}
			size := common.HashLength - slotPos
			if len(b)-res < common.HashLength {
				size = len(b) - res
				copy(b[res:], s.slot[slotPos:size])
			} else {
				copy(b[res:], s.slot[slotPos:])
			}
			wb := s.slot[slotPos : slotPos+size]
			res += len(wb)
			err = s.addSlot(off+size, func(hash common.Hash) Slot {
				return Slot(s.stateDb.GetState(s.tokenAddress, hash))
			})

			if err != nil {
				return 0, err
			}
			off = 0
		} else {
			res = copy(b, s.slot[off:])
		}
	}
	log.Printf("REsult Read--------%v", res)
	return res, nil

}

func (s *StorageStream) Flush() {
	for k, v := range s.slots {
		s.stateDb.SetState(s.tokenAddress, k, common.Hash(*v))
	}
}

func (s *StorageStream) addSlot(off int, getSlot func(common.Hash) Slot) error {
	slotIndex, err := slot(off)
	if err != nil {
		return err
	}

	slotKey := common.Hash{}
	binary.BigEndian.PutUint64(slotKey[:], slotIndex)
	slot := getSlot(slotKey)
	s.slots[slotKey] = &slot
	s.slot = &slot

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
