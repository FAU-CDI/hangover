package imap

import "github.com/FAU-CDI/hangover/internal/triplestore/impl"

// cspell:words imap

// Map represents the backend of an Imap and creates appropriate key-value stores.
type Map interface {
	Forward() (HashMap[impl.Label, TripleID], error)
	Reverse() (HashMap[impl.ID, impl.Label], error)
}

// HashMap is something that stores key-value pairs.
type HashMap[Key comparable, Value any] interface {
	// Grow resizes this hash map to the given size.
	// if the HashMap already has data in it, may be a no-op.
	Grow(size uint64) error

	// Close closes this store
	Close() error

	// Compact informs the store to perform any optimizations or compaction of internal data structures.
	Compact() error

	// Finalize indicates to the implementation that no more mutating calls will be made.
	// A mutating call is one to Compact, Set or Delete.
	Finalize() error

	// Set sets the given key to the given value
	Set(key Key, value Value) error

	// Get retrieves the value for Key from the given storage.
	// The second value indicates if the value was found.
	Get(key Key) (Value, bool, error)

	// GetZero is like Get, but when the value does not exist returns the zero value
	GetZero(key Key) (Value, error)

	// Has is like Get, but returns only the second value.
	Has(key Key) (bool, error)

	// Delete deletes the given key from this storage
	Delete(key Key) error

	// Iterate calls f for all entries in Storage.
	//
	// When any f returns a non-nil error, that error is returned immediately to the caller
	// and iteration stops.
	//
	// There is no guarantee on order.
	Iterate(f func(Key, Value) error) error

	// Count counts the number of elements in this store
	Count() (uint64, error)
}
