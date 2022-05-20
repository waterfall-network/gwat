package creator

import "github.com/waterfall-foundation/gwat/common"

// Assignment represents consensus data of block creators assignment
type Assignment struct {
	Slot     uint64
	Epoch    uint64
	Creators []common.Address
}
