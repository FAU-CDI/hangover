// package igraph provides Index
package igraph

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

// cSpell:words igraph imap

// Index represents a searchable index of a directed labeled graph with optionally attached Data.
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
// Index may not be modified concurrently, however it is possible to run several queries concurrently.
type Index struct {
	data      imap.HashMap[impl.ID, impl.Datum]    // holds data mappings
	inverses  imap.HashMap[impl.ID, impl.ID]       // inverse ids for a given id
	language  imap.HashMap[impl.ID, impl.Language] // ids for the language of each datum
	psoIndex  ThreeStorage                         // <predicate> <subject> <object>
	posIndex  ThreeStorage                         // <predicate> <object> <subject>
	triples   imap.HashMap[impl.ID, IndexTriple]   // values for the triples
	pMask     map[impl.ID]struct{}                 // mask for predicates
	dMask     map[impl.ID]struct{}                 // mask for data
	labels    imap.IMap
	stats     Stats
	finalized atomic.Bool
	triple    impl.ID // id for a given triple
}

// Stats returns statistics from this graph
func (index *Index) Stats() Stats {
	return index.stats
}

// TripleCount returns the total number of (distinct) triples in this graph.
// Triples which have been identified will only count once.
func (index *Index) TripleCount() (count uint64, err error) {
	if index == nil {
		return 0, nil
	}
	return index.triples.Count()
}

// Triple returns the triple with the given id
func (index *Index) Triple(id impl.ID) (triple Triple, err error) {
	t, _, err := index.triples.Get(id)
	if err != nil {
		return triple, err
	}

	triple.Role = t.Role

	triple.Subject, err = index.labels.Reverse(t.Items[0].Literal)
	if err != nil {
		return triple, err
	}
	triple.SSubject, err = index.labels.Reverse(t.Items[0].Canonical)
	if err != nil {
		return triple, err
	}

	triple.Predicate, err = index.labels.Reverse(t.Items[1].Literal)
	if err != nil {
		return triple, err
	}
	triple.SPredicate, err = index.labels.Reverse(t.Items[1].Canonical)
	if err != nil {
		return triple, err
	}

	triple.Object, err = index.labels.Reverse(t.Items[2].Literal)
	if err != nil {
		return triple, err
	}
	triple.SObject, err = index.labels.Reverse(t.Items[2].Canonical)
	if err != nil {
		return triple, err
	}

	triple.Datum, _, err = index.data.Get(t.Items[2].Literal)
	if err != nil {
		return triple, err
	}

	triple.Language, err = index.language.GetZero(t.Items[2].Literal)
	if err != nil {
		return triple, err
	}

	triple.ID = id
	return triple, nil
}

// Reset resets this index and prepares all internal structures for use.
func (index *Index) Reset(engine Engine) (err error) {

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

	index.language, err = engine.Language()
	if err != nil {
		return
	}
	closers = append(closers, index.language)

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

	// reset mask and triples
	index.pMask = nil
	index.dMask = nil
	var s Stats
	index.stats = s

	return nil
}

// SetPredicateMask sets the masks for predicates
func (index *Index) SetPredicateMask(predicates map[impl.Label]struct{}) error {
	return index.setMask(predicates, &index.pMask)
}

// SetDataMask sets the masks for data
func (index *Index) SetDataMask(predicates map[impl.Label]struct{}) error {
	return index.setMask(predicates, &index.dMask)
}

func (index *Index) setMask(predicates map[impl.Label]struct{}, dest *map[impl.ID]struct{}) error {
	mask := make(map[impl.ID]struct{}, len(predicates))
	for label := range predicates {
		ids, err := index.labels.Add(label)
		if err != nil {
			return err
		}
		mask[ids.Canonical] = struct{}{}
	}

	*dest = mask
	return nil
}

func (index *Index) addMask(predicate impl.Label, mask map[impl.ID]struct{}) (imap.TripleID, bool, error) {
	// no mask => add normally
	if mask == nil {
		id, err := index.labels.Add(predicate)
		return id, true, err
	}

	// check if we have an id
	ids, ok, err := index.labels.Get(predicate)
	if err != nil {
		return ids, false, err
	}
	if !ok {
		return ids, false, err
	}

	// make sure it's contained in the map
	_, ok = mask[ids.Literal]
	return ids, ok, nil
}

// AddTriple inserts a subject-predicate-object triple into the index.
// Adding a triple more than once has no effect.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *Index) AddTriple(subject, predicate, object impl.Label) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	// add a predicate (and check if it is masked)
	p, masked, err := index.addMask(predicate, index.pMask)
	if err != nil {
		return err
	}

	// was masked out!
	if !masked {
		index.stats.MaskedPredTriples++
		return nil
	}

	// store the labels for the triple values
	s, err := index.labels.Add(subject)
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
		Role:  Regular,
		Items: [3]imap.TripleID{s, p, o},
	})

	conflicted, err := index.insert(s.Canonical, p.Canonical, o.Canonical, id)
	if err != nil {
		return err
	}
	if !conflicted {
		index.stats.DirectTriples++
	}

	i, ok, err := index.inverses.Get(p.Canonical)
	if err != nil {
		return err
	}
	if ok {
		// reverse id
		iid := index.triple.Inc()
		index.triples.Set(iid, IndexTriple{
			Role: Inverse,
			Items: [3]imap.TripleID{
				{
					Canonical: o.Canonical,
					Literal:   s.Literal,
				},
				{
					Canonical: i,
					Literal:   p.Literal,
				},
				{
					Canonical: s.Canonical,
					Literal:   o.Literal,
				},
			},
		})

		conflicted, err := index.insert(o.Canonical, i, s.Canonical, iid)
		if err != nil {
			return err
		}
		if !conflicted {
			index.stats.InverseTriples++
		}
	}
	return nil
}

// AddData inserts a non-language subject-predicate-data triple into the index.
// Adding multiple items to a specific subject with a specific predicate is supported.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *Index) AddData(subject, predicate impl.Label, object impl.Datum) error {
	return index.AddLangData(subject, predicate, object, "")
}

// AddLangData inserts a language-specific subject-predicate-data triple into the index.
// Adding multiple items to a specific subject with a specific predicate is supported.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *Index) AddLangData(subject, predicate impl.Label, object impl.Datum, lang impl.Language) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	// add a predicate (and check if it is masked)
	p, masked, err := index.addMask(predicate, index.dMask)
	if err != nil {
		return err
	}

	// was masked out!
	if !masked {
		index.stats.MaskedDataTriples++
		return nil
	}

	// get labels for subject and object
	o := index.labels.Next()
	if err := index.data.Set(o, object); err != nil {
		return err
	}

	// store the language (unless it is the default)
	if lang != "" {
		if err := index.language.Set(o, lang); err != nil {
			return err
		}
	}

	s, err := index.labels.Add(subject)
	if err != nil {
		return err
	}

	// store the original triple
	id := index.triple.Inc()
	if err := index.triples.Set(id, IndexTriple{
		Role: Data,
		Items: [3]imap.TripleID{
			s,
			p,
			{
				Canonical: o,
				Literal:   o,
			},
		},
	}); err != nil {
		return err
	}

	conflicted, err := index.insert(s.Canonical, p.Canonical, o, id)
	if err == nil && !conflicted {
		index.stats.DatumTriples++
	}
	return err
}

var errResolveConflictCorrupt = errors.New("errResolveConflict: Corrupted triple data")

func (index *Index) resolveLabelConflict(old, new impl.ID) (impl.ID, error) {
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
func (index *Index) insert(subject, predicate, object impl.ID, label impl.ID) (conflicted bool, err error) {
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
func (index *Index) MarkIdentical(new, old impl.Label) error {
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
func (index *Index) MarkInverse(left, right impl.Label) error {
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
	if err := index.inverses.Set(l.Canonical, r.Canonical); err != nil {
		return err
	}
	if err := index.inverses.Set(r.Canonical, l.Canonical); err != nil {
		return err
	}
	return nil
}

// IdentityMap writes all Labels for which has a semantically equivalent label.
// See [imap.Storage.IdentityMap].
func (index *Index) IdentityMap(storage imap.HashMap[impl.Label, impl.Label]) error {
	return index.labels.IdentityMap(storage)
}

// Compact informs the implementation to perform any internal optimizations.
func (index *Index) Compact() error {
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
func (index *Index) Finalize() error {
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
func (index *Index) Close() error {
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
