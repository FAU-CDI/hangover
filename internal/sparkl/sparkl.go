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

// ParsePredicateString parses a value of comma-or-newline-separated value into a list of impl.Labels
// Empty values are ignored.
func ParsePredicateString(target *[]impl.Label, value string) {
	if value == "" {
		*target = nil
		return
	}

	var values []string

	csplit := strings.Split(value, ",")
	for _, c := range csplit {
		values = append(values, strings.Split(c, "\n")...)
	}

	*target = make([]impl.Label, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			*target = append(*target, impl.Label(value))
		}
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
