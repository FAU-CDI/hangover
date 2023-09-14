package main

import (
	"encoding/json"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

func doJSON(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, st *stats.Stats) {
	var err error

	// generate bundles
	var bundles map[string][]wisski.Entity
	st.DoStage(stats.StageExtractBundles, func() error {
		bundles, err = sparkl.LoadPathbuilder(pb, index, bEngine, st)
		return err
	})
	if err != nil {
		st.LogFatal("extract bundles", err)
	}

	st.DoStage(stats.StageExportJSON, func() error {
		return json.NewEncoder(os.Stdout).Encode(bundles)
	})
	if err != nil {
		st.LogFatal("write json", err)
	}
}
