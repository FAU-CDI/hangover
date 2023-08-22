// Command n2r turns an nquads file into a json file
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/perf"
	"github.com/FAU-CDI/hangover/pkg/progress"
	"github.com/pkg/profile"
)

// cspell:words nquads pathbuilder

func main() {
	if debugProfile != "" {
		defer profile.Start(profile.ProfilePath(debugProfile)).Stop()
	}

	if mysql != "" && sqlite != "" {
		log.Print("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		log.Fatal("both -sqlite and -mysql were given")
	}

	// find the paths
	nqp, pbp, _, err := hangover.FindSource(false, nArgs...)
	if err != nil {
		log.Print("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		log.Fatal(err)
	}

	// read the pathbuilder
	var pb pathbuilder.Pathbuilder
	{
		start := perf.Now()
		pb, err = pbxml.Load(pbp)
		pbT := perf.Since(start)

		if err != nil {
			log.Fatalf("Unable to load Pathbuilder: %s", err)
		}
		log.Printf("loaded pathbuilder, took %s", pbT)
	}

	var predicates sparkl.Predicates
	sparkl.ParsePredicateString(&predicates.SameAs, sameAs)
	sparkl.ParsePredicateString(&predicates.InverseOf, inverseOf)

	// make an engine
	engine := sparkl.NewEngine(cache)
	bEngine := storages.NewBundleEngine(cache)

	if cache != "" {
		log.Printf("caching data on-disk at %s", cache)
	}

	// build an index
	var index *sparkl.Index
	{
		start := perf.Now()
		index, err = sparkl.LoadIndex(nqp, predicates, engine, sparkl.DefaultIndexOptions(&pb), &progress.Progress{
			Rewritable: progress.Rewritable{
				FlushInterval: progress.DefaultFlushInterval,
				Writer:        os.Stderr,
			},
		})
		indexT := perf.Since(start)

		if err != nil {
			log.Fatalf("Unable to build index: %s", err)
		}
		defer index.Close()

		log.Printf("built index, stats %s, took %s", index.Stats(), indexT)
	}

	switch {
	case mysql != "":
		doSQL(&pb, index, bEngine, "mysql", mysql)
	case sqlite != "":
		doSQL(&pb, index, bEngine, "sqlite", sqlite)
	default:
		doJSON(&pb, index, bEngine)
	}
}

// ===================

var nArgs []string
var cache string
var sameAs = string(wisski.SameAs)
var inverseOf = string(wisski.InverseOf)
var debugProfile = ""

var sqlite string
var mysql string

var sqlSeperator string = ","
var sqlFieldTables bool

func init() {
	var legalFlag bool = false
	flag.BoolVar(&legalFlag, "legal", legalFlag, "Display legal notices and exit")

	flag.StringVar(&sameAs, "sameas", sameAs, "SameAs Properties")
	flag.StringVar(&inverseOf, "inverseof", inverseOf, "InverseOf Properties")

	flag.StringVar(&cache, "cache", cache, "During indexing, cache data in the given directory as opposed to memory")
	flag.StringVar(&sqlite, "sqlite", sqlite, "Export an sqlite database to the given path")
	flag.StringVar(&sqlite, "mysql", mysql, "Export a mysql database. Use a connection string of the form `username:password@host/database`")

	flag.StringVar(&sqlSeperator, "sql-seperator", sqlSeperator, "Use seperator on multi-valued fields")
	flag.BoolVar(&sqlFieldTables, "sql-field-tables", sqlFieldTables, "Store values for fields in seperate tables")

	flag.StringVar(&debugProfile, "debug-profile", debugProfile, "write out a debugging profile to the given path")

	defer func() {
		if legalFlag {
			fmt.Print(hangover.LegalText())
			os.Exit(0)
		}
	}()

	flag.Parse()
	nArgs = flag.Args()
}
