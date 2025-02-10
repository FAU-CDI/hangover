package wisski2

import (
	"context"
	"slices"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
)

const (
	Type impl.Label = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" // the "Type" Predicate
)

// Value holds information for a single instantiated path inside WissKI.
type Value struct {
	// Path and Pathbuilder hold information about the path
	Path        *pathbuilder.Path
	Pathbuilder *pathbuilder.Pathbuilder

	// Ordered nodes that make up this path.
	// Guaranteed to be of the same length as the concepts in the path array
	Nodes []impl.Label

	// Datum held by this value.
	// If an only if there is a datatype property.
	Datum impl.Datum

	// Language of the datum (if applicable).
	Language impl.Language

	// Triples that are of relevance to this path.
	// No guarantee on order, may include sameAs relationships.
	Triples []igraph.Triple
}

// Values extracts all values for the given path from the index.
// The resulting channel is closed once no more values are available, or once the context is closed.
// Once value is closed, the error will be returned.
func Values(context context.Context, index *igraph.Index, path *pathbuilder.Path, pathbuilder *pathbuilder.Pathbuilder) (<-chan Value, <-chan error) {
	valChan := make(chan Value)    // for returning values
	errChan := make(chan error, 1) // for returning errors

	go func() {
		defer close(errChan)
		defer close(valChan)

		// Generate an alternating "path array".
		// It contains an alternating list of concepts and predicates to query.
		pathArray := slices.Clone(path.PathArray)
		if len(pathArray) == 0 { // nothing to query
			return
		}
		if datatype := path.Datatype(); !path.IsGroup && datatype != "" {
			pathArray = append(pathArray, datatype)
		}

		// Start building a query, starting at the first concept
		query, err := index.PathsStarting(Type, impl.Label(pathArray[0]))
		if err != nil {
			errChan <- err
			return
		}

		// iteratively update
		for i := 1; i < len(pathArray); i++ {
			if i%2 == 0 { // concept
				if err := query.Ending(Type, impl.Label(pathArray[i])); err != nil {
					errChan <- err
					return
				}
			} else { // predicate
				if err := query.Connected(impl.Label(pathArray[i])); err != nil {
					errChan <- err
					return
				}
			}
		}

		// create an actual iterator for the result
		result := query.Paths()
		defer result.Close()

		// iterate over the result and return it to the caller.
		for result.Next() {

			// compute the current instance of the path we got returned.
			// this is quick for a single instance.
			instance := result.Datum()
			value := Value{
				Path:        path,
				Pathbuilder: pathbuilder,

				Nodes:    instance.Nodes,
				Datum:    instance.Datum,
				Language: instance.Language,

				Triples: instance.Triples,
			}

			// if the context is closed, don't bother sending back a value.
			select {
			case <-context.Done():
				errChan <- err
				return
			default:
			}

			// send or bail out!
			select {
			case valChan <- value:
			case <-context.Done():
				errChan <- err
				return
			}
		}

		// check if the result had an error
		if err := result.Err(); err != nil {
			errChan <- err
			return
		}
	}()

	return valChan, errChan
}

// EntityURI returns the URI of the entity this value is associated to.
func (value Value) EntityURI() impl.Label {
	panic("not implemented")
}
