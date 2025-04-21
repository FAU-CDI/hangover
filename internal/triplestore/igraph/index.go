// package igraph provides Index
package igraph

import (
	"errors"
	"fmt"
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
	data      imap.HashMap[impl.ID, impl.Datum]  // holds data mappings
	inverses  imap.HashMap[impl.ID, impl.ID]     // inverse ids for a given id
	psoIndex  ThreeStorage                       // <predicate> <subject> <object>
	posIndex  ThreeStorage                       // <predicate> <object> <subject>
	triples   imap.HashMap[impl.ID, IndexTriple] // values for the triples
	pMask     map[impl.ID]struct{}               // mask for predicates
	dMask     map[impl.ID]struct{}               // mask for data
	labels    imap.IMap
	stats     Stats
	finalized atomic.Bool
	triple    impl.ID // id for a given triple

	// map between sources
	// TODO: extend this more
	sources       map[impl.Source]impl.ID
	reverseSource map[impl.ID]impl.Source
}

// Stats returns statistics from this graph.
func (index *Index) Stats() Stats {
	return index.stats
}

// TripleCount returns the total number of (distinct) triples in this graph.
// Triples which have been identified will only count once.
func (index *Index) TripleCount() (count uint64, err error) {
	if index == nil {
		return 0, nil
	}
	count, err = index.triples.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to count triples: %w", err)
	}
	return count, nil
}

// Triple returns the triple with the given id.
func (index *Index) Triple(id impl.ID) (triple Triple, err error) {
	t, _, err := index.triples.Get(id)
	if err != nil {
		return triple, fmt.Errorf("failed to resolve id: %w", err)
	}

	triple.Role = t.Role

	triple.Subject, err = index.labels.Reverse(t.Items[0].Literal)
	if err != nil {
		return triple, fmt.Errorf("failed to resolve reserve label: %w", err)
	}
	triple.SSubject, err = index.labels.Reverse(t.Items[0].Canonical)
	if err != nil {
		return triple, fmt.Errorf("failed to reverse semantic subject: %w", err)
	}

	triple.Predicate, err = index.labels.Reverse(t.Items[1].Literal)
	if err != nil {
		return triple, fmt.Errorf("failed to reverse predicate: %w", err)
	}
	triple.SPredicate, err = index.labels.Reverse(t.Items[1].Canonical)
	if err != nil {
		return triple, fmt.Errorf("failed to reverse semantic predicate: %w", err)
	}

	triple.Object, err = index.labels.Reverse(t.Items[2].Literal)
	if err != nil {
		return triple, fmt.Errorf("failed to reverse object: %w", err)
	}
	triple.SObject, err = index.labels.Reverse(t.Items[2].Canonical)
	if err != nil {
		return triple, fmt.Errorf("failed to reverse semantic object: %w", err)
	}

	triple.Datum, _, err = index.data.Get(t.Items[2].Literal)
	if err != nil {
		return triple, fmt.Errorf("failed to resolve datum: %w", err)
	}

	triple.Source, err = index.getSource(t.Source)
	if err != nil {
		return triple, fmt.Errorf("failed to get source: %w", err)
	}

	triple.ID = id
	return triple, nil
}

// Reset resets this index and prepares all internal structures for use.
func (index *Index) Reset(engine Engine) (err error) {
	if err = index.Close(); err != nil {
		return fmt.Errorf("failed to close index: %w", err)
	}

	var closers []io.Closer
	defer func() {
		if err != nil {
			errs := make([]error, 1, len(closers))
			errs[0] = err
			for _, closer := range closers {
				if err := closer.Close(); err != nil {
					errs = append(errs, fmt.Errorf("failed to close closer: %w", err))
				}
			}
			// errs always contains err
			// if there is more, make a joined error
			if len(errs) > 1 {
				err = errors.Join(errs...)
			}
		}
	}()

	if err := index.labels.Reset(engine); err != nil {
		return fmt.Errorf("failed to reset labels index: %w", err)
	}
	closers = append(closers, &index.labels)

	index.data, err = engine.Data()
	if err != nil {
		return fmt.Errorf("failed to initialize data: %w", err)
	}
	closers = append(closers, index.data)

	index.inverses, err = engine.Inverses()
	if err != nil {
		return fmt.Errorf("failed to initialize inverse index: %w", err)
	}
	closers = append(closers, index.inverses)

	index.psoIndex, err = engine.PSOIndex()
	if err != nil {
		return fmt.Errorf("failed to initialize PSOIndex: %w", err)
	}
	closers = append(closers, index.psoIndex)

	index.posIndex, err = engine.POSIndex()
	if err != nil {
		return fmt.Errorf("failed to initialize POS index: %w", err)
	}
	closers = append(closers, index.posIndex)

	index.triples, err = engine.Triples()
	if err != nil {
		return fmt.Errorf("failed to initialize triples: %w", err)
	}

	index.triple.Reset()
	index.finalized.Store(false)

	// reset mask and triples
	index.pMask = nil
	index.dMask = nil
	var s Stats
	index.stats = s

	index.sources = make(map[impl.Source]impl.ID)
	index.reverseSource = make(map[impl.ID]impl.Source)

	return nil
}

// SetPredicateMask sets the masks for predicates.
func (index *Index) SetPredicateMask(predicates map[impl.Label]struct{}) error {
	return index.setMask(predicates, &index.pMask)
}

// SetDataMask sets the masks for data.
func (index *Index) SetDataMask(predicates map[impl.Label]struct{}) error {
	return index.setMask(predicates, &index.dMask)
}

func (index *Index) setMask(predicates map[impl.Label]struct{}, dest *map[impl.ID]struct{}) error {
	mask := make(map[impl.ID]struct{}, len(predicates))
	for label := range predicates {
		ids, err := index.labels.Add(label)
		if err != nil {
			return fmt.Errorf("failed to add predicate label: %w", err)
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
		if err != nil {
			return id, true, fmt.Errorf("failed to add predicate label: %w", err)
		}
		return id, true, nil
	}

	// check if we have an id
	ids, ok, err := index.labels.Get(predicate)
	if err != nil {
		return ids, false, fmt.Errorf("failed to resolve predicate label: %w", err)
	}
	if !ok {
		return ids, false, nil
	}

	// make sure it's contained in the map
	_, ok = mask[ids.Literal]
	return ids, ok, nil
}

// Grow attempts to reserve space for as many data as is created by calls to AddData.
// This must happen before any call to AddData to have any effect.
// May also have no effect at all if the backend doesn't do anything.
func (index *Index) Grow(data uint64) error {
	if index.finalized.Load() {
		return ErrFinalized
	}
	if err := index.data.Grow(data); err != nil {
		return fmt.Errorf("failed to grow data: %w", err)
	}
	return nil
}

// AddTriple inserts a subject-predicate-object triple into the index.
// Adding a triple more than once has no effect.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *Index) AddTriple(subject, predicate, object impl.Label, source impl.Source) error {
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
		return fmt.Errorf("failed to add subject label: %w", err)
	}
	o, err := index.labels.Add(object)
	if err != nil {
		return fmt.Errorf("failed to add object label: %w", err)
	}
	// get the source
	g := index.addSource(source)

	// forward id
	id := index.triple.Inc()
	if err := index.triples.Set(id, IndexTriple{
		Role:   Regular,
		Source: g,
		Items:  [3]imap.TripleID{s, p, o},
	}); err != nil {
		return fmt.Errorf("failed to add triple to index: %w", err)
	}

	conflicted, err := index.insert(s.Canonical, p.Canonical, o.Canonical, id)
	if err != nil {
		return fmt.Errorf("failed to insert canonical into index: %w", err)
	}
	if !conflicted {
		index.stats.DirectTriples++
	}

	i, ok, err := index.inverses.Get(p.Canonical)
	if err != nil {
		return fmt.Errorf("failed to get inverse: %w", err)
	}
	if ok {
		// reverse id
		iid := index.triple.Inc()
		if err := index.triples.Set(iid, IndexTriple{
			Role:   Inverse,
			Source: g,
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
		}); err != nil {
			return fmt.Errorf("failed to add to index: %w", err)
		}

		conflicted, err := index.insert(o.Canonical, i, s.Canonical, iid)
		if err != nil {
			return fmt.Errorf("failed to insert: %w", err)
		}
		if !conflicted {
			index.stats.InverseTriples++
		}
	}
	return nil
}

// addSource returns the id for the given source.
func (index *Index) addSource(source impl.Source) impl.ID {
	if id, ok := index.sources[source]; ok {
		return id
	}

	next := index.triple.Inc()
	index.sources[source] = next
	index.reverseSource[next] = source
	return next
}

var errUnknownSource = errors.New("unknown source")

func (index *Index) getSource(id impl.ID) (impl.Source, error) {
	source, ok := index.reverseSource[id]
	if !ok {
		return impl.Source{}, errUnknownSource
	}
	return source, nil
}

// AddLangData inserts a subject-predicate-data triple into the index.
// Adding multiple items to a specific subject with a specific predicate is supported.
//
// Reset must have been called, or this function may panic.
// After all Add operations have finished, Finalize must be called.
func (index *Index) AddData(subject, predicate impl.Label, object impl.Datum, source impl.Source) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	// add a predicate (and check if it is masked)
	p, masked, err := index.addMask(predicate, index.dMask)
	if err != nil {
		return fmt.Errorf("failed to add object data: %w", err)
	}

	// was masked out!
	if !masked {
		index.stats.MaskedDataTriples++
		return nil
	}

	// store the new datum for the object
	o := index.labels.Next()
	if err := index.data.Set(o, object); err != nil {
		return fmt.Errorf("failed to add object data: %w", err)
	}

	// add the subject
	s, err := index.labels.Add(subject)
	if err != nil {
		return fmt.Errorf("failed to add subject label: %w", err)
	}

	// get the source
	g := index.addSource(source)

	// store the original triple
	id := index.triple.Inc()
	if err := index.triples.Set(id, IndexTriple{
		Role:   Data,
		Source: g,
		Items: [3]imap.TripleID{
			s,
			p,
			{
				Canonical: o,
				Literal:   o,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to add triple to index: %w", err)
	}

	conflicted, err := index.insert(s.Canonical, p.Canonical, o, id)
	if err == nil && !conflicted {
		index.stats.DatumTriples++
	}
	return err
}

var errResolveConflictCorrupt = errors.New("errResolveConflict: Corrupted triple data")

func (index *Index) resolveLabelConflict(old, conflicting impl.ID) (impl.ID, error) {
	if old == conflicting {
		return old, nil
	}

	index.stats.ConflictTriples++

	// lod the old triple
	ot, ok, err := index.triples.Get(old)
	if !ok {
		return old, errResolveConflictCorrupt
	}
	if err != nil {
		return old, fmt.Errorf("failed to resolve old triple: %w", err)
	}

	// load the new triple
	nt, ok, err := index.triples.Get(conflicting)
	if !ok {
		return old, errResolveConflictCorrupt
	}
	if err != nil {
		return conflicting, fmt.Errorf("failed to resolve conflicting: %w", err)
	}

	// use the one with the smaller role
	if nt.Role < ot.Role {
		return conflicting, nil
	}
	return old, nil
}

// insert inserts the provided (subject, predicate, object) ids into the graph.
func (index *Index) insert(subject, predicate, object impl.ID, label impl.ID) (conflicted bool, err error) {
	var conflicted1, conflicted2 bool

	conflicted1, err = index.psoIndex.Add(predicate, subject, object, label, index.resolveLabelConflict)
	if err != nil {
		return false, fmt.Errorf("failed to add to pso index: %w", err)
	}
	if conflicted2, err = index.posIndex.Add(predicate, object, subject, label, index.resolveLabelConflict); err != nil {
		return false, fmt.Errorf("failed to add to pos index: %w", err)
	}
	return conflicted1 || conflicted2, nil
}

// MarkIdentical identifies the same and old labels.
// See [imap.IMap.MarkIdentical].
func (index *Index) MarkIdentical(same, old impl.Label) error {
	if index.finalized.Load() {
		return ErrFinalized
	}

	_, err := index.labels.MarkIdentical(same, old)
	if err != nil {
		return fmt.Errorf("failed to set identical: %w", err)
	}
	return nil
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
		return fmt.Errorf("failed to add left label: %w", err)
	}

	r, err := index.labels.Add(right)
	if err != nil {
		return fmt.Errorf("failed to add right label: %w", err)
	}

	if l == r {
		return nil
	}

	// store the inverses of the left and right
	if err := index.inverses.Set(l.Canonical, r.Canonical); err != nil {
		return fmt.Errorf("failed to set left-right canonical inverse: %w", err)
	}
	if err := index.inverses.Set(r.Canonical, l.Canonical); err != nil {
		return fmt.Errorf("failed to set right-left canonical inverse: %w", err)
	}
	return nil
}

// IdentityMap writes all Labels for which has a semantically equivalent label.
// See [imap.Storage.IdentityMap].
func (index *Index) IdentityMap(storage imap.HashMap[impl.Label, impl.Label]) error {
	if err := index.labels.IdentityMap(storage); err != nil {
		return fmt.Errorf("failed to get identity map: %w", err)
	}
	return nil
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

// Close closes any storages attached to this storage.
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
