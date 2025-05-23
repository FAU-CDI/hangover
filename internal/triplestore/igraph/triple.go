//spellchecker:words igraph
package igraph

//spellchecker:words errors github hangover internal triplestore imap impl anglo korean
import (
	"errors"
	"fmt"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/anglo-korean/rdf"
)

// Stats holds statistics about triples in the index.
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

// IndexTriple represents a triple stored inside the index.
type IndexTriple struct {
	Role   // Why was this triple stored?
	Source impl.ID
	Items  [3]imap.TripleID
}

func MarshalTriple(triple IndexTriple) ([]byte, error) {
	result := make([]byte, 7*impl.IDLen+1)
	err := impl.MarshalIDs(
		result[1:],
		triple.Source,
		triple.Items[0].Literal,
		triple.Items[1].Literal,
		triple.Items[2].Literal,
		triple.Items[0].Canonical,
		triple.Items[1].Canonical,
		triple.Items[2].Canonical,
	)
	result[0] = byte(triple.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ids: %w", err)
	}
	return result, nil
}

var errDecodeTriple = errors.New("DecodeTriple: src too short")

func UnmarshalTriple(dest *IndexTriple, src []byte) error {
	if len(src) < 7*impl.IDLen+1 {
		return errDecodeTriple
	}
	dest.Role = Role(src[0])
	if err := impl.UnmarshalIDs(
		src[1:],
		&dest.Source,
		&(dest.Items[0].Literal),
		&(dest.Items[1].Literal),
		&(dest.Items[2].Literal),
		&(dest.Items[0].Canonical),
		&(dest.Items[1].Canonical),
		&(dest.Items[2].Canonical),
	); err != nil {
		return fmt.Errorf("failed to unmarshal ids: %w", err)
	}
	return nil
}

// Triple represents a triple found inside a graph.
type Triple struct {
	// the literal SPO for this triple, as found in the original data.
	Subject   impl.Label
	Predicate impl.Label
	Object    impl.Label

	// the canonical (normalized for sameAs) for this triple.
	// FIXME: change this to say canonical in the name
	SSubject   impl.Label
	SPredicate impl.Label
	SObject    impl.Label

	// Datum of this triple
	Datum  impl.Datum
	Source impl.Source

	// ID uniquely identifies this triple.
	// Two triples are identical iff their IDs are identical.
	ID impl.ID

	// Why was this triple inserted?
	Role Role
}

// Triple returns this Triple as an rdf triple.
func (triple Triple) Triple(canonical bool) (spo rdf.Triple, err error) {
	var subject, predicate string
	if !canonical {
		subject = string(triple.Subject)
		predicate = string(triple.Predicate)
	} else {
		subject = string(triple.SSubject)
		predicate = string(triple.SPredicate)
	}

	spo.Subj, err = rdf.NewIRI(subject)
	if err != nil {
		return rdf.Triple{}, fmt.Errorf("failed to create IRI for subject: %w", err)
	}

	spo.Pred, err = rdf.NewIRI(predicate)
	if err != nil {
		return rdf.Triple{}, fmt.Errorf("failed to create IRI for predicate: %w", err)
	}

	if triple.Role != Data {
		var object string
		if !canonical {
			object = string(triple.Object)
		} else {
			object = string(triple.SObject)
		}

		spo.Obj, err = rdf.NewIRI(object)
		if err != nil {
			return rdf.Triple{}, fmt.Errorf("failed to create IRI for object: %w", err)
		}
	} else {
		var err error

		if triple.Datum.Language != "" {
			spo.Obj, err = rdf.NewLangLiteral(triple.Datum.Value, triple.Datum.Language)
		} else {
			spo.Obj, err = rdf.NewLiteral(triple.Datum.Value)
		}
		if err != nil {
			return rdf.Triple{}, fmt.Errorf("failed to create literal: %w", err)
		}
	}

	return
}

// Compare compares this triple to another triple based on it's id.
func (triple Triple) Compare(other Triple) int {
	return triple.ID.Compare(other.ID)
}

// Inferred returns if this triple has been inferred.
func (triple Triple) Inferred() bool {
	return triple.Role == Inverse
}

// Role represents the role of the triple.
type Role uint8

const (
	// Regular represents a regular (non-inferred) triple.
	Regular Role = iota

	// Inverse represents an inferred inverse triple.
	Inverse

	// Data represents a data triple.
	Data
)
