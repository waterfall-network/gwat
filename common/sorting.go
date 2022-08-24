// implements sorting interface for array of uint64

package common

type SorterAskU64 []uint64
type SorterDeskU64 []uint64

func (a SorterAskU64) Len() int           { return len(a) }
func (a SorterAskU64) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SorterAskU64) Less(i, j int) bool { return a[i] < a[j] }

func (a SorterDeskU64) Len() int           { return len(a) }
func (a SorterDeskU64) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SorterDeskU64) Less(i, j int) bool { return a[i] > a[j] }
