package igraph

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/FAU-CDI/hangover/pkg/imap"
	"github.com/tkw1536/pkglib/iterator"
)

// Paths represents a set of paths in a related GraphIndex.
// It implements a very simple sparql-like query engine.
//
// A Paths object is stateful.
// A Paths object should only be created from a GraphIndex; the zero value is invalid.
// It can be further refined using the [Connected] and [Ending] methods.
type Paths[Label comparable, Datum any] struct {
	index      *IGraph[Label, Datum]
	predicates []imap.ID

	elements iterator.Iterator[element]
	size     int
}

// PathsStarting creates a new [PathSet] that represents all one-element paths
// starting at a vertex which is connected to object with the given predicate
func (index *IGraph[Label, Datum]) PathsStarting(predicate, object Label) (*Paths[Label, Datum], error) {
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
func (index *IGraph[URI, Datum]) newQuery(source func(sender iterator.Generator[element])) (q *Paths[URI, Datum]) {
	q = &Paths[URI, Datum]{
		index:    index,
		elements: iterator.New(source),
		size:     -1,
	}
	return q
}

// Connected extends the sets of in this PathSet by those which
// continue the existing paths using an edge labeled with predicate.
func (set *Paths[Label, Datum]) Connected(predicate Label) error {
	p, err := set.index.labels.Forward(predicate)
	if err != nil {
		return err
	}
	set.predicates = append(set.predicates, p)
	return set.expand(p)
}

var errAborted = errors.New("paths: aborted")

// expand expands the nodes in this query by adding a link to each element found in the index
func (set *Paths[URI, Datum]) expand(p imap.ID) error {
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
func (set *Paths[URI, Datum]) Ending(predicate URI, object URI) error {
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
func (set *Paths[URI, Datum]) restrict(p, o imap.ID) error {
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
func (set *Paths[Label, Datum]) Size() (int, error) {
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
func (set *Paths[Label, Datum]) Paths() iterator.Iterator[Path[Label, Datum]] {
	return iterator.Map(set.elements, set.makePath)
}

// makePath creates a path from an element
func (set *Paths[Label, Datum]) makePath(elem element) (path Path[Label, Datum]) {
	path.index = set.index
	path.edgeIDs = set.predicates

	// insert nodes and triples
	e := &elem
	for {
		path.nodeIDs = append(path.nodeIDs, e.Node)
		path.tripleIDs = append(path.tripleIDs, e.Triples...)
		e = e.Parent
		if e == nil {
			break
		}
	}

	// reverse the triples and nodes
	for i := len(path.nodeIDs)/2 - 1; i >= 0; i-- {
		opp := len(path.nodeIDs) - 1 - i
		path.nodeIDs[i], path.nodeIDs[opp] = path.nodeIDs[opp], path.nodeIDs[i]
	}

	for i := len(path.tripleIDs)/2 - 1; i >= 0; i-- {
		opp := len(path.tripleIDs) - 1 - i
		path.tripleIDs[i], path.tripleIDs[opp] = path.tripleIDs[opp], path.tripleIDs[i]
	}
	return path
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
type Path[Label comparable, Datum any] struct {
	// index is the index this Path belonges to
	index *IGraph[Label, Datum]

	errNodes  error
	nodeIDs   []imap.ID
	nodesOnce sync.Once
	nodes     []Label
	hasDatum  bool
	datum     Datum

	errEdges  error
	edgeIDs   []imap.ID
	edgesOnce sync.Once
	edges     []Label

	errTriples  error
	tripleIDs   []imap.ID
	triplesOnce sync.Once
	triples     []Triple[Label, Datum]
}

// Nodes returns the nodes this path consists of, in order.
func (path *Path[Label, Datum]) Nodes() ([]Label, error) {
	path.processNodes()
	return path.nodes, path.errNodes
}

// Node returns the label of the node at the given index of path.
func (path *Path[Label, Datum]) Node(index int) (Label, error) {
	switch {
	case len(path.nodes) > index:
		// already computed!
		return path.nodes[index], nil
	case index >= len(path.nodeIDs):
		// path does not exist
		var label Label
		return label, nil
	case index == len(path.nodeIDs)-1:
		// check if the last element has data associated with it
		last := path.nodeIDs[len(path.nodeIDs)-1]
		has, err := path.index.data.Has(last)
		if has || err != nil {
			var label Label
			return label, err
		}
		fallthrough
	default:
		// return the index
		return path.index.labels.Reverse(path.nodeIDs[index])
	}
}

// Datum returns the datum attached to the last node of this path, if any.
func (path *Path[Label, Datum]) Datum() (datum Datum, ok bool, err error) {
	path.processNodes()
	return path.datum, path.hasDatum, path.errNodes
}

func (path *Path[Label, Datum]) processNodes() {
	path.nodesOnce.Do(func() {
		if len(path.nodeIDs) == 0 {
			return
		}

		// split off the last value as a datum (if any)
		last := path.nodeIDs[len(path.nodeIDs)-1]
		path.datum, path.hasDatum, path.errNodes = path.index.data.Get(last)
		if path.errNodes != nil {
			return
		}
		if path.hasDatum {
			path.nodeIDs = path.nodeIDs[:len(path.nodeIDs)-1]
		}

		// turn the nodes into a set of labels
		path.nodes = make([]Label, len(path.nodeIDs))
		for j, label := range path.nodeIDs {
			path.nodes[j], path.errNodes = path.index.labels.Reverse(label)
			if path.errNodes != nil {
				return
			}
		}
	})
}

// Edges returns the labels of the edges this path consists of.
func (path *Path[Label, Datum]) Edges() ([]Label, error) {
	path.processEdges()
	return path.edges, path.errEdges
}

func (path *Path[Label, Datum]) processEdges() {
	path.edgesOnce.Do(func() {
		path.edges = make([]Label, len(path.edgeIDs))
		for j, label := range path.edgeIDs {
			path.edges[j], path.errEdges = path.index.labels.Reverse(label)
			if path.errEdges != nil {
				return
			}
		}
	})
}

// Triples returns the triples that this Path consists of.
// Triples are guaranteed to be returned in query order, that is in the order they were required for the query to be fullfilled.
func (path *Path[Label, Datum]) Triples() ([]Triple[Label, Datum], error) {
	path.processTriples()
	return path.triples, path.errTriples
}

func (path *Path[Label, Datum]) processTriples() {
	path.triplesOnce.Do(func() {
		path.triples = make([]Triple[Label, Datum], len(path.tripleIDs))
		for j, label := range path.tripleIDs {
			path.triples[j], path.errTriples = path.index.Triple(label)
			if path.errEdges != nil {
				return
			}
		}
	})
}

// String turns this result into a string
//
// NOTE(twiesing): This is for debugging only, and ignores all errors.
// It should not be used in producion code.
func (result *Path[URI, Datum]) String() string {
	var builder strings.Builder

	result.processNodes()
	result.processEdges()

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
