package main

import (
	"encoding/json"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/status"
	"github.com/FAU-CDI/hangover/internal/triplestore/igraph"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

func doJSON(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine, stats *status.Status) {
	var err error

	// generate bundles
	var bundles map[string][]wisski.Entity
	stats.DoStage(status.StageExtractBundles, func() error {
		bundles, err = sparkl.LoadPathbuilder(pb, index, bEngine, stats)
		return err
	})
	if err != nil {
		stats.LogFatal("extract bundles", err)
	}

	stats.DoStage(status.StageExportJSON, func() error {
		return json.NewEncoder(os.Stdout).Encode(bundles)
	})
	if err != nil {
		stats.LogFatal("write json", err)
	}
}
