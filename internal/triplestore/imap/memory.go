package imap

import (
	"errors"
	"runtime"
)

// Memory contains the main in-memory value
type Memory[Key comparable, Value any] struct {
	mp map[Key]Value
}

func (m Memory[Key, Value]) IsNil() bool {
	return m.mp == nil
}

// MakeMemory makes a new memory instance
func MakeMemory[Key comparable, Value any](size int) Memory[Key, Value] {
	return Memory[Key, Value]{
		mp: make(map[Key]Value, size),
	}
}

var errMemoryUnintialized = errors.New("map not initalized")

// Compact causes any changes to be flushed to disk and performs common cleanup tasks
func (Memory[Key, Value]) Compact() error {
	runtime.GC()
	return nil
}

// Finalize makes this map read-only.
// It is a no-op.
func (ims Memory[Key, Value]) Finalize() error {
	return errors.Join(ims.Compact(), nil)
}

func (ims Memory[Key, Value]) Set(key Key, value Value) error {
	if ims.mp == nil {
		return errMemoryUnintialized
	}

	ims.mp[key] = value
	return nil
}

// Get returns the given value if it exists
func (ims Memory[Key, Value]) Get(key Key) (Value, bool, error) {
	value, ok := ims.mp[key]
	return value, ok, nil
}

// GetZero returns the value associated with Key, or the zero value otherwise.
func (ims Memory[Key, Value]) GetZero(key Key) (Value, error) {
	return ims.mp[key], nil
}

func (ims Memory[Key, Value]) Has(key Key) (bool, error) {
	_, ok := ims.mp[key]
	return ok, nil
}

// Delete deletes the given key from this storage
func (ims Memory[Key, Value]) Delete(key Key) error {
	delete(ims.mp, key)
	return nil
}

// Iterate calls f for all entries in Storage.
// there is no guarantee on order.
func (ims Memory[Key, Value]) Iterate(f func(Key, Value) error) error {
	for key, value := range ims.mp {
		if err := f(key, value); err != nil {
			return err
		}
	}
	return nil
}

// Close closes this MapStorage, deleting all values
func (ims *Memory[Key, Value]) Close() error {
	ims.mp = nil
	runtime.GC() // re-claim all the memory if needed
	return nil
}

func (ims Memory[Key, Value]) Count() (uint64, error) {
	return uint64(len(ims.mp)), nil
}
