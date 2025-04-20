// Package sparkl provides facilities to generate an Index for a WissKI
package sparkl

import (
	"errors"
	"fmt"
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
func LoadIndex(path string, predicates Predicates, engine igraph.Engine, opts IndexOptions, st *stats.Stats) (idx *igraph.Index, e error) {
	reader, err := os.Open(path) // #nosec G304 -- explicit parameter
	if err != nil {
		return nil, fmt.Errorf("failed to open path: %w", err)
	}
	defer func() {
		if e2 := reader.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close reader: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(err, e2)
			}
		}
	}()

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
		return nil, fmt.Errorf("failed to reset index: %w", err)
	}

	closeIndex := func(err error) (*igraph.Index, error) {
		e2 := index.Close()
		if e2 == nil {
			return nil, err
		}

		e2 = fmt.Errorf("failed to close index: %w", e2)
		if err == nil {
			return nil, e2
		}

		return nil, errors.Join(err, e2)
	}

	// setup the mask for indexing the data
	if err := setMask(&index, opts.Mask); err != nil {
		return closeIndex(fmt.Errorf("failed to set mask: %w", err))
	}

	// read the "same as" triples first
	var totalCount, dataCount int
	err := st.DoStage(stats.StageScanSameAs, func() (err error) {
		totalCount, dataCount, err = indexSameAs(source, &index, predicates.SameAs, opts, st)
		return
	})
	if err != nil {
		return closeIndex(fmt.Errorf("failed to do %v stage: %w", stats.StageScanSameAs, err))
	}
	st.LogDebug("index count", "total", totalCount, "data", dataCount)

	// update stats
	st.StoreIndexStats(index.Stats())

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		return closeIndex(fmt.Errorf("failed to compact error: %w", err))
	}

	// read the "inverse" triples next
	err = st.DoStage(stats.StageScanInverse, func() error {
		return indexInverseOf(source, &index, predicates.InverseOf, totalCount, opts, st)
	})
	if err != nil {
		return closeIndex(fmt.Errorf("failed to do %v stage: %w", stats.StageScanInverse, err))
	}

	// update stats
	st.StoreIndexStats(index.Stats())

	// compact the index, or close on failure!
	if err := index.Compact(); err != nil {
		return closeIndex(fmt.Errorf("failed to compact index: %w", err))
	}

	// and then read all the other data
	err = st.DoStage(stats.StageScanTriples, func() error {
		return indexData(source, &index, totalCount, dataCount, opts, st)
	})
	if err != nil {
		return closeIndex(fmt.Errorf("failed to do %v stage: %w", stats.StageScanTriples, err))
	}

	if err := index.Compact(); err != nil {
		return closeIndex(fmt.Errorf("failed to compact index: %w", err))
	}

	// update stats
	st.StoreIndexStats(index.Stats())

	// and finalize the index
	if err := index.Finalize(); err != nil {
		return closeIndex(fmt.Errorf("failed to finalize index: %w", err))
	}

	return &index, nil
}

// set mask sets a mask while building the index, causing several triples to not be indexed at all.
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

// indexSameAs inserts SameAs pairs into the index.
func indexSameAs(source Source, index *igraph.Index, sameAsPredicates []impl.Label, opts IndexOptions, stats *stats.Stats) (allCount, dataCount int, err error) {
	err = source.Open()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open source: %w", err)
	}
	defer func() {
		if e2 := source.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close source: %w", e2)
			if err == nil {
				err = e2
			} else {
				err = errors.Join(err, e2)
			}
		}
	}()

	sameAss := make(map[impl.Label]struct{}, len(sameAsPredicates))
	for _, sameAs := range sameAsPredicates {
		sameAss[sameAs] = struct{}{}
	}

	for {
		tok := source.Next()

		if err := stats.SetCT(allCount, allCount); err != nil {
			return 0, 0, fmt.Errorf("failed to update stats: %w", err)
		}
		allCount++

		// check if we should compact
		if opts.shouldCompact(allCount) {
			if err := index.Compact(); err != nil {
				return 0, 0, fmt.Errorf("failed to compact index: %w", err)
			}
		}

		switch {
		case errors.Is(tok.Err, io.EOF):
			return allCount, dataCount, nil
		case tok.Err != nil:
			return 0, 0, tok.Err
		case !tok.HasDatum:
			if _, ok := sameAss[tok.Predicate]; ok {
				if err := index.MarkIdentical(tok.Subject, tok.Object); err != nil {
					return 0, 0, fmt.Errorf("failed to mark as identical: %w", err)
				}
			}
		default:
			dataCount++
		}
	}
}

// indexInverseOf inserts InverseOf pairs into the index.
func indexInverseOf(source Source, index *igraph.Index, inversePredicates []impl.Label, total int, opts IndexOptions, stats *stats.Stats) (e error) {
	if len(inversePredicates) == 0 {
		return nil
	}

	err := source.Open()
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer func() {
		if e2 := source.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close source: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	inverses := make(map[impl.Label]struct{})
	for _, inverse := range inversePredicates {
		inverses[inverse] = struct{}{}
	}

	var counter int
	for {
		tok := source.Next()

		counter++
		if err := stats.SetCT(counter, total); err != nil {
			return fmt.Errorf("failed to update total: %w", err)
		}

		// check if we should compact
		if opts.shouldCompact(counter) {
			if err := index.Compact(); err != nil {
				return err
			}
		}

		switch {
		case errors.Is(tok.Err, io.EOF):
			return nil
		case tok.Err != nil:
			return tok.Err
		case !tok.HasDatum:
			if _, ok := inverses[tok.Predicate]; ok {
				if err := index.MarkInverse(tok.Subject, tok.Object); err != nil {
					return fmt.Errorf("failed to mark inverse: %w", err)
				}
			}
		}
	}
}

var errIndexDataNegative = errors.New("indexData: dataCount < 0")

// indexData inserts data into the index.
func indexData(source Source, index *igraph.Index, totalCount, dataCount int, opts IndexOptions, stats *stats.Stats) (e error) {
	err := source.Open()
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer func() {
		if e2 := source.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close source: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	if dataCount < 0 {
		return errIndexDataNegative
	}
	if err := index.Grow(uint64(dataCount)); err != nil {
		return fmt.Errorf("failed to grow index: %w", err)
	}

	var counter int
	for {
		tok := source.Next()
		counter++
		if err := stats.SetCT(counter, totalCount); err != nil {
			return fmt.Errorf("failed to update stats: %w", err)
		}

		// check if we should compact
		if opts.shouldCompact(counter) {
			if err := index.Compact(); err != nil {
				return fmt.Errorf("failed to compact index: %w", err)
			}
		}

		switch {
		case errors.Is(tok.Err, io.EOF):
			return nil
		case tok.Err != nil:
			return tok.Err
		case tok.HasDatum:
			if err := index.AddData(tok.Subject, tok.Predicate, tok.Datum, tok.Source); err != nil {
				return fmt.Errorf("failed to add data triple: %w", err)
			}
		case !tok.HasDatum:
			if err := index.AddTriple(tok.Subject, tok.Predicate, tok.Object, tok.Source); err != nil {
				return fmt.Errorf("failed to add triple: %w", err)
			}
		}
	}
}
