package storages

import (
	"sync"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/tkw1536/pkglib/iterator"
)

type MemoryEngine struct{}

func (MemoryEngine) NewStorage(bundle *pathbuilder.Bundle) (BundleStorage, error) {
	return &Memory{
		bundle:        bundle,
		childStorages: make(map[string]BundleStorage),

		lookup: make(map[impl.Label]int),
	}, nil
}

// Memory implements an in-memory bundle storage
type Memory struct {
	bundle        *pathbuilder.Bundle
	childStorages map[string]BundleStorage
	lookup        map[impl.Label]int
	Entities      []wisski.Entity
	addField      sync.Mutex
	addChild      sync.Mutex
}

// Add adds an entity to this BundleSlice
func (bs *Memory) Add(uri impl.Label, path []impl.Label, triples []igraph.Triple) error {
	bs.lookup[uri] = len(bs.Entities)
	entity := wisski.Entity{
		URI:      uri,
		Path:     path,
		Triples:  triples,
		Fields:   make(map[string][]wisski.FieldValue, len(bs.bundle.ChildFields)),
		Children: make(map[string][]wisski.Entity, len(bs.bundle.ChildBundles)),
	}

	for _, field := range bs.bundle.ChildFields {
		entity.Fields[field.MachineName()] = make([]wisski.FieldValue, 0, field.MakeCardinality())
	}

	for _, bundle := range bs.bundle.ChildBundles {
		entity.Children[bundle.MachineName()] = make([]wisski.Entity, 0, bundle.Path.MakeCardinality())
	}

	bs.Entities = append(bs.Entities, entity)
	return nil
}

func (bs *Memory) AddFieldValue(uri impl.Label, field string, value impl.Datum, path []impl.Label, triples []igraph.Triple) error {
	id, ok := bs.lookup[uri]
	if !ok {
		return ErrNoEntity
	}

	bs.addField.Lock()
	defer bs.addField.Unlock()

	bs.Entities[id].Fields[field] = append(bs.Entities[id].Fields[field], wisski.FieldValue{
		Value:   value,
		Path:    path,
		Triples: triples,
	})

	return nil
}

func (bs *Memory) RegisterChildStorage(bundle string, storage BundleStorage) error {
	bs.childStorages[bundle] = storage
	return nil
}

func (bs *Memory) AddChild(parent impl.Label, bundle string, child impl.Label) error {
	id, ok := bs.lookup[parent]
	if !ok {
		return ErrNoEntity
	}

	bs.addChild.Lock()
	defer bs.addChild.Unlock()

	entity, err := bs.childStorages[bundle].Load(child)
	if err != nil {
		return err
	}
	bs.Entities[id].Children[bundle] = append(bs.Entities[id].Children[bundle], entity)
	return nil
}

func (bs *Memory) Finalize() error {
	return nil
}

func (bs *Memory) Get(parentPathIndex int) iterator.Iterator[LabelWithParent] {
	return iterator.New(func(sender iterator.Generator[LabelWithParent]) {
		defer sender.Return()

		for _, entity := range bs.Entities {
			var parent impl.Label
			if parentPathIndex > -1 {
				parent = entity.Path[parentPathIndex]
			}

			if sender.Yield(LabelWithParent{
				Label:  entity.URI,
				Parent: parent,
			}) {
				break
			}
		}
	})
}

func (bs *Memory) Count() (int64, error) {
	return int64(len(bs.Entities)), nil
}

func (bs *Memory) Load(uri impl.Label) (entity wisski.Entity, err error) {
	index, ok := bs.lookup[uri]
	if !ok {
		return entity, ErrNoEntity
	}
	return bs.Entities[index], nil
}

func (bs *Memory) Close() error {
	bs.lookup = nil
	bs.childStorages = nil
	return nil
}
