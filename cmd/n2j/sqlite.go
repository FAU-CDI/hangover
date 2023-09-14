package main

import (
	"database/sql"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/exporter"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
)

const (
	sqliteMaxQueryVar = 32766 // see https://www.sqlite.org/limits.html
	sqlLiteBatchSize  = 1000
)

func doSQL(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, proto, addr string, st *stats.Stats) {
	var err error

	// setup the sqlite
	db, err := sql.Open(proto, addr)
	if err != nil {
		st.LogFatal("open sql", err)
	}

	// and do the export
	err = st.DoStage(stats.StageExportSQL, func() error {
		return sparkl.Export(pb, index, bEngine, &exporter.SQL{
			DB: db,

			BatchSize:   sqlLiteBatchSize,
			MaxQueryVar: sqliteMaxQueryVar,

			MakeFieldTables: sqlFieldTables,

			Separator: sqlSeperator,
		}, st)
	})
	if err != nil {
		st.LogFatal("export sql", err)
	}
}
