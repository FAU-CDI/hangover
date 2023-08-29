package viewer

import (
	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/imap"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:words pathbuilder

// findBundle returns a bundle by machine name and makes sure that appropriate caches are filled
func (viewer *Viewer) findBundle(machine string) (bundle *pathbuilder.Bundle, ok bool) {
	bundle = viewer.Pathbuilder.Bundle(machine)
	if bundle == nil {
		return nil, false
	}

	return bundle, true
}

// findEntity finds an entity by the given bundle machine name
func (viewer *Viewer) findEntity(bundle_machine string, uri imap.Label) (bundle *pathbuilder.Bundle, entity *wisski.Entity, ok bool) {
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

// getEntityURIs returns the URIs belonging to a single bundle
// TODO: Make this stream
func (viewer *Viewer) getEntityURIs(id string) (bundle *pathbuilder.Bundle, uris []imap.Label, ok bool) {
	bundle, ok = viewer.findBundle(id)
	if !ok {
		return nil, nil, false
	}

	entities := viewer.Cache.Entities(bundle.MachineName())
	uris = make([]imap.Label, len(entities))
	for i, entity := range entities {
		uris[i] = entity.URI
	}
	return bundle, uris, true
}

// getEntityURIs returns the URIs belonging to a single bundle
// TODO: Make this stream
func (viewer *Viewer) getEntity(id string, uri imap.Label) (entity *wisski.Entity, ok bool) {
	_, entity, ok = viewer.findEntity(id, uri)
	return
}
