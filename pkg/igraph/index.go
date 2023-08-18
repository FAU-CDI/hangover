package igraph

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

// cSpell:words imap

// IGraph represents a searchable index of a directed labeled graph with optionally attached Data.
//
// Labels are used for nodes and edges.
// This means that the graph is defined by triples of the form (subject Label, predicate Label, object Label).
// See [AddTriple].
//
// Datum is used for data associated with the specific nodes.
// See [AddDatum].
//
// The zero value represents an empty index, but is otherwise not ready to be used.
// To fill an index, it first needs to be [Reset], and then [Finalize]d.
//
// IGraph may not be modified concurrently, however it is possible to run several queries concurrently.
type IGraph[Label comparable, Datum any] struct {
	finalized atomic.Bool

	stats Stats

	labels imap.IMap[Label]

	// data holds mappings between internal IDs and data
	data imap.KeyValueStore[imap.ID, Datum]

	inverses imap.KeyValueStore[imap.ID, imap.ID] // inverse ids for a given id

	// the triple indexes, forward and backward
	psoIndex ThreeStorage // <predicate> <subject> <object>
	posIndex ThreeStorage // <predicate> <object> <subject>

	// the id for a given triple
	triple  imap.ID
	triples imap.KeyValueStore[imap.ID, IndexTriple]
}

// Stats returns statistics from this graph
func (index *IGraph[Label, Datum]) Stats() Stats {
	return index.stats
}

// TripleCount returns the total number of (distinct) triples in this graph.
// Triples which have been identified will only count once.
func (index *IGraph[Label, Datum]) TripleCount() (count uint64, err error) {
	if index == nil {
		return 0, nil
	}
	return index.triples.Count()
}

// Triple returns the triple with the given
func (index *IGraph[Label, Datum]) Triple(id imap.ID) (triple Triple[Label, Datum], err error) {
	t, _, err := index.triples.Get(id)
	if err != nil {
		return triple, err
	}

	triple.Role = t.Role

	triple.Subject, err = index.labels.Reverse(t.Items[0])
	if err != nil {
		return triple, err
	}
	triple.SSubject, err = index.labels.Reverse(t.SItems[0])
	if err != nil {
		return triple, err
	}

	triple.Predicate, err = index.labels.Reverse(t.Items[1])
	if err != nil {
		return triple, err
	}
	triple.SPredicate, err = index.labels.Reverse(t.SItems[1])
	if err != nil {
		return triple, err
	}

	triple.Object, err = index.labels.Reverse(t.Items[2])
	if err != nil {
		return triple, err
	}
	triple.SObject, err = index.labels.Reverse(t.SItems[2])
	if err != nil {
		return triple, err
	}

	triple.Datum, _, err = index.data.Get(t.Items[2])
	if err != nil {
		return triple, err
	}

	triple.ID = id
	return triple, nil
}

// Reset resets this index and prepares all internal structures for use.
func (index *IGraph[Label, Datum]) Reset(engine Engine[Label, Datum]) (err error) {

	if err = index.Close(); err != nil {
		return err
	}

	var closers []io.Closer
	defer func() {
		if err != nil {
			for _, closer := range closers {
				closer.Close()
			}
		}
	}()

	if err := index.labels.Reset(engine); err != nil {
		return err
	}
	closers = append(closers, &index.labels)

	index.data, err = engine.Data()
	if err != nil {
		return
	}
	closers = append(closers, index.data)

	index.inverses, err = engine.Inverses()
	if err != nil {
		return
	}
	closers = append(closers, index.inverses)

	index.psoIndex, err = engine.PSOIndex()
	if err != nil {
		return
	}
	closers = append(closers, index.psoIndex)

	index.posIndex, err = engine.POSIndex()
	if err != nil {
		return
	}
	closers = append(closers, index.posIndex)

	index.triples, err = engine.Triples()
	if err != nil {
		return
	}

	index.triple.Reset()
	index.finalized.Store(false)

	return nil
}

// AddTriple inserts a subject-predicate-object triple into the index.
// Adding a triple more than once has no effect.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *IGraph[Label, Datum]) AddTriple(subject, predicate, object Label) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	// store the labels for the triple values
	s, err := index.labels.Add(subject)
	if err != nil {
		return err
	}
	p, err := index.labels.Add(predicate)
	if err != nil {
		return err
	}
	o, err := index.labels.Add(object)
	if err != nil {
		return err
	}

	// forward id
	id := index.triple.Inc()
	index.triples.Set(id, IndexTriple{
		Role:   Regular,
		SItems: [3]imap.ID{s[0], p[0], o[0]},
		Items:  [3]imap.ID{s[1], p[1], o[1]},
	})

	conflicted, err := index.insert(s[0], p[0], o[0], id)
	if err != nil {
		return err
	}
	if !conflicted {
		index.stats.DirectTriples++
	}

	i, ok, err := index.inverses.Get(p[0])
	if err != nil {
		return err
	}
	if ok {
		// reverse id
		iid := index.triple.Inc()
		index.triples.Set(iid, IndexTriple{
			Role:   Inverse,
			SItems: [3]imap.ID{o[0], i, s[0]},
			Items:  [3]imap.ID{s[1], p[1], o[1]},
		})

		conflicted, err := index.insert(o[0], i, s[0], iid)
		if err != nil {
			return err
		}
		if !conflicted {
			index.stats.InverseTriples++
		}
	}
	return nil
}

// AddData inserts a subject-predicate-data triple into the index.
// Adding multiple items to a specific subject with a specific predicate is supported.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *IGraph[Label, Datum]) AddData(subject, predicate Label, object Datum) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	// get labels for subject, predicate and object
	o := index.labels.Next()
	if err := index.data.Set(o, object); err != nil {
		return err
	}

	s, err := index.labels.Add(subject)
	if err != nil {
		return err
	}

	p, err := index.labels.Add(predicate)
	if err != nil {
		return err
	}

	// store the original triple
	id := index.triple.Inc()
	index.triples.Set(id, IndexTriple{
		Role:   Data,
		SItems: [3]imap.ID{s[0], p[0], o},
		Items:  [3]imap.ID{s[1], p[1], o},
	})

	conflicted, err := index.insert(s[0], p[0], o, id)
	if err == nil && !conflicted {
		index.stats.DatumTriples++
	}
	return err
}

var errResolveConflictCorrupt = errors.New("errResolveConflict: Corrupted triple data")

func (index *IGraph[Label, Datum]) resolveLabelConflict(old, new imap.ID) (imap.ID, error) {
	if old == new {
		return old, nil
	}

	index.stats.ConflictTriples++

	// lod the old triple
	ot, ok, err := index.triples.Get(old)
	if !ok {
		return old, errResolveConflictCorrupt
	}
	if err != nil {
		return old, err
	}

	// load the new triple
	nt, ok, err := index.triples.Get(new)
	if !ok {
		return old, errResolveConflictCorrupt
	}
	if err != nil {
		return new, err
	}

	// use the one with the smaller role
	if nt.Role < ot.Role {
		return new, nil
	}
	return old, nil

}

// insert inserts the provided (subject, predicate, object) ids into the graph
func (index *IGraph[Label, Datum]) insert(subject, predicate, object imap.ID, label imap.ID) (conflicted bool, err error) {
	var conflicted1, conflicted2 bool

	conflicted1, err = index.psoIndex.Add(predicate, subject, object, label, index.resolveLabelConflict)
	if err != nil {
		return false, err
	}
	if conflicted2, err = index.posIndex.Add(predicate, object, subject, label, index.resolveLabelConflict); err != nil {
		return false, err
	}
	return conflicted1 || conflicted2, err
}

// MarkIdentical identifies the new and old labels.
// See [imap.IMap.MarkIdentical].
func (index *IGraph[Label, Datum]) MarkIdentical(new, old Label) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	_, err := index.labels.MarkIdentical(new, old)
	return err
}

// MarkInverse marks the left and right Labels as inverse properties of each other.
// After calls to MarkInverse, no more calls to MarkIdentical should be made.
//
// Each label is assumed to have at most one inverse.
// A label may not be it's own inverse.
//
// This means that each call to AddTriple(s, left, o) will also result in a call to AddTriple(o, right, s).
func (index *IGraph[Label, Datum]) MarkInverse(left, right Label) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	l, err := index.labels.Add(left)
	if err != nil {
		return err
	}

	r, err := index.labels.Add(right)
	if err != nil {
		return err
	}

	if l == r {
		return nil
	}

	// store the inverses of the left and right
	if err := index.inverses.Set(l[0], r[0]); err != nil {
		return err
	}
	if err := index.inverses.Set(r[0], l[0]); err != nil {
		return err
	}
	return nil
}

// IdentityMap writes all Labels for which has a semantically equivalent label.
// See [imap.Storage.IdentityMap].
func (index *IGraph[Label, Datum]) IdentityMap(storage imap.KeyValueStore[Label, Label]) error {
	return index.labels.IdentityMap(storage)
}

// Compact informs the implementation to perform any internal optimizations.
func (index *IGraph[Label, Datum]) Compact() error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	var wg sync.WaitGroup
	wg.Add(6)

	var errs [6]error

	go func() {
		defer wg.Done()
		errs[0] = index.labels.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[1] = index.data.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[2] = index.inverses.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[3] = index.posIndex.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[4] = index.psoIndex.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[5] = index.triples.Compact()
	}()

	wg.Wait()
	return errors.Join(errs[:]...)
}

var ErrFinalized = errors.New("IGraph: Finalized")

// Finalize finalizes any adding operations into this graph.
//
// Finalize must be called before any query is performed,
// but after any calls to the Add* methods.
// Calling finalize multiple times is invalid.
func (index *IGraph[Label, Datum]) Finalize() error {
	if index.finalized.Swap(true) {
		return ErrFinalized
	}

	var wg sync.WaitGroup
	wg.Add(6)

	var errs [6]error

	go func() {
		defer wg.Done()
		errs[0] = index.labels.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[1] = index.data.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[2] = index.inverses.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[3] = index.posIndex.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[4] = index.psoIndex.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[5] = index.triples.Finalize()
	}()

	wg.Wait()
	return errors.Join(errs[:]...)
}

// Close closes any storages attached to this storage
func (index *IGraph[Label, Datum]) Close() error {
	var errors [5]error
	errors[0] = index.labels.Close()

	if index.data != nil {
		errors[1] = index.data.Close()
		index.data = nil
	}

	if index.inverses != nil {
		errors[2] = index.inverses.Close()
		index.inverses = nil
	}

	if index.psoIndex != nil {
		errors[3] = index.psoIndex.Close()
		index.psoIndex = nil
	}

	if index.posIndex != nil {
		errors[4] = index.posIndex.Close()
		index.posIndex = nil
	}

	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
