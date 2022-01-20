package creator

import "github.com/ethereum/go-ethereum/common"

type Assignment struct {
	Slot     uint64
	Epoch    uint64
	Creators []common.Address
}
