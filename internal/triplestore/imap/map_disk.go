//spellchecker:words imap
package imap

//spellchecker:words encoding json errors path filepath github hangover internal triplestore impl syndtr goleveldb leveldb util
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DiskMap represents an engine that persistently stores data on disk.
type DiskMap struct {
	Path string
}

var (
	_ Map = (*DiskMap)(nil)
)

func (de DiskMap) Forward() (HashMap[impl.Label, TripleID], error) {
	forward := filepath.Join(de.Path, "forward.leveldb")

	ds, err := NewDiskStorage[impl.Label, TripleID](forward)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = func(key impl.Label) ([]byte, error) {
		return impl.LabelAsByte(key), nil
	}
	ds.UnmarshalKey = func(dest *impl.Label, src []byte) error {
		*dest = impl.ByteAsLabel(src)
		return nil
	}

	ds.MarshalValue = (TripleID).Marshal
	ds.UnmarshalValue = (*TripleID).Unmarshal

	return ds, nil
}

func (de DiskMap) Reverse() (HashMap[impl.ID, impl.Label], error) {
	reverse := filepath.Join(de.Path, "reverse.leveldb")

	ds, err := NewDiskStorage[impl.ID, impl.Label](reverse)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = impl.MarshalID
	ds.UnmarshalKey = impl.UnmarshalID

	ds.MarshalValue = func(key impl.Label) ([]byte, error) {
		return impl.LabelAsByte(key), nil
	}
	ds.UnmarshalValue = func(dest *impl.Label, src []byte) error {
		*dest = impl.ByteAsLabel(src)
		return nil
	}

	return ds, nil
}

// NewDiskStorage creates a new disk-based storage with the given options.
// If the filepath already exists, it is deleted.
func NewDiskStorage[Key comparable, Value any](path string) (*DiskStorage[Key, Value], error) {
	// If the path already exists, wipe it
	_, err := os.Stat(path)
	if err == nil {
		if e2 := os.RemoveAll(path); e2 != nil {
			err = errors.Join(err, fmt.Errorf("failed to cleanup path: %w", e2))
		}
		return nil, err
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file: %w", err)
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

// DiskStorage implements Storage as an in-memory storage.
type DiskStorage[Key comparable, Value any] struct {
	DB *leveldb.DB

	MarshalKey     func(key Key) ([]byte, error)
	UnmarshalKey   func(dest *Key, src []byte) error
	MarshalValue   func(value Value) ([]byte, error)
	UnmarshalValue func(dest *Value, src []byte) error
}

func (ds *DiskStorage[Key, Value]) Grow(size uint64) error {
	// not supported
	return nil
}

func (ds *DiskStorage[Key, Value]) Set(key Key, value Value) error {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}
	valueB, err := ds.MarshalValue(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := ds.DB.Put(keyB, valueB, nil); err != nil {
		return fmt.Errorf("failed to set value for key: %w", err)
	}
	return nil
}

// Get returns the given value if it exists.
func (ds *DiskStorage[Key, Value]) Get(key Key) (v Value, b bool, err error) {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return v, b, err
	}

	valueB, err := ds.DB.Get(keyB, nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return v, false, nil
	}
	if err != nil {
		return v, b, fmt.Errorf("failed to get key from database: %w", err)
	}

	if err := ds.UnmarshalValue(&v, valueB); err != nil {
		return v, b, fmt.Errorf("failed to unmarshal value: %w", err)
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
		return false, fmt.Errorf("failed to check database for key: %w", err)
	}
	return ok, nil
}

// Delete deletes the given key from this storage.
func (ds *DiskStorage[Key, Value]) Delete(key Key) error {
	keyB, err := ds.MarshalKey(key)
	if err != nil {
		return err
	}

	if err := ds.DB.Delete(keyB, nil); err != nil {
		return fmt.Errorf("failed to delete key from disk: %w", err)
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
			return fmt.Errorf("function returned error: %w", err)
		}
	}
	if err := it.Error(); err != nil {
		return fmt.Errorf("failed to iterate database: %w", err)
	}
	return nil
}

func (ds *DiskStorage[Key, Value]) Compact() error {
	if err := ds.DB.CompactRange(util.Range{}); err != nil {
		return fmt.Errorf("failed to compact database: %w", err)
	}
	return nil
}

func (ds *DiskStorage[Key, Value]) Finalize() error {
	return errors.Join(ds.Compact(), ds.DB.SetReadOnly())
}

func (ds *DiskStorage[Key, Value]) Close() error {
	var err error

	if ds.DB != nil {
		err = ds.DB.Close()
	}
	ds.DB = nil
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
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
