// Package glass provides Glass
package glass

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"os"
	"runtime/debug"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/perf"
	"github.com/FAU-CDI/hangover/pkg/progress"
	"github.com/FAU-CDI/hangover/pkg/sgob"
)

// cspell:words WissKI pathbuilder nquads

const GlassVersion = 1

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
func Create(pathbuilderPath string, nquadsPath string, cacheDir string, flags viewer.RenderFlags, output io.Writer) (drincw Glass, err error) {
	log := log.New(output, "", log.LstdFlags)

	// read the pathbuilder
	var pbPerf perf.Diff
	{
		start := perf.Now()
		drincw.Pathbuilder, err = pbxml.Load(pathbuilderPath)
		pbPerf = perf.Since(start)

		if err != nil {
			log.Fatalf("Unable to load Pathbuilder: %s", err)
			return drincw, err
		}
		log.Printf("loaded pathbuilder, took %s", pbPerf)
	}

	// make an engine
	engine := sparkl.NewEngine(cacheDir)
	bEngine := storages.NewBundleEngine(cacheDir)
	if cacheDir != "" {
		log.Printf("caching data on-disk at %s", cacheDir)
	}

	// build an index
	var index *igraph.Index
	var indexPerf perf.Diff
	{
		start := perf.Now()
		index, err = sparkl.LoadIndex(nquadsPath, flags.Predicates, engine, sparkl.DefaultIndexOptions(&drincw.Pathbuilder), &progress.Progress{
			Rewritable: progress.Rewritable{
				FlushInterval: progress.DefaultFlushInterval,
				Writer:        output,
			},
		})
		indexPerf = perf.Since(start)

		if err != nil {
			log.Fatalf("Unable to build index: %s", err)
			return drincw, err
		}
		defer index.Close()

		log.Printf("built index, stats %s, took %s", index.Stats(), indexPerf)
	}

	// generate bundles
	var bundles map[string][]wisski.Entity
	var bundlesPerf perf.Diff
	{
		start := perf.Now()
		bundles, err = sparkl.LoadPathbuilder(&drincw.Pathbuilder, index, bEngine)
		if err != nil {
			log.Fatalf("Unable to load pathbuilder: %s", err)
		}
		bundlesPerf = perf.Since(start)
		log.Printf("extracted bundles, took %s", bundlesPerf)
	}

	// generate cache
	var cachePerf perf.Diff
	{
		start := perf.Now()

		identities := imap.MakeMemory[impl.Label, impl.Label](0)
		index.IdentityMap(&identities)

		cache, err := sparkl.NewCache(bundles, &identities)
		if err != nil {
			log.Fatalf("unable to build cache: %s", err)
		}
		drincw.Cache = &cache

		cachePerf = perf.Since(start)
		log.Printf("built cache, took %s", cachePerf)
	}

	index.Close()        // We close the index early, because it's no longer needed
	debug.FreeOSMemory() // force returning memory to the os

	drincw.Flags = flags
	return drincw, nil
}

// Export writes a glass to the given path
func Export(path string, drincw Glass, output io.Writer) (err error) {
	log := log.New(output, "", log.LstdFlags)

	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to create export: %s", err)
		return err
	}
	defer f.Close()

	writer, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		log.Fatalf("Unable to create export: %s", err)
		return err
	}
	defer writer.Flush()

	{
		start := perf.Now()

		counter := &progress.Writer{
			Writer: writer,

			Rewritable: progress.Rewritable{
				FlushInterval: progress.DefaultFlushInterval,
				Writer:        output,
			},
		}
		err = drincw.EncodeTo(gob.NewEncoder(counter))
		counter.Flush(true)
		os.Stderr.WriteString("\r")
		if err != nil {
			log.Fatalf("Unable to encode export: %s", err)
		}
		log.Printf("wrote export, took %s", perf.Since(start).SetBytes(counter.Bytes))
	}

	return err
}

var errInvalidVersion = errors.New("Glass Export: Invalid version")

// Import loads a glass from disk
func Import(path string, output io.Writer) (drincw Glass, err error) {
	log := log.New(output, "", log.LstdFlags)

	defer debug.FreeOSMemory() // force clearing free memory

	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Unable to open export: %s", err)
		return
	}
	defer f.Close()

	reader, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalf("Unable to open export: %s", err)
		return
	}

	{
		start := perf.Now()

		counter := &progress.Reader{
			Reader: reader,

			Rewritable: progress.Rewritable{
				FlushInterval: progress.DefaultFlushInterval,
				Writer:        os.Stderr,
			},
		}
		err = drincw.DecodeFrom(gob.NewDecoder(counter))
		counter.Flush(true)
		os.Stderr.WriteString("\r")
		if err != nil {
			log.Fatalf("Unable to decode export: %s", err)
		}
		log.Printf("read export, took %s", perf.Since(start).SetBytes(counter.Bytes))
	}

	return
}
