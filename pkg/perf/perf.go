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
	s.Bytes, s.Objects = measureHeapCount()
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

// Since computes the diff between now, and the previous point in time
func Since(start Snapshot) Diff {
	bytes, objects := measureHeapCount()
	return Diff{
		Time:    time.Since(start.Time),
		Bytes:   bytes - start.Bytes,
		Objects: objects - start.Objects,
	}
}

const (
	measureHeapThreshold = 10 * 1024                           // number of bytes to be considered stable time
	measureHeapSleep     = 50 * time.Millisecond               // amount of time to sleep between measuring cycles
	measureMaxCycles     = int(time.Second / measureHeapSleep) // maximal cycles to run
)

// measureHeapCount measures the current use of the heap
func measureHeapCount() (heapcount int64, objects int64) {
	// NOTE(twiesing): This has been vaguely adapted from https://dev.to/vearutop/estimating-memory-footprint-of-dynamic-structures-in-go-2apf

	var stats runtime.MemStats

	var prevHeapUse, currentHeapUse uint64
	var prevGCCount, currentGCCount uint32

	for i := 0; i < measureMaxCycles; i++ {
		runtime.ReadMemStats(&stats)
		currentGCCount = stats.NumGC
		currentHeapUse = stats.HeapInuse

		if prevGCCount != 0 && currentGCCount > prevGCCount && math.Abs(float64(currentHeapUse-prevHeapUse)) < measureHeapThreshold {
			break
		}

		prevHeapUse = currentHeapUse
		prevGCCount = currentGCCount

		time.Sleep(measureHeapSleep)
		runtime.GC()
	}

	return int64(currentHeapUse + stats.StackInuse), int64(stats.HeapObjects)
}
