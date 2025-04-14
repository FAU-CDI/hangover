package igraph

import (
	"maps"
	"slices"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

// MemoryEngine represents an engine that stores everything in memory.
type MemoryEngine struct {
	imap.MemoryMap
}

func (MemoryEngine) Data() (imap.HashMap[impl.ID, impl.Datum], error) {
	ms := imap.MakeMemory[impl.ID, impl.Datum](0)
	return &ms, nil
}
func (MemoryEngine) Triples() (imap.HashMap[impl.ID, IndexTriple], error) {
	ms := imap.MakeMemory[impl.ID, IndexTriple](0)
	return &ms, nil
}
func (MemoryEngine) Inverses() (imap.HashMap[impl.ID, impl.ID], error) {
	ms := imap.MakeMemory[impl.ID, impl.ID](0)
	return &ms, nil
}
func (MemoryEngine) PSOIndex() (ThreeStorage, error) {
	th := make(ThreeHash)
	return &th, nil
}
func (MemoryEngine) POSIndex() (ThreeStorage, error) {
	th := make(ThreeHash)
	return &th, nil
}

// ThreeHash implements ThreeStorage in memory.
type ThreeHash map[impl.ID]map[impl.ID]*ThreeItem

func (th *ThreeHash) Compact() error {
	return nil // do nothing
}

type ThreeItem struct {
	Data map[impl.ID]impl.ID
	Keys []impl.ID
}

func (tlm ThreeHash) Add(a, b, c impl.ID, l impl.ID, conflict func(old, new impl.ID) (impl.ID, error)) (conflicted bool, err error) {
	switch {
	case tlm[a] == nil:
		tlm[a] = make(map[impl.ID]*ThreeItem)
		fallthrough
	case tlm[a][b] == nil:
		tlm[a][b] = &ThreeItem{
			Data: make(map[impl.ID]impl.ID, 1),
		}
		fallthrough
	default:
		var old impl.ID
		old, conflicted = tlm[a][b].Data[c]
		if conflicted {
			l, err = conflict(old, l)
			if err != nil {
				return false, err
			}
		}
		tlm[a][b].Data[c] = l
	}
	return conflicted, nil
}

func (tlm ThreeHash) Count() (total int64, err error) {
	for _, a := range tlm {
		for _, b := range a {
			total += int64(len(b.Keys))
		}
	}
	return total, nil
}

func (tlm ThreeHash) Finalize() error {
	for _, a := range tlm {
		for _, b := range a {
			b.Keys = slices.AppendSeq(make([]impl.ID, 0, len(b.Data)), maps.Keys(b.Data))
			slices.SortFunc(b.Keys, impl.ID.Compare)
		}
	}
	return nil
}

func (tlm ThreeHash) Fetch(a, b impl.ID, f func(c impl.ID, l impl.ID) error) error {
	three := tlm[a][b]
	if three == nil {
		return nil
	}
	for _, c := range three.Keys {
		if err := f(c, three.Data[c]); err != nil {
			return err
		}
	}

	return nil
}

func (tlm ThreeHash) Has(a, b, c impl.ID) (impl.ID, bool, error) {
	three := tlm[a][b]
	if three == nil {
		var invalid impl.ID
		return invalid, false, nil
	}
	l, ok := three.Data[c]
	return l, ok, nil
}

func (tlm *ThreeHash) Close() error {
	*tlm = nil
	return nil
}
