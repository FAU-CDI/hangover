package igraph

import (
	"errors"
	"fmt"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

// Stats holds statistics about triples in the index
type Stats struct {
	DirectTriples   int64
	DatumTriples    int64
	InverseTriples  int64
	ConflictTriples int64
}

func (stats Stats) String() string {
	return fmt.Sprintf("{direct:%d,datum:%d,inverse:%d,conflict:%d}", stats.DirectTriples, stats.DatumTriples, stats.InverseTriples, stats.ConflictTriples)
}

// IndexTriple represents a triple stored inside the index
type IndexTriple struct {
	Role              // Why was this triple stored?
	Items  [3]imap.ID // What were the *original* items in this triple
	SItems [3]imap.ID // What were the *semantic* items in this triple
}

func MarshalTriple(triple IndexTriple) ([]byte, error) {
	result := make([]byte, 6*imap.IDLen+1)
	imap.MarshalIDs(
		result[1:],
		triple.Items[0],
		triple.Items[1],
		triple.Items[2],
		triple.SItems[0],
		triple.SItems[1],
		triple.SItems[2],
	)
	result[0] = byte(triple.Role)
	return result, nil
}

var errDecodeTriple = errors.New("DecodeTriple: src too short")

func UnmarshalTriple(dest *IndexTriple, src []byte) error {
	if len(src) < 6*imap.IDLen+1 {
		return errDecodeTriple
	}
	dest.Role = Role(src[0])
	imap.UnmarshalIDs(
		src[1:],
		&(dest.Items[0]),
		&(dest.Items[1]),
		&(dest.Items[2]),
		&(dest.SItems[0]),
		&(dest.SItems[1]),
		&(dest.SItems[2]),
	)
	return nil
}

// Triple represents a triple found inside a graph
type Triple[Label comparable, Datum any] struct {
	// ID uniquely identifies this triple.
	// Two Triples are identical iff their IDs are identical.
	ID imap.ID

	Role Role

	Subject, Predicate, Object    Label
	SSubject, SPredicate, SObject Label // the "semantic" version of the datum

	Datum Datum
}

// Inferred returns if this triple has been inferred
func (triple Triple[Label, Datum]) Inferred() bool {
	return triple.Role == Inverse
}

// Role represents the role of the triple
type Role uint8

const (
	// Regular represents a regular (non-inferred) triple
	Regular Role = iota

	// Inverse represents an inferred inverse triple
	Inverse

	// Data represents a data triple
	Data
)
