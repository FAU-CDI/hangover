//spellchecker:words wisski
package wisski

//spellchecker:words slices github hangover internal triplestore igraph impl anglo korean
import (
	"context"
	"fmt"
	"io"

	"slices"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/anglo-korean/rdf"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
)

//spellchecker:words Wiss KI

// Entity represents an Entity inside a WissKI Bundle.
type Entity struct {
	Fields   map[string][]FieldValue // values for specific fields
	Children map[string][]Entity     // child paths for child bundles
	URI      impl.Label
	Path     []impl.Label
	Triples  []igraph.Triple
}

func (entity Entity) WriteGraphViz(ctx context.Context, format graphviz.Format, w io.Writer) (err error) {
	g, err := graphviz.New(ctx)
	if err != nil {
		return fmt.Errorf("failed to instantiate graphviz: %w", err)
	}

	graph, err := g.Graph(graphviz.WithDirectedType(cgraph.StrictDirected))
	if err != nil {
		return fmt.Errorf("failed to create graph: %w", err)
	}

	graph.SetRankDir(cgraph.LRRank)

	nodes := make(map[string]*cgraph.Node)
	makeNode := func(name string) (*cgraph.Node, error) {
		node, ok := nodes[name]
		if ok {
			return node, nil
		}
		nodes[name], err = graph.CreateNodeByName(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
		return nodes[name], nil
	}

	counter := 0

	// add all the triples
	for _, triple := range entity.AllTriples() {
		subject, err := makeNode(string(triple.SSubject))
		if err != nil {
			return fmt.Errorf("failed to create subject node: %w", err)
		}

		var object *cgraph.Node

		if triple.Role != igraph.Data {
			object, err = makeNode(string(triple.SObject))
			if err != nil {
				return fmt.Errorf("failed to create object node: %w", err)
			}
		} else {
			counter++
			object, err = graph.CreateNodeByName(fmt.Sprintf("_:%d", counter))
			if err != nil {
				return fmt.Errorf("failed to create datum node: %w", err)
			}
			if triple.Datum.Language != "" {
				object.SetLabel(fmt.Sprintf("%q@%s", triple.Datum.Value, triple.Datum.Language))
			} else {
				object.SetLabel(fmt.Sprintf("%q", triple.Datum.Value))
			}

			object.SetShape(cgraph.BoxShape)
		}

		{
			_, err := graph.CreateEdgeByName(string(triple.SPredicate), subject, object)
			if err != nil {
				return fmt.Errorf("failed to create edge: %w", err)
			}
		}
	}

	// and render it!
	if err := g.Render(ctx, graph, format, w); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}
	return nil
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
			return fmt.Errorf("failed to encode canonical triple: %w", err)
		}

		if err := writer.Encode(triple); err != nil {
			return fmt.Errorf("failed to encode triple: %w", err)
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
