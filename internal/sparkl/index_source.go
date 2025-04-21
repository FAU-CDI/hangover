//spellchecker:words sparkl
package sparkl

//spellchecker:words github hangover internal triplestore impl cayleygraph quad nquads
import (
	"fmt"
	"io"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/nquads"
)

//spellchecker:words triplestore

// Source represents a source of triples.
type Source interface {
	// Open opens this data source.
	//
	// It is valid to call open more than once after Next() returns a token with err = io.EOF.
	// In this case the second call to open should reset the data source.
	Open() error

	// Close closes this source.
	// Close may only be called once a token with err != io.EOF is called.
	Close() error

	// Next scans the next token
	Next() Token
}

// In the case of 3, Error == nil && HasDatum = True.
type Token struct {
	Datum     impl.Datum
	Err       error
	Subject   impl.Label
	Predicate impl.Label
	Object    impl.Label
	Source    impl.Source
	HasDatum  bool
}

// QuadSource reads triples from a quad file.
type QuadSource struct {
	Reader io.ReadSeeker
	reader *nquads.Reader
}

func (qs *QuadSource) Open() error {
	// if we previously had a reader
	// then we need to reset the state
	if qs.reader != nil {
		if err := qs.reader.Close(); err != nil {
			return fmt.Errorf("failed to close reader: %w", err)
		}
		_, err := qs.Reader.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to seek back to start: %w", err)
		}
	}

	qs.reader = nquads.NewReader(qs.Reader, true)
	return nil
}

// Next reads the next token from the QuadSource.
func (qs *QuadSource) Next() Token {
	for {
		value, err := qs.reader.ReadQuad()
		if err != nil {
			return Token{Err: err}
		}

		var source impl.Source
		source.Graph, _ = asLabel(value.Label)

		sI, sOK := asLabel(value.Subject)
		pI, pOK := asLabel(value.Predicate)
		if !sOK || !pOK {
			continue
		}

		oI, oOK := asLabel(value.Object)
		if oOK {
			return Token{
				Subject:   sI,
				Predicate: pI,
				Object:    oI,
				Source:    source,
			}
		} else {
			var datum impl.Datum

			// if this is a language string
			ldatum, ok := value.Object.(quad.LangString)
			if ok {
				datum.Value = ldatum.Native().(string)
				datum.Language = ldatum.Lang
			} else {
				datum.Value = fmt.Sprint(value.Object.Native())
			}

			return Token{
				Subject:   sI,
				Predicate: pI,
				HasDatum:  true,
				Datum:     datum,
				Source:    source,
			}
		}
	}
}

func (qs *QuadSource) Close() error {
	if qs.reader == nil {
		return nil
	}

	if err := qs.reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader: %w", err)
	}
	return nil
}

func asLabel(value quad.Value) (uri impl.Label, ok bool) {
	switch datum := value.(type) {
	case quad.IRI:
		return impl.Label(datum), true
	case quad.BNode:
		return impl.Label(datum), true
	default:
		return "", false
	}
}
