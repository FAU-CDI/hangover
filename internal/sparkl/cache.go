package sparkl

import (
	"encoding/gob"
	"errors"
	"runtime"

	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/imap"
	"github.com/FAU-CDI/hangover/pkg/sgob"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// cspell:Words WIssKI imap

// Cache represents an easily accessible cache of WissKIObjects.
// It is held entirely in memory.
type Cache struct {
	beIndex map[string][]wisski.Entity // mappings from bundles to entities
	biIndex map[string]map[imap.ID]int // index into beIndex by uri
	ebIndex map[imap.ID]string         // index from entity uri into bundle

	bundleNames []string // names of all bundles

	sameAs  map[imap.ID]imap.ID   // canonical name mappings from entities
	aliasOf map[imap.ID][]imap.ID // opposite of sameAs

	engine imap.MemoryMap // the engine used for the imap
	uris   *imap.IMap     // holds mappings between ids and uris
}

func (cache *Cache) Close() error {
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

// EncodeTo encodes this cache object to the given gob.Encoder.
func (cache *Cache) EncodeTo(encoder *gob.Encoder) error {
	for _, obj := range []any{
		cache.beIndex,
		cache.biIndex,
		cache.ebIndex,
		cache.bundleNames,
		cache.sameAs,
		cache.aliasOf,
		cache.engine.FStorage,
		cache.engine.RStorage,
	} {
		if err := sgob.Encode(encoder, obj); err != nil {
			return err
		}
	}

	return nil
}

func (cache *Cache) DecodeFrom(decoder *gob.Decoder) error {
	if cache.uris != nil {
		if err := cache.uris.Close(); err != nil {
			return err
		}
	}

	for _, obj := range []any{
		&cache.beIndex,
		&cache.biIndex,
		&cache.ebIndex,
		&cache.bundleNames,
		&cache.sameAs,
		&cache.aliasOf,
		&cache.engine.FStorage,
		&cache.engine.RStorage,
	} {
		if err := sgob.Decode(decoder, obj); err != nil {
			return err
		}
	}

	cache.uris = &imap.IMap{}
	return cache.uris.Reset(&cache.engine)
}

func (cache Cache) Entities(bundle_machine string) []wisski.Entity {
	return cache.beIndex[bundle_machine]
}

func (cache Cache) BundleNames() []string {
	return cache.bundleNames
}

// TODO: Do we want to use an IMap here?

// NewCache creates a new cache from a bundle-entity-map
func NewCache(Data map[string][]wisski.Entity, SameAs imap.HashMap[imap.Label, imap.Label]) (c Cache, err error) {
	// reset the uris
	c.uris = &imap.IMap{}
	c.uris.Reset(&c.engine)

	// store the bundle-entity index
	c.beIndex = Data
	c.biIndex = make(map[string]map[imap.ID]int, len(c.beIndex))
	c.ebIndex = make(map[imap.ID]string)
	for bundle, entities := range c.beIndex {
		c.biIndex[bundle] = make(map[imap.ID]int, len(entities))
		for i, entity := range entities {
			id, err := c.uris.Add(entity.URI)
			if err != nil {
				return c, err
			}
			c.biIndex[bundle][id.Canonical] = i
			c.ebIndex[id.Canonical] = bundle
		}
	}

	c.bundleNames = maps.Keys(c.beIndex)
	slices.Sort(c.bundleNames)

	sameAsCount, err := SameAs.Count()
	if err != nil {
		return c, err
	}

	// setup same-as and same-as-in
	c.sameAs = make(map[imap.ID]imap.ID, sameAsCount)
	c.aliasOf = make(map[imap.ID][]imap.ID, sameAsCount)

	err = SameAs.Iterate(func(alias, canon imap.Label) error {
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

func (c Cache) canonical(uri imap.Label) imap.ID {
	id, err := c.uris.Forward(uri)
	if err != nil {
		return id
	}
	if cid, ok := c.sameAs[id]; ok {
		return cid
	}
	return id
}

// Canonical returns the canonical version of the given uri
func (c Cache) Canonical(uri imap.Label) imap.Label {
	canon, _ := c.uris.Reverse(c.canonical(uri))
	return canon
}

// Aliases returns the Aliases of the given imap.Label, excluding itself
func (c Cache) Aliases(uri imap.Label) []imap.Label {
	id, err := c.uris.Forward(uri)
	if err != nil {
		return nil
	}

	aids := c.aliasOf[id]
	aliases := make([]imap.Label, 0, len(aids))
	for _, id := range aids {
		alias, err := c.uris.Reverse(id)
		if err != nil {
			continue
		}
		aliases = append(aliases, alias)
	}
	return aliases
}

// Bundle returns the bundle of the given uri, if any
func (c Cache) Bundle(uri imap.Label) (string, bool) {
	cid := c.canonical(uri)
	bundle, ok := c.ebIndex[cid]
	return bundle, ok
}

// FirstBundle returns the first bundle for which the given imap.Label exists
func (c Cache) FirstBundle(uris ...imap.Label) (uri imap.Label, bundle string, ok bool) {
	for _, uri := range uris {
		bundle, ok = c.Bundle(uri)
		if ok {
			return uri, bundle, true
		}
	}
	return
}

// Entity looks up the given entity
func (c Cache) Entity(uri imap.Label, bundle string) (*wisski.Entity, bool) {
	index, ok := c.biIndex[bundle][c.canonical(uri)]
	if !ok {
		return nil, false
	}
	return &c.beIndex[bundle][index], true
}
