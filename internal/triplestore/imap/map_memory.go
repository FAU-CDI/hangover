//spellchecker:words imap
package imap

//spellchecker:words errors github hangover internal triplestore impl
import (
	"errors"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

// MemoryMap holds forward and backward maps in memory.
// It implements Map.
type MemoryMap struct {
	FStorage Memory[impl.Label, TripleID]
	RStorage Memory[impl.ID, impl.Label]
}

func (me *MemoryMap) Close() error {
	return errors.Join(
		me.FStorage.Close(),
		me.RStorage.Close(),
	)
}

var (
	_ Map = (*MemoryMap)(nil)
)

func (me *MemoryMap) Forward() (HashMap[impl.Label, TripleID], error) {
	if me.FStorage.IsNil() {
		me.FStorage = MakeMemory[impl.Label, TripleID](0)
	}
	return &me.FStorage, nil
}

func (me *MemoryMap) Reverse() (HashMap[impl.ID, impl.Label], error) {
	if me.RStorage.IsNil() {
		me.RStorage = MakeMemory[impl.ID, impl.Label](0)
	}
	return &me.RStorage, nil
}
