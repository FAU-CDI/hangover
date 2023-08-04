package wisski

import (
	"github.com/FAU-CDI/hangover/pkg/igraph"
	"golang.org/x/exp/slices"
)

// Entity represents an Entity inside a WissKI Bundle
type Entity struct {
	URI     URI      // URI of this entity
	Path    []URI    // the path of this entity
	Triples []Triple // the triples that define this entity itself

	Fields   map[string][]FieldValue // values for specific fields
	Children map[string][]Entity     // child paths for specific entities
}

// AllTriples returns all triples that are related to this entity.
// Concretetly this means:
//
// - Any Triple defining the entity itself.
// - Any Triple defining any field of the entity.
// - Any Triple defining any child entity.
//
// Triples are returned in globally consistent order.
// Triples are guaranteed not to be repeated.
// This means that any two calls to AllTriples() use the same order.
func (entity Entity) AllTriples() (triples []Triple) {
	triples = entity.appendTriples(triples)
	slices.SortFunc(triples, func(left, right Triple) int {
		// TODO: fixme
		if left == right {
			return 0
		}
		if left.ID.Less(right.ID) {
			return -1
		}
		return 1
	})

	return slices.CompactFunc(triples, func(left, right Triple) bool {
		return left.ID == right.ID
	})
}

// appendTriples appends triples for this entity to triples
// It does not deduplicate, and does not return
func (entity Entity) appendTriples(triples []Triple) []Triple {
	triples = append(triples, entity.Triples...)
	for _, fields := range entity.Fields {
		for _, field := range fields {
			triples = append(triples, field.Triples...)
		}
	}

	for _, children := range entity.Children {
		for _, child := range children {
			triples = child.appendTriples(triples)
		}
	}
	return triples
}

// FieldValue represents the value of a field inside an entity
type FieldValue struct {
	Path    []URI
	Triples []Triple
	Value   any
}

// Triple represents a triple of WissKI Data
type Triple = igraph.Triple[URI, any]
