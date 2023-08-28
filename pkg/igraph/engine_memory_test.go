package igraph

import (
	"testing"
)

func TestMemoryEngine(t *testing.T) {
	graphTest(t, &MemoryEngine{}, 100_000)
}
