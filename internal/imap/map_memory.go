package imap

import "errors"

// MemoryMap holds forward and backward maps in memory.
// It implements Map.
type MemoryMap struct {
	FStorage Memory[Label, TripleID]
	RStorage Memory[ID, Label]
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

func (me *MemoryMap) Forward() (HashMap[Label, TripleID], error) {
	if me.FStorage.IsNil() {
		me.FStorage = MakeMemory[Label, TripleID](0)
	}
	return &me.FStorage, nil
}

func (me *MemoryMap) Reverse() (HashMap[ID, Label], error) {
	if me.RStorage.IsNil() {
		me.RStorage = MakeMemory[ID, Label](0)
	}
	return &me.RStorage, nil
}
