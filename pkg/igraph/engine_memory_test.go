package igraph

import "testing"

func TestMemoryEngine(t *testing.T) {
	graphTest(t, &MemoryEngine[int, string]{}, 100_000)
}
