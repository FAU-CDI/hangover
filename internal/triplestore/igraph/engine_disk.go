package igraph

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/syndtr/goleveldb/leveldb"
	leveldberrors "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DiskEngine represents an engine that stores everything on disk
type DiskEngine struct {
	imap.DiskMap
}

func (de DiskEngine) Language() (imap.HashMap[impl.ID, impl.Language], error) {
	data := filepath.Join(de.Path, "language.leveldb")

	ds, err := imap.NewDiskStorage[impl.ID, impl.Language](data)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = impl.MarshalID
	ds.UnmarshalKey = impl.UnmarshalID

	ds.MarshalValue = func(value impl.Language) ([]byte, error) {
		return impl.LanguageAsByte(value), nil
	}
	ds.UnmarshalValue = func(dest *impl.Language, src []byte) error {
		*dest = impl.ByteAsLanguage(src)
		return nil
	}
	return ds, nil
}

func (de DiskEngine) Data() (imap.HashMap[impl.ID, impl.Datum], error) {
	data := filepath.Join(de.Path, "data.leveldb")

	ds, err := imap.NewDiskStorage[impl.ID, impl.Datum](data)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = impl.MarshalID
	ds.UnmarshalKey = impl.UnmarshalID

	ds.MarshalValue = func(value impl.Datum) ([]byte, error) {
		return impl.DatumAsByte(value), nil
	}
	ds.UnmarshalValue = func(dest *impl.Datum, src []byte) error {
		*dest = impl.ByteAsDatum(src)
		return nil
	}
	return ds, nil
}

func (de DiskEngine) Triples() (imap.HashMap[impl.ID, IndexTriple], error) {
	data := filepath.Join(de.Path, "triples.leveldb")

	ds, err := imap.NewDiskStorage[impl.ID, IndexTriple](data)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = impl.MarshalID
	ds.UnmarshalKey = impl.UnmarshalID

	ds.MarshalValue = MarshalTriple
	ds.UnmarshalValue = UnmarshalTriple

	return ds, nil
}

func (de DiskEngine) Inverses() (imap.HashMap[impl.ID, impl.ID], error) {
	inverses := filepath.Join(de.Path, "inverses.leveldb")

	ds, err := imap.NewDiskStorage[impl.ID, impl.ID](inverses)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = impl.MarshalID
	ds.UnmarshalKey = impl.UnmarshalID

	ds.MarshalValue = impl.MarshalID
	ds.UnmarshalValue = impl.UnmarshalID

	return ds, nil
}
func (de DiskEngine) PSOIndex() (ThreeStorage, error) {
	pso := filepath.Join(de.Path, "pso.leveldb")
	return NewDiskHash(pso)
}
func (de DiskEngine) POSIndex() (ThreeStorage, error) {
	pos := filepath.Join(de.Path, "pos.leveldb")
	return NewDiskHash(pos)
}

func NewDiskHash(path string) (ThreeStorage, error) {
	// If the path already exists, wipe it
	_, err := os.Stat(path)
	if err == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, err
		}
	}

	level, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	dh := &ThreeDiskHash{
		DB: level,
	}
	return dh, nil
}

// ThreeHash implements ThreeStorage in memory
type ThreeDiskHash struct {
	DB *leveldb.DB
}

func (tlm *ThreeDiskHash) Add(a, b, c impl.ID, l impl.ID, conflict func(old, new impl.ID) (impl.ID, error)) (conflicted bool, err error) {
	key := impl.EncodeIDs(a, b, c)
	value, err := tlm.DB.Get(key, nil)
	switch err {
	case nil:
		l, err = conflict(impl.DecodeID(value, 0), l)
		if err != nil {
			return false, err
		}
		conflicted = true
	case leveldberrors.ErrNotFound:
	}
	return conflicted, tlm.DB.Put(impl.EncodeIDs(a, b, c), impl.EncodeIDs(l), nil)
}

func (tlm *ThreeDiskHash) Count() (total int64, err error) {
	iterator := tlm.DB.NewIterator(nil, nil)
	defer iterator.Release()

	for iterator.Next() {
		total++
	}

	if err := iterator.Error(); err != nil {
		return 0, err
	}

	return total, nil
}

func (tlm ThreeDiskHash) Compact() error {
	return tlm.DB.CompactRange(util.Range{})
}

func (tlm ThreeDiskHash) Finalize() error {
	return errors.Join(tlm.Compact(), tlm.DB.SetReadOnly())
}

func (tlm *ThreeDiskHash) Fetch(a, b impl.ID, f func(c impl.ID, l impl.ID) error) error {
	iterator := tlm.DB.NewIterator(util.BytesPrefix(impl.EncodeIDs(a, b)), nil)
	defer iterator.Release()

	for iterator.Next() {
		c := impl.DecodeID(iterator.Key(), 2)
		l := impl.DecodeID(iterator.Value(), 0)
		if err := f(c, l); err != nil {
			return err
		}
	}

	if err := iterator.Error(); err != nil {
		return err
	}

	return nil
}

func (tlm *ThreeDiskHash) Has(a, b, c impl.ID) (id impl.ID, ok bool, err error) {
	value, err := tlm.DB.Get(impl.EncodeIDs(a, b, c), nil)
	if err == leveldberrors.ErrNotFound {
		var invalid impl.ID
		return invalid, false, nil
	}

	err = impl.UnmarshalID(&id, value)
	if err != nil {
		return id, false, err
	}
	return id, true, nil
}

func (tlm *ThreeDiskHash) Close() (err error) {
	if tlm.DB != nil {
		err = tlm.DB.Close()
		tlm.DB = nil
	}
	return
}
