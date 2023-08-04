package imap

import "testing"

func TestDiskEngine(t *testing.T) {
	dir := t.TempDir()
	engineTest(t, DiskEngine[string]{
		Path: dir,
	}, 100_000)
}
