//spellchecker:words igraph
package igraph_test

//spellchecker:words testing github hangover internal triplestore igraph
import (
	"testing"

	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
)

func TestMemoryEngine(t *testing.T) {
	t.Parallel()

	graphTest(t, &igraph.MemoryEngine{}, 100_000)
}
