package wca

import (
	"fmt"
	"time"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover/internal/sqlitey"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const MANIFEST_VERSION = "1.0"

// Manifest represents the manifest of a WCA archive
type Manifest struct {
	Description string
	Created     time.Time
	Pathbuilder pathbuilder.Pathbuilder
	Version     string
}

// writeTo writes the given manifest into the archive
func (manifest *Manifest) writeTo(archive *Archive) error {
	// marshal the pathbuilder
	pathbuilder, err := pbxml.Marshal(manifest.Pathbuilder)
	if err != nil {
		return fmt.Errorf("unable to format pathbuilder: %w", err)
	}

	// and do the insert
	{
		err := sqlitex.ExecuteTransient(
			archive.conn,
			"INSERT OR REPLACE INTO `wca_manifest` (`Description`, `Created`, `Pathbuilder`, `Version`) VALUES (?, ?, ?, ?)",
			&sqlitex.ExecOptions{
				Args: []any{
					manifest.Description,
					manifest.Created.Unix(),
					pathbuilder,
					MANIFEST_VERSION,
				},
			},
		)
		if err != nil {
			return fmt.Errorf("unable to insert into `wca_manifest` table: %w", err)
		}
	}

	return nil
}

// readFrom unmarshals a manifest from the given archive.
// If no valid manifest is found in the archive, returns an error.
func (manifest *Manifest) readFrom(archive *Archive) error {
	// make sure there is exactly one row!
	{
		count, err := sqlitey.Result(
			archive.conn,
			"SELECT count(*) FROM `wca_manifest` WHERE `Version` = ?",
			[]any{MANIFEST_VERSION},
			sqlitex.ResultInt64,
		)
		if err != nil {
			return fmt.Errorf("unable to query `wca_manifest` table: %w", err)
		}
		if count != 1 {
			return fmt.Errorf("error reading `wca_manifest` table: expected to find exactly 1 row for version, but found %d rows", count)
		}
	}

	// query that row!
	sawRow := false
	err := sqlitex.ExecuteTransient(
		archive.conn, "SELECT `Description`, `Created`, `Pathbuilder`, count(*) as `COUNT` FROM `wca_manifest` WHERE `Version` = ? LIMIT 1",
		&sqlitex.ExecOptions{
			Args: []any{MANIFEST_VERSION},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				sawRow = true

				manifest.Description = stmt.GetText("Description")
				manifest.Created = time.Unix(stmt.GetInt64("Created"), 0)
				manifest.Version = MANIFEST_VERSION

				bytes := make([]byte, stmt.ColumnLen(stmt.ColumnIndex("Pathbuilder")))
				stmt.GetBytes("Pathbuilder", bytes)

				var err error
				manifest.Pathbuilder, err = pbxml.Unmarshal(bytes)
				return err
			},
		},
	)
	if err != nil {
		return fmt.Errorf("unable to query `wca_manifest` table: %w", err)
	}
	if !sawRow {
		return fmt.Errorf("error reading `wca_manifest` table: no row read")
	}

	return nil
}
