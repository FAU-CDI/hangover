// Package glass provides Glass
//
//spellchecker:words glass
package glass

//spellchecker:words errors runtime debug github drincw pathbuilder pbxml hangover internal sparkl storages stats triplestore imap impl viewer wisski
import (
	"errors"
	"fmt"
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

//spellchecker:words Wiss KI pathbuilder nquads

const GlassVersion = 2

// Glass represents a stand-alone representation of a WissKI.
type Glass struct {
	Pathbuilder pathbuilder.Pathbuilder
	Cache       *sparkl.Cache
	Flags       viewer.RenderFlags
}

func (glass *Glass) Close() error {
	if err := glass.Cache.Close(); err != nil {
		return fmt.Errorf("failed to close cache: %w", err)
	}
	return nil
}

// Create creates a new glass from the given pathbuilder and nquads.
// output is written to output.
func Create(pathbuilderPath string, nquadsPath string, cacheDir string, flags viewer.RenderFlags, st *stats.Stats) (drincw Glass, e error) {
	// read the pathbuilder
	if err := st.DoStage(stats.StageReadPathbuilder, func() (err error) {
		drincw.Pathbuilder, err = pbxml.Load(pathbuilderPath)
		if err != nil {
			return fmt.Errorf("failed to load pathbuilder xml: %w", err)
		}
		return nil
	}); err != nil {
		return drincw, fmt.Errorf("failed to load pathbuilder: %w", err)
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
		return drincw, fmt.Errorf("failed to load index: %w", err)
	}

	st.Log("finished indexing", "stats", st.IndexStats())
	defer func() {
		if e2 := index.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close index: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	// extract the bundles
	var bundles map[string][]wisski.Entity
	if err := st.DoStage(stats.StageExtractBundles, func() (err error) {
		bundles, err = sparkl.LoadPathbuilder(&drincw.Pathbuilder, index, bEngine, st)
		if err != nil {
			return fmt.Errorf("failed to load pathbuilder: %w", err)
		}
		return nil
	}); err != nil {
		return drincw, fmt.Errorf("failed to extract bundles: %w", err)
	}

	// extract the cache

	identities := imap.MakeMemory[impl.Label, impl.Label](0)

	if err := st.DoStage(stats.StageExtractSameAs, func() error {
		return index.IdentityMap(&identities)
	}); err != nil {
		return Glass{}, fmt.Errorf("failed to extract same as: %w", err)
	}

	if err := st.DoStage(stats.StageExtractCache, func() error {
		cache, err := sparkl.NewCache(bundles, &identities, st)
		if err != nil {
			return fmt.Errorf("failed to create new cache: %w", err)
		}
		drincw.Cache = &cache
		return nil
	}); err != nil {
		return Glass{}, fmt.Errorf("failed to extract cache: %w", err)
	}

	// We close the index early, because it's no longer needed
	if err := index.Close(); err != nil {
		return drincw, fmt.Errorf("failed to close index: %w", err)
	}
	// force returning memory to the os
	debug.FreeOSMemory()

	drincw.Flags = flags
	return drincw, nil
}
