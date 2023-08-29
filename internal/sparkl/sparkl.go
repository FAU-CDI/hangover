// Package sparkl implements a very primitive graph index
package sparkl

import (
	"strings"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

// cspell:words sparkl

// Predicates represent special predicates
type Predicates struct {
	SameAs    []impl.Label
	InverseOf []impl.Label
}

// ParsePredicateString parses a value of comma-separate value into a list of impl.Labels
func ParsePredicateString(target *[]impl.Label, value string) {
	if value == "" {
		*target = nil
		return
	}

	values := strings.Split(value, ",")
	*target = make([]impl.Label, len(values))
	for i, value := range values {
		(*target)[i] = impl.Label(value)
	}
}

// NewEngine creates an engine that stores data at the specified path.
// When path is the empty string, stores data in memory.
func NewEngine(path string) igraph.Engine {
	if path == "" {
		return &igraph.MemoryEngine{}
	}

	var de igraph.DiskEngine
	de.Path = path
	return de
}
