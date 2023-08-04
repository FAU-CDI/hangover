package imap

import "testing"

func TestMemoryEngine(t *testing.T) {
	engineTest(t, &MemoryEngine[string]{}, 1_000_000)
}
