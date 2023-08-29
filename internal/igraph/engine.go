package igraph

import (
	"io"

	"github.com/FAU-CDI/hangover/internal/imap"
)

// Engine represents an object that creates storages for an IGraph
type Engine interface {
	imap.Map

	Data() (imap.HashMap[imap.ID, imap.Datum], error)
	Triples() (imap.HashMap[imap.ID, IndexTriple], error)
	Inverses() (imap.HashMap[imap.ID, imap.ID], error)
	PSOIndex() (ThreeStorage, error)
	POSIndex() (ThreeStorage, error)
}

type ThreeStorage interface {
	io.Closer

	// Add adds a new mapping for the given (a, b, c).
	//
	// l acts as a label for the insert.
	// when the given edge already exists, the conflict function should be called to resolve the conflict
	Add(a, b, c imap.ID, l imap.ID, conflict func(old, new imap.ID) (imap.ID, error)) (conflicted bool, err error)

	// Count counts the overall number of entries in the index
	Count() (int64, error)

	// Compact indicates to the caller to perform internal optimizations of all data structures.
	Compact() error

	// Finalize informs the storage that no more mutable calls will be made.
	// A mutable call is one to Compact or Add.
	Finalize() error

	// Fetch iterates over all triples (a, b, c) in c-order.
	// l is the last label that was created for the triple.
	// If an error occurs, iteration stops and is returned to the caller
	Fetch(a, b imap.ID, f func(c imap.ID, l imap.ID) error) error

	// Has checks if the given mapping exists and returns the label (if any)
	Has(a, b, c imap.ID) (imap.ID, bool, error)
}
