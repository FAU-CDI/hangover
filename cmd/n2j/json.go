//spellchecker:words main
package main

//spellchecker:words encoding json github drincw pathbuilder hangover internal sparkl storages stats triplestore igraph wisski
import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

func doJSON(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, st *stats.Stats) (err error) {
	// generate bundles
	var bundles map[string][]wisski.Entity
	if err := st.DoStage(stats.StageExtractBundles, func() error {
		bundles, err = sparkl.LoadPathbuilder(pb, index, bEngine, st)
		return fmt.Errorf("failed to load pathbuilder: %w", err)
	}); err != nil {
		st.LogFatal("extract bundles", err)
	}

	if err := st.DoStage(stats.StageExportJSON, func() error {
		return json.NewEncoder(os.Stdout).Encode(bundles)
	}); err != nil {
		st.LogFatal("write json", err)
	}
	return nil
}
