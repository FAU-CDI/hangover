package imap_test

import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
)

func TestMemoryMap(t *testing.T) {
	t.Parallel()

	mapTest(t, &imap.MemoryMap{}, 1_000_000)
}
