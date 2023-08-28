package storages

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/igraph"
	"github.com/FAU-CDI/hangover/pkg/imap"
	"github.com/tkw1536/pkglib/iterator"
)

// BundleEngine is a function that initializes and returns a new BundleStorage
type BundleEngine interface {
	NewStorage(bundle *pathbuilder.Bundle) (BundleStorage, error)
}

// NewBundleEngine creates a new BundleEngine backed by the disk at the provided path.
// If path is the empty string, return a memory-backed engine instead.
func NewBundleEngine(path string) BundleEngine {
	if path == "" {
		return MemoryEngine{}
	}
	return DiskEngine{
		Path: filepath.Join(path, "bundles"),
	}
}

// BundleStorage is responsible for storing entities for a single bundle
type BundleStorage interface {
	io.Closer

	// Add adds a new entity with the given URI (and optional path information)
	// to this bundle.
	//
	// Calls to add for a specific bundle storage are serialized.
	Add(uri imap.Label, path []imap.Label, triples []igraph.Triple) error

	// AddFieldValue adds a value to the given field for the entity with the given uri.
	//
	// Concurrent calls to distinct fields may take place, however within each field calls are always synchronized.
	//
	// A non-existing parent should return ErrNoEntity.
	AddFieldValue(uri imap.Label, field string, value any, path []imap.Label, triples []igraph.Triple) error

	// RegisterChildStorage register the given storage as a BundleStorage for the child bundle.
	// The Storage should delete the reference to the child storage when it is closed.
	RegisterChildStorage(bundle string, storage BundleStorage) error

	// AddChild adds a child entity of the given bundle to the given entity.
	//
	// Multiple concurrent calls to AddChild may take place, but every concurrent call will be for a different bundle.
	//
	// A non-existing parent should return ErrNoEntity.
	AddChild(parent imap.Label, bundle string, child imap.Label) error

	// Finalize is called to signal to this storage that no more write operations will take place.
	Finalize() error

	// Get returns an iterator that iterates over the url of every entity in this bundle, along with their parent URIs.
	// The iterator is guaranteed to iterate in some consistent order, but no further guarantees beyond that.
	//
	// parentPathIndex returns the index of the parent uri in child paths.
	Get(parentPathIndex int) iterator.Iterator[LabelWithParent]

	// Count counts the number of entities in this storage.
	Count() (int64, error)

	// Load loads an entity with the given URI from this storage.
	// A non-existing entity should return err = ErrNoEntity.
	Load(uri imap.Label) (wisski.Entity, error)
}

var (
	ErrNoEntity = errors.New("no such entity")
)

// LabelWithParent represents a URI along with it's parent
type LabelWithParent struct {
	Label  imap.Label
	Parent imap.Label
}
