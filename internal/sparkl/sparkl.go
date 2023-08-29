// Package sparkl implements a very primitive graph index
package sparkl

import (
	"strings"

	"github.com/FAU-CDI/hangover/internal/igraph"
	"github.com/FAU-CDI/hangover/internal/imap"
)

// cspell:words sparkl

// Predicates represent special predicates
type Predicates struct {
	SameAs    []imap.Label
	InverseOf []imap.Label
}

// ParsePredicateString parses a value of comma-separate value into a list of imap.Labels
func ParsePredicateString(target *[]imap.Label, value string) {
	if value == "" {
		*target = nil
		return
	}

	values := strings.Split(value, ",")
	*target = make([]imap.Label, len(values))
	for i, value := range values {
		(*target)[i] = imap.Label(value)
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
