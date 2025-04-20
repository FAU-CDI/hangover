package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

func doSQL(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, proto, addr string, skipClose bool, st *stats.Stats) (*sql.DB, error) {
	var err error

	// setup the sqlite
	db, err := sql.Open(proto, addr)
	if err != nil {
		st.LogFatal("open sql", err)
	}

	// and do the export
	err = st.DoStage(stats.StageExportSQL, func() error {
		return sparkl.Export(pb, index, bEngine, &exporter.SQL{
			SkipClose: skipClose,
			DB:        db,

			BatchSize:   sqlLiteBatchSize,
			MaxQueryVar: sqliteMaxQueryVar,

			MakeFieldTables: sqlFieldTables,

			Separator: sqlSeperator,
		}, st)
	})
	if err != nil {
		st.LogFatal("export sql", err)
	}
	return db, err
}

func doCSV(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, path string, st *stats.Stats) (e error) {
	// turn it into an sqlite first
	db, err := doSQL(pb, index, bEngine, "sqlite", ":memory:", true, st)
	if err != nil {
		return
	}
	defer func() {
		if e2 := db.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close db: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	// query the list of tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if e2 := rows.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close rows: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	// make it a list
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			st.LogFatal("scanning table names", err)
		}
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		st.LogFatal("scanning table names", err)
	}

	for _, table := range tables {
		st.Log("exporting table", "name", table)
		if err := doCSVTable(db, table, path); err != nil {
			st.LogFatal("writing csv table", err)
		}
	}
	return nil
}

func doCSVTable(db *sql.DB, table string, path string) (e error) {
	// open a csv file matching the name
	file, err := os.Create(filepath.Join(path, table+".csv")) // #nosec G304 -- this is intended
	if err != nil {
		return fmt.Errorf("failed to create for table %q: %w", table, err)
	}
	defer func() {
		if e2 := file.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close csv file: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	// create a writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// query everything in the table
	rows, err := db.Query("SELECT * FROM " + table) // #nosec G202 -- can't parametrize table name
	if err != nil {
		return err
	}
	defer func() {
		if e2 := rows.Close(); e2 != nil {
			e2 = fmt.Errorf("failed to close rows: %w", e2)
			if e == nil {
				e = e2
			} else {
				e = errors.Join(e, e2)
			}
		}
	}()

	// write the header
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if err := writer.Write(columns); err != nil {
		return err
	}

	// Write the data
	for rows.Next() {
		// create some values
		values := make([]any, len(columns))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}

		// read everything
		if err := rows.Scan(values...); err != nil {
			return err
		}

		// convert into strings
		strings := make([]string, len(values))
		for i, val := range values {
			b := val.(*sql.RawBytes)
			strings[i] = string(*b)
		}

		// and write out
		if err := writer.Write(strings); err != nil {
			return err
		}
	}

	// check for errors
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}
