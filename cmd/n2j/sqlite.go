package main

import (
	"database/sql"
	"log"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/exporter"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/pkg/perf"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
)

const (
	sqliteMaxQueryVar = 32766 // see https://www.sqlite.org/limits.html
	sqlLiteBatchSize  = 1000
)

func doSQL(pb *pathbuilder.Pathbuilder, index *sparkl.Index, bEngine storages.BundleEngine, proto, addr string) {
	var err error

	// setup the sqlite
	db, err := sql.Open(proto, addr)
	if err != nil {
		log.Fatal(err)
	}

	// and do the export
	{
		start := perf.Now()
		err = sparkl.Export(pb, index, bEngine, &exporter.SQL{
			DB: db,

			BatchSize:   sqlLiteBatchSize,
			MaxQueryVar: sqliteMaxQueryVar,

			MakeFieldTables: sqlFieldTables,

			Separator: sqlSeperator,
		})
		if err != nil {
			log.Fatalf("Unable to export sql: %s", err)
		}
		bundleT := perf.Since(start)
		log.Printf("wrote bundles, took %s", bundleT)
	}
}
