package imap

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"
)

func ExampleIMap() {

	var mp IMap[string]
	mp.Reset(&MemoryEngine[string]{})

	lid := func(prefix string) func(id ID, err error) {
		return func(id ID, err error) {
			fmt.Println(prefix, id, err)
		}
	}

	lid2 := func(prefix string) func(id [2]ID, err error) {
		return func(id [2]ID, err error) {
			fmt.Println(prefix, id[0], err)
		}
	}

	lstr := func(prefix string) func(value string, err error) {
		return func(value string, err error) {
			fmt.Println(prefix, value, err)
		}
	}

	lid2("add")(mp.Add("hello"))
	lid2("add")(mp.Add("world"))
	lid2("add")(mp.Add("earth"))

	lid2("add<again>")(mp.Add("hello"))
	lid2("add<again>")(mp.Add("world"))
	lid2("add<again>")(mp.Add("earth"))

	lid("get")(mp.Forward("hello"))
	lid("get")(mp.Forward("world"))
	lid("get")(mp.Forward("earth"))

	lstr("reverse")(mp.Reverse(*new(ID).LoadInt(big.NewInt(1))))
	lstr("reverse")(mp.Reverse(*new(ID).LoadInt(big.NewInt(2))))
	lstr("reverse")(mp.Reverse(*new(ID).LoadInt(big.NewInt(3))))

	mp.MarkIdentical("earth", "world")

	lstr("reverse<again>")(mp.Reverse(*new(ID).LoadInt(big.NewInt(1))))
	lstr("reverse<again>")(mp.Reverse(*new(ID).LoadInt(big.NewInt(3))))

	lid2("add<again>")(mp.Add("hello"))
	lid2("add<again>")(mp.Add("world"))
	lid2("add<again>")(mp.Add("earth"))

	// Output: add ID(1) <nil>
	// add ID(2) <nil>
	// add ID(3) <nil>
	// add<again> ID(1) <nil>
	// add<again> ID(2) <nil>
	// add<again> ID(3) <nil>
	// get ID(1) <nil>
	// get ID(2) <nil>
	// get ID(3) <nil>
	// reverse hello <nil>
	// reverse world <nil>
	// reverse earth <nil>
	// reverse<again> hello <nil>
	// reverse<again> earth <nil>
	// add<again> ID(1) <nil>
	// add<again> ID(3) <nil>
	// add<again> ID(3) <nil>
}

// engineTest performs a test for a given engine
func engineTest(t *testing.T, engine Engine[string], N int) {
	var mp IMap[string]
	mp.Reset(engine)
	defer mp.Close()

	// make i == i + 1
	for i := 0; i < N; i += 2 {
		canon, err := mp.MarkIdentical(strconv.Itoa(i), strconv.Itoa(i+1))
		if err != nil {
			t.Fatalf("MarkIdentical returned error %s", err)
		}
		got := canon.Int(big.NewInt(0)).Int64()
		want := int64(i + 1)
		if got != want {
			t.Errorf("MarkIdentical() got id = %s, want = %d", canon, want)
		}
	}

	// check that forward mappings work
	for i := 0; i < N; i++ {
		id, err := mp.Forward(strconv.Itoa(i))
		if err != nil {
			t.Errorf("Forward() returned error %s", err)
		}
		got := int(id.Int(big.NewInt(0)).Int64())
		want := i - (i % 2) + 1
		if got != want {
			t.Errorf("Forward() got = %d, want = %d", got, want)
		}
	}

	// check that reverse mappings work
	var id ID
	var big big.Int
	for i := 1; i < N; i++ {
		big.SetInt64(int64(i))

		got, err := mp.Reverse(*id.LoadInt(&big))
		if err != nil {
			t.Errorf("Reverse() returned error %s", err)
		}
		want := strconv.Itoa(i - 1)

		if got != want {
			t.Errorf("Reverse(%s) got = %q, want = %q", &big, got, want)
		}
	}
}
