package igraph

import (
	"github.com/FAU-CDI/hangover/pkg/imap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// MemoryEngine represents an engine that stores everything in memory
type MemoryEngine[Label comparable, Datum any] struct {
	imap.MemoryMap[Label]
}

func (MemoryEngine[Label, Datum]) Data() (imap.HashMap[imap.ID, Datum], error) {
	ms := imap.MakeMemory[imap.ID, Datum](0)
	return &ms, nil
}
func (MemoryEngine[Label, Datum]) Triples() (imap.HashMap[imap.ID, IndexTriple], error) {
	ms := imap.MakeMemory[imap.ID, IndexTriple](0)
	return &ms, nil
}
func (MemoryEngine[Label, Datum]) Inverses() (imap.HashMap[imap.ID, imap.ID], error) {
	ms := imap.MakeMemory[imap.ID, imap.ID](0)
	return &ms, nil
}
func (MemoryEngine[Label, Datum]) PSOIndex() (ThreeStorage, error) {
	th := make(ThreeHash)
	return &th, nil

}
func (MemoryEngine[Label, Datum]) POSIndex() (ThreeStorage, error) {
	th := make(ThreeHash)
	return &th, nil
}

// ThreeHash implements ThreeStorage in memory
type ThreeHash map[imap.ID]map[imap.ID]*ThreeItem

func (th *ThreeHash) Compact() error {
	return nil // do nothing
}

type ThreeItem struct {
	Keys []imap.ID
	Data map[imap.ID]imap.ID
}

func (tlm ThreeHash) Add(a, b, c imap.ID, l imap.ID, conflict func(old, new imap.ID) (imap.ID, error)) (conflicted bool, err error) {
	switch {
	case tlm[a] == nil:
		tlm[a] = make(map[imap.ID]*ThreeItem)
		fallthrough
	case tlm[a][b] == nil:
		tlm[a][b] = &ThreeItem{
			Data: make(map[imap.ID]imap.ID, 1),
		}
		fallthrough
	default:
		var old imap.ID
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
			b.Keys = maps.Keys(b.Data)
			slices.SortFunc(b.Keys, func(a imap.ID, b imap.ID) int {
				if a.Less(b) {
					return -1
				}
				if a == b {
					return 0
				}
				return 1
			})
		}
	}
	return nil
}

func (tlm ThreeHash) Fetch(a, b imap.ID, f func(c imap.ID, l imap.ID) error) error {
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

func (tlm ThreeHash) Has(a, b, c imap.ID) (imap.ID, bool, error) {
	three := tlm[a][b]
	if three == nil {
		var invalid imap.ID
		return invalid, false, nil
	}
	l, ok := three.Data[c]
	return l, ok, nil
}

func (tlm *ThreeHash) Close() error {
	*tlm = nil
	return nil
}
