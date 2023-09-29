// Package stats provides Stats
package stats

// spellchecker:words rewritable

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/pkg/progress"
	"github.com/tkw1536/pkglib/lazy"
	"github.com/tkw1536/pkglib/perf"
)

// Stats holds statistical information about the current stage of the previous.
// Updating the status writes out detailed information to an underlying io.Writer.
//
// Stats is safe to access concurrently, however the caller is responsible for only logging to one stage at a time.
//
// A nil Stats is valid, and discards any information written to it.
type Stats struct {
	// done indicates if this value is finished.
	// if this status is done, no further edits may be made.
	// if it is not done, edits may be made.
	//
	// if it is, no further changes to any changes may be made.
	done atomic.Bool
	m    sync.RWMutex // m protects changes to current and all

	logger     *slog.Logger
	rewritable *progress.Rewritable

	istats lazy.Lazy[igraph.Stats]

	current StageStats   // current holds information about the current stage
	all     []StageStats // all hold information about the old stages

	// OnUpdate is called every time this stats updates.
	// OnUpdate may be nil.
	OnUpdate func(*Stats)
}

// NewStats creates a new stats object that writes statistics to the given output.
func NewStats(w io.Writer) *Stats {
	if w == nil {
		return &Stats{}
	}
	return &Stats{
		logger:     slog.New(slog.NewTextHandler(w, nil)),
		rewritable: &progress.Rewritable{Writer: w, FlushInterval: progress.DefaultFlushInterval},
	}
}

// Rewritable returns the rewritable associated with this status.
// It is automatically closed at the end of each stage
func (st *Stats) Rewritable() *progress.Rewritable {
	if st == nil {
		return nil
	}
	return st.rewritable
}

// Current returns a copy of the current StageStats
func (st *Stats) Current() StageStats {
	if st == nil {
		var zero StageStats
		return zero
	}
	st.m.RLock()
	defer st.m.RUnlock()
	return st.current
}

// onUpdate calls the onUpdate handler.
func (st *Stats) onUpdate() {
	if st == nil || st.OnUpdate == nil {
		return
	}
	st.OnUpdate(st)
}

// StoreIndexStats optionally stores index statistics.
// If st is nil or done, this call has no effect
func (st *Stats) StoreIndexStats(stats igraph.Stats) {
	defer st.onUpdate()

	if st == nil || st.done.Load() {
		return
	}

	st.istats.Set(stats)
}

// IndexStats returns the current stats for the index
func (st *Stats) IndexStats() igraph.Stats {
	if st == nil {
		var zero igraph.Stats
		return zero
	}
	return st.istats.Get(nil)
}

// Current returns a copy of the current StageStats
func (st *Stats) All() []StageStats {
	if st == nil {
		return []StageStats{}
	}

	st.m.RLock()
	defer st.m.RUnlock()

	all := append([]StageStats{}, st.all...)
	if st.current.Stage != StageInitial {
		all = append(all, st.current)
	}
	return all
}

type Progress struct {
	Done bool // Done indicates if the viewer is currently done

	Stage          Stage
	Current, Total int
}

// Progress returns information about the current stage
func (st *Stats) Progress() (progress Progress) {
	// fast path: we're already done
	if st.Done() {
		return Progress{Done: true}
	}

	// load the current stage
	st.m.RLock()
	{
		progress.Stage = st.current.Stage
		progress.Current = st.current.Current
		progress.Total = st.current.Total
	}
	st.m.RUnlock()

	// check again if we're done now
	if st.Done() {
		return Progress{Done: true}
	}

	return progress
}

// Log logs an informational message with the provided key, value field pairs.
//
// When status or the associated logger are nil, no logging occurs.
func (st *Stats) Log(message string, fields ...any) {
	if st == nil || st.logger == nil {
		return
	}
	st.logger.Info(message, fields...)
}

// Close marks this status as done.
// Future edits will have no effect.
func (st *Stats) Close() {
	if st == nil {
		return
	}
	st.done.Store(true)
}

// Done checks if further edits made to this status have any effect.
func (st *Stats) Done() bool {
	return st == nil || st.done.Load()
}

// Log logs a debug message with the provided key, value field pairs.
//
// When status or the associated logger are nil, no logging occurs.
func (st *Stats) LogDebug(message string, fields ...any) {
	if st == nil || st.logger == nil {
		return
	}
	st.logger.Debug(message, fields...)
}

// LogError logs an error message containing the provided error and the provided key, value field pairs.
//
// When status or the associated logger are nil, no logging occurs.
func (st *Stats) LogError(message string, err error, fields ...any) {
	if st == nil || st.logger == nil {
		return
	}

	st.logger.Error("FAILED "+message, append([]any{"err", err}, fields...)...)
}

// LogFatal is like LogError followed by os.Exit(1).
// When the associated logger are nil, os.Exit(1) is called immediately.
func (st *Stats) LogFatal(message string, err error) {
	st.LogError(message, err)
	os.Exit(1)
}

// Diff returns a performance diff starting at the first, and ending at the last stage.
// If status is nil, a nil diff is returned.
func (st *Stats) Diff() perf.Diff {
	// if there is no status, don't do a diff
	if st == nil || st.done.Load() {
		var zero perf.Diff
		return zero
	}

	st.m.RLock()
	defer st.m.RUnlock()

	min := st.current.Start
	max := st.current.End

	for _, ss := range st.all {
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
// If st is done or nil, this function has no effect.
func (st *Stats) Start(stage Stage) {
	if st == nil || st.done.Load() {
		return
	}

	defer st.onUpdate()

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
func (st *Stats) End() (prev StageStats) {
	if st == nil || st.done.Load() {
		return
	}

	defer st.onUpdate()

	st.m.Lock()
	defer st.m.Unlock()

	return st.end()
}

// end implements End.
// st must not be nil st.m must be held for writing.
func (st *Stats) end() (prev StageStats) {
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
	if st.rewritable != nil {
		st.rewritable.Flush(true)
		st.rewritable.Close() // reset it!
	}

	// log that we finished the stage
	// and write out the perf
	if st.logger != nil {
		if prev.Total != 0 || prev.Current != 0 {
			st.logger.Info("end", "stage", prev.Stage, "took", prev.Diff(), "current", prev.Current, "total", prev.Total)
		} else {
			st.logger.Info("end", "stage", prev.Stage, "took", prev.Diff())
		}
	}
	return
}

// DoStage is a convenience wrapper to start a new stage, call f, and log the resulting error if any.
//
// If st is nil, immediately invokes f.
func (st *Stats) DoStage(stage Stage, f func() error) error {
	if st == nil || st.done.Load() {
		return f()
	}

	st.Start(stage)

	err := f()

	st.m.Lock()
	defer st.m.Unlock()

	// an err occurred => write the stats
	if err != nil {

		st.end()

		if st.rewritable != nil {
			st.rewritable.Close()
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

// SetCT sets the current and total for the given stage.
// It the status is nil, or the status is done, has no effect.
func (st *Stats) SetCT(current, total int) {
	if st == nil || st.done.Load() {
		return
	}

	// update the process and make a copy
	var progress string

	defer st.onUpdate()

	st.m.Lock()
	{
		st.current.Current = current
		st.current.Total = total
		progress = st.current.Progress()
	}
	st.m.Unlock()

	// and write out the rewritable
	if st.rewritable != nil {
		st.rewritable.Write(progress)
	}
}

// Progress returns a string holding progress information on the current stage
func (ss StageStats) Progress() string {
	if ss.Total == 0 {
		return ""
	}
	if ss.Current < ss.Total {
		return fmt.Sprintf("%s: %d/%d", string(ss.Stage), ss.Current, ss.Total)
	} else {
		return fmt.Sprintf("%s: %d", string(ss.Stage), ss.Current)
	}
}

// Diff returns a diff of the given stage
func (ss StageStats) Diff() perf.Diff {
	return ss.End.Sub(ss.Start)
}

// Stage represents a stage used for statistics
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
	StageExtractSameAs   Stage = "sameas"
	StageExtractBundles  Stage = "bundles"
	StageExtractCache    Stage = "cache"
	StageHandler         Stage = "handler"
)
