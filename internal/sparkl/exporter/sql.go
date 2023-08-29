package exporter

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/imap"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/huandu/go-sqlbuilder"
)

// cspell:words twiesing

// SQL implements an exporter for storing data inside an sql database.
// TODO(twiesing): For now this only supports string-like fields.
type SQL struct {
	DB              *sql.DB
	batches         map[string][]wisski.Entity
	Separator       string // separator for database multi-valued fields
	BatchSize       int    // BatchSize for top-level bundles
	MaxQueryVar     int    // Maximum number of query variables (overrides BatchSize)
	dbLock          sync.Mutex
	batchLock       sync.Mutex
	MakeFieldTables bool // create tables for field values (if false, they get joined with "separator")
}

// exec executes an sql query
func (sql *SQL) exec(query string, args []any) (err error) {
	sql.dbLock.Lock()
	defer sql.dbLock.Unlock()

	_, err = sql.DB.Exec(query, args...)
	return
}

// execInsert executes an insert into the given table, the given columns, and the given values.
// When this would exceed limits on maximum number of query variables, multiple inserts are executed.
func (sql *SQL) execInsert(table string, columns []string, values [][]any) error {
	// nothing to insert!
	if len(values) == 0 {
		return nil
	}

	// determine the chink size based on total number of query variables
	chunkSize := sql.MaxQueryVar / len(columns)
	if chunkSize == 0 {
		return errInsufficientQueryVars
	}

	// maybe the user requested an even smaller batch size!
	if sql.BatchSize < chunkSize {
		chunkSize = sql.BatchSize
	}

	for i := 0; i < len(values); i += chunkSize {
		insert := sqlbuilder.InsertInto(table)
		insert.Cols(columns...)

		// determine the true chunk size
		chunkStart := i
		chunkEnd := i + chunkSize
		if chunkEnd > len(values) {
			chunkEnd = len(values)
		}

		// and add the values for this chunk
		for _, v := range values[chunkStart:chunkEnd] {
			insert.Values(v...)
		}

		// perform this insert
		if err := sql.exec(insert.Build()); err != nil {
			return err
		}
	}

	return nil
}

func (sql *SQL) Begin(bundle *pathbuilder.Bundle, count int64) error {
	// make sure that the batches are initialized
	func() {
		sql.batchLock.Lock()
		defer sql.batchLock.Unlock()

		if sql.batches == nil {
			sql.batches = make(map[string][]wisski.Entity)
		}
	}()

	// create a table for the given bundle
	return sql.createBundleTable(bundle)
}

const (
	uriColumn    = "uri"
	parentColumn = "parent"
	valueColumn  = "value"

	fieldColumnPrefix = "field__"

	bundleTablePrefix = "bundle__"

	fieldTablePrefix = "field__"
	fieldTableInfix  = "__"
)

func (*SQL) BundleTable(bundle *pathbuilder.Bundle) string {
	return bundleTablePrefix + bundle.MachineName()
}

func (*SQL) FieldTable(bundle *pathbuilder.Bundle, field pathbuilder.Field) string {
	return fieldTablePrefix + bundle.MachineName() + fieldTableInfix + field.MachineName()
}

func (*SQL) FieldColumn(field pathbuilder.Field) string {
	return fieldColumnPrefix + field.MachineName()
}

// createBundleTable creates a table for the given bundle
func (sql *SQL) createBundleTable(bundle *pathbuilder.Bundle) error {
	// build all the child tables first!
	for _, child := range bundle.ChildBundles {
		if err := sql.createBundleTable(child); err != nil {
			return err
		}
	}

	// drop the table if it already exists
	if err := sql.exec("DROP TABLE IF EXISTS "+sql.BundleTable(bundle)+";", nil); err != nil {
		return err
	}

	// create a table with fields for every field, and the child field
	table := sqlbuilder.CreateTable(sql.BundleTable(bundle)).IfNotExists()
	table.Define(uriColumn, "TEXT", "NOT NULL")
	if !bundle.IsToplevel() {
		table.Define(parentColumn, "TEXT")
	}
	for _, field := range bundle.ChildFields {
		if !sql.MakeFieldTables {
			table.Define(sql.FieldColumn(field))
		} else {
			if err := sql.CreateFieldTable(bundle, field); err != nil {
				return err
			}
		}
	}

	// run the query
	return sql.exec(table.Build())
}

// CreateFieldTable creates a table for the given field
func (sql *SQL) CreateFieldTable(bundle *pathbuilder.Bundle, field pathbuilder.Field) error {
	table := sqlbuilder.CreateTable(sql.FieldTable(bundle, field)).IfNotExists()
	table.Define(uriColumn, "TEXT")
	table.Define(valueColumn, "TEXT")
	return sql.exec(table.Build())
}

func (sql *SQL) Add(bundle *pathbuilder.Bundle, entity *wisski.Entity) (err error) {
	name := bundle.MachineName()

	batch := func() []wisski.Entity {
		sql.batchLock.Lock()
		defer sql.batchLock.Unlock()

		// add the current entity to the bundle
		sql.batches[name] = append(sql.batches[name], *entity)
		if len(sql.batches[name]) < sql.BatchSize {
			return nil
		}

		// extract current batch
		entities := make([]wisski.Entity, sql.BatchSize)
		count := copy(entities, sql.batches[name])
		entities = entities[:count]

		// remove them from the remaining batches
		rest := copy(sql.batches[name], sql.batches[name][count:]) // slide to the left
		sql.batches[name] = sql.batches[name][:rest]

		return entities
	}()

	// no current batch to insert
	if len(batch) == 0 {
		return nil
	}

	return sql.insert(bundle, "", batch)
}

func (sql *SQL) End(bundle *pathbuilder.Bundle) error {
	name := bundle.MachineName()

	// extract the remaining entities to be inserted
	rest := func() []wisski.Entity {
		sql.batchLock.Lock()
		defer sql.batchLock.Unlock()

		result := sql.batches[name]
		delete(sql.batches, name)
		return result
	}()

	// split them into batches
	for len(rest) > 0 {
		batch := rest
		rest = nil
		if len(batch) > sql.BatchSize {
			rest = batch[sql.BatchSize:]
			batch = batch[:sql.BatchSize]
		}

		// and insert them all
		if err := sql.insert(bundle, "", batch); err != nil {
			return err
		}
	}

	return nil
}

func (sql *SQL) Close() error {
	return sql.DB.Close() // close the database
}

var (
	nullString               sql.NullString
	errInsufficientQueryVars = errors.New("insufficient query variables")
)

// inserts performs inserts into the table for the provided bundle.
func (sql *SQL) insert(bundle *pathbuilder.Bundle, parent imap.Label, entities []wisski.Entity) error {

	// 1. insert into the bundle table
	if err := sql.insertBundleTable(bundle, parent, entities); err != nil {
		return err
	}

	// 2. insert into the field table(s) (if any)
	for _, field := range bundle.ChildFields {
		if err := sql.insertFieldTables(bundle, field, entities); err != nil {
			return err
		}
	}

	// 3. insert any children into table(s)
	bundles := bundle.ChildBundles
	for _, entity := range entities {
		for _, bundle := range bundles {
			if err := sql.insertChildTables(entity, bundle); err != nil {
				return err
			}
		}
	}

	return nil
}

func (sql *SQL) insertFieldTables(bundle *pathbuilder.Bundle, field pathbuilder.Field, entities []wisski.Entity) error {
	if !sql.MakeFieldTables {
		// user requested *not* to make the field tables
		return nil
	}

	// insert into the uri and value columns for each field
	columns := []string{uriColumn, valueColumn}
	values := make([][]any, 0)
	for _, entity := range entities {
		for _, value := range entity.Fields[field.MachineName()] {
			values = append(values, []any{
				entity.URI,
				fmt.Sprintf("%v", value.Value),
			})
		}
	}

	// do the actual insert!
	return sql.execInsert(sql.FieldTable(bundle, field), columns, values)
}

func (sql *SQL) insertBundleTable(bundle *pathbuilder.Bundle, parent imap.Label, entities []wisski.Entity) error {
	// determine all the columns to insert
	var columns []string
	columns = append(columns, uriColumn)
	if !bundle.IsToplevel() {
		columns = append(columns, parentColumn)
	}

	fields := bundle.ChildFields // the child fields to iterate over
	if !sql.MakeFieldTables {
		for _, field := range fields {
			columns = append(columns, sql.FieldColumn(field))
		}
	}

	// make all the strings
	var builder strings.Builder
	values := make([][]any, len(entities))
	for i, entity := range entities {
		values[i] = make([]any, 0, len(values))

		// uri and parent
		values[i] = append(values[i], string(entity.URI))
		if !bundle.IsToplevel() {
			values[i] = append(values[i], string(parent))
		}

		if sql.MakeFieldTables { // don't have to insert
			continue
		}

		// values for the actual fields
		for _, field := range fields {
			fvalues := entity.Fields[field.MachineName()]
			if len(fvalues) == 0 {
				values[i] = append(values[i], nullString)
				continue
			}
			for _, v := range fvalues {
				fmt.Fprintf(&builder, "%v%s", v.Value, sql.Separator)
			}
			values[i] = append(values[i], builder.String()[:builder.Len()-len(sql.Separator)])
			builder.Reset()
		}
	}
	return sql.execInsert(sql.BundleTable(bundle), columns, values)
}

func (sql *SQL) insertChildTables(parent wisski.Entity, bundle *pathbuilder.Bundle) error {
	children := parent.Children[bundle.MachineName()]
	return sql.insert(bundle, parent.URI, children)
}
