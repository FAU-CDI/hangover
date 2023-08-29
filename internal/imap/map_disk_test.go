package imap

import "testing"

func TestDiskMap(t *testing.T) {
	dir := t.TempDir()
	mapTest(t, DiskMap{
		Path: dir,
	}, 100)
}
