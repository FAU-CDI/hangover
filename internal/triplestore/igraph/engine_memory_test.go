package igraph_test

import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
)

func TestMemoryEngine(t *testing.T) {
	t.Parallel()

	graphTest(t, &igraph.MemoryEngine{}, 100_000)
}
