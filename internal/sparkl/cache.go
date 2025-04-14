package sparkl

import (
	"errors"
	"runtime"

	"maps"
	"slices"

	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:Words WIssKI imap

// Cache represents an easily accessible cache of WissKIObjects.
// It is held entirely in memory.
type Cache struct {
	engine      imap.MemoryMap             // the engine used for the imap
	beIndex     map[string][]wisski.Entity // mappings from bundles to entities
	biIndex     map[string]map[impl.ID]int // index into beIndex by uri
	ebIndex     map[impl.ID]string         // index from entity uri into bundle
	sameAs      map[impl.ID]impl.ID        // canonical name mappings from entities
	aliasOf     map[impl.ID][]impl.ID      // opposite of sameAs
	uris        *imap.IMap                 // holds mappings between ids and uris
	bundleNames []string                   // names of all bundles
}

func (cache *Cache) Close() error {
	if cache == nil {
		return nil
	}

	defer runtime.GC()

	cache.beIndex = nil
	cache.biIndex = nil
	cache.ebIndex = nil
	cache.bundleNames = nil
	cache.sameAs = nil
	cache.aliasOf = nil

	return errors.Join(
		cache.engine.Close(),
		cache.uris.Close(),
	)
}

func (cache Cache) Entities(bundle_machine string) []wisski.Entity {
	return cache.beIndex[bundle_machine]
}

func (cache Cache) BundleNames() []string {
	return cache.bundleNames
}

// TODO: Do we want to use an IMap here?

// NewCache creates a new cache from a bundle-entity-map.
func NewCache(Data map[string][]wisski.Entity, SameAs imap.HashMap[impl.Label, impl.Label], st *stats.Stats) (c Cache, err error) {
	var counter int
	progress := func() {
		counter++
		st.SetCT(counter, counter)
	}

	// reset the uris
	c.uris = &imap.IMap{}
	c.uris.Reset(&c.engine)

	// store the bundle-entity index
	c.beIndex = Data
	c.biIndex = make(map[string]map[impl.ID]int, len(c.beIndex))
	c.ebIndex = make(map[impl.ID]string)
	for bundle, entities := range c.beIndex {
		c.biIndex[bundle] = make(map[impl.ID]int, len(entities))
		for i, entity := range entities {
			id, err := c.uris.Add(entity.URI)
			if err != nil {
				return c, err
			}
			c.biIndex[bundle][id.Canonical] = i
			c.ebIndex[id.Canonical] = bundle

			progress()
		}
	}

	c.bundleNames = slices.AppendSeq(make([]string, 0, len(c.beIndex)), maps.Keys(c.beIndex))
	slices.Sort(c.bundleNames)

	sameAsCount, err := SameAs.Count()
	if err != nil {
		return c, err
	}

	// setup same-as and same-as-in
	c.sameAs = make(map[impl.ID]impl.ID, sameAsCount)
	c.aliasOf = make(map[impl.ID][]impl.ID, sameAsCount)

	err = SameAs.Iterate(func(alias, canon impl.Label) error {
		defer progress()

		aliass, err := c.uris.Add(alias)
		if err != nil {
			return err
		}
		canons, err := c.uris.Add(canon)
		if err != nil {
			return err
		}

		c.sameAs[aliass.Canonical] = canons.Canonical
		c.aliasOf[canons.Canonical] = append(c.aliasOf[canons.Canonical], aliass.Canonical)

		return nil
	})
	if err != nil {
		return c, err
	}

	return c, nil
}

func (c Cache) canonical(uri impl.Label) impl.ID {
	id, err := c.uris.Forward(uri)
	if err != nil {
		return id
	}
	if cid, ok := c.sameAs[id]; ok {
		return cid
	}
	return id
}

// Canonical returns the canonical version of the given uri.
func (c Cache) Canonical(uri impl.Label) impl.Label {
	canon, _ := c.uris.Reverse(c.canonical(uri))
	return canon
}

// Aliases returns the Aliases of the given impl.Label, excluding itself.
func (c Cache) Aliases(uri impl.Label) []impl.Label {
	id, err := c.uris.Forward(uri)
	if err != nil {
		return nil
	}

	aids := c.aliasOf[id]
	aliases := make([]impl.Label, 0, len(aids))
	for _, id := range aids {
		alias, err := c.uris.Reverse(id)
		if err != nil {
			continue
		}
		aliases = append(aliases, alias)
	}
	return aliases
}

// Bundle returns the bundle of the given uri, if any.
func (c Cache) Bundle(uri impl.Label) (string, bool) {
	cid := c.canonical(uri)
	bundle, ok := c.ebIndex[cid]
	return bundle, ok
}

// FirstBundle returns the first bundle for which the given impl.Label exists.
func (c Cache) FirstBundle(uris ...impl.Label) (uri impl.Label, bundle string, ok bool) {
	for _, uri := range uris {
		bundle, ok = c.Bundle(uri)
		if ok {
			return uri, bundle, true
		}
	}
	return
}

// Entity looks up the given entity.
func (c Cache) Entity(uri impl.Label, bundle string) (*wisski.Entity, bool) {
	index, ok := c.biIndex[bundle][c.canonical(uri)]
	if !ok {
		return nil, false
	}
	return &c.beIndex[bundle][index], true
}
