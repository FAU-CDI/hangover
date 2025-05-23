// Command n2r turns an nquads file into a json file
//
//spellchecker:words main
package main

//spellchecker:words embed errors flag github drincw pathbuilder pbxml hangover internal sparkl storages stats triplestore igraph wisski profile
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
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/pkg/profile"
)

//spellchecker:words nquads pathbuilder

var errBothSqliteAndMysql = errors.New("both -sqlite and -mysql were given")

func main() {
	// create a new status
	st := stats.NewStats(os.Stderr, debug)

	if debugProfile != "" {
		defer profile.Start(profile.ProfilePath(debugProfile)).Stop()
	}

	var selected int
	if mysql != "" {
		selected++
	}
	if sqlite != "" {
		selected++
	}
	if csvPath != "" {
		selected++
	}

	if selected > 1 {
		st.Log("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		st.LogError("parse arguments", errBothSqliteAndMysql)
	}

	// find the paths
	nqp, pbp, err := hangover.FindSource(nArgs...)
	if err != nil {
		st.Log("Usage: n2j [-help] [...flags] /path/to/pathbuilder /path/to/nquads")
		st.LogFatal("find source", err)
	}

	// read the pathbuilder
	var pb pathbuilder.Pathbuilder
	err = st.DoStage(stats.StageReadPathbuilder, func() (err error) {
		pb, err = pbxml.Load(pbp)
		return
	})
	if err != nil {
		st.LogFatal("pathbuilder load", err)
	}

	var predicates sparkl.Predicates
	predicates.SameAs = sparkl.ParsePredicateString(sameAs)
	predicates.InverseOf = sparkl.ParsePredicateString(inverseOf)

	// make an engine
	engine := sparkl.NewEngine(cache)
	bEngine := storages.NewBundleEngine(cache)

	if cache != "" {
		st.Log("caching data on-disk", "path", cache)
	}

	// build an index
	var index *igraph.Index
	index, err = sparkl.LoadIndex(nqp, predicates, engine, sparkl.DefaultIndexOptions(&pb), st)
	if err != nil {
		st.LogFatal("unable to load index", err)
	}
	st.Log("finished indexing", "stats", index.Stats())

	{
		var err error
		switch {
		case mysql != "":
			_, err = doSQL(&pb, index, bEngine, "mysql", mysql, false, st)
		case sqlite != "":
			_, err = doSQL(&pb, index, bEngine, "sqlite", sqlite, false, st)
		case csvPath != "":
			err = doCSV(&pb, index, bEngine, csvPath, st)
		default:
			err = doJSON(&pb, index, bEngine, st)
		}

		if err != nil {
			st.LogFatal("failed to export", err)
		}
	}
}

// ===================

var nArgs []string
var cache string
var sameAs = string(wisski.DefaultSameAsProperties)
var inverseOf = string(wisski.InverseOf)
var debugProfile = ""

var sqlite string
var csvPath string
var mysql string

var debug bool

var sqlSeparator string = ","
var sqlFieldTables bool

func init() {
	var legalFlag = false
	flag.BoolVar(&legalFlag, "legal", legalFlag, "Display legal notices and exit")

	flag.StringVar(&sameAs, "sameas", sameAs, "SameAs Properties")
	flag.StringVar(&inverseOf, "inverseof", inverseOf, "InverseOf Properties")

	flag.StringVar(&cache, "cache", cache, "During indexing, cache data in the given directory as opposed to memory")
	flag.StringVar(&sqlite, "sqlite", sqlite, "Export an sqlite database to the given path")
	flag.StringVar(&csvPath, "csv", csvPath, "Export CSV files at the given path")
	flag.StringVar(&sqlite, "mysql", mysql, "Export a mysql database. Use a connection string of the form `username:password@host/database`")

	flag.BoolVar(&debug, "debug", debug, "Setup debug logging")

	flag.StringVar(&sqlSeparator, "sql-seperator", sqlSeparator, "Use seperator on multi-valued fields")
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
