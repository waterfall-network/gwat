package operation

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"math/big"
	"testing"
)

func TestCreateOperationOperation(t *testing.T) {
	type decodedOp struct {
		op          Std
		name        []byte
		symbol      []byte
		decimals    uint8
		totalSupply *big.Int
		baseURI     []byte
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        opName,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
				baseURI:     nil,
			},
			encoded: []byte{
				243, 12, 212, 20, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 100,
			},
			errs: []error{ErrNoBaseURI},
		},
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:          StdWRC721,
				name:        opName,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
				baseURI:     opBaseURI,
			},
			encoded: []byte{
				243, 12, 227, 130, 2, 209, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 128, 128, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109,
			},
			errs: []error{ErrNoTokenSupply},
		},
		{
			caseName: "No empty symbol",
			decoded: decodedOp{
				op:          0,
				name:        opName,
				symbol:      nil,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
				baseURI:     opBaseURI,
			},
			encoded: []byte{
				243, 12, 223, 128, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 128, 128, 100, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109,
			},
			errs: []error{ErrNoTokenSupply, ErrNoSymbol},
		},
		{
			caseName: "No empty token supply ",
			decoded: decodedOp{
				op:          0,
				name:        opName,
				symbol:      opSymbol,
				decimals:    0,
				totalSupply: opTotalSupply,
				baseURI:     nil,
			},
			encoded: []byte{
				243, 12, 214, 130, 2, 209, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 128,
			},
			errs: []error{ErrNoTokenSupply},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		switch o.op {
		case StdWRC20:
			createOp, err := NewWrc20CreateOperation(
				o.name,
				o.symbol,
				&o.decimals,
				o.totalSupply,
			)
			if err != nil {
				return err
			}

			equalOpBytes(t, createOp, b)
		case StdWRC721:
			createOp, err := NewWrc721CreateOperation(
				o.name,
				o.symbol,
				o.baseURI,
			)
			if err != nil {
				return err
			}

			equalOpBytes(t, createOp, b)
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Create)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		tS, ok := opDecoded.TotalSupply()
		if !ok {
			return ErrNoTokenSupply
		}

		testutils.CompareBigInt(t, tS, o.totalSupply)

		if opDecoded.Decimals() != o.decimals {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.decimals, opDecoded.Decimals())
		}

		if !bytes.Equal(opDecoded.Name(), o.name) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.name, opDecoded.Name())
		}

		if !bytes.Equal(opDecoded.Symbol(), o.symbol) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.symbol, opDecoded.Symbol())
		}

		uri, ok := opDecoded.BaseURI()
		if !ok {
			return ErrNoBaseURI
		}
		if !bytes.Equal(uri, o.baseURI) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.baseURI, uri)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
