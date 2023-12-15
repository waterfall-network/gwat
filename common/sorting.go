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
