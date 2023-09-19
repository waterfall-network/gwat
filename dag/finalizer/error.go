package finalizer

import (
	"errors"
)

var (
	// ErrBusy returned when unknown block detected in finalizing chain.
	ErrBusy = errors.New("busy")

	// ErrSpineNotFound if spine block not found
	ErrSpineNotFound = errors.New("spine not found")

	// ErrFinNrrUsed throws if fin nr is already used
	ErrFinNrrUsed = errors.New("fin nr is already used")

	// ErrBadParams returned when received unacceptable params.
	ErrBadParams = errors.New("bad params")
)
