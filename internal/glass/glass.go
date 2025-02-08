// Package glass provides Glass
package glass

import (
	"runtime/debug"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:words WissKI pathbuilder nquads

const GlassVersion = 2

// Glass represents a stand-alone representation of a WissKI
type Glass struct {
	Pathbuilder pathbuilder.Pathbuilder
	Cache       *sparkl.Cache
	Flags       viewer.RenderFlags
}

func (glass *Glass) Close() error {
	return glass.Cache.Close()
}

// Create creates a new glass from the given pathbuilder and nquads.
// output is written to output.
func Create(pathbuilderPath string, nquadsPath string, cacheDir string, flags viewer.RenderFlags, st *stats.Stats) (drincw Glass, err error) {
	// read the pathbuilder
	if err := st.DoStage(stats.StageReadPathbuilder, func() (err error) {
		drincw.Pathbuilder, err = pbxml.Load(pathbuilderPath)
		return err
	}); err != nil {
		return drincw, err
	}

	// make an engine
	engine := sparkl.NewEngine(cacheDir)
	bEngine := storages.NewBundleEngine(cacheDir)
	if cacheDir != "" {
		st.Log("caching data on-disk", "path", cacheDir)
	}

	// build an index
	index, err := sparkl.LoadIndex(nquadsPath, flags.Predicates, engine, sparkl.DefaultIndexOptions(&drincw.Pathbuilder), st)
	if err != nil {
		return drincw, err
	}

	st.Log("finished indexing", "stats", st.IndexStats())
	defer index.Close()

	// extract the bundles
	var bundles map[string][]wisski.Entity
	st.DoStage(stats.StageExtractBundles, func() (err error) {
		bundles, err = sparkl.LoadPathbuilder(&drincw.Pathbuilder, index, bEngine, st)
		return err
	})
	if err != nil {
		return drincw, err
	}

	// extract the cache

	identities := imap.MakeMemory[impl.Label, impl.Label](0)

	if err := st.DoStage(stats.StageExtractSameAs, func() error {
		return index.IdentityMap(&identities)
	}); err != nil {
		return Glass{}, err
	}

	if err := st.DoStage(stats.StageExtractCache, func() error {
		cache, err := sparkl.NewCache(bundles, &identities, st)
		if err != nil {
			return err
		}
		drincw.Cache = &cache
		return nil
	}); err != nil {
		return Glass{}, err
	}

	if err != nil {
		return drincw, err
	}

	index.Close()        // We close the index early, because it's no longer needed
	debug.FreeOSMemory() // force returning memory to the os

	drincw.Flags = flags
	return drincw, nil
}
