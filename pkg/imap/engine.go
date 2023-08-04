package imap

// Engine creates Storages for an IMap
type Engine[Label comparable] interface {
	Forward() (KeyValueStore[Label, [2]ID], error)
	Reverse() (KeyValueStore[ID, Label], error)
}

// KeyValueStore is something that holds a set of key-value pairs.
//
// Key-Value stores support various read and write operations
type KeyValueStore[Key comparable, Value any] interface {
	// Close closes this key
	Close() error

	// Set sets the given key to the given value
	Set(key Key, value Value) error

	// Get retrieves the value for Key from the given storage.
	// The second value indiciates if the value was found.
	Get(key Key) (Value, bool, error)

	// GetZero is like Get, but when the value does not exist returns the zero value
	GetZero(key Key) (Value, error)

	// Has is like Get, but returns only the second value.
	Has(key Key) (bool, error)

	// Delete deletes the given key from this storage
	Delete(key Key) error

	// Iterate calls f for all entries in Storage.
	//
	// When any f returns a non-nil error, that error is returned immediatly to the caller
	// and iteration stops.
	//
	// There is no guarantee on order.
	Iterate(f func(Key, Value) error) error

	// Count counts the number of elements in this store
	Count() (uint64, error)
}
