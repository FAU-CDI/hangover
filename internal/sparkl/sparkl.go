// Package sparkl implements a very primitive graph index
package sparkl

import (
	"strings"

	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/igraph"
)

type (
	Entity = wisski.Entity
	URI    = wisski.URI

	BundleStorage = storages.BundleStorage
	BundleEngine  = storages.BundleEngine

	Engine       = igraph.Engine[URI, any]
	MemoryEngine = igraph.MemoryEngine[URI, any]
	DiskEngine   = igraph.DiskEngine[URI, any]

	Triple = igraph.Triple[URI, any] // Triple inside the index
	Index  = igraph.IGraph[URI, any] // Index represents an index of a RDF Graph
	Paths  = igraph.Paths[URI, any]  // Set of Paths inside the index
	Path   = igraph.Path[URI, any]   // Singel Path in the index
)

// Predicates represent special predicates
type Predicates struct {
	SameAs    []URI
	InverseOf []URI
}

// ParsePredicateString parses a value of comma-seperate value into a list of URIs
func ParsePredicateString(target *[]URI, value string) {
	if value == "" {
		*target = nil
		return
	}

	values := strings.Split(value, ",")
	*target = make([]URI, len(values))
	for i, value := range values {
		(*target)[i] = URI(value)
	}
}

// NewEngine creates an engine that stores data at the specified path.
// When path is the empty string, stores data in memory.
func NewEngine(path string) Engine {
	if path == "" {
		return &MemoryEngine{}
	}

	var de DiskEngine
	de.Path = path
	return de
}
