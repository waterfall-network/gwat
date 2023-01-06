package types

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

func TestFinalizationParams_Copy(t *testing.T) {
	src_1 := &FinalizationParams{
		Spines: common.HashArray{
			common.Hash{0x22, 0x22},
			common.Hash{0x33, 0x33},
		},
		BaseSpine: &common.Hash{0x11, 0x11},
	}

	tests := []struct {
		name string
		src  *FinalizationParams
		want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: fmt.Sprintf("%#v", src_1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("%#v", tt.src.Copy())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestFinalizationParams_MarshalJSON(t *testing.T) {
	src_1 := &FinalizationParams{
		Spines: common.HashArray{
			common.Hash{0x22, 0x22},
			common.Hash{0x33, 0x33},
		},
		BaseSpine: &common.Hash{0x11, 0x11},
	}
	//fmt.Println()
	exp := "{\"spines\":[\"0x2222000000000000000000000000000000000000000000000000000000000000\"," +
		"\"0x3333000000000000000000000000000000000000000000000000000000000000\"]," +
		"\"baseSpine\":\"0x1111000000000000000000000000000000000000000000000000000000000000\"}"

	tests := []struct {
		name string
		src  *FinalizationParams
		want driver.Value
	}{
		{
			name: "Marshal-1",
			src:  src_1,
			want: fmt.Sprintf("%s", exp),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := tt.src.MarshalJSON()
			got := fmt.Sprintf("%s", res)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestFinalizationParams_UnMarshalJSON(t *testing.T) {
	src_1 := &FinalizationParams{
		Spines: common.HashArray{
			common.Hash{0x22, 0x22},
			common.Hash{0x33, 0x33},
		},
		BaseSpine: &common.Hash{0x11, 0x11},
	}
	input, _ := src_1.MarshalJSON()
	tests := []struct {
		name  string
		input []byte
		want  driver.Value
	}{
		{
			name:  "Marshal-1",
			input: input,
			want:  fmt.Sprintf("%#v", src_1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &FinalizationParams{}
			res.UnmarshalJSON(tt.input)
			got := fmt.Sprintf("%#v", res)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}
