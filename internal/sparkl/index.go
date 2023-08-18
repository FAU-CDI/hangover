// Package sparkl provides facilities to generate an Index for a WissKI
package sparkl

import (
	"io"
	"os"

	"github.com/FAU-CDI/hangover/pkg/progress"
)

// cspell:words nquads WissKI sparkl

// LoadIndex is like MakeIndex, but reads nquads from the given path.
// When err != nil, the caller must eventually close the index.
func LoadIndex(path string, predicates Predicates, engine Engine, opts IndexOptions, p *progress.Progress) (*Index, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return MakeIndex(&QuadSource{Reader: reader}, predicates, engine, opts, p)
}

func DefaultIndexOptions() IndexOptions {
	return IndexOptions{CompactInterval: 100_000}
}

type IndexOptions struct {
	CompactInterval int // Interval during which to call internal compact. Set <= 0 to disable.
}

func (io IndexOptions) shouldCompact(index int) bool {
	return io.CompactInterval > 0 && index > 0 && index%io.CompactInterval == 0
}

// MakeIndex creates a new Index from the given source.
// When err != nil, the caller must eventually close the index.
func MakeIndex(source Source, predicates Predicates, engine Engine, opts IndexOptions, p *progress.Progress) (*Index, error) {
	// create a new index
	var index Index
	if err := index.Reset(engine); err != nil {
		return nil, err
	}

	// read the "same as" triples first
	total, err := indexSameAs(source, &index, predicates.SameAs, opts, p)
	if err != nil {
		index.Close()
		return nil, err
	}

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	// read the "inverse" triples next
	if err := indexInverseOf(source, &index, predicates.InverseOf, total, opts, p); err != nil {
		index.Close()
		return nil, err
	}

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	// and then read all the other data
	if err := indexData(source, &index, total, opts, p); err != nil {
		index.Close()
		return nil, err
	}

	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	if p != nil {
		p.Close()
	}

	// and finalize the index
	if err := index.Finalize(); err != nil {
		index.Close()
		return nil, err
	}

	return &index, nil
}

// indexSameAs inserts SameAs pairs into the index
func indexSameAs(source Source, index *Index, sameAsPredicates []URI, opts IndexOptions, p *progress.Progress) (count int, err error) {
	err = source.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()

	sameAss := make(map[URI]struct{}, len(sameAsPredicates))
	for _, sameAs := range sameAsPredicates {
		sameAss[sameAs] = struct{}{}
	}

	for {
		tok := source.Next()
		if p != nil {
			count++
			p.Set("Scan 1/3: SameAs   ", count, count)
		}

		// check if we should compact
		if opts.shouldCompact(count) {
			if err := index.Compact(); err != nil {
				return 0, err
			}
		}

		switch {
		case tok.Err == io.EOF:
			return count, nil
		case tok.Err != nil:
			return 0, tok.Err
		case !tok.HasDatum:
			if _, ok := sameAss[tok.Predicate]; ok {
				index.MarkIdentical(tok.Subject, tok.Object)
			}
		}
	}
}

// indexInverseOf inserts InverseOf pairs into the index
func indexInverseOf(source Source, index *Index, inversePredicates []URI, total int, opts IndexOptions, p *progress.Progress) error {
	if len(inversePredicates) == 0 {
		return nil
	}

	err := source.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	inverses := make(map[URI]struct{})
	for _, inverse := range inversePredicates {
		inverses[inverse] = struct{}{}
	}

	var counter int
	for {
		tok := source.Next()
		if p != nil {
			counter++
			p.Set("Scan 2/3: InverseOf", counter, total)
		}

		// check if we should compact
		if opts.shouldCompact(counter) {
			if err := index.Compact(); err != nil {
				return err
			}
		}

		switch {
		case tok.Err == io.EOF:
			return nil
		case tok.Err != nil:
			return tok.Err
		case !tok.HasDatum:
			if _, ok := inverses[tok.Predicate]; ok {
				index.MarkInverse(tok.Subject, tok.Object)
			}
		}
	}
}

// indexData inserts data into the index
func indexData(source Source, index *Index, total int, opts IndexOptions, p *progress.Progress) error {
	err := source.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	var counter int
	for {
		tok := source.Next()
		if p != nil {
			counter++
			p.Set("Scan 3/3: Triples  ", counter, total)
		}

		// check if we should compact
		if opts.shouldCompact(counter) {
			if err := index.Compact(); err != nil {
				return err
			}
		}

		switch {
		case tok.Err == io.EOF:
			return nil
		case tok.Err != nil:
			return tok.Err
		case tok.HasDatum:
			index.AddData(tok.Subject, tok.Predicate, tok.Datum)
		case !tok.HasDatum:
			index.AddTriple(tok.Subject, tok.Predicate, tok.Object)
		}
	}
}
