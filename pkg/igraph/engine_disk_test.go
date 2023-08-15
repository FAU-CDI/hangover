package igraph

import (
	"testing"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

func TestDiskEngine(t *testing.T) {
	dir := t.TempDir()
	graphTest(t, &DiskEngine[int, string]{
		DiskMap: imap.DiskMap[int]{
			Path: dir,
		},
	}, 100_000)
}
