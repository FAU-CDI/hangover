package imap

import "testing"

func TestDiskMap(t *testing.T) {
	dir := t.TempDir()
	mapTest(t, DiskMap[string]{
		Path: dir,
	}, 1_000_000)
}
