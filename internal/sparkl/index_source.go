package sparkl

import (
	"fmt"
	"io"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/nquads"
)

// cspell:words triplestore

// Source represents a source of triples
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

// Token represents a token read from a triplestore file.
//
// It can represent one of three states:
//
// 1. an error token
// 1. a (subject, predicate, object) token
// 2. a (subject, predicate, datum) token
//
// In the case of 1, Error != nil.
// In the case of 2, Error == nil && HasDatum = False
// In the case of 3, Error == nil && HasDatum = True
type Token struct {
	Datum     impl.Datum
	Err       error
	Subject   impl.Label
	Predicate impl.Label
	Object    impl.Label
	Language  impl.Language
	HasDatum  bool
}

// QuadSource reads triples from a quad file
type QuadSource struct {
	Reader io.ReadSeeker
	reader *nquads.Reader
}

func (qs *QuadSource) Open() error {
	// if we previously had a reader
	// then we need to reset the state
	if qs.reader != nil {
		if err := qs.reader.Close(); err != nil {
			return err
		}
		_, err := qs.Reader.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
	}

	qs.reader = nquads.NewReader(qs.Reader, true)
	return nil
}

// Next reads the next token from the QuadSource
func (qs *QuadSource) Next() Token {
	for {
		value, err := qs.reader.ReadQuad()
		if err != nil {
			return Token{Err: err}
		}

		sI, sOK := asURILike(value.Subject)
		pI, pOK := asURILike(value.Predicate)
		if !(sOK && pOK) {
			continue
		}

		oI, oOK := asURILike(value.Object)
		if oOK {
			return Token{
				Subject:   sI,
				Predicate: pI,
				Object:    oI,
			}
		} else {
			var language impl.Language
			var datum impl.Datum

			// if this is a language string
			ldatum, ok := value.Object.(quad.LangString)
			if ok {
				language = impl.Language(ldatum.Lang)
				datum = impl.Datum(ldatum.Native().(string))
			} else {
				datum = impl.Datum(fmt.Sprint(value.Object.Native()))
			}

			return Token{
				Subject:   sI,
				Predicate: pI,
				HasDatum:  true,
				Datum:     datum,
				Language:  language,
			}
		}
	}
}

func (qs *QuadSource) Close() error {
	if qs.reader != nil {
		return qs.reader.Close()
	}
	return nil
}

func asURILike(value quad.Value) (uri impl.Label, ok bool) {
	switch datum := value.(type) {
	case quad.IRI:
		return impl.Label(string(datum)), true
	case quad.BNode:
		return impl.Label(string(datum)), true
	default:
		return "", false
	}
}
