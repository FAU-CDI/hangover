package impl

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
)

func ExampleID() {
	// create a new zero -- which isn't valid
	var zero ID
	fmt.Println(zero)
	fmt.Println(zero.Valid())

	// increment the id -- it is now valid
	fmt.Println(zero.Inc())
	fmt.Println(zero.Valid())

	// create the value 10
	var ten ID
	for range 10 {
		ten.Inc()
	}

	// compare it with other ids
	fmt.Println(zero.Compare(ten)) // 0 < 10
	fmt.Println(ten.Compare(zero)) // 10 > 0
	fmt.Println(ten.Compare(ten))  // 10 == 10

	// Output: ID(0)
	// false
	// ID(1)
	// true
	// -1
	// 1
	// 0
}

// maximum numbers for the ID "torture tests".
const (
	testIDLarge = (1 << (3 * 8)) // use a full 3 bytes
	testIDSmall = 10             // (1 << (8 + (8 / 2))) // use 2.5 bytes
)

func TestID_Int(t *testing.T) {
	var (
		id ID      // ID representation
		bi big.Int // Big Integer representation
	)

	// increment, which is guaranteed to have that value
	for i := range testIDLarge {
		bi.SetInt64(-1) // store a dirty value into the integer
		id.Int(&bi)     // decode the value

		value := int(bi.Int64())
		if value != i {
			t.Error("failed to decode incrementally", i)
		}
		id.Inc()
	}

	// next encode and decode again
	// then check if the values are identical
	for i := range testIDLarge {
		id.LoadInt(bi.SetInt64(int64(i))) // store the integer into the id

		bi.SetInt64(-1) // set a dirty value in the bigint
		id.Int(&bi)     // decode the id again

		got := int(bi.Int64())
		if got != i {
			t.Error("failed to round trip", i)
		}
	}
}

func BenchmarkID_Inc(b *testing.B) {
	var id ID
	for range b.N {
		id.Reset()

		for range testIDSmall {
			id.Inc()
		}
	}
}

func BenchmarkID_Load(b *testing.B) {
	var id ID
	var bi big.Int
	for range b.N {
		id.LoadInt(&bi)
	}
}

// Test that only non-zero values are valid.
func TestID_Valid(t *testing.T) {
	var (
		id ID      // ID representation
		bi big.Int // to load big integers
	)

	for i := range testIDLarge {
		id.LoadInt(bi.SetInt64(int64(i)))

		got := id.Valid()
		want := i != 0

		if got != want {
			t.Errorf("ID(%d).Valid() = %v, want = %v", i, got, want)
		}
	}
}

func BenchmarkID_Compare(b *testing.B) {
	b.StopTimer()

	var idI, idJ ID

	idI.LoadInt(big.NewInt(10000))
	idJ.LoadInt(big.NewInt(12))

	b.StartTimer()

	for range b.N {
		idI.Compare(idJ)
	}
}

// Test that the order of ids behaves as expected.
func TestID_Compare(t *testing.T) {
	var (
		idI, idJ ID
		big      big.Int
	)

	bytesI := make([]byte, IDLen)
	bytesJ := make([]byte, IDLen)

	// check that the .Compare() method indeed implements the order
	// that was induced by their generation
	for i := range testIDSmall {
		idI.LoadInt(big.SetInt64(int64(i))) // set i to the right value
		idI.Encode(bytesI)                  // and decode the bytes

		for j := range testIDSmall {
			idJ.LoadInt(big.SetInt64(int64(j))) // set j
			idJ.Encode(bytesJ)                  // and decode the bytes

			{
				got := idI.Compare(idJ)

				var want int
				if i < j {
					want = -1
				} else if i > j {
					want = 1
				} else {
					want = 0
				}

				if got != want {
					t.Errorf("id(%d) <> id(%d) = %d, want %d", i, j, got, want)
				}
			}

			{
				got := idJ.Compare(idI)
				var want int
				if j < i {
					want = -1
				} else if j > i {
					want = 1
				} else {
					want = 0
				}
				if got != want {
					t.Errorf("id(%d) <> id(%d) = %d, want %d", j, i, got, want)
				}
			}
		}
	}
}

const (
	testEncodeIDsMax  = 1000
	testEncodeIDsSeed = 1000
	testEncodeIDsN    = 100
)

func TestEncodeIDs(t *testing.T) {
	reader := rand.New(rand.NewSource(testEncodeIDsSeed))

	var big big.Int

	for n := 1; n < testEncodeIDsN; n++ {
		// setup a random range of ids [0 ... n)
		ids := make([]ID, n)
		values := make([]int64, n)
		for i := range ids {
			value := reader.Int63n(testEncodeIDsMax)

			values[i] = value
			ids[i].LoadInt(big.SetInt64(value))
		}

		// encode as a slice of bytes
		bytes := EncodeIDs(ids...)

		// check that random access decoding works
		for i := range n {
			DecodeID(bytes, i).Int(&big)
			got := big.Int64()
			want := values[i]
			if got != want {
				t.Errorf("DecodeID() got = %d, want = %d", got, want)
			}
		}

		// check that overall decoding works
		got := DecodeIDs(bytes)
		if !reflect.DeepEqual(ids, got) {
			t.Errorf("DecodeIDs() got = %d, want = %d", got, ids)
		}
	}
}
