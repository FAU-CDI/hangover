// Package status provides Status
package status

// spellchecker:words rewritable

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/FAU-CDI/hangover/pkg/perf"
	"github.com/FAU-CDI/hangover/pkg/progress"
)

type Rewritable = *progress.Rewritable

// Status holds statistical information about the current stage of the previous.
// Updating the status writes out detailed information to an underlying io.Writer.
//
// Status is safe to access concurrently, however the caller is responsible for only logging to one stage at a time.
//
// A nil Status is valid, and discards any information written to it.
type Status struct {
	m sync.RWMutex // m protects changes to current and all

	logger     *slog.Logger
	Rewritable Rewritable

	current StageStats   // current holds information about the current stage
	all     []StageStats // all hold information about the old stages
}

// NewStatus creates a new status which writes output to the given io.Writer.
// If w is nil, returns a nil Status.
func NewStatus(w io.Writer) *Status {
	if w == nil {
		return nil
	}
	return &Status{
		logger:     slog.New(slog.NewTextHandler(w, nil)),
		Rewritable: &progress.Rewritable{Writer: w, FlushInterval: progress.DefaultFlushInterval},
	}
}

// Log logs an informational message with the provided key, value field pairs.
// When status or the associated logger are nil, no logging occurs.
func (status *Status) Log(message string, fields ...any) {
	if status == nil || status.logger == nil {
		return
	}
	status.logger.Info(message, fields...)
}

// Log logs a debug message with the provided key, value field pairs.
// When status or the associated logger are nil, no logging occurs.
func (status *Status) LogDebug(message string, fields ...any) {
	if status == nil || status.logger == nil {
		return
	}
	status.logger.Info(message, fields...)
}

// LogError logs an error message containing the provided error and the provided key, value field pairs.
func (status *Status) LogError(message string, err error, fields ...any) {
	if status == nil || status.logger == nil {
		return
	}

	status.logger.Error("FAILED "+message, append([]any{"err", err}, fields...)...)
}

// LogFatal is like LogError followed by os.Exit(1).
// When status or the associated logger are nil, os.Exit(1) is called immediately.
func (status *Status) LogFatal(message string, err error) {
	status.LogError(message, err)
	os.Exit(1)
}

// Diff returns a performance diff starting at the first, and ending at the last stage.
// If status is nil, a nil diff is returned.
func (status *Status) Diff() perf.Diff {
	// if there is no status, don't do a diff
	if status == nil {
		var zero perf.Diff
		return zero
	}

	status.m.RLock()
	defer status.m.RUnlock()

	min := status.current.Start
	max := status.current.End

	for _, ss := range status.all {
		if min.Time.IsZero() || ss.Start.Time.Before(min.Time) {
			min = ss.Start
		}
		if max.Time.IsZero() || ss.End.Time.After(max.Time) {
			max = ss.End
		}
	}

	return max.Sub(min)
}

// Start starts a new stage, updating the current property.
// Any changes are written to the underlying writer.
//
// If st is nil, this function has no effect.
func (st *Status) Start(stage Stage) {
	if st == nil {
		return
	}

	st.m.Lock()
	defer st.m.Unlock()

	// end the previous stage (if any)
	st.end()

	// start a new stage
	st.current.Stage = stage
	st.current.Start = perf.Now()

	// log out the changes
	if st.logger != nil {
		st.logger.Info("start", "stage", stage)
	}
}

// End ends the current stage if any.
// Any changes are flushed to the underlying writer.
//
// If st is nil, this function has no effect.
func (st *Status) End() (prev StageStats) {
	if st == nil {
		return
	}

	st.m.Lock()
	defer st.m.Unlock()

	return st.end()
}

// end implements End.
// st.m must be held for writing.
func (st *Status) end() (prev StageStats) {
	// store the current stage (if any)
	if st.current.Stage != StageInitial {
		st.current.End = perf.Now()
		st.all = append(st.all, st.current)
		prev = st.current
	}

	// and reset the current stage
	st.current = *new(StageStats)

	// don't do anything
	if prev.Stage == StageInitial {
		return
	}

	// write the final status into the rewritable
	// and force a rewrite!
	if st.Rewritable != nil {
		st.Rewritable.Flush(true)
		st.Rewritable.Close() // reset it!
	}

	// log that we finished the stage
	// and write out the perf
	if st.logger != nil {
		st.logger.Info("end", "stage", prev.Stage, "took", prev.Diff())
	}
	return
}

// DoStage is a convenience wrapper to start a new stage, call f, and log the resulting error if any.
//
// If st is nil, immediately invokes f.
func (st *Status) DoStage(stage Stage, f func() error) error {
	if st == nil {
		return f()
	}

	st.Start(stage)

	err := f()

	st.m.Lock()
	defer st.m.Unlock()

	// an err occurred => write the stats
	if err != nil {

		st.end()

		if st.Rewritable != nil {
			st.Rewritable.Close()
		}
		st.LogError("failed stage", err, "stage", stage)
		return err
	}

	st.end()
	return nil
}

// StageStats holds the stats for a specific stage
type StageStats struct {
	Stage Stage

	Start perf.Snapshot // At the start of the stage
	End   perf.Snapshot // At the end of the stage

	Current int
	Total   int
}

// SetCT sets the current and total for the given stage
func (status *Status) SetCT(current, total int) {
	status.current.Current = current
	status.current.Total = total
	status.current.Rewrite(status.Rewritable)
}

// Rewrite writes the current stage to the given rewritable
func (ss StageStats) Rewrite(r Rewritable) {
	if r == nil {
		return
	}
	if ss.Current < ss.Total {
		r.Write(fmt.Sprintf("%s: %d/%d", string(ss.Stage), ss.Current, ss.Total))
	} else {
		r.Write(fmt.Sprintf("%s: %d", string(ss.Stage), ss.Current))
	}
}

// Diff returns a diff of the given stage
func (ss StageStats) Diff() perf.Diff {
	return ss.End.Sub(ss.Start)
}

// Stage represents an export stage
type Stage string

const (
	StageInitial         Stage = ""
	StageImportIndex     Stage = "import"
	StageExportIndex     Stage = "export"
	StageExportSQL       Stage = "export/sql"
	StageExportJSON      Stage = "export/json"
	StageReadPathbuilder Stage = "pathbuilder"
	StageScanSameAs      Stage = "index/sameas"
	StageScanInverse     Stage = "index/inverse"
	StageScanTriples     Stage = "index/triples"
	StageExtractBundles  Stage = "bundles"
	StageExtractCache    Stage = "cache"
	StageHandler         Stage = "handler"
)
