package igraph

import (
	"errors"
	"fmt"
	"strings"

	"github.com/FAU-CDI/hangover/pkg/imap"
	"github.com/tkw1536/pkglib/iterator"
)

// cspell:words sparql twiesing

// Paths represents a set of paths in a related GraphIndex.
// It implements a very simple sparql-like query engine.
//
// A Paths object is stateful.
// A Paths object should only be created from a GraphIndex; the zero value is invalid.
// It can be further refined using the [Connected] and [Ending] methods.
type Paths struct {
	index      *Index
	predicates []imap.ID

	elements iterator.Iterator[element]
	size     int
}

// PathsStarting creates a new [PathSet] that represents all one-element paths
// starting at a vertex which is connected to object with the given predicate
func (index *Index) PathsStarting(predicate, object imap.Label) (*Paths, error) {
	p, err := index.labels.Forward(predicate)
	if err != nil {
		return nil, err
	}

	o, err := index.labels.Forward(object)
	if err != nil {
		return nil, err
	}

	return index.newQuery(func(sender iterator.Generator[element]) {
		err := index.posIndex.Fetch(p, o, func(s imap.ID, l imap.ID) error {
			if sender.Yield(element{
				Node:    s,
				Triples: []imap.ID{l},
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
func (index *Index) newQuery(source func(sender iterator.Generator[element])) (q *Paths) {
	q = &Paths{
		index:    index,
		elements: iterator.New(source),
		size:     -1,
	}
	return q
}

// Connected extends the sets of in this PathSet by those which
// continue the existing paths using an edge labeled with predicate.
func (set *Paths) Connected(predicate imap.Label) error {
	p, err := set.index.labels.Forward(predicate)
	if err != nil {
		return err
	}
	set.predicates = append(set.predicates, p)
	return set.expand(p)
}

var errAborted = errors.New("paths: aborted")

// expand expands the nodes in this query by adding a link to each element found in the index
func (set *Paths) expand(p imap.ID) error {
	set.elements = iterator.Connect(set.elements, func(subject element, sender iterator.Generator[element]) (stop bool) {
		err := set.index.psoIndex.Fetch(p, subject.Node, func(object imap.ID, l imap.ID) error {
			if sender.Yield(element{
				Node:    object,
				Triples: []imap.ID{l},
				Parent:  &subject,
			}) {
				return errAborted
			}
			return nil
		})

		if err != errAborted {
			sender.YieldError(err)
		}
		return err != nil && err != errAborted
	})
	set.size = -1
	return nil
}

// Ending restricts this set of paths to those that end in a node
// which is connected to object via predicate.
func (set *Paths) Ending(predicate, object imap.Label) error {
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
func (set *Paths) restrict(p, o imap.ID) error {
	set.elements = iterator.Connect(set.elements, func(subject element, sender iterator.Generator[element]) bool {
		tid, has, err := set.index.posIndex.Has(p, o, subject.Node)
		if err != nil {
			sender.YieldError(err)
			return true
		}
		if !has {
			return false
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
	if set.size != -1 {
		return set.size, nil
	}

	// we don't know the size, so we need to fully expand it
	all, err := iterator.Drain(set.elements)
	if err != nil {
		return 0, err
	}
	set.size = len(all)
	set.elements = iterator.Slice(all)
	return set.size, nil
}

// Paths returns an iterator over paths contained in this Paths.
// It may only be called once, afterwards further calls may be invalid.
func (set *Paths) Paths() iterator.Iterator[Path] {
	return iterator.New(func(generator iterator.Generator[Path]) {
		defer generator.Return()

		for set.elements.Next() {
			element := set.elements.Datum()
			path, err := set.makePath(element)
			if generator.YieldError(err) {
				return
			}
			if generator.Yield(path) {
				return
			}
		}

		generator.YieldError(set.elements.Err())
	})
}

// makePath creates a path from an element
func (set *Paths) makePath(elem element) (path Path, err error) {
	var rNodes, rTriples []imap.ID

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
	// node this path ends at
	Node imap.ID

	// triple this label had (if applicable)
	Triples []imap.ID

	// previous element of this path (if any)
	Parent *element
}

// Path represents a path inside a GraphIndex
type Path struct {
	nodes   []imap.Label
	edges   []imap.Label
	triples []Triple

	hasDatum bool
	datum    imap.Datum
}

// newPath creates a new path from the given index, with the given ids
// an "r" in front of the variable indicates it is passed in reverse order
func newPath(index *Index, rNodeIDs []imap.ID, edgeIDs []imap.ID, rTripleIDs []imap.ID) (path Path, err error) {
	// process nodes
	if len(rNodeIDs) != 0 {
		// split off the first value to use as a datum (if any)
		path.datum, path.hasDatum, err = index.data.Get(rNodeIDs[0])
		if err != nil {
			return Path{}, err
		}
		if path.hasDatum {
			rNodeIDs = rNodeIDs[1:]
		}

		// turn the nodes into a set of labels
		// reverse the passed nodes here!
		path.nodes = make([]imap.Label, len(rNodeIDs))
		last := len(rNodeIDs) - 1
		for j, label := range rNodeIDs {
			path.nodes[last-j], err = index.labels.Reverse(label)
			if err != nil {
				return Path{}, err
			}
		}
	}

	// process edges
	path.edges = make([]imap.Label, len(edgeIDs))
	last := len(path.edges) - 1
	for j, label := range edgeIDs {
		path.edges[last-j], err = index.labels.Reverse(label)
		if err != nil {
			return Path{}, err
		}
	}

	// process triples
	path.triples = make([]Triple, len(rTripleIDs))
	for j, label := range rTripleIDs {
		path.triples[j], err = index.Triple(label)
		if err != nil {
			return Path{}, err
		}
	}

	return

}

// Nodes returns the nodes this path consists of, in order.
func (path *Path) Nodes() ([]imap.Label, error) {
	return path.nodes, nil
}

var errOutOfBounds = errors.New("Path.Node: index out of bounds")

// Node returns the label of the node at the given index of path.
func (path *Path) Node(index int) (label imap.Label, err error) {
	if index < 0 || index > len(path.nodes) {
		return label, errOutOfBounds
	}
	return path.nodes[index], nil
}

// Datum returns the datum attached to the last node of this path, if any.
func (path *Path) Datum() (datum imap.Datum, ok bool, err error) {
	return path.datum, path.hasDatum, nil
}

// Edges returns the labels of the edges this path consists of.
func (path *Path) Edges() ([]imap.Label, error) {
	return path.edges, nil
}

// Triples returns the triples that this Path consists of.
// Triples are guaranteed to be returned in query order, that is in the order they were required for the query to be fulfilled.
func (path *Path) Triples() ([]Triple, error) {
	return path.triples, nil
}

// String turns this result into a string
//
// NOTE(twiesing): This is for debugging only, and ignores all errors.
// It should not be used in production code.
func (result *Path) String() string {
	var builder strings.Builder

	for i, edge := range result.edges {
		fmt.Fprintf(&builder, "%v %v ", result.nodes[i], edge)
	}

	if len(result.nodes) > 0 {
		fmt.Fprintf(&builder, "%v", result.nodes[len(result.nodes)-1])
	}
	if result.hasDatum {
		fmt.Fprintf(&builder, " %#v", result.datum)
	}
	return builder.String()
}
