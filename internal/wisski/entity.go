package wisski

import (
	"github.com/FAU-CDI/hangover/internal/igraph"
	"github.com/FAU-CDI/hangover/internal/imap"
	"golang.org/x/exp/slices"
)

// cspell:words WissKI

// Entity represents an Entity inside a WissKI Bundle
type Entity struct {
	Fields   map[string][]FieldValue // values for specific fields
	Children map[string][]Entity     // child paths for child bundles
	URI      imap.Label
	Path     []imap.Label
	Triples  []igraph.Triple
}

// AllTriples returns all triples that are related to this entity.
// Concretely this means:
//
// - Any Triple defining the entity itself.
// - Any Triple defining any field of the entity.
// - Any Triple defining any child entity.
//
// Triples are returned in globally consistent order.
// Triples are guaranteed not to be repeated.
// This means that any two calls to AllTriples() use the same order.
func (entity Entity) AllTriples() (triples []igraph.Triple) {
	triples = entity.appendTriples(triples)
	slices.SortFunc(triples, func(left, right igraph.Triple) int {
		// TODO: fixme
		if left == right {
			return 0
		}
		if left.ID.Less(right.ID) {
			return -1
		}
		return 1
	})

	return slices.CompactFunc(triples, func(left, right igraph.Triple) bool {
		return left.ID == right.ID
	})
}

// appendTriples appends triples for this entity to triples
// It does not deduplicate, and does not return
func (entity Entity) appendTriples(triples []igraph.Triple) []igraph.Triple {
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
	Value   any
	Path    []imap.Label
	Triples []igraph.Triple
}
