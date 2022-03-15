package creator

import "github.com/ethereum/go-ethereum/common"

// Assignment represents consensus data of block creators assignment
type Assignment struct {
	Slot     uint64
	Epoch    uint64
	Creators []common.Address
}
