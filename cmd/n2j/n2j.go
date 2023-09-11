// Command n2r turns an nquads file into a json file
package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/status"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/pkg/profile"
)

// cspell:words nquads pathbuilder

var errBothSqliteAndMysql = errors.New("both -sqlite and -mysql were given")

func main() {
	// create a new status
	stats := status.NewStats(os.Stderr)

	if debugProfile != "" {
		defer profile.Start(profile.ProfilePath(debugProfile)).Stop()
	}

	if mysql != "" && sqlite != "" {
		stats.Log("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		stats.LogError("parse arguments", errBothSqliteAndMysql)
	}

	// find the paths
	nqp, pbp, _, err := hangover.FindSource(false, nArgs...)
	if err != nil {
		stats.Log("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		stats.LogFatal("find source", err)
	}

	// read the pathbuilder
	var pb pathbuilder.Pathbuilder
	err = stats.DoStage(status.StageReadPathbuilder, func() (err error) {
		pb, err = pbxml.Load(pbp)
		return
	})
	if err != nil {
		stats.LogFatal("pathbuilder load", err)
	}

	var predicates sparkl.Predicates
	sparkl.ParsePredicateString(&predicates.SameAs, sameAs)
	sparkl.ParsePredicateString(&predicates.InverseOf, inverseOf)

	// make an engine
	engine := sparkl.NewEngine(cache)
	bEngine := storages.NewBundleEngine(cache)

	if cache != "" {
		stats.Log("caching data on-disk", "path", cache)
	}

	// build an index
	var index *igraph.Index
	index, err = sparkl.LoadIndex(nqp, predicates, engine, sparkl.DefaultIndexOptions(&pb), stats)
	if err != nil {
		stats.LogFatal("unable to load index", err)
	}
	stats.Log("finished indexing", "stats", index.Stats())

	switch {
	case mysql != "":
		doSQL(&pb, index, bEngine, "mysql", mysql, stats)
	case sqlite != "":
		doSQL(&pb, index, bEngine, "sqlite", sqlite, stats)
	default:
		doJSON(&pb, index, bEngine, stats)
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
