package igraph

import (
	"errors"
	"fmt"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

// Stats holds statistics about triples in the index
type Stats struct {
	DirectTriples     uint64
	DatumTriples      uint64
	MaskedPredTriples uint64
	MaskedDataTriples uint64
	InverseTriples    uint64
	ConflictTriples   uint64
}

func (stats Stats) String() string {
	return fmt.Sprintf("{direct:%d,datum:%d,mask(pred):%d,mask(data):%d,inverse:%d,conflict:%d}", stats.DirectTriples, stats.DatumTriples, stats.MaskedPredTriples, stats.MaskedDataTriples, stats.InverseTriples, stats.ConflictTriples)
}

// IndexTriple represents a triple stored inside the index
type IndexTriple struct {
	Role  // Why was this triple stored?
	Items [3]imap.TripleID
}

func MarshalTriple(triple IndexTriple) ([]byte, error) {
	result := make([]byte, 6*imap.IDLen+1)
	imap.MarshalIDs(
		result[1:],
		triple.Items[0].Literal,
		triple.Items[1].Literal,
		triple.Items[2].Literal,
		triple.Items[0].Canonical,
		triple.Items[1].Canonical,
		triple.Items[2].Canonical,
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
		&(dest.Items[0].Literal),
		&(dest.Items[1].Literal),
		&(dest.Items[2].Literal),
		&(dest.Items[0].Canonical),
		&(dest.Items[1].Canonical),
		&(dest.Items[2].Canonical),
	)
	return nil
}

// Triple represents a triple found inside a graph
type Triple struct {
	// the literal SPO for this triple, as found in the original data.
	Subject   imap.Label
	Predicate imap.Label
	Object    imap.Label

	// the canonical (normalized for sameAs) for this triple.
	// FIXME: change this to say canonical in the name
	SSubject   imap.Label
	SPredicate imap.Label
	SObject    imap.Label
	Datum      imap.Datum

	// ID uniquely identifies this triple.
	// Two triples are identical iff their IDs are identical.
	ID imap.ID

	// Why was this triple inserted?
	Role Role
}

// Inferred returns if this triple has been inferred
func (triple Triple) Inferred() bool {
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
