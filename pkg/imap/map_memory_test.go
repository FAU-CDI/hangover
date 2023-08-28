package imap

import "testing"

func TestMemoryMap(t *testing.T) {
	mapTest(t, &MemoryMap{}, 1_000_000)
}
