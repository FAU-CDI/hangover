// Package imap
//
//spellchecker:words imap
package imap

//spellchecker:words errors sync atomic github hangover internal triplestore impl
import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

//spellchecker:words twiesing imap

// IMap holds forward and reverse mapping from Labels to IDs.
// An IMap may be read concurrently; however any operations which change internal state are not safe to access concurrently.
//
// The zero map is not ready for use; it should be initialized using a call to [Reset].
type IMap struct {
	forward HashMap[impl.Label, TripleID] // mapping from labels to the ids of their trippings
	reverse HashMap[impl.ID, impl.Label]  // mapping from literal back to their labels

	finalized atomic.Bool // stores if the map has been finalized
	id        impl.ID     // last id inserted
}

// TripleID represents the id of a tripleID.
//
//nolint:recvcheck
type TripleID struct {
	// Canonical holds the id of this triple, that is normalized for inverses and identities.
	Canonical impl.ID

	// Literal is the original id of the triple found in the original triple.
	// It always refers to the original value, no matter which value it actually has.
	Literal impl.ID
}

// Marshal marshals this TripleID into a []byte.
func (ti TripleID) Marshal() ([]byte, error) {
	return impl.EncodeIDs(ti.Canonical, ti.Literal), nil
}

// Unmarshal reads this TripleID from a []byte.
func (ti *TripleID) Unmarshal(src []byte) error {
	err := impl.UnmarshalIDs(src, &(ti.Canonical), &(ti.Literal))
	if err != nil {
		return fmt.Errorf("failed to unmarshal ids: %w", err)
	}
	return nil
}

var ErrFinalized = errors.New("IMap is finalized")

// Reset resets this IMap to be empty, closing any previously opened files.
func (mp *IMap) Reset(engine Map) error {
	if err := mp.Close(); err != nil {
		return err
	}

	var err error
	var closers []io.Closer

	mp.forward, err = engine.Forward()
	if err != nil {
		return fmt.Errorf("failed to initialize forward mapping: %w", err)
	}
	closers = append(closers, mp.forward)

	mp.reverse, err = engine.Reverse()
	if err != nil {
		errs := make([]error, 1, len(closers)+1)
		errs[0] = err
		for _, closer := range closers {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close: %w", err))
			}
		}
		return errors.Join(errs...)
	}

	mp.id.Reset()
	mp.finalized.Store(false)
	return nil
}

// Next returns a new unused id within this map
// It is always valid.
func (mp *IMap) Next() impl.ID {
	return mp.id.Inc()
}

// Compact indicates to the implementation to perform any optimization of internal data structures.
func (mp *IMap) Compact() error {
	var errs [2]error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errs[0] = mp.forward.Compact()
	}()

	go func() {
		defer wg.Done()
		errs[1] = mp.reverse.Compact()
	}()

	wg.Wait()
	return errors.Join(errs[:]...)
}

// Finalize indicates that no more mutating calls will be made.
// A mutable call is one made to Compact, Add, AddNew or MarkIdentical.
func (mp *IMap) Finalize() error {
	// store that we finalized!
	if mp.finalized.Swap(true) {
		return ErrFinalized
	}

	var errs [2]error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errs[0] = mp.forward.Finalize()
	}()

	go func() {
		defer wg.Done()
		errs[1] = mp.reverse.Finalize()
	}()

	wg.Wait()
	return errors.Join(errs[:]...)
}

// Add inserts label into this IMap and returns a pair of corresponding ids.
// The first is the canonical id (for use in lookups) whereas the second is the original id.
//
// When label (or any object marked identical to ID) already exists in this IMap, returns the corresponding ID.
func (mp *IMap) Add(label impl.Label) (ids TripleID, err error) {
	// we were already finalized!
	if mp.finalized.Load() {
		return ids, ErrFinalized
	}

	ids, _, err = mp.AddNew(label)
	if err != nil {
		return ids, fmt.Errorf("failed to add label: %w", err)
	}
	return ids, nil
}

// Get behaves like Add, but in case the label has no associated mappings returns ok = false and does not modify the state.
func (mp *IMap) Get(label impl.Label) (ids TripleID, ok bool, err error) {
	ids, ok, err = mp.forward.Get(label)
	if err != nil {
		return ids, ok, fmt.Errorf("failed to get label: %w", err)
	}
	return ids, ok, nil
}

// AddNew behaves like Add, except additionally returns a boolean indicating if the returned id existed previously.
func (mp *IMap) AddNew(label impl.Label) (ids TripleID, old bool, err error) {
	// we were already finalized!
	if mp.finalized.Load() {
		return ids, false, ErrFinalized
	}

	// fetch the mapping (if any)
	ids, old, err = mp.forward.Get(label)
	if err != nil {
		return
	}
	if old {
		return
	}

	// create a new map for the identical triple
	{
		id := mp.id.Inc()
		ids.Canonical = id
		ids.Literal = id
	}

	// store mappings in both directions
	if err := mp.forward.Set(label, ids); err != nil {
		return TripleID{}, false, fmt.Errorf("failed to store forward ids: %w", err)
	}
	if err := mp.reverse.Set(ids.Literal, label); err != nil {
		return TripleID{}, false, fmt.Errorf("failed to store reverse ids: %w", err)
	}

	// return the id
	return
}

// MarkIdentical marks the two labels as being identical.
// It returns the ID corresponding to the label same.
//
// Once applied, all future calls to [Forward] or [Add] with old will act as if being called by same.
// A previous ID corresponding to old (if any) is no longer valid.
//
// NOTE(twiesing): Each call to MarkIdentical potentially requires iterating over all calls that were previously added to this map.
// This is a potentially slow operation and should be avoided.
func (mp *IMap) MarkIdentical(same, old impl.Label) (canonical impl.ID, err error) {
	// we were already finalized!
	if mp.finalized.Load() {
		return canonical, ErrFinalized
	}

	canonicals, err := mp.Add(same)
	canonical = canonicals.Canonical // we use the "new" version of canonicals
	if err != nil {
		return canonical, fmt.Errorf("failed to add id for 'same' element: %w", err)
	}
	aliass, aliasIsOld, err := mp.AddNew(old)
	alias := aliass.Canonical // we use the canonical
	if err != nil {
		return canonical, fmt.Errorf("failed to add id for 'old' element: %w", err)
	}

	// the canonical variant of the alias
	// is already set to be canonical
	if canonical == alias {
		return canonical, nil
	}

	// if the alias was added fresh, then it can't be touched by anything else
	// so we can just give it a new canonical identifier!
	if !aliasIsOld {
		aliass.Canonical = canonical
		if err := mp.forward.Set(old, aliass); err != nil {
			return canonical, fmt.Errorf("failed to set alias to old id: %w", err)
		}
		return
	}

	// alias wasn't fresh, so we need to update everything that currently maps to it
	// that is stored in the "canonical" map of the first element.
	err = mp.forward.Iterate(func(label impl.Label, ids TripleID) error {
		if ids.Canonical != alias || label == same {
			return nil
		}

		// set the canonical id
		ids.Canonical = canonical
		if err := mp.forward.Set(label, ids); err != nil {
			return fmt.Errorf("failed to set forward id: %w", err)
		}

		// we do not delete anything
		return nil
	})
	if err != nil {
		return canonical, fmt.Errorf("failed to iterate through ids: %w", err)
	}
	return canonical, nil
}

// Forward returns the id corresponding to the given label.
//
// If the label is not contained in this map, the zero ID is returned.
// The zero ID is never returned for a valid id.
func (mp *IMap) Forward(label impl.Label) (impl.ID, error) {
	// TODO: This stores, but discards the original value.
	value, err := mp.forward.GetZero(label)
	if err != nil {
		return value.Canonical, fmt.Errorf("failed to get forward id: %w", err)
	}
	return value.Canonical, nil
}

// Reverse returns the label corresponding to the given id.
// When id is not contained in this map, the zero value of the label type is contained.
func (mp *IMap) Reverse(id impl.ID) (impl.Label, error) {
	label, err := mp.reverse.GetZero(id)
	if err != nil {
		return label, fmt.Errorf("failed to get reverse id: %w", err)
	}
	return label, nil
}

// IdentityMap writes canonical label mappings to the given storage.
//
// Concretely a pair (L1, L2) is written to storage iff
//
//	mp.Reverse(mp.Forward(L1)) == L2 && L1 != L2
func (mp *IMap) IdentityMap(storage HashMap[impl.Label, impl.Label]) error {
	// TODO: Do we really want this right now
	if err := mp.forward.Iterate(func(label impl.Label, id TripleID) error {
		value, err := mp.reverse.GetZero(id.Canonical)
		if err != nil {
			return fmt.Errorf("failed to get canonical id: %w", err)
		}
		if value != label {
			if err := storage.Set(label, value); err != nil {
				return fmt.Errorf("failed to store canonical id: %w", err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to iterate forward mappings: %w", err)
	}
	return nil
}

// Close closes any storages related to this IMap.
//
// Calling close multiple times results in err = nil.
func (mp *IMap) Close() error {
	var errs [2]error

	if mp.forward != nil {
		errs[0] = mp.forward.Close()
		mp.forward = nil
	}
	if mp.reverse != nil {
		errs[1] = mp.reverse.Close()
		mp.reverse = nil
	}

	return errors.Join(errs[0], errs[1])
}
