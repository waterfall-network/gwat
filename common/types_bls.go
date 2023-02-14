package common

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
)

// Lengths
const (
	// BlsPubKeyLength is the expected length of the bls public key
	BlsPubKeyLength = 48
	// BlsSigLength is the expected length of the bls public key
	BlsSigLength = 96
)

var (
	blsPubKeyT = reflect.TypeOf(BlsPubKey{})
	blsSigT    = reflect.TypeOf(BlsSignature{})
)

// BlsPubKey represents the 32 byte Keccak256 hash of arbitrary data.
type BlsPubKey [BlsPubKeyLength]byte

// BytesToBlsPubKey sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToBlsPubKey(b []byte) BlsPubKey {
	var h BlsPubKey
	h.SetBytes(b)
	return h
}

// BigToBlsPubKey sets byte representation of b to bls hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToBlsPubKey(b *big.Int) BlsPubKey { return BytesToBlsPubKey(b.Bytes()) }

// HexToBlsPubKey sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToBlsPubKey(s string) BlsPubKey { return BytesToBlsPubKey(FromHex(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h BlsPubKey) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h BlsPubKey) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h BlsPubKey) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h BlsPubKey) TerminalString() string {
	return fmt.Sprintf("%x..%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h BlsPubKey) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter.
// BlsPubKey supports the %v, %s, %q, %x, %X and %d format verbs.
func (h BlsPubKey) Format(s fmt.State, c rune) {
	hexb := make([]byte, 2+len(h)*2)
	copy(hexb, "0x")
	hex.Encode(hexb[2:], h[:])

	switch c {
	case 'x', 'X':
		if !s.Flag('#') {
			hexb = hexb[2:]
		}
		if c == 'X' {
			hexb = bytes.ToUpper(hexb)
		}
		fallthrough
	case 'v', 's':
		s.Write(hexb)
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(hexb)
		s.Write(q)
	case 'd':
		fmt.Fprint(s, ([len(h)]byte)(h))
	default:
		fmt.Fprintf(s, "%%!%c(hash=%x)", c, h)
	}
}

// UnmarshalText parses a hash in hex syntax.
func (h *BlsPubKey) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlsPubKey", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *BlsPubKey) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(blsPubKeyT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h BlsPubKey) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *BlsPubKey) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-BlsPubKeyLength:]
	}

	copy(h[BlsPubKeyLength-len(b):], b)
}

// Generate implements testing/quick.Generator.
func (h BlsPubKey) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Scan implements Scanner for database/sql.
func (h *BlsPubKey) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into BlsPubKey", src)
	}
	if len(srcB) != BlsPubKeyLength {
		return fmt.Errorf("can't scan []byte of len %d into BlsPubKey, want %d", len(srcB), BlsPubKeyLength)
	}
	copy(h[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (h BlsPubKey) Value() (driver.Value, error) {
	return h[:], nil
}

// ImplementsGraphQLType returns true if BlsPubKey implements the specified GraphQL type.
func (BlsPubKey) ImplementsGraphQLType(name string) bool { return name == "Bytes32" }

// UnmarshalGraphQL unmarshals the provided GraphQL query data.
func (h *BlsPubKey) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = h.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for BlsPubKey", input)
	}
	return err
}

// UnprefixedBlsPubKey allows marshaling a BlsPubKey without 0x prefix.
type UnprefixedBlsPubKey BlsPubKey

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedBlsPubKey) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedBlsPubKey", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedBlsPubKey) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

// BlsSignature represents the 32 byte Keccak256 hash of arbitrary data.
type BlsSignature [BlsSigLength]byte

// BytesToBlsSig sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToBlsSig(b []byte) BlsSignature {
	var h BlsSignature
	h.SetBytes(b)
	return h
}

// BigToBlsSig sets byte representation of b to bls hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToBlsSig(b *big.Int) BlsSignature { return BytesToBlsSig(b.Bytes()) }

// HexToBlsSig sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToBlsSig(s string) BlsSignature { return BytesToBlsSig(FromHex(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h BlsSignature) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h BlsSignature) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h BlsSignature) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h BlsSignature) TerminalString() string {
	return fmt.Sprintf("%x..%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h BlsSignature) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter.
// BlsSignature supports the %v, %s, %q, %x, %X and %d format verbs.
func (h BlsSignature) Format(s fmt.State, c rune) {
	hexb := make([]byte, 2+len(h)*2)
	copy(hexb, "0x")
	hex.Encode(hexb[2:], h[:])

	switch c {
	case 'x', 'X':
		if !s.Flag('#') {
			hexb = hexb[2:]
		}
		if c == 'X' {
			hexb = bytes.ToUpper(hexb)
		}
		fallthrough
	case 'v', 's':
		s.Write(hexb)
	case 'q':
		q := []byte{'"'}
		s.Write(q)
		s.Write(hexb)
		s.Write(q)
	case 'd':
		fmt.Fprint(s, ([len(h)]byte)(h))
	default:
		fmt.Fprintf(s, "%%!%c(hash=%x)", c, h)
	}
}

// UnmarshalText parses a hash in hex syntax.
func (h *BlsSignature) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlsSignature", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *BlsSignature) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(blsSigT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h BlsSignature) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *BlsSignature) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-BlsSigLength:]
	}

	copy(h[BlsSigLength-len(b):], b)
}

// Generate implements testing/quick.Generator.
func (h BlsSignature) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(len(h))
	for i := len(h) - 1; i > m; i-- {
		h[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Scan implements Scanner for database/sql.
func (h *BlsSignature) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into BlsSignature", src)
	}
	if len(srcB) != BlsSigLength {
		return fmt.Errorf("can't scan []byte of len %d into BlsSignature, want %d", len(srcB), BlsSigLength)
	}
	copy(h[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (h BlsSignature) Value() (driver.Value, error) {
	return h[:], nil
}

// ImplementsGraphQLType returns true if BlsSignature implements the specified GraphQL type.
func (BlsSignature) ImplementsGraphQLType(name string) bool { return name == "Bytes32" }

// UnmarshalGraphQL unmarshals the provided GraphQL query data.
func (h *BlsSignature) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		err = h.UnmarshalText([]byte(input))
	default:
		err = fmt.Errorf("unexpected type %T for BlsSignature", input)
	}
	return err
}

// UnprefixedBlsSig allows marshaling a BlsSignature without 0x prefix.
type UnprefixedBlsSig BlsSignature

// UnmarshalText decodes the hash from hex. The 0x prefix is optional.
func (h *UnprefixedBlsSig) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedUnprefixedText("UnprefixedBlsSig", input, h[:])
}

// MarshalText encodes the hash as hex.
func (h UnprefixedBlsSig) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}
