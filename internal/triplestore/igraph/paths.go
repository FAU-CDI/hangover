package igraph

import (
	"errors"
	"fmt"
	"strings"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/tkw1536/pkglib/traversal"
)

// cspell:words sparql twiesing

const invalidSize = -1

// Paths represents a set of paths in a related GraphIndex.
// It implements a very simple sparql-like query engine.
//
// A Paths object is stateful.
// A Paths object should only be created from a GraphIndex; the zero value is invalid.
// It can be further refined using the [Connected] and [Ending] methods.
type Paths struct {
	elements   traversal.Iterator[element]
	index      *Index
	predicates []impl.ID
	size       int // if known, otherwise invalidSize
}

// PathsStarting creates a new [PathSet] that represents all one-element paths
// starting at a vertex which is connected to object with the given predicate
func (index *Index) PathsStarting(predicate, object impl.Label) (*Paths, error) {
	p, err := index.labels.Forward(predicate)
	if err != nil {
		return nil, err
	}

	o, err := index.labels.Forward(object)
	if err != nil {
		return nil, err
	}

	return index.newQuery(func(sender traversal.Generator[element]) {
		err := index.posIndex.Fetch(p, o, func(s impl.ID, l impl.ID) error {
			if !sender.Yield(element{
				Node:    s,
				Triples: []impl.ID{l},
				Parent:  nil,
			}) {
				return errAborted
			}
			return nil
		})

		if err != errAborted {
			sender.YieldError(err)
		}
	}), nil
}

// newQuery creates a new Query object that contains nodes with the given ids
func (index *Index) newQuery(source func(sender traversal.Generator[element])) (q *Paths) {
	q = &Paths{
		index:    index,
		elements: traversal.New(source),
		size:     invalidSize,
	}
	return q
}

// Connected extends the sets of in this PathSet by those which
// continue the existing paths using an edge labeled with predicate.
func (set *Paths) Connected(predicate impl.Label) error {
	p, err := set.index.labels.Forward(predicate)
	if err != nil {
		return err
	}
	set.predicates = append(set.predicates, p)
	return set.expand(p)
}

var errAborted = errors.New("paths: aborted")

// expand expands the nodes in this query by adding a link to each element found in the index
func (set *Paths) expand(p impl.ID) error {
	set.elements = traversal.Connect(set.elements, func(subject element, sender traversal.Generator[element]) (ok bool) {
		err := set.index.psoIndex.Fetch(p, subject.Node, func(object impl.ID, l impl.ID) error {
			if !sender.Yield(element{
				Node:    object,
				Triples: []impl.ID{l},
				Parent:  &subject,
			}) {
				return errAborted
			}
			return nil
		})

		// if we have a "real" error, yield it and stop!
		if err != nil && err != errAborted {
			return sender.YieldError(err)
		}
		return true
	})
	set.size = -1
	return nil
}

// Ending restricts this set of paths to those that end in a node
// which is connected to object via predicate.
func (set *Paths) Ending(predicate, object impl.Label) error {
	p, err := set.index.labels.Forward(predicate)
	if err != nil {
		return err
	}
	o, err := set.index.labels.Forward(object)
	if err != nil {
		return err
	}
	return set.restrict(p, o)
}

// restrict restricts the set of nodes by those mapped in the index
func (set *Paths) restrict(p, o impl.ID) error {
	set.elements = traversal.Connect(set.elements, func(subject element, sender traversal.Generator[element]) bool {
		tid, has, err := set.index.posIndex.Has(p, o, subject.Node)
		if err != nil {
			return sender.YieldError(err)
		}
		if !has {
			return true
		}

		subject.Triples = append(subject.Triples, tid)
		return sender.Yield(subject)
	})
	set.size = -1
	return nil
}

// Size returns the number of elements in this path.
//
// NOTE(twiesing): This potentially takes a lot of memory, because we need to expand the stream.
func (set *Paths) Size() (int, error) {
	if set.size != invalidSize {
		return set.size, nil
	}

	// we don't know the size, so we need to fully expand it
	all, err := traversal.Drain(set.elements)
	if err != nil {
		return 0, err
	}
	set.size = len(all)
	set.elements = traversal.Slice(all)
	return set.size, nil
}

// Paths returns an iterator over paths contained in this Paths.
// It may only be called once, afterwards further calls may be invalid.
func (set *Paths) Paths() traversal.Iterator[Path] {
	return traversal.New(func(generator traversal.Generator[Path]) {
		defer generator.Return()

		for set.elements.Next() {
			element := set.elements.Datum()
			path, err := set.makePath(element)
			if !generator.YieldError(err) {
				return
			}
			if !generator.Yield(path) {
				return
			}
		}

		generator.YieldError(set.elements.Err())
	})
}

// makePath creates a path from an element
func (set *Paths) makePath(elem element) (path Path, err error) {
	var rNodes, rTriples []impl.ID

	e := &elem
	for {
		rNodes = append(rNodes, e.Node)
		rTriples = append(rTriples, e.Triples...)
		e = e.Parent
		if e == nil {
			break
		}
	}

	// make a new path
	return newPath(
		set.index,
		rNodes,
		set.predicates,
		rTriples,
	)
}

// element represents an element of a path
type element struct {
	Parent  *element
	Triples []impl.ID
	Node    impl.ID
}

// Path represents a path inside a GraphIndex
type Path struct {
	Datum    impl.Datum
	Language impl.Language
	Nodes    []impl.Label
	Edges    []impl.Label
	Triples  []Triple
	HasDatum bool
}

// newPath creates a new path from the given index, with the given ids
// an "r" in front of the variable indicates it is passed in reverse order
func newPath(index *Index, rNodeIDs []impl.ID, edgeIDs []impl.ID, rTripleIDs []impl.ID) (path Path, err error) {
	// process nodes
	if len(rNodeIDs) != 0 {
		// split off the first value to use as a datum (if any)
		path.Datum, path.HasDatum, err = index.data.Get(rNodeIDs[0])
		if err != nil {
			return Path{}, err
		}

		// if we have a datum, get the language and the node id!
		if path.HasDatum {
			path.Language, err = index.language.GetZero(rNodeIDs[0])
			if err != nil {
				return Path{}, err
			}
			rNodeIDs = rNodeIDs[1:]
		}

		// turn the nodes into a set of labels
		// reverse the passed nodes here!
		path.Nodes = make([]impl.Label, len(rNodeIDs))
		last := len(rNodeIDs) - 1
		for j, label := range rNodeIDs {
			path.Nodes[last-j], err = index.labels.Reverse(label)
			if err != nil {
				return Path{}, err
			}
		}
	}

	// process edges
	path.Edges = make([]impl.Label, len(edgeIDs))
	for j, label := range edgeIDs {
		path.Edges[j], err = index.labels.Reverse(label)
		if err != nil {
			return Path{}, err
		}
	}

	// process triples
	path.Triples = make([]Triple, len(rTripleIDs))
	last := len(rTripleIDs) - 1
	for j, label := range rTripleIDs {
		path.Triples[last-j], err = index.Triple(label)
		if err != nil {
			return Path{}, err
		}
	}

	return

}

// Value returns the value corresponding to a field represented by this path.
//
// The value returned tries the following options in order:
//
// - the datum corresponding to the path
// - the last node, interpreted as a datum
// - the zero value of the datum type
func (path Path) Value() (value impl.Datum) {
	if path.HasDatum {
		return path.Datum
	}

	if len(path.Nodes) > 0 {
		return impl.Datum(path.Nodes[len(path.Nodes)-1])
	}

	return
}

// String turns this result into a string
//
// NOTE(twiesing): This is for debugging only, and ignores all errors.
// It should not be used in production code.
func (path Path) String() string {
	var builder strings.Builder

	for i, edge := range path.Edges {
		fmt.Fprintf(&builder, "%v %v ", path.Nodes[i], edge)
	}

	if len(path.Nodes) > 0 {
		fmt.Fprintf(&builder, "%v", path.Nodes[len(path.Nodes)-1])
	}
	if path.HasDatum {
		fmt.Fprintf(&builder, " %#v", path.Datum)
	}
	return builder.String()
}
