package headsync

import (
	"errors"
)

var (
	// ErrBusy returned when process busy.
	ErrBusy = errors.New("busy")

	// ErrNotReady returned if state not ready to head sync.
	ErrNotReady = errors.New("not ready")

	// ErrBadParams returned when received unacceptable params.
	ErrBadParams = errors.New("bad params")

	// ErrUnknownHash returned if bc.getBlock(hash) == nil
	ErrUnknownHash = errors.New("unknown spine hash")

	// ErrCheckpointNotFin returned if checkpoint isn't finalized
	ErrCheckpointNotFin = errors.New("checkpoint not finalized")

	// ErrCheckpointBadNr  returned when height of checkpoint mismatch to finalizing number.
	ErrCheckpointBadNr = errors.New("checkpoint bad number")

	// ErrCheckpointNoState returned if checkpoint state not found
	ErrCheckpointNoState = errors.New("checkpoint state not found")
)
