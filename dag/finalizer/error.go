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

	// ErrBadFinalizedSequence  returned when finalizing numbers of candidates contain already finalized numbers
	// but related hashes does not match.
	ErrBadFinalizedSequence = errors.New("sequence of candidates does not match to the sequence of finalized blocks")

	// ErrSpineNotFound if spine block not found
	ErrSpineNotFound = errors.New("spine not found")

	// ErrInvalidBlock if block validation failed
	ErrInvalidBlock = errors.New("invalid block")

	// ErrFinNrrUsed throws if fin nr is already used
	ErrFinNrrUsed = errors.New("fin nr is already used")

	// ErrBadParams returned when received unacceptable params.
	ErrBadParams = errors.New("bad params")
)
