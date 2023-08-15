package imap

import "testing"

func TestMemoryMap(t *testing.T) {
	mapTest(t, &MemoryMap[string]{}, 1_000_000)
}
