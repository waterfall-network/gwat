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
		OpType:    2,
		ProcEpoch: 45645,
		Index:     45645,
		Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:    new(big.Int),
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

func TestValidatorSync_MarshalJSON(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:    2,
		ProcEpoch: 45645,
		Index:     45645,
		Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:    new(big.Int),
	}
	src_1.Amount.SetString("32789456000000", 10)

	//fmt.Println()
	exp := "{\"opType\":\"0x2\"," +
		"\"procEpoch\":\"0xb24d\"," +
		"\"index\":\"0xb24d\"," +
		"\"creator\":\"0xffffffffffffffffffffffffffffffffffffffff\"," +
		"\"amount\":\"0x1dd263e09400\"}"
	tests := []struct {
		name string
		src  *ValidatorSync
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

func TestValidatorSync_UnMarshalJSON(t *testing.T) {
	src_1 := &ValidatorSync{
		OpType:    2,
		ProcEpoch: 45645,
		Index:     45645,
		Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Amount:    new(big.Int),
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
		Epoch: 45645,
		Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}

	want := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb2, 0x4d,
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
		0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	want := &Checkpoint{
		Epoch: 45645,
		Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
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
		Epoch: 45645,
		Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
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
		Epoch: 45645,
		Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
	}
	//fmt.Println()
	exp := "{\"epoch\":\"0xb24d\"," +
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
		Epoch: 45645,
		Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
		Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
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
			Epoch: 45645,
			Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{&ValidatorSync{
			OpType:    2,
			ProcEpoch: 45645,
			Index:     45645,
			Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:    new(big.Int),
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
			Epoch: 45645,
			Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{&ValidatorSync{
			OpType:    2,
			ProcEpoch: 45645,
			Index:     45645,
			Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:    new(big.Int),
		}},
	}
	src_1.ValSyncData[0].Amount.SetString("32789456000000", 10)

	//fmt.Println()
	exp := "{\"spines\":[\"0x2222000000000000000000000000000000000000000000000000000000000000\"," +
		"\"0x3333000000000000000000000000000000000000000000000000000000000000\"]," +
		"\"baseSpine\":\"0x1111000000000000000000000000000000000000000000000000000000000000\"," +
		"\"checkpoint\":{" +
		"\"epoch\":\"0xb24d\"," +
		"\"root\":\"0x1111111111111111110000000000000000000000000000000000000000000000\"," +
		"\"spine\":\"0x2222222222222222220000000000000000000000000000000000000000000000\"}," +
		"\"valSyncData\":[{\"opType\":\"0x2\",\"procEpoch\":\"0xb24d\",\"index\":\"0xb24d\"," +
		"\"creator\":\"0xffffffffffffffffffffffffffffffffffffffff\",\"amount\":\"0x1dd263e09400\"}]}"

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
		Checkpoint: &Checkpoint{
			Epoch: 45645,
			Root:  common.Hash{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11},
			Spine: common.Hash{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22},
		},
		ValSyncData: []*ValidatorSync{&ValidatorSync{
			OpType:    2,
			ProcEpoch: 45645,
			Index:     45645,
			Creator:   common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			Amount:    new(big.Int),
		}},
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
