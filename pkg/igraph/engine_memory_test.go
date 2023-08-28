package igraph

import (
	"testing"

	"github.com/FAU-CDI/hangover/pkg/imap"
)

func TestMemoryEngine(t *testing.T) {
	graphTest(t, &MemoryEngine[imap.Label, imap.Datum]{}, 100_000)
}
