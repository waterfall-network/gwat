// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
