package wca

import (
	"errors"
	"fmt"

	"zombiezen.com/go/sqlite"
)

// OpenFile opens the given file as a read-only WissKI Archive.
func OpenFile(path string) (*Archive, error) {
	archive, err := newArchive(path, sqlite.OpenReadOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to create new archive: %w", err)
	}

	// read the manifest, or bail out!
	if err := archive.manifest.readFrom(archive); err != nil {
		if e2 := archive.Close(); e2 != nil {
			err = errors.Join(err, fmt.Errorf("failed to close archive: %w", e2))
		}
		return nil, fmt.Errorf("failed to read from archive: %w", err)
	}

	// and done!
	return archive, nil
}

// CreateArchive creates a new archive with the given filename.
func CreateArchive(path string, manifest *Manifest) (*Archive, error) {
	// create the archive
	archive, err := newArchive(path, sqlite.OpenReadWrite, sqlite.OpenCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create new archive: %w", err)
	}

	// initialize with the manifest
	if err := archive.initialize(manifest); err != nil {
		if e2 := archive.Close(); e2 != nil {
			err = errors.Join(err, fmt.Errorf("failed to close archive: %w", e2))
		}
		return nil, fmt.Errorf("failed to initialize archive: %w", err)
	}

	return archive, nil
}

// newArchive opens an archive from the given connection string.
func newArchive(path string, flags ...sqlite.OpenFlags) (*Archive, error) {
	conn, err := sqlite.OpenConn(path, flags...)
	if err != nil {
		return nil, fmt.Errorf("unable to open archive: %w", err)
	}
	return &Archive{conn: conn}, nil
}
