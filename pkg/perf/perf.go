// Package perf provides a means of capturing metrics
package perf

import (
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

// cspell:words twiesing

// Snapshot holds metrics at a specific instance
type Snapshot struct {
	// Time the snapshot was captured
	Time time.Time

	// memory in use
	Bytes int64

	// number of objects on the heap
	Objects int64
}

// BytesString returns a human-readable string representing the bytes
func (snapshot Snapshot) BytesString() string {
	return human(snapshot.Bytes)
}

// ObjectsString returns a human-readable string representing the number of objects
func (snapshot Snapshot) ObjectsString() string {
	if snapshot.Objects == 1 {
		return "1 object"
	}
	return fmt.Sprintf("%d objects", snapshot.Objects)
}

func (snapshot Snapshot) String() string {
	return fmt.Sprintf("%s (%s) used at %s", snapshot.BytesString(), snapshot.ObjectsString(), snapshot.Time.Format(time.Stamp))
}

// Sub subtracts the other snapshot from this snapshot.
func (s Snapshot) Sub(other Snapshot) Diff {
	return Diff{
		Time:    s.Time.Sub(other.Time),
		Bytes:   s.Bytes - other.Bytes,
		Objects: s.Objects - other.Objects,
	}
}

// Now returns a snapshot for the current time
func Now() (s Snapshot) {
	s.Time = time.Now()
	s.Bytes, s.Objects = perf()
	return
}

// Diff represents the difference between two snapshots
type Diff struct {
	Time    time.Duration
	Bytes   int64
	Objects int64
}

// BytesString returns a human-readable string representing the bytes
func (diff Diff) BytesString() string {
	return human(diff.Bytes)
}

// ObjectsString returns a human-readable string representing the number of objects
func (diff Diff) ObjectsString() string {
	if diff.Objects == 1 {
		return "1 object"
	}
	return fmt.Sprintf("%d objects", diff.Objects)
}

func (diff Diff) String() string {
	return fmt.Sprintf("%s, %s, %s", diff.Time, diff.BytesString(), diff.ObjectsString())
}

func human(bytes int64) string {
	if bytes < 0 {
		return "-" + humanize.Bytes(uint64(-bytes))
	}
	return humanize.Bytes(uint64(bytes))
}

// Since returns changes in metrics since start.
// It is a shortcut for start.Sub(perf.Now())
func Since(start Snapshot) (diff Diff) {
	diff.Bytes, diff.Objects = perf()
	diff.Time = time.Since(start.Time)

	diff.Bytes -= start.Bytes
	diff.Objects -= start.Objects

	return
}

const (
	measureHeapThreshold = 10 * 1024                           // number of bytes to be considered stable time
	measureHeapSleep     = 50 * time.Millisecond               // amount of time to sleep between measuring cycles
	measureMaxCycles     = int(time.Second / measureHeapSleep) // maximal cycles to run
)

// perf computes the current performance statistics.
//
// bytes hold the amount of memory used by stack and heap together in bytes.
// objects holds the number of objects on the heap.
//
// perf performs multiple measurement cycles, until the used heap memory is stable.
// the limits and maximum used values are defined by appropriate constants in this package.
func perf() (bytes int64, objects int64) {
	// NOTE(twiesing): This has been vaguely adapted from https://dev.to/vearutop/estimating-memory-footprint-of-dynamic-structures-in-go-2apf

	var stats runtime.MemStats

	var prevHeapUse, currentHeapUse uint64
	var prevGCCount, currentGCCount uint32

	for i := 0; i < measureMaxCycles; i++ {
		// read heap statistics
		runtime.ReadMemStats(&stats)
		currentGCCount = stats.NumGC
		currentHeapUse = stats.HeapInuse

		// check that there has been a garbage collection cycle
		// and the heap has been sufficiently stable
		if i != 0 && currentGCCount > prevGCCount && math.Abs(float64(currentHeapUse-prevHeapUse)) < measureHeapThreshold {
			break
		}

		// store the previous values
		prevHeapUse = currentHeapUse
		prevGCCount = currentGCCount

		// wait some time, and run the garbage collector
		// for the next iteration
		time.Sleep(measureHeapSleep)
		runtime.GC()
	}

	// compute the overall memory used, and the given number of objects on the heap
	return int64(stats.HeapInuse + stats.StackInuse), int64(stats.HeapObjects)
}
