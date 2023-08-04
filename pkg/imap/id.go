package imap

import (
	"errors"
	"fmt"
	"math/big"
)

// ID represents an ID of a specific element.
// Not all IDs are valid, see [Valid].
//
// Users should not rely on the exact size of this data type.
// Instead they should use appropriate methods to compare values.
//
// Internally, an ID is represented in big endian array of bytes.
// It effectively corresponds to a uint32.
type ID [4]byte

// IDLen is the length of the ID type
const IDLen = len(ID{})

// Valid checks if this ID is valid
func (id ID) Valid() bool {
	// NOTE(twiesing): We start this loop from the back as a performance optimisation
	// because most IDs are expected to be small, so most of them will be faster
	for i := IDLen - 1; i >= 0; i-- {
		if id[i] != 0 {
			return true
		}
	}
	return false
}

// Reset resets this id to an invalid value
func (id *ID) Reset() {
	// TODO: Untested
	for i := 0; i < IDLen; i++ {
		(*id)[i] = 0
	}
}

// Inc increments this ID, returning a copy of the new value.
// It is the equivalent of the "++" operator.
//
// When Inc() exceeds the maximum possible value for an ID, panics.
func (id *ID) Inc() ID {
	for i := IDLen - 1; i >= 0; i-- {
		(*id)[i]++
		if (*id)[i] != 0 {
			return *id
		}
	}

	// NOTE(twiesing): If this line is ever reached we should increase the size of the ID type.
	panic("Inc: Overflow")
}

// Int writes the numerical value of this id into the given big int.
// The big.Int is returned for convenience.
func (id ID) Int(value *big.Int) *big.Int {
	bytes := make([]byte, IDLen)
	id.Encode(bytes)
	return value.SetBytes(bytes)
}

// LoadInt sets the value of this id as an integer and returns it.
// Trying to load an integer bigger than the maximal id results in a panic.
//
// The ID is returned for convenience.
func (id *ID) LoadInt(value *big.Int) *ID {
	id.Decode(value.FillBytes(make([]byte, IDLen)))
	return id
}

// Less compares this ID to another id.
// An id is less than another id iff Inc() has been called fewer times.
func (id ID) Less(other ID) bool {
	for i := 0; i < IDLen; i++ {
		if id[i] < other[i] {
			return true
		}
		if id[i] > other[i] {
			return false
		}
	}
	return false
}

// String formats this id as a string.
// It is only intended for debugging, and should not be used for production code.
func (id ID) String() string {
	return fmt.Sprintf("ID(%v)", id.Int(big.NewInt(0)))
}

// Encode encodes id using a big endian encoding into dest.
// dest must be of at least size [IDLen].
//
// Comparing two distinct slices using [bytes.Compare] produces the same result
// as using appropriate calls [Less].
func (id ID) Encode(dest []byte) {
	_ = dest[IDLen-1] // boundary hint to compiler
	copy(dest[:], id[:])
}

// Decode sets this id to be the values that has been decoded from src.
// src must be of at least size IDLen, or a runtime panic occurs.
func (id *ID) Decode(src []byte) {
	_ = src[IDLen-1] // boundary hint to compiler
	copy((*id)[:], src[:])
}

var errMarshal = errors.New("MarshalIDs: invalid length")

// MarshalIDs is like EncodeIDs, but also takes a []byte to write to
func MarshalIDs(dst []byte, ids ...ID) error {
	if len(dst) < len(ids)*IDLen {
		return errMarshal
	}
	for i := 0; i < len(ids); i++ {
		ids[i].Encode(dst[i*IDLen:])
	}
	return nil
}

// MarshalID is like MarshalIDs, but takes takes only a single value
func MarshalID(value ID) ([]byte, error) {
	dest := make([]byte, IDLen)
	return dest, MarshalIDs(dest, value)
}

// EncodeIDs encodes IDs into a new slice of bytes.
// Each id is encoded sequentially using [Encode].
func EncodeIDs(ids ...ID) []byte {
	bytes := make([]byte, len(ids)*IDLen)
	MarshalIDs(bytes, ids...)
	return bytes
}

// DecodeIDs decodes a set of ids encoded with [EncodeIDs].
// The behaviour of slices that do not evenly divide into IDs is not defined.
func DecodeIDs(src []byte) []ID {
	ids := make([]ID, len(src)/IDLen)
	for i := 0; i < len(ids); i++ {
		ids[i].Decode(src[i*IDLen:])
	}
	return ids
}

// DecodeID works like DecodeIDs, but only decodes the id with index i
func DecodeID(src []byte, index int) (id ID) {
	id.Decode(src[index*IDLen:])
	return
}

// MarshalIDPair is like MarshalID but takes two ids
func MarshalIDPair(values [2]ID) ([]byte, error) {
	return EncodeIDs(values[0], values[1]), nil
}

var errUnmarshal = errors.New("unmarshalID: invalid length")

// UnmarshalID behaves like [dest.Decode], but produces an error
// when there are insufficient number of bytes in src.
func UnmarshalID(dest *ID, src []byte) error {
	if len(src) < IDLen {
		return errUnmarshal
	}
	dest.Decode(src)
	return nil
}

// UnmarshalIDs is like UnmarshalID but decodes into every destination passed.
func UnmarshalIDs(src []byte, dests ...*ID) error {
	if len(src) < len(dests)*IDLen {
		return errUnmarshal
	}
	for i, dest := range dests {
		dest.Decode(src[i*IDLen:])
	}
	return nil
}

// UnmarshalIDPair is like UnmarshalID but takes two ids.
func UnmarshalIDPair(dest *[2]ID, src []byte) error {
	return UnmarshalIDs(src, &(*dest)[0], &(*dest)[1])
}
