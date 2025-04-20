package wisski

import (
	"fmt"
	"io"

	"slices"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/anglo-korean/rdf"
)

// cspell:words WissKI

// Entity represents an Entity inside a WissKI Bundle.
type Entity struct {
	Fields   map[string][]FieldValue // values for specific fields
	Children map[string][]Entity     // child paths for child bundles
	URI      impl.Label
	Path     []impl.Label
	Triples  []igraph.Triple
}

// WriteTo writes triples representing this entity into w.
func (entity Entity) WriteAllTriples(w io.Writer, canonical bool, f rdf.Format) (err error) {
	writer := rdf.NewTripleEncoder(w, f)
	defer func() {
		werr := writer.Close()
		if err == nil && werr != nil {
			err = fmt.Errorf("failed to close triple encoder: %w", err)
		}
	}()

	for _, triple := range entity.AllTriples() {
		triple, err := triple.Triple(canonical)
		if err != nil {
			return err
		}

		if err := writer.Encode(triple); err != nil {
			return err
		}
	}

	return nil
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
	slices.SortFunc(triples, igraph.Triple.Compare)

	return slices.CompactFunc(triples, func(left, right igraph.Triple) bool {
		return left.ID == right.ID
	})
}

// It does not deduplicate, and does not return.
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

// FieldValue represents the value of a field inside an entity.
type FieldValue struct {
	Datum   impl.Datum
	Path    []impl.Label
	Triples []igraph.Triple
}
