package sparkl

import (
	"errors"
	"fmt"
	"sync"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl/exporter"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:words pathbuilder

// Export loads all top-level paths from the given path-builder from the index into the given engine.
// Afterwards it is exported into the given exporter.
func Export(pb *pathbuilder.Pathbuilder, index *igraph.Index, engine storages.BundleEngine, exporter exporter.Exporter, stats *stats.Stats) (e error) {
	bundles := pb.Bundles()

	storages, closer, err := StoreBundles(bundles, index, engine, stats)
	if closer != nil {
		defer func() {
			if e2 := closer(); e2 != nil {
				e2 = fmt.Errorf("failed to close bundle storage: %w", e2)
				if e == nil {
					e = e2
				} else {
					e = errors.Join(e, e2)
				}
			}
		}()
	}
	if err != nil {
		return err
	}

	var errOnce sync.Once
	var gErr error

	var wg sync.WaitGroup
	for i := range storages {
		wg.Add(1)
		go (func(i int) {
			defer wg.Done()

			err := func() (e error) {
				storage := storages[i]
				bundle := bundles[i]
				defer func() {
					if e2 := storage.Close(); e2 != nil {
						e2 = fmt.Errorf("failed to close storage: %w", e2)
						if e == nil {
							e = e2
						} else {
							e = errors.Join(e, e2)
						}
					}
				}()

				// count the number of elements
				count, err := storage.Count()
				if err != nil {
					return err
				}

				// start the exporter
				if err := exporter.Begin(bundle, count); err != nil {
					errOnce.Do(func() { gErr = err })
					return err
				}

				// make sure it is also closed
				defer func() {
					err := exporter.End(bundle)
					if e == nil {
						e = err
					}
				}()

				// load uris from storage

				// load all the entities
				for element, err := range storage.Get(-1) {
					if err != nil {
						return err
					}
					entity, err := storage.Load(element.Label)
					if err != nil {
						return err
					}
					if err := exporter.Add(bundle, &entity); err != nil {
						return err
					}
				}

				return nil
			}()

			if err != nil {
				errOnce.Do(func() { gErr = err })
			}
		})(i)
	}
	wg.Wait()

	// close the exporter
	{
		err := exporter.Close()
		if gErr == nil {
			gErr = err
		}
	}

	return gErr
}

// LoadPathbuilder loads all paths in the given pathbuilder.
func LoadPathbuilder(pb *pathbuilder.Pathbuilder, index *igraph.Index, engine storages.BundleEngine, stats *stats.Stats) (map[string][]wisski.Entity, error) {
	mp := exporter.Map{
		Data: make(map[string][]wisski.Entity, len(pb.Bundles())),
	}
	err := Export(pb, index, engine, &mp, stats)
	if err != nil {
		return nil, err
	}
	return mp.Data, nil
}
