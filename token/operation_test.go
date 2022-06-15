package token

import (
	"bytes"
	"log"
	"math/big"
	"testing"
)

var c = &createOperation{
	operation:   operation{},
	name:        []byte("token"),
	symbol:      []byte("t"),
	decimals:    5,
	totalSupply: big.NewInt(100),
	baseURI:     nil,
}

func TestCreateOperationMarshalBinary(t *testing.T) {
	want := []byte{
		202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
	}

	decimals := c.Decimals()
	totalSupply, ok := c.TotalSupply()
	if !ok {
		t.Error("can`t read totalSupply field")
	}

	createOp, err := NewWrc20CreateOperation(c.Name(), c.Symbol(), &decimals, totalSupply)
	if err != nil {
		t.Error(err)
	}

	have, err := createOp.MarshalBinary()

	if !bytes.Equal(want, have) {
		t.Errorf("values do not match: want: %+v\nhave: %+v", want, have)
	}

	std := createOp.Standard()
	log.Println(std)
	if std != StdWRC20 {
		log.Printf("STD %v", std)
	}
}
