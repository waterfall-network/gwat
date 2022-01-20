package finalizer

import (
	"errors"
)

var (
	// ErrChainGap returned when finalizing chain has gap.
	ErrChainGap = errors.New("chain gap")

	// ErrMissmatchHeadNr returned when the finalizing chain first number mismatch with head number.
	ErrMismatchHeadNr = errors.New("mismatch with head number")

	// ErrUnknownBlock returned when unknown block detected in finalizing chain.
	ErrUnknownBlock = errors.New("unknown block detected")

	// ErrBusy returned when unknown block detected in finalizing chain.
	ErrBusy = errors.New("busy")

	// ErrBusy returned when unknown block detected in finalizing chain.
	ErrSyncing = errors.New("synchronizing")

	// ErrBlockFinalizationFailed returned when process of block finalization failed.
	ErrBlockFinalizationFailed = errors.New("block finalization failed")

	// ErrBadDag returned when dag is bad.
	ErrBadDag = errors.New("bad dag chain")

	// ErrMismatchFinalisingPosition  returned when height of blue block mismatch to finalizing number.
	ErrMismatchFinalisingPosition = errors.New("mismatch finalising block position")
)
