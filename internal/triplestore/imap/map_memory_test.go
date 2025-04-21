//spellchecker:words imap
package imap_test

//spellchecker:words testing github hangover internal triplestore imap
import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
)

func TestMemoryMap(t *testing.T) {
	t.Parallel()

	mapTest(t, &imap.MemoryMap{}, 1_000_000)
}
