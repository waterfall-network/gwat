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

// implements sorting interface for array of uint64

package common

type SorterAscU64 []uint64
type SorterDescU64 []uint64

func (a SorterAscU64) Len() int           { return len(a) }
func (a SorterAscU64) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SorterAscU64) Less(i, j int) bool { return a[i] < a[j] }

func (a SorterDescU64) Len() int           { return len(a) }
func (a SorterDescU64) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SorterDescU64) Less(i, j int) bool { return a[i] > a[j] }
