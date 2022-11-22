package creator

import "gitlab.waterfall.network/waterfall/protocol/gwat/common"

// Assignment represents consensus data of block creators assignment
type Assignment struct {
	Slot     uint64
	Creators []common.Address
}
