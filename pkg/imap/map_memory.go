package imap

// MemoryMap holds forward and backward maps in memory
type MemoryMap[Label comparable] struct {
	FStorage Memory[Label, TripleID]
	RStorage Memory[ID, Label]
}

func (me *MemoryMap[Label]) Forward() (HashMap[Label, TripleID], error) {
	if me.FStorage.IsNil() {
		me.FStorage = MakeMemory[Label, TripleID](0)
	}
	return &me.FStorage, nil
}

func (me *MemoryMap[Label]) Reverse() (HashMap[ID, Label], error) {
	if me.RStorage.IsNil() {
		me.RStorage = MakeMemory[ID, Label](0)
	}
	return &me.RStorage, nil
}
