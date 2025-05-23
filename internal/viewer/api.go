//spellchecker:words viewer
package viewer

//spellchecker:words github drincw pathbuilder hangover internal stats triplestore igraph impl wisski pkglib perf
import (
	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/tkw1536/pkglib/perf"
)

//spellchecker:words pathbuilder

// findBundle returns a bundle by machine name and makes sure that appropriate caches are filled.
func (viewer *Viewer) findBundle(machine string) (bundle *pathbuilder.Bundle, ok bool) {
	bundle = viewer.Pathbuilder.Bundle(machine)
	if bundle == nil {
		return nil, false
	}

	return bundle, true
}

// findEntity finds an entity by the given bundle machine name.
func (viewer *Viewer) findEntity(bundle_machine string, uri impl.Label) (bundle *pathbuilder.Bundle, entity *wisski.Entity, ok bool) {
	bundle, ok = viewer.findBundle(bundle_machine)
	if !ok {
		return nil, nil, false
	}

	entity, ok = viewer.Cache.Entity(uri, bundle.MachineName())
	if !ok {
		return nil, nil, false
	}

	return
}

func (viewer *Viewer) getBundles() (bundles []*pathbuilder.Bundle, ok bool) {
	names := viewer.Cache.BundleNames()
	bundles = make([]*pathbuilder.Bundle, 0, len(names))
	for _, name := range names {
		bundle := viewer.Pathbuilder.Bundle(name)
		if bundle == nil {
			// If this happens, something in the pathbuilder is very corrupt.
			// This should never happen.
			// you should never hit this case.
			continue
		}
		bundles = append(bundles, bundle)
	}
	return bundles, true
}

// TODO: Make this stream.
func (viewer *Viewer) getEntityURIs(id string) (bundle *pathbuilder.Bundle, uris []impl.Label, ok bool) {
	bundle, ok = viewer.findBundle(id)
	if !ok {
		return nil, nil, false
	}

	entities := viewer.Cache.Entities(bundle.MachineName())
	uris = make([]impl.Label, len(entities))
	for i, entity := range entities {
		uris[i] = entity.URI
	}
	return bundle, uris, true
}

// TODO: Make this stream.
func (viewer *Viewer) getEntity(id string, uri impl.Label) (entity *wisski.Entity, ok bool) {
	_, entity, ok = viewer.findEntity(id, uri)
	return
}

// Perf represents viewer performance.
type Perf struct {
	Stages []stats.StageStats
	Index  igraph.Stats
	Now    perf.Snapshot
}

func (viewer *Viewer) Perf() Perf {
	return Perf{
		Stages: viewer.Stats.All(),
		Index:  viewer.Stats.IndexStats(),
	}
}
