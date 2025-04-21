//spellchecker:words imap
package imap_test

//spellchecker:words math strconv testing github hangover internal triplestore imap impl
import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

//spellchecker:words itol

func ExampleIMap() {
	var mp imap.IMap
	_ = mp.Reset(&imap.MemoryMap{})

	lid := func(prefix impl.Label) func(id impl.ID, err error) {
		return func(id impl.ID, err error) {
			fmt.Println(prefix, id, err)
		}
	}

	lid2 := func(prefix impl.Label) func(id imap.TripleID, err error) {
		return func(id imap.TripleID, err error) {
			fmt.Println(prefix, id.Canonical, err)
		}
	}

	lstr := func(prefix impl.Label) func(value impl.Label, err error) {
		return func(value impl.Label, err error) {
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

	lstr("reverse")(mp.Reverse(*new(impl.ID).LoadInt(big.NewInt(1))))
	lstr("reverse")(mp.Reverse(*new(impl.ID).LoadInt(big.NewInt(2))))
	lstr("reverse")(mp.Reverse(*new(impl.ID).LoadInt(big.NewInt(3))))

	_, _ = mp.MarkIdentical("earth", "world")

	lstr("reverse<again>")(mp.Reverse(*new(impl.ID).LoadInt(big.NewInt(1))))
	lstr("reverse<again>")(mp.Reverse(*new(impl.ID).LoadInt(big.NewInt(3))))

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

// itol is like strconv.itoa, but returns a label.
func itol(i int) impl.Label {
	return impl.Label(strconv.Itoa(i))
}

// mapTest performs a test for a given engine.
func mapTest(t *testing.T, engine imap.Map, n int) {
	t.Helper()

	var mp imap.IMap
	if err := mp.Reset(engine); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := mp.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// make i == i + 1
	for i := 0; i < n; i += 2 {
		canon, err := mp.MarkIdentical(itol(i), itol(i+1))
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
	for i := range n {
		id, err := mp.Forward(itol(i))
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
	var id impl.ID
	var big big.Int
	for i := 1; i < n; i++ {
		big.SetInt64(int64(i))

		got, err := mp.Reverse(*id.LoadInt(&big))
		if err != nil {
			t.Errorf("Reverse() returned error %s", err)
		}
		want := itol(i - 1)

		if got != want {
			t.Errorf("Reverse(%s) got = %q, want = %q", &big, got, want)
		}
	}
}
