// Package wca implements WissKI Column Archive functionality
package wca

import (
	"fmt"
	"time"

	"github.com/FAU-CDI/hangover/internal/sqlitey"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	_ "embed"
)

// Archive implements an opened WissKI Column Archive
type Archive struct {
	conn *sqlite.Conn // database connection

	manifest Manifest // manifest
}

// Close closes this archive
func (archive *Archive) Close() error {
	return archive.conn.Close()
}

//go:embed wca.sql
var wcaSQL string

// initialize initializes the archive, copying relevant values from the given manifest
func (archive *Archive) initialize(manifest *Manifest) error {
	// check that we have a fresh schema!
	{
		schema_version, err := sqlitey.Result(archive.conn, "PRAGMA schema_version;", nil, sqlitex.ResultInt64)
		if err != nil {
			return fmt.Errorf("initialize: unable to check schema_version: %w", err)
		}
		if schema_version != 0 {
			return fmt.Errorf("initialize: not a fresh database: expected schema_version 0, but got %d", schema_version)
		}
	}

	// create the basic table structure
	if err := sqlitex.ExecuteScript(archive.conn, wcaSQL, nil); err != nil {
		return fmt.Errorf("initialize: unable to create table structure: %w", err)
	}

	// copy over the manifest unless it's nil
	if manifest != nil {
		archive.manifest = *manifest
	}

	// setup defaults for specific flags
	if archive.manifest.Created.IsZero() {
		archive.manifest.Created = time.Now()
	}
	if archive.manifest.Version == "" {
		archive.manifest.Version = MANIFEST_VERSION
	}

	// and write out the manifest
	if err := archive.manifest.writeTo(archive); err != nil {
		return fmt.Errorf("initialize: unable to write manifest: %w", err)
	}

	return nil
}
