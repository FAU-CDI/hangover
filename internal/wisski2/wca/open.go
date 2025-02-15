package wca

import (
	"fmt"

	"zombiezen.com/go/sqlite"
)

// OpenFile opens the given file as a read-only WissKI Archive.
func OpenFile(path string) (*Archive, error) {
	archive, err := newArchive(path, sqlite.OpenReadOnly)
	if err != nil {
		return nil, fmt.Errorf("OpenFile: %w", err)
	}

	// read the manifest, or bail out!
	if err := archive.manifest.readFrom(archive); err != nil {
		archive.Close()
		return nil, fmt.Errorf("OpenFile: %w", err)
	}

	// and done!
	return archive, nil
}

// CreateArchive creates a new archive with the given filename
func CreateArchive(path string, manifest *Manifest) (*Archive, error) {
	// create the archive
	archive, err := newArchive(path, sqlite.OpenReadWrite, sqlite.OpenCreate)
	if err != nil {
		return nil, fmt.Errorf("CreateArchive: %w", err)
	}

	// initialize with the manifest
	if err := archive.initialize(manifest); err != nil {
		archive.Close()
		return nil, fmt.Errorf("CreateArchive: %w", err)
	}

	return archive, nil
}

// newArchive opens an archive from the given connection string
func newArchive(path string, flags ...sqlite.OpenFlags) (*Archive, error) {
	conn, err := sqlite.OpenConn(path, flags...)
	if err != nil {
		return nil, fmt.Errorf("unable to open archive: %w", err)
	}
	return &Archive{conn: conn}, nil
}
