package storage

import (
	"errors"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
)

var ErrInvalidOff = errors.New("negative offset")

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

func (s *StorageStream) WriteAt(b []byte, off *big.Int) (int, error) {

	return s.do(b, off, func(streamBuf, b []byte) int {
		return copy(streamBuf, b)
	})
}

func (s *StorageStream) ReadAt(b []byte, off *big.Int) (int, error) {
	return s.do(b, off, func(streamBuf, b []byte) int {
		return copy(b, streamBuf)
	})
}

func (s *StorageStream) Flush() {
	for k, v := range s.bufferedSlots {
		s.stateDb.SetState(s.tokenAddress, k, common.Hash(*v))
	}
}

func (s *StorageStream) do(b []byte, off *big.Int, action func(streamBuf, b []byte) int) (int, error) {
	if off.Cmp(big.NewInt(0)) < 0 {
		return 0, ErrInvalidOff
	}

	slotPos, err := position(off)
	if err != nil {
		return 0, err
	}

	var res int
	for res = 0; res < len(b); {
		slotOffset := big.NewInt(0)
		err = s.getSlot(slotOffset.Add(off, big.NewInt(int64(res))))
		if err != nil {
			return 0, err
		}

		wb := action(s.buf[slotPos:], b[res:])
		res += wb
		slotPos = 0
	}

	return res, nil
}

func (s *StorageStream) getSlot(off *big.Int) error {
	slotKey, err := slot(off)
	if err != nil {
		return err
	}

	slot, ok := s.bufferedSlots[slotKey]
	if !ok {
		tmp := Slot(s.stateDb.GetState(s.tokenAddress, slotKey))
		slot = &tmp
		s.bufferedSlots[slotKey] = slot
	}

	s.buf = slot[:]

	return nil
}

func slot(shift *big.Int) (common.Hash, error) {
	res := new(big.Int)

	if shift.Cmp(big.NewInt(0)) < 0 {
		return common.Hash{}, ErrInvalidOff
	}

	res.Div(shift, big.NewInt(int64(len(Slot{}))))

	s := common.Hash{}
	res.FillBytes(s[:])

	return s, nil
}

func position(shift *big.Int) (int, error) {
	res := new(big.Int)

	if shift.Cmp(big.NewInt(0)) < 0 {
		return 0, ErrInvalidOff
	}

	res.Rem(shift, big.NewInt(int64(len(Slot{}))))

	// It`s safe because slot has 32 bytes size.
	return int(res.Uint64()), nil
}
