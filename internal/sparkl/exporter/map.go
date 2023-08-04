package exporter

import (
	"sync"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// Map implements an exporter that stores data inside a map.
type Map struct {
	Data map[string][]wisski.Entity
	l    sync.Mutex
}

// Begin signals that count entities will be transmitted for the given bundle
func (mp *Map) Begin(bundle *pathbuilder.Bundle, count int64) error {
	mp.l.Lock()
	defer mp.l.Unlock()
	mp.Data[bundle.MachineName()] = make([]wisski.Entity, 0, int(count))
	return nil
}

// Add adds entities for the given bundle
func (mp *Map) Add(bundle *pathbuilder.Bundle, entity *wisski.Entity) error {
	mp.l.Lock()
	defer mp.l.Unlock()
	mp.Data[bundle.MachineName()] = append(mp.Data[bundle.MachineName()], *entity)
	return nil
}

// End signals that no more entities will be submitted for the given bundle
func (mp *Map) End(bundle *pathbuilder.Bundle) error {
	return nil // no-op
}

func (mp *Map) Close() error {
	return nil // no-op
}
