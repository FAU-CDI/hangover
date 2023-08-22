package imap

import "errors"

// MemoryMap holds forward and backward maps in memory
type MemoryMap[Label comparable] struct {
	FStorage MemoryStorage[Label, TripleID]
	RStorage MemoryStorage[ID, Label]
}

func (me *MemoryMap[Label]) Forward() (KeyValueStore[Label, TripleID], error) {
	if me.FStorage == nil {
		me.FStorage = make(MemoryStorage[Label, TripleID])
	}
	return &me.FStorage, nil
}

func (me *MemoryMap[Label]) Reverse() (KeyValueStore[ID, Label], error) {
	if me.RStorage == nil {
		me.RStorage = make(MemoryStorage[ID, Label])
	}
	return &me.RStorage, nil
}

// MemoryStorage implements Storage as an in-memory map
type MemoryStorage[Key comparable, Value any] map[Key]Value

func (MemoryStorage[Key, Value]) Compact() error {
	return nil
}

func (ims MemoryStorage[Key, Value]) Finalize() error {
	return errors.Join(ims.Compact(), nil)
}

func (ims MemoryStorage[Key, Value]) Set(key Key, value Value) error {
	ims[key] = value
	return nil
}

// Get returns the given value if it exists
func (ims MemoryStorage[Key, Value]) Get(key Key) (Value, bool, error) {
	value, ok := ims[key]
	return value, ok, nil
}

// GetZero returns the value associated with Key, or the zero value otherwise.
func (ims MemoryStorage[Key, Value]) GetZero(key Key) (Value, error) {
	return ims[key], nil
}

func (ims MemoryStorage[Key, Value]) Has(key Key) (bool, error) {
	_, ok := ims[key]
	return ok, nil
}

// Delete deletes the given key from this storage
func (ims MemoryStorage[Key, Value]) Delete(key Key) error {
	delete(ims, key)
	return nil
}

// Iterate calls f for all entries in Storage.
// there is no guarantee on order.
func (ims MemoryStorage[Key, Value]) Iterate(f func(Key, Value) error) error {
	for key, value := range ims {
		if err := f(key, value); err != nil {
			return err
		}
	}
	return nil
}

// Close closes this MapStorage, deleting all values
func (ims *MemoryStorage[Key, Value]) Close() error {
	*ims = nil
	return nil
}

func (ims *MemoryStorage[Key, Value]) Count() (uint64, error) {
	return uint64(len(*ims)), nil
}
