package types

import (
	"database/sql/driver"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

func TestValidatorSync_Copy(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:     2,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_1.Amount.SetString("32789456000000", 10)

	tests := []struct {
		name string
		src  *ValidatorSync
		want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: src_1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.Copy()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestValidatorSync_Key(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:     2,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		InitTxHash: common.Hash{0x01, 0x02, 0x03},
	}
	src_1.Amount.SetString("32789456000000", 10)

	want := common.Hash{0x01, 0x02, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	tests := []struct {
		name string
		src  *ValidatorSync
		want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.Key()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestValidatorSync_MarshalJSON(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:     2,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		TxHash:     nil,
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_1.Amount.SetString("32789456000000", 10)

	//fmt.Println()
	exp := []byte{123, 34, 105, 110, 105, 116, 84, 120, 72, 97, 115, 104, 34, 58, 34, 48, 120, 48, 49, 48, 50, 48, 51, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 111, 112, 84, 121, 112, 101, 34, 58, 34, 48, 120, 50, 34, 44, 34, 112, 114, 111, 99, 69, 112, 111, 99, 104, 34, 58, 34, 48, 120, 98, 50, 52, 100, 34, 44, 34, 105, 110, 100, 101, 120, 34, 58, 34, 48, 120, 98, 50, 52, 100, 34, 44, 34, 99, 114, 101, 97, 116, 111, 114, 34, 58, 34, 48, 120, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 34, 44, 34, 97, 109, 111, 117, 110, 116, 34, 58, 34, 48, 120, 49, 100, 100, 50, 54, 51, 101, 48, 57, 52, 48, 48, 34, 44, 34, 116, 120, 72, 97, 115, 104, 34, 58, 110, 117, 108, 108, 125}

	tests := []struct {
		name string
		src  *ValidatorSync
		want driver.Value
	}{
		{

			name: "Marshal-1",
			src:  src_1,
			want: exp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := tt.src.MarshalJSON()
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("got:  %v\nwant: %v", tt.src, tt.want)
			}
		})
	}
}

func TestValidatorSync_UnMarshalJSON(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:     2,
		ProcEpoch:  45645,
		Index:      45645,
		Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:     new(big.Int),
		InitTxHash: common.Hash{1, 2, 3},
	}
	src_1.Amount.SetString("32789456000000", 10)

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
			res := &ValidatorSync{}
			res.UnmarshalJSON(tt.input)
			got := fmt.Sprintf("%#v", res)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestChecpoint_ToBytes(t *testing.T) {
	src_1 := &Checkpoint{
		Epoch:    45645,
		FinEpoch: 45647,
		Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}

	want := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb2, 0x4d,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb2, 0x4F,
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	tests := []struct {
		name string
		src  *Checkpoint
		want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: fmt.Sprintf("%#x", want),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("%#x", tt.src.Bytes())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestChecpoint_BytesToCheckpoint(t *testing.T) {
	src_1 := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb2, 0x4d,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb2, 0x4F,
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	want := &Checkpoint{
		Epoch:    45645,
		FinEpoch: 45647,
		Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}

	tests := []struct {
		name string
		src  []byte
		want *Checkpoint
		//want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := BytesToCheckpoint(tt.src)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestChecpoint_Copy(t *testing.T) {
	src_1 := &Checkpoint{
		Epoch:    45645,
		FinEpoch: 45647,
		Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}

	tests := []struct {
		name string
		src  *Checkpoint
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

func TestCheckpoint_MarshalJSON(t *testing.T) {
	src_1 := &Checkpoint{
		Epoch:    45645,
		FinEpoch: 45647,
		Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}
	//fmt.Println()
	exp := "{\"epoch\":\"0xb24d\"," +
		"\"finEpoch\":\"0xb24f\"," +
		"\"root\":\"0x1111111111111111110000000000000000000000000000000000000000000000\"," +
		"\"spine\":\"0x2222222222222222220000000000000000000000000000000000000000000000\"}"
	tests := []struct {
		name string
		src  *Checkpoint
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

func TestCheckpoint_UnMarshalJSON(t *testing.T) {
	src_1 := &Checkpoint{
		Epoch:    45645,
		FinEpoch: 45647,
		Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
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
			res := &Checkpoint{}
			res.UnmarshalJSON(tt.input)
			got := fmt.Sprintf("%#v", res)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func TestFinalizationParams_Copy(t *testing.T) {
	src_1 := &FinalizationParams{
		Spines: common.HashArray{
			common.Hash{0x22, 0x22},
			common.Hash{0x33, 0x33},
		},
		BaseSpine: &common.Hash{0x11, 0x11},
		Checkpoint: &Checkpoint{
			Epoch:    45645,
			FinEpoch: 45647,
			Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{{
			OpType:     2,
			ProcEpoch:  45645,
			Index:      45645,
			Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:     new(big.Int),
			TxHash:     &common.Hash{4, 5, 6},
			InitTxHash: common.Hash{7, 8, 9},
		}},
	}
	src_1.ValSyncData[0].Amount.SetString("32789456000000", 10)

	tests := []struct {
		name string
		src  *FinalizationParams
		want driver.Value
	}{
		{
			name: "Copy-1",
			src:  src_1,
			want: src_1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src.Copy()
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
		Checkpoint: &Checkpoint{
			Epoch:    45645,
			FinEpoch: 45647,
			Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{{
			OpType:     2,
			ProcEpoch:  45645,
			Index:      45645,
			Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:     new(big.Int),
			InitTxHash: common.Hash{1, 2, 3},
		}},
		SyncMode: HeadSync,
	}
	src_1.ValSyncData[0].Amount.SetString("32789456000000", 10)

	//fmt.Println()
	exp := []byte{123, 34, 115, 112, 105, 110, 101, 115, 34, 58, 91, 34, 48, 120, 50, 50, 50, 50, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 48, 120, 51, 51, 51, 51, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 93, 44, 34, 98, 97, 115, 101, 83, 112, 105, 110, 101, 34, 58, 34, 48, 120, 49, 49, 49, 49, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 99, 104, 101, 99, 107, 112, 111, 105, 110, 116, 34, 58, 123, 34, 101, 112, 111, 99, 104, 34, 58, 34, 48, 120, 98, 50, 52, 100, 34, 44, 34, 102, 105, 110, 69, 112, 111, 99, 104, 34, 58, 34, 48, 120, 98, 50, 52, 102, 34, 44, 34, 114, 111, 111, 116, 34, 58, 34, 48, 120, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 49, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 115, 112, 105, 110, 101, 34, 58, 34, 48, 120, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 50, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 125, 44, 34, 118, 97, 108, 83, 121, 110, 99, 68, 97, 116, 97, 34, 58, 91, 123, 34, 105, 110, 105, 116, 84, 120, 72, 97, 115, 104, 34, 58, 34, 48, 120, 48, 49, 48, 50, 48, 51, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 34, 44, 34, 111, 112, 84, 121, 112, 101, 34, 58, 34, 48, 120, 50, 34, 44, 34, 112, 114, 111, 99, 69, 112, 111, 99, 104, 34, 58, 34, 48, 120, 98, 50, 52, 100, 34, 44, 34, 105, 110, 100, 101, 120, 34, 58, 34, 48, 120, 98, 50, 52, 100, 34, 44, 34, 99, 114, 101, 97, 116, 111, 114, 34, 58, 34, 48, 120, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 102, 34, 44, 34, 97, 109, 111, 117, 110, 116, 34, 58, 34, 48, 120, 49, 100, 100, 50, 54, 51, 101, 48, 57, 52, 48, 48, 34, 44, 34, 116, 120, 72, 97, 115, 104, 34, 58, 110, 117, 108, 108, 125, 93, 44, 34, 115, 121, 110, 99, 77, 111, 100, 101, 34, 58, 34, 48, 120, 50, 34, 125}
	tests := []struct {
		name string
		src  *FinalizationParams
		want driver.Value
	}{
		{
			name: "Marshal-1",
			src:  src_1,
			want: exp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := tt.src.MarshalJSON()
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("got:  %v\nwant: %v", res, tt.want)
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
		Checkpoint: &Checkpoint{
			Epoch:    45645,
			FinEpoch: 45647,
			Root:     common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine:    common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{{
			OpType:     2,
			ProcEpoch:  45645,
			Index:      45645,
			Creator:    common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:     new(big.Int),
			InitTxHash: common.Hash{1, 2, 3},
		}},
		SyncMode: MainSync,
	}
	src_1.ValSyncData[0].Amount.SetString("32789456000000", 10)

	input, _ := src_1.MarshalJSON()
	tests := []struct {
		name  string
		input []byte
		want  driver.Value
	}{
		{
			name:  "Marshal-1",
			input: input,
			want:  src_1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &FinalizationParams{}
			res.UnmarshalJSON(tt.input)
			got := res
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}
