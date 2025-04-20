package imap_test

import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
)

func TestDiskMap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mapTest(t, imap.DiskMap{
		Path: dir,
	}, 100)
}
