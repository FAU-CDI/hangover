// Package glass provides Glass
package glass

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"os"
	"runtime/debug"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/progress"
	"github.com/FAU-CDI/hangover/pkg/sgob"
)

// cspell:words WissKI pathbuilder nquads

const GlassVersion = 2

// Glass represents a stand-alone representation of a WissKI
type Glass struct {
	Pathbuilder pathbuilder.Pathbuilder
	Cache       *sparkl.Cache
	Flags       viewer.RenderFlags
}

func (glass *Glass) Close() error {
	return glass.Cache.Close()
}

// EncodeTo encodes a glass to a given encoder
func (glass *Glass) EncodeTo(encoder *gob.Encoder) error {
	// encode the pathbuilder as xml
	pbxml, err := pbxml.Marshal(glass.Pathbuilder)
	if err != nil {
		return err
	}

	// encode all the fields
	for _, obj := range []any{
		GlassVersion,
		pbxml,
		glass.Flags,
	} {
		if err := sgob.Encode(encoder, obj); err != nil {
			return err
		}
	}

	// encode the payload
	return glass.Cache.EncodeTo(encoder)
}

// DecodeFrom decodes a glass from the given decoder
func (glass *Glass) DecodeFrom(decoder *gob.Decoder) (err error) {
	var version int
	var xml []byte
	for _, obj := range []any{
		&version,
		&xml,
		&glass.Flags,
	} {
		if err := sgob.Decode(decoder, obj); err != nil {
			return err
		}
	}

	// decode the xml again
	glass.Pathbuilder, err = pbxml.Unmarshal(xml)
	if err != nil {
		return err
	}

	if version != GlassVersion {
		return errInvalidVersion
	}

	glass.Cache = new(sparkl.Cache)
	return glass.Cache.DecodeFrom(decoder)
}

// Create creates a new glass from the given pathbuilder and nquads.
// output is written to output.
func Create(pathbuilderPath string, nquadsPath string, cacheDir string, flags viewer.RenderFlags, st *stats.Stats) (drincw Glass, err error) {
	// read the pathbuilder
	if err := st.DoStage(stats.StageReadPathbuilder, func() (err error) {
		drincw.Pathbuilder, err = pbxml.Load(pathbuilderPath)
		return err
	}); err != nil {
		return drincw, err
	}

	// make an engine
	engine := sparkl.NewEngine(cacheDir)
	bEngine := storages.NewBundleEngine(cacheDir)
	if cacheDir != "" {
		st.Log("caching data on-disk", "path", cacheDir)
	}

	// build an index
	index, err := sparkl.LoadIndex(nquadsPath, flags.Predicates, engine, sparkl.DefaultIndexOptions(&drincw.Pathbuilder), st)
	if err != nil {
		return drincw, err
	}

	st.Log("finished indexing", "stats", st.IndexStats())
	defer index.Close()

	// extract the bundles
	var bundles map[string][]wisski.Entity
	st.DoStage(stats.StageExtractBundles, func() (err error) {
		bundles, err = sparkl.LoadPathbuilder(&drincw.Pathbuilder, index, bEngine, st)
		return err
	})
	if err != nil {
		return drincw, err
	}

	// extract the cache

	identities := imap.MakeMemory[impl.Label, impl.Label](0)

	if err := st.DoStage(stats.StageExtractSameAs, func() error {
		return index.IdentityMap(&identities)
	}); err != nil {
		return Glass{}, err
	}

	if err := st.DoStage(stats.StageExtractCache, func() error {
		cache, err := sparkl.NewCache(bundles, &identities, st)
		if err != nil {
			return err
		}
		drincw.Cache = &cache
		return nil
	}); err != nil {
		return Glass{}, err
	}

	if err != nil {
		return drincw, err
	}

	index.Close()        // We close the index early, because it's no longer needed
	debug.FreeOSMemory() // force returning memory to the os

	drincw.Flags = flags
	return drincw, nil
}

// Export writes a glass to the given path
func Export(path string, drincw Glass, st *stats.Stats) (err error) {
	f, err := os.Create(path)
	if err != nil {
		st.LogError("create export", err)
		return err
	}
	defer f.Close()

	writer, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		st.LogError("create gzip writer", err)
		return err
	}
	defer writer.Flush()

	return st.DoStage(stats.StageExportIndex, func() error {
		counter := &progress.Writer{
			Writer:     writer,
			Rewritable: *st.Rewritable(),
		}

		err = drincw.EncodeTo(gob.NewEncoder(counter))
		counter.Flush(true)

		if err != nil {
			st.LogError("encode export", err)
		}
		return err
	})
}

var errInvalidVersion = errors.New("Glass Export: Invalid version")

// Import loads a glass from disk
func Import(path string, st *stats.Stats) (drincw Glass, err error) {
	defer debug.FreeOSMemory() // force clearing free memory

	f, err := os.Open(path)
	if err != nil {
		st.LogError("open export", err)
		return drincw, err
	}
	defer f.Close()

	reader, err := gzip.NewReader(f)
	if err != nil {
		st.LogError("open export", err)
		return drincw, err
	}

	err = st.DoStage(stats.StageImportIndex, func() error {
		counter := &progress.Reader{
			Reader:     reader,
			Rewritable: *st.Rewritable(),
		}
		err = drincw.DecodeFrom(gob.NewDecoder(counter))
		counter.Flush(true)
		if err != nil {
			st.LogError("decode export", err)
			return err
		}
		return nil
	})
	return drincw, err
}
