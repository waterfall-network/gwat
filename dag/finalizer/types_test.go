package finalizer

import (
	"database/sql/driver"
	"fmt"
	"github.com/waterfall-foundation/gwat/common"
	"reflect"
	"testing"
)

func TestNrHashMap_ToBytes(t *testing.T) {
	nhm := make(NrHashMap, 4)
	nhm[55] = nil
	nhm[56] = &common.Hash{0x11}
	nhm[57] = &common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22}
	nhm[58] = &common.Hash{0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33}

	bts := []byte{
		0, 0, 0, 0, 0, 0, 0, 55,

		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 56,

		17, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 57,

		34, 34, 34, 34, 34, 34, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 58,

		51, 51, 51, 51, 51, 51, 51, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,
	}

	tests := []struct {
		name    string
		inst    NrHashMap
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "ToBytes",
			inst:    nhm,
			want:    fmt.Sprintf("%v", bts),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//got := fmt.Sprintf("%v", tt.inst.ToBytes())
			got := fmt.Sprintf("%v", nhm.ToBytes())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\ngot  %v,\nwant %v", got, tt.want)
			}
		})
	}
}

func TestNrHashMap_SetBytes(t *testing.T) {
	nhm := make(NrHashMap, 4)
	//nhm[55] = &common.Hash{}
	nhm[55] = nil
	nhm[56] = &common.Hash{0x11}
	nhm[57] = &common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22}
	nhm[58] = &common.Hash{0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33}

	bts := []byte{
		0, 0, 0, 0, 0, 0, 0, 55,

		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 56,

		17, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 57,

		34, 34, 34, 34, 34, 34, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,

		0, 0, 0, 0, 0, 0, 0, 58,

		51, 51, 51, 51, 51, 51, 51, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0,
	}

	tests := []struct {
		name    string
		inst    NrHashMap
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "ToBytes",
			inst:    NrHashMap{},
			want:    fmt.Sprintf("%v", &nhm),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("%v", tt.inst.SetBytes(bts))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\ngot  %v,\nwant %v", got, tt.want)
			}
		})
	}
}
