package igraph

import (
	"os"
	"path/filepath"

	"github.com/FAU-CDI/hangover/pkg/imap"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DiskEngine represents an engine that stores everything on disk
type DiskEngine[Label comparable, Datum any] struct {
	imap.DiskMap[Label]

	MarshalDatum   func(datum Datum) ([]byte, error)
	UnmarshalDatum func(dest *Datum, src []byte) error
}

func (de DiskEngine[Label, Datum]) Data() (imap.KeyValueStore[imap.ID, Datum], error) {
	data := filepath.Join(de.Path, "data.leveldb")

	ds, err := imap.NewDiskStorage[imap.ID, Datum](data)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = imap.MarshalID
	ds.UnmarshalKey = imap.UnmarshalID

	if de.MarshalDatum != nil && de.UnmarshalDatum != nil {
		ds.MarshalValue = de.MarshalDatum
		ds.UnmarshalValue = de.UnmarshalDatum
	}

	return ds, nil
}

func (de DiskEngine[Label, Datum]) Triples() (imap.KeyValueStore[imap.ID, IndexTriple], error) {
	data := filepath.Join(de.Path, "triples.leveldb")

	ds, err := imap.NewDiskStorage[imap.ID, IndexTriple](data)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = imap.MarshalID
	ds.UnmarshalKey = imap.UnmarshalID

	ds.MarshalValue = MarshalTriple
	ds.UnmarshalValue = UnmarshalTriple

	return ds, nil
}

func (de DiskEngine[Label, Datum]) Inverses() (imap.KeyValueStore[imap.ID, imap.ID], error) {
	inverses := filepath.Join(de.Path, "inverses.leveldb")

	ds, err := imap.NewDiskStorage[imap.ID, imap.ID](inverses)
	if err != nil {
		return nil, err
	}

	ds.MarshalKey = imap.MarshalID
	ds.UnmarshalKey = imap.UnmarshalID

	ds.MarshalValue = imap.MarshalID
	ds.UnmarshalValue = imap.UnmarshalID

	return ds, nil
}
func (de DiskEngine[Label, Datum]) PSOIndex() (ThreeStorage, error) {
	pso := filepath.Join(de.Path, "pso.leveldb")
	return NewDiskHash(pso)
}
func (de DiskEngine[Label, Datum]) POSIndex() (ThreeStorage, error) {
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

func (tlm *ThreeDiskHash) Add(a, b, c imap.ID, l imap.ID, conflict func(old, new imap.ID) (imap.ID, error)) (conflicted bool, err error) {
	key := imap.EncodeIDs(a, b, c)
	value, err := tlm.DB.Get(key, nil)
	switch err {
	case nil:
		l, err = conflict(imap.DecodeID(value, 0), l)
		if err != nil {
			return false, err
		}
		conflicted = true
	case errors.ErrNotFound:
	}
	return conflicted, tlm.DB.Put(imap.EncodeIDs(a, b, c), imap.EncodeIDs(l), nil)
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

func (tlm ThreeDiskHash) Finalize() error {
	if err := tlm.DB.CompactRange(util.Range{}); err != nil {
		return err
	}
	return tlm.DB.SetReadOnly()
}

func (tlm *ThreeDiskHash) Fetch(a, b imap.ID, f func(c imap.ID, l imap.ID) error) error {
	iterator := tlm.DB.NewIterator(util.BytesPrefix(imap.EncodeIDs(a, b)), nil)
	defer iterator.Release()

	for iterator.Next() {
		c := imap.DecodeID(iterator.Key(), 2)
		l := imap.DecodeID(iterator.Value(), 0)
		if err := f(c, l); err != nil {
			return err
		}
	}

	if err := iterator.Error(); err != nil {
		return err
	}

	return nil
}

func (tlm *ThreeDiskHash) Has(a, b, c imap.ID) (id imap.ID, ok bool, err error) {
	value, err := tlm.DB.Get(imap.EncodeIDs(a, b, c), nil)
	if err == errors.ErrNotFound {
		var invalid imap.ID
		return invalid, false, nil
	}

	err = imap.UnmarshalID(&id, value)
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
