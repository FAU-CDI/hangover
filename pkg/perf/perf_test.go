package perf_test

import (
	"fmt"
	"time"

	"github.com/FAU-CDI/hangover/pkg/perf"
)

func ExampleDiff() {
	// Diff holds both the amount of time an operation took,
	// the number of bytes consumed, and the total number of allocated objects.
	diff := perf.Diff{
		Time:    15 * time.Second,
		Bytes:   100,
		Objects: 100,
	}
	fmt.Println(diff)
	// Output: 15s, 100 B, 100 objects
}
