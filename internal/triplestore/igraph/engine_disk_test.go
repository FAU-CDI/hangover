//spellchecker:words igraph
package igraph_test

//spellchecker:words testing github hangover internal triplestore igraph imap
import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/triplestore/imap"
)

func TestDiskEngine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	graphTest(t, &igraph.DiskEngine{
		DiskMap: imap.DiskMap{
			Path: dir,
		},
	}, 100_000)
}
