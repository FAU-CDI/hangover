//spellchecker:words storages
package storages

//spellchecker:words errors iter path filepath sync atomic github drincw pathbuilder hangover internal triplestore igraph impl wisski syndtr goleveldb leveldb lerrors
import (
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/syndtr/goleveldb/leveldb"
	lerrors "github.com/syndtr/goleveldb/leveldb/errors"
)

type DiskEngine struct {
	Path string
}

func (de DiskEngine) NewStorage(bundle *pathbuilder.Bundle) (BundleStorage, error) {
	path := filepath.Join(de.Path, bundle.Path.Bundle)

	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("failed to remove previous contents: %w", err)
		}
	}

	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Disk{
		DB: db,

		childStorages: make(map[string]BundleStorage, len(bundle.ChildBundles)),
	}, nil
}

// Disk represents a disk-backed storage.
type Disk struct {
	DB            *leveldb.DB
	childStorages map[string]BundleStorage
	count         int64
	l             sync.RWMutex // protects modifying data on disk
}

func (ds *Disk) put(f func(*sEntity) error) error {
	entity := sEntityPool.Get().(*sEntity)
	entity.Reset()
	defer sEntityPool.Put(entity)

	if err := f(entity); err != nil {
		return fmt.Errorf("f returned error: %w", err)
	}

	data, err := entity.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode entity: %w", err)
	}

	ds.l.Lock()
	defer ds.l.Unlock()

	if err := ds.DB.Put([]byte(entity.URI), data, nil); err != nil {
		return fmt.Errorf("failed to store entity: %w", err)
	}
	return nil
}

func (ds *Disk) get(uri impl.Label, f func(*sEntity) error) error {
	entity := sEntityPool.Get().(*sEntity)
	entity.Reset()
	defer sEntityPool.Put(entity)

	ds.l.RLock()
	defer ds.l.RUnlock()

	// get the entity or return an error
	data, err := ds.DB.Get([]byte(uri), nil)
	if errors.Is(err, lerrors.ErrNotFound) {
		return ErrNoEntity
	}
	if err != nil {
		return fmt.Errorf("failed to get entity from database: %w", err)
	}

	// decode the entity!
	if err := entity.Decode(data); err != nil {
		return fmt.Errorf("failed to decode entity: %w", err)
	}

	// handle the entity!
	if err := f(entity); err != nil {
		return fmt.Errorf("failed to handle entity: %w", err)
	}
	return nil
}

func (ds *Disk) decode(data []byte, f func(*sEntity) error) error {
	entity := sEntityPool.Get().(*sEntity)
	entity.Reset()
	defer sEntityPool.Put(entity)

	if err := entity.Decode(data); err != nil {
		return fmt.Errorf("failed to decode entity: %w", err)
	}

	return f(entity)
}

func (ds *Disk) update(uri impl.Label, update func(*sEntity) error) error {
	entity := sEntityPool.Get().(*sEntity)
	entity.Reset()
	defer sEntityPool.Put(entity)

	ds.l.Lock()
	defer ds.l.Unlock()

	// get the entity or return an error
	data, err := ds.DB.Get([]byte(uri), nil)
	if errors.Is(err, lerrors.ErrNotFound) {
		return ErrNoEntity
	}
	if err != nil {
		return fmt.Errorf("failed to get entity: %w", err)
	}

	// decode the entity!
	if err := entity.Decode(data); err != nil {
		return fmt.Errorf("failed to decode entity: %w", err)
	}

	// perform the entity
	if err := update(entity); err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	// encoded the entity again
	data, err = entity.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode entity: %w", err)
	}

	// and put it back!
	if err := ds.DB.Put([]byte(entity.URI), data, nil); err != nil {
		return fmt.Errorf("failed to put into database: %w", err)
	}
	return nil
}

// Add adds an entity to this BundleSlice.
func (ds *Disk) Add(uri impl.Label, path []impl.Label, triples []igraph.Triple) error {
	atomic.AddInt64(&ds.count, 1)
	return ds.put(func(se *sEntity) error {
		se.URI = uri
		se.Path = path
		se.Triples = triples
		se.Fields = make(map[string][]wisski.FieldValue)
		se.Children = make(map[string][]impl.Label)
		return nil
	})
}

func (ds *Disk) AddFieldValue(uri impl.Label, field string, value impl.Datum, path []impl.Label, triples []igraph.Triple) error {
	return ds.update(uri, func(se *sEntity) error {
		if se.Fields == nil {
			se.Fields = make(map[string][]wisski.FieldValue)
		}
		se.Fields[field] = append(se.Fields[field], wisski.FieldValue{
			Datum:   value,
			Path:    path,
			Triples: triples,
		})
		return nil
	})
}

func (ds *Disk) RegisterChildStorage(bundle string, storage BundleStorage) error {
	ds.childStorages[bundle] = storage
	return nil
}

func (ds *Disk) AddChild(parent impl.Label, bundle string, child impl.Label) error {
	return ds.update(parent, func(se *sEntity) error {
		if se.Children == nil {
			se.Children = make(map[string][]impl.Label)
		}
		se.Children[bundle] = append(se.Children[bundle], child)
		return nil
	})
}

func (ds *Disk) Finalize() error {
	if err := ds.DB.SetReadOnly(); err != nil {
		return fmt.Errorf("failed to set database to read only: %w", err)
	}
	return nil
}

func (ds *Disk) Get(parentPathIndex int) iter.Seq2[LabelWithParent, error] {
	return func(yield func(LabelWithParent, error) bool) {
		it := ds.DB.NewIterator(nil, nil)
		defer it.Release()

		for it.Next() {
			var uri LabelWithParent
			var err error

			if parentPathIndex > 0 {
				err = ds.decode(it.Value(), func(se *sEntity) error {
					uri.Label = se.URI
					if parentPathIndex != -1 {
						uri.Parent = se.Path[parentPathIndex]
					}
					return nil
				})
			} else {
				uri.Label = impl.Label(it.Key())
			}

			if err != nil {
				yield(LabelWithParent{}, err)
				return
			}

			if !yield(uri, nil) {
				return
			}
		}

		if err := it.Error(); err != nil {
			yield(LabelWithParent{}, it.Error())
		}
	}
}

func (ds *Disk) Count() (int64, error) {
	return atomic.LoadInt64(&ds.count), nil
}

func (ds *Disk) Load(uri impl.Label) (entity wisski.Entity, err error) {
	err = ds.get(uri, func(se *sEntity) error {
		// copy simple fields
		entity.URI = se.URI
		entity.Path = se.Path
		entity.Triples = se.Triples
		entity.Fields = se.Fields

		// load all the child entities
		entity.Children = make(map[string][]wisski.Entity)
		for bundle, value := range se.Children {
			entity.Children[bundle] = make([]wisski.Entity, len(value))
			for i, uri := range value {
				entity.Children[bundle][i], err = ds.childStorages[bundle].Load(uri)
				if err != nil {
					return fmt.Errorf("failed to load WissKI entity: %w", err)
				}
			}
		}
		return nil
	})
	return
}

func (ds *Disk) Close() (err error) {
	if ds.DB != nil {
		err = ds.DB.Close()
		ds.DB = nil
		ds.childStorages = nil
	}
	if err != nil {
		return fmt.Errorf("failed to close DB: %w", err)
	}
	return nil
}
