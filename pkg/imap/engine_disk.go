package imap

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
)

// DiskEngine represents an engine that persistently stores data on disk.
type DiskEngine[Label comparable] struct {
	Path string

	MarshalLabel   func(label Label) ([]byte, error)
	UnmarshalLabel func(dest *Label, src []byte) error
}

func (de DiskEngine[Label]) Forward() (KeyValueStore[Label, [2]ID], error) {
	forward := filepath.Join(de.Path, "forward.leveldb")

	ds, err := NewDiskStorage[Label, [2]ID](forward)
	if err != nil {
		return nil, err
	}

	if de.MarshalLabel != nil && de.UnmarshalLabel != nil {
		ds.MarshalKey = de.MarshalLabel
		ds.UnmarshalKey = de.UnmarshalLabel
	}

	ds.MarshalValue = MarshalIDPair
	ds.UnmarshalValue = UnmarshalIDPair

	return ds, nil
}

func (de DiskEngine[Label]) Reverse() (KeyValueStore[ID, Label], error) {
	reverse := filepath.Join(de.Path, "reverse.leveldb")

	ds, err := NewDiskStorage[ID, Label](reverse)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = MarshalID
	ds.UnmarshalKey = UnmarshalID

	if de.MarshalLabel != nil && de.UnmarshalLabel != nil {
		ds.MarshalValue = de.MarshalLabel
		ds.UnmarshalValue = de.UnmarshalLabel
	}

	return ds, nil
}

// NewDiskStorage creates a new disk-based storage with the given options.
// If the filepath already exists, it is deleted.
func NewDiskStorage[Key comparable, Value any](path string) (*DiskStorage[Key, Value], error) {

	// If the path already exists, wipe it
	_, err := os.Stat(path)
	if err == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, err
		}
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	storage := &DiskStorage[Key, Value]{
		DB: db,

		MarshalKey: func(key Key) ([]byte, error) {
			return json.Marshal(key)
		},
		UnmarshalKey: func(dest *Key, src []byte) error {
			return json.Unmarshal(src, dest)
		},
		MarshalValue: func(value Value) ([]byte, error) {
			return json.Marshal(value)
		},
		UnmarshalValue: func(dest *Value, src []byte) error {
			return json.Unmarshal(src, dest)
		},
	}
	return storage, nil
}

// DiskStorage implements Storage as an in-memory storage
type DiskStorage[Key comparable, Value any] struct {
	DB *leveldb.DB

	MarshalKey     func(key Key) ([]byte, error)
	UnmarshalKey   func(dest *Key, src []byte) error
	MarshalValue   func(value Value) ([]byte, error)
	UnmarshalValue func(dest *Value, src []byte) error
}

func (ds *DiskStorage[Key, Value]) Set(key Key, value Value) error {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return err
	}
	valueB, err := ds.MarshalValue(value)
	if err != nil {
		return err
	}

	return ds.DB.Put(keyB, valueB, nil)
}

// Get returns the given value if it exists
func (ds *DiskStorage[Key, Value]) Get(key Key) (v Value, b bool, err error) {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return v, b, err
	}

	valueB, err := ds.DB.Get(keyB, nil)
	if err == leveldb.ErrNotFound {
		return v, false, nil
	}
	if err != nil {
		return v, b, err
	}

	if err := ds.UnmarshalValue(&v, valueB); err != nil {
		return v, b, err
	}

	return v, true, nil
}

// GetZero returns the value associated with Key, or the zero value otherwise.
func (ds *DiskStorage[Key, Value]) GetZero(key Key) (Value, error) {
	value, _, err := ds.Get(key)
	return value, err
}

func (ds *DiskStorage[Key, Value]) Has(key Key) (bool, error) {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return false, err
	}

	ok, err := ds.DB.Has(keyB, nil)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Delete deletes the given key from this storage
func (ds *DiskStorage[Key, Value]) Delete(key Key) error {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return err
	}

	if err := ds.DB.Delete(keyB, nil); err != nil {
		return err
	}

	return nil
}

// Iterate calls f for all entries in Storage.
// there is no guarantee on order.
func (ds *DiskStorage[Key, Value]) Iterate(f func(Key, Value) error) error {
	it := ds.DB.NewIterator(nil, nil)
	defer it.Release()

	for it.Next() {
		var key Key
		if err := ds.UnmarshalKey(&key, it.Key()); err != nil {
			return err
		}
		var value Value
		if err := ds.UnmarshalValue(&value, it.Value()); err != nil {
			return err
		}
		if err := f(key, value); err != nil {
			return err
		}
	}
	return it.Error()
}

func (ds *DiskStorage[Key, Value]) Close() error {
	var err error

	if ds.DB != nil {
		err = ds.DB.Close()
	}
	ds.DB = nil
	return err
}

// Count returns the number of objects in this DiskStorage.
func (ds *DiskStorage[Key, Value]) Count() (count uint64, err error) {
	it := ds.DB.NewIterator(nil, nil)
	for it.Next() {
		count++
	}
	err = it.Error()
	if err != nil {
		count = 0
	}
	return
}
