package perf_test

import (
	"fmt"
	"runtime"
	"time"

	"github.com/FAU-CDI/hangover/pkg/perf"
)

// An example of capturing performance metrics
func ExampleNow() {
	metrics := perf.Now()

	const ARRAY_SIZE = 1000
	const SLEEP = 1 * time.Second

	// some fancy and slow task
	{
		var stuff [ARRAY_SIZE]int32
		defer runtime.KeepAlive(stuff)
		time.Sleep(SLEEP)
	}

	// capture the new metrics
	diff := perf.Since(metrics)

	// check that we slept long enough
	if diff.Time >= SLEEP {
		fmt.Println("a lot of time has passed")
	}

	// check that enough memory was allocated
	if diff.Bytes >= 4*ARRAY_SIZE {
		fmt.Println("a lot of memory was allocated")
	}

	// Output: a lot of time has passed
	// a lot of memory was allocated
}

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
