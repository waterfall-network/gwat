package creator

import (
	"errors"
)

var (
	// ErrCreatorStopped throws if creator not running
	ErrCreatorStopped = errors.New("creator stopped")

	// ErrSlotLocked throws if current consensus epoch-slot
	// less or equal then last handled.
	ErrSlotLocked = errors.New("slot locked")

	// ErrSynchronization throws if synchronization process running
	ErrSynchronization = errors.New("synchronization")

	// ErrCreatorNotActive throws if current creator is not in
	// creators list of consensus
	ErrCreatorNotActive = errors.New("creator not active")

	// ErrNoTxs throws if no assigned transactions
	ErrNoTxs = errors.New("no assigned txs")
)
