package sparkl

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/status"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/tkw1536/pkglib/iterator"
)

// StoreBundle loads all entities from the given bundle into a new storage, which is then returned.
//
// Storages for any child bundles, and the bundle itself, are created using the makeStorage function.
// The storage for this bundle is returned.
func StoreBundle(bundle *pathbuilder.Bundle, index *igraph.Index, engine storages.BundleEngine, stats *status.Stats) (storages.BundleStorage, func() error, error) {
	storages, closer, err := StoreBundles([]*pathbuilder.Bundle{bundle}, index, engine, stats)
	if err != nil {
		return nil, nil, err
	}
	return storages[0], closer, err
}

// StoreBundles is like StoreBundle, but takes multiple bundles
func StoreBundles(bundles []*pathbuilder.Bundle, index *igraph.Index, engine storages.BundleEngine, stats *status.Stats) ([]storages.BundleStorage, func() error, error) {
	context := &Context{
		Index:  index,
		Engine: engine,
		Stats:  stats,
	}
	context.Open()

	// compute the total number of bundles
	var total int
	for _, b := range bundles {
		total += totalBundleCount(b)
	}
	stats.SetCT(0, total)

	// update the counter every time we finish a bundle
	var counter atomic.Int64
	storages := make([]storages.BundleStorage, len(bundles))
	for i := range storages {
		storages[i] = context.Store(bundles[i], func() {
			current := int(counter.Add(1))
			stats.SetCT(current, total)
		})
	}
	err := context.Wait()

	return storages, context.Close, err
}

// totalBundleCount recursively counts the number of bundles on the tree under bundle.
func totalBundleCount(bundle *pathbuilder.Bundle) (counter int) {
	if bundle == nil {
		return 0
	}
	counter++
	for _, b := range bundle.ChildBundles {
		counter += totalBundleCount(b)
	}
	return
}

// Context represents a context to extract bundle data from index into storages.
//
// A Context must be opened, and eventually waited on.
// See [Open] and [Close].
type Context struct {
	Engine       storages.BundleEngine
	err          error          // error that occurred (if any)
	Index        *igraph.Index  // index being used
	closers      chan io.Closer // what to close when done
	extractWait  sync.WaitGroup // waiting on extracting entities in all bundles
	childAddWait sync.WaitGroup // loading child entities wait
	errOnce      sync.Once      // to set the error

	Stats *status.Stats
}

// Open opens this context, and signals that multiple calls to Store() may follow.
//
// Multiple calls to Open are invalid.
func (context *Context) Open() {
	context.extractWait.Add(1)
	context.closers = make(chan io.Closer)
}

// Wait signals this context that no more bundles will be loaded.
// And then waits for all bundle extracting to finish.
//
// Multiple calls to Wait() are invalid.
func (context *Context) Wait() error {
	context.extractWait.Done()
	context.extractWait.Wait()
	context.childAddWait.Wait()
	return context.err
}

// Close closes this context
func (context *Context) Close() (err error) {
	for {
		select {
		case closer := <-context.closers:
			cErr := closer.Close()
			if err == nil {
				err = cErr
			}
		default:
			return nil
		}
	}
}

// reportError stores an error in this context
// if error is non-nil, returns true.
func (context *Context) reportError(err error) bool {
	if err == nil {
		return false
	}
	context.errOnce.Do(func() {
		context.err = err
	})
	return true
}

// Store creates a new Storage for the given bundle and schedules entities to be loaded.
// Once onFinish is finished
// May only be called between calls [Open] and [Wait].
//
// Any error that occurs is returned only by Wait.
func (context *Context) Store(bundle *pathbuilder.Bundle, onFinish func()) storages.BundleStorage {
	context.extractWait.Add(1)

	// create a new context
	storage, err := context.Engine.NewStorage(bundle)
	if context.reportError(err) {
		context.extractWait.Done()
		onFinish()
		return nil
	}

	go func() {
		defer onFinish()
		defer context.extractWait.Done()

		// determine the index of the URI within the paths describing this bundle
		// this is the length of the parent path, or zero (if it does not exist).
		var entityURIIndex int
		if bundle.Parent != nil {
			entityURIIndex = len(bundle.Path.PathArray) / 2
		}

		// stage 1: load the entities themselves
		err := (func() error {
			paths := extractPath(bundle.Path, context.Index, context.Stats)
			defer paths.Close()

			for paths.Next() {
				path := paths.Datum()
				nodes := path.Nodes
				triples := path.Triples
				storage.Add(nodes[entityURIIndex], nodes, triples)
			}

			return paths.Err()
		})()
		if context.reportError(err) {
			return
		}

		// stage 2: fill all the fields
		for _, field := range bundle.Fields() {
			context.extractWait.Add(1)
			go func(field pathbuilder.Field) {
				defer context.extractWait.Done()

				paths := extractPath(field.Path, context.Index, context.Stats)
				defer paths.Close()

				for paths.Next() {
					path := paths.Datum()

					err = storage.AddFieldValue(path.Nodes[entityURIIndex], field.MachineName(), path.Value(), path.Language, path.Nodes, path.Triples)
					if err != storages.ErrNoEntity {
						context.reportError(err)
					}
				}
				context.reportError(paths.Err())
			}(field)
		}

		// stage 3: read child paths
		cstorages := make([]storages.BundleStorage, len(bundle.ChildBundles))
		for i, bundle := range bundle.ChildBundles {
			cstorages[i] = context.Store(bundle, onFinish)
			if cstorages[i] == nil {
				// creating the storage has failed, so we don't need to continue
				// and we can return immediately.
				return
			}

			err := storage.RegisterChildStorage(bundle.MachineName(), cstorages[i])
			context.reportError(err)
		}

		context.childAddWait.Add(len(cstorages))

		// stage 4: register all the child entities
		go func() {
			context.extractWait.Wait()

			var wg sync.WaitGroup

			for i, cstorage := range cstorages {
				wg.Add(1)
				go func(cstorage storages.BundleStorage, bundle *pathbuilder.Bundle) {
					defer wg.Done()
					defer context.childAddWait.Done()

					children := cstorage.Get(entityURIIndex)
					for children.Next() {
						child := children.Datum()
						err := storage.AddChild(child.Parent, bundle.MachineName(), child.Label)
						if err != storages.ErrNoEntity {
							context.reportError(err)
						}
					}
					context.reportError(children.Err())
				}(cstorage, bundle.ChildBundles[i])
			}

			wg.Wait()
			storage.Finalize() // no more writing!

			// tell the storage to be closed on a call to Close()
			context.closers <- storage
		}()
	}()

	return storage
}

const (
	debugLogAllPaths = false // turn this on to log all paths being queried
)

var debugLogID int64 // id of the current log id

// extractPath extracts values for a single path from the index.
// The returned channel is never nil.
//
// Any values found along the path are written to the returned channel which is then closed.
// If an error occurs, it is written to errDst before the channel is closed.
func extractPath(path pathbuilder.Path, index *igraph.Index, stats *status.Stats) iterator.Iterator[igraph.Path] {
	// start with the path array
	uris := append([]string{}, path.PathArray...)
	if len(uris) == 0 {
		return iterator.Empty[igraph.Path](nil)
	}

	// add the datatype property if are not a group
	// and it is not empty
	if datatype := path.Datatype(); !path.IsGroup && datatype != "" {
		uris = append(uris, datatype)
	}

	// if debugging is enabled, set it up
	var debugID int64
	if debugLogAllPaths {
		debugID = atomic.AddInt64(&debugLogID, 1)
	}

	set, err := index.PathsStarting(wisski.Type, impl.Label(uris[0]))
	if err != nil {
		return iterator.Empty[igraph.Path](err)
	}
	if debugLogAllPaths {
		size, err := set.Size()
		if err != nil {
			return iterator.Empty[igraph.Path](err)
		}
		stats.LogDebug("path", "id", debugID, "uri", uris[0], "size", size)
	}

	for i := 1; i < len(uris); i++ {
		if i%2 == 0 {
			if err := set.Ending(wisski.Type, impl.Label(uris[i])); err != nil {
				return iterator.Empty[igraph.Path](err)
			}
		} else {
			if err := set.Connected(impl.Label(uris[i])); err != nil {
				return iterator.Empty[igraph.Path](err)
			}
		}

		if debugLogAllPaths {
			size, err := set.Size()
			if err != nil {
				return iterator.Empty[igraph.Path](err)
			}
			stats.LogDebug("uri", "id", debugID, "uris", uris[i], "size", size)
		}
	}

	return set.Paths()
}
