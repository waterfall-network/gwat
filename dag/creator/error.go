package creator

import (
	"errors"
)

var (
	ErrCreatorStopped   = errors.New("creator stopped")
	ErrSlotLocked       = errors.New("slot locked")
	ErrSynchronization  = errors.New("synchronization")
	ErrCreatorNotActive = errors.New("creator not active")
	ErrNoTxs            = errors.New("no assigned txs")
)
