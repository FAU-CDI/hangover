// Package sparkl provides facilities to generate an Index for a WissKI
package sparkl

import (
	"errors"
	"io"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// cspell:words nquads WissKI sparkl pathbuilder

// LoadIndex is like MakeIndex, but reads nquads from the given path.
// When err != nil, the caller must eventually close the index.
func LoadIndex(path string, predicates Predicates, engine igraph.Engine, opts IndexOptions, st *stats.Stats) (*igraph.Index, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return MakeIndex(&QuadSource{Reader: reader}, predicates, engine, opts, st)
}

func DefaultIndexOptions(pb *pathbuilder.Pathbuilder) IndexOptions {
	return IndexOptions{CompactInterval: 100_000, Mask: pb}
}

type IndexOptions struct {
	Mask            *pathbuilder.Pathbuilder // Pathbuilder to use as a mask when indexing
	CompactInterval int                      // Interval during which to call internal compact. Set <= 0 to disable.
}

func (io IndexOptions) shouldCompact(index int) bool {
	return io.CompactInterval > 0 && index > 0 && index%io.CompactInterval == 0
}

// MakeIndex creates a new Index from the given source.
// When err != nil, the caller must eventually close the index.
func MakeIndex(source Source, predicates Predicates, engine igraph.Engine, opts IndexOptions, st *stats.Stats) (*igraph.Index, error) {
	// create a new index
	var index igraph.Index
	if err := index.Reset(engine); err != nil {
		return nil, err
	}

	// setup the mask for indexing the data
	if err := setMask(&index, opts.Mask); err != nil {
		index.Close()
		return nil, err
	}

	// read the "same as" triples first
	var total int
	err := st.DoStage(stats.StageScanSameAs, func() (err error) {
		total, err = indexSameAs(source, &index, predicates.SameAs, opts, st)
		return
	})
	if err != nil {
		index.Close()
		return nil, err
	}

	// update stats
	st.StoreIndexStats(index.Stats())

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	// read the "inverse" triples next
	err = st.DoStage(stats.StageScanInverse, func() error {
		return indexInverseOf(source, &index, predicates.InverseOf, total, opts, st)
	})
	if err != nil {
		index.Close()
		return nil, err
	}

	// update stats
	st.StoreIndexStats(index.Stats())

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	// and then read all the other data
	err = st.DoStage(stats.StageScanTriples, func() error {
		return indexData(source, &index, total, opts, st)
	})
	if err != nil {
		index.Close()
		return nil, err
	}

	if err := index.Compact(); err != nil {
		index.Close()
		return nil, err
	}

	// update stats
	st.StoreIndexStats(index.Stats())

	// and finalize the index
	if err := index.Finalize(); err != nil {
		index.Close()
		return nil, err
	}

	return &index, nil
}

// set mask sets a mask while building the index, causing several triples to not be indexed at all
func setMask(index *igraph.Index, pb *pathbuilder.Pathbuilder) error {
	if pb == nil {
		return nil
	}

	dmask := make(map[impl.Label]struct{})

	pmask := make(map[impl.Label]struct{})
	pmask[wisski.Type] = struct{}{}

	for _, bundle := range pb.Bundles() {
		addBundleToMasks(pmask, dmask, bundle)
	}

	return errors.Join(
		index.SetPredicateMask(pmask),
		index.SetDataMask(dmask),
	)
}

func addBundleToMasks(pmask map[impl.Label]struct{}, dmask map[impl.Label]struct{}, bundle *pathbuilder.Bundle) {
	addPathArrayToMasks(pmask, bundle.PathArray)
	for _, field := range bundle.Fields() {
		addPathArrayToMasks(pmask, field.PathArray)
		dmask[impl.Label(field.DatatypeProperty)] = struct{}{}
	}
	for _, child := range bundle.ChildBundles {
		addBundleToMasks(pmask, dmask, child)
	}
}

func addPathArrayToMasks(pmask map[impl.Label]struct{}, ary []string) {
	for i, pth := range ary {
		if i%2 == 1 {
			pmask[impl.Label(pth)] = struct{}{}
		}
	}
}

// indexSameAs inserts SameAs pairs into the index
func indexSameAs(source Source, index *igraph.Index, sameAsPredicates []impl.Label, opts IndexOptions, stats *stats.Stats) (count int, err error) {
	err = source.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()

	sameAss := make(map[impl.Label]struct{}, len(sameAsPredicates))
	for _, sameAs := range sameAsPredicates {
		sameAss[sameAs] = struct{}{}
	}

	for {
		tok := source.Next()

		stats.SetCT(count, count)
		count++

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
func indexInverseOf(source Source, index *igraph.Index, inversePredicates []impl.Label, total int, opts IndexOptions, stats *stats.Stats) error {
	if len(inversePredicates) == 0 {
		return nil
	}

	err := source.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	inverses := make(map[impl.Label]struct{})
	for _, inverse := range inversePredicates {
		inverses[inverse] = struct{}{}
	}

	var counter int
	for {
		tok := source.Next()

		counter++
		stats.SetCT(counter, total)

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
func indexData(source Source, index *igraph.Index, total int, opts IndexOptions, stats *stats.Stats) error {
	err := source.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	var counter int
	for {
		tok := source.Next()
		counter++
		stats.SetCT(counter, total)

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
			index.AddLangData(tok.Subject, tok.Predicate, tok.Datum, tok.Language)
		case !tok.HasDatum:
			index.AddTriple(tok.Subject, tok.Predicate, tok.Object)
		}
	}
}
