// Package perf captures performance metrics
package perf

import (
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/dustin/go-humanize"
)

// Snapshot represents metrics at a specific point in time
type Snapshot struct {
	Time    time.Time
	Bytes   int64
	Objects int64
}

func (snapshot Snapshot) String() string {
	var plural string
	if snapshot.Objects != 1 {
		plural = "s"
	}
	return fmt.Sprintf("%s (%d object%s) used at %s", humanize.Bytes(uint64(snapshot.Bytes)), snapshot.Objects, plural, snapshot.Time.Format(time.Stamp))
}

// Sub subtracts the metrics for a difference snapshots
func (s Snapshot) Sub(other Snapshot) Diff {
	return Diff{
		Time: s.Time.Sub(other.Time),
	}
}

// Now returns a snapshot for the current moment
func Now() (s Snapshot) {
	s.Time = time.Now()
	s.Bytes, s.Objects = measureHeapCount()
	return
}

// Diff represents the difference between two points in time
type Diff struct {
	Time    time.Duration
	Bytes   int64
	Objects int64
}

func (diff Diff) SetBytes(bytes int64) Diff {
	diff.Bytes = bytes
	return diff
}

func (diff Diff) String() string {
	var plural string
	if diff.Objects != 1 {
		plural = "s"
	}

	return fmt.Sprintf("%s, %s, %d object%s", diff.Time, human(diff.Bytes), diff.Objects, plural)
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
	measureHeapSleep     = 50 * time.Millisecond               // amount of time to sleep between measuring cyles
	measureMaxCyles      = int(time.Second / measureHeapSleep) // maximal cycles to run
)

// measureHeapCount measures the current use of the heap
func measureHeapCount() (heapcount int64, objects int64) {
	// NOTE(twiesing): This has been vaguely adapted from https://dev.to/vearutop/estimating-memory-footprint-of-dynamic-structures-in-go-2apf

	var stats runtime.MemStats

	var prevHeapUse, currentHeapUse uint64
	var prevGCCount, currentGCCount uint32

	for i := 0; i < measureMaxCyles; i++ {
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
