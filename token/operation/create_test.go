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
		percentFee  uint8
	}

	cases := []operationTestCase{
		{
			caseName: "WRC20_Correct",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        opName,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
			},
			encoded: []byte{
				243, 12, 214, 20, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 100, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "WRC20_EmptyName",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        nil,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
			},
			encoded: []byte{243, 12, 201, 20, 128, 130, 84, 84, 130, 39, 15, 100},
			errs:    []error{ErrNoName},
		},
		{
			caseName: "WRC20_EmptySymbol",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        opName,
				symbol:      nil,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
			},
			encoded: []byte{
				243, 12, 223, 128, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 128, 128, 100, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109,
			},
			errs: []error{ErrNoSymbol},
		},
		{
			caseName: "WRC20_EmptyTokenSupply",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        opName,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: nil,
			},
			encoded: []byte{243, 12, 210, 20, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 128, 100},
			errs:    []error{ErrNoTokenSupply},
		},
		{
			caseName: "WRC20_ZeroDecimals",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        opName,
				symbol:      opSymbol,
				decimals:    0, // will be use DefaultDecimals
				totalSupply: opTotalSupply,
			},
			encoded: []byte{243, 12, 214, 20, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 18, 128, 128},
			errs:    []error{},
		},
		{
			caseName: "WRC721_Correct",
			decoded: decodedOp{
				op:          StdWRC721,
				name:        opName,
				symbol:      opSymbol,
				decimals:    opDecimals,
				totalSupply: opTotalSupply,
				baseURI:     opBaseURI,
				percentFee:  opPercentFee,
			},
			encoded: []byte{
				243, 12, 228, 130, 2, 209, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 128, 128, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109, 10,
			},
			errs: []error{},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		switch o.op {
		case StdWRC20:
			decPtr := &o.decimals
			if o.decimals == 0 {
				decPtr = nil
			}
			createOp, err := NewWrc20CreateOperation(
				o.name,
				o.symbol,
				decPtr,
				o.totalSupply,
			)
			if err != nil {
				return err
			}

			return equalOpBytes(createOp, b)
		case StdWRC721:
			createOp, err := NewWrc721CreateOperation(
				o.name,
				o.symbol,
				o.baseURI,
				&o.percentFee,
			)
			if err != nil {
				return err
			}

			return equalOpBytes(createOp, b)
		default:
			return ErrStandardNotValid
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

		err = checkOpCodeAndStandard(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		if !bytes.Equal(opDecoded.Name(), o.name) {
			return fmt.Errorf("name do not match:\nwant: %+v\nhave: %+v", o.name, opDecoded.Name())
		}

		if !bytes.Equal(opDecoded.Symbol(), o.symbol) {
			return fmt.Errorf("symbol do not match:\nwant: %+v\nhave: %+v", o.symbol, opDecoded.Symbol())
		}

		switch o.op {
		case StdWRC20:
			expDec := o.decimals
			if expDec == 0 {
				expDec = DefaultDecimals
			}
			if opDecoded.Decimals() != expDec {
				return fmt.Errorf("values do not match:\nwant: %d, have: %d", expDec, opDecoded.Decimals())
			}

			tS, ok := opDecoded.TotalSupply()
			if !ok || len(tS.Bytes()) == 0 {
				return ErrNoTokenSupply
			}

			if !testutils.BigIntEquals(tS, o.totalSupply) {
				return fmt.Errorf("expected totalSupply: %s, got: %s", o.totalSupply.String(), tS.String())
			}

			expDecimals := o.decimals
			if expDecimals == 0 {
				expDecimals = DefaultDecimals
			}
			if opDecoded.Decimals() != expDecimals {
				return fmt.Errorf("expected decimal: %d, got: %d", expDecimals, opDecoded.Decimals())
			}
		case StdWRC721:
			uri, _ := opDecoded.BaseURI()
			if !bytes.Equal(uri, o.baseURI) {
				return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.baseURI, uri)
			}

			percentFee := opDecoded.PercentFee()
			if o.percentFee != percentFee {
				return fmt.Errorf("expected percentFee: %d, got: %d", o.percentFee, percentFee)
			}
		default:
			return ErrStandardNotValid
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
