package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/igraph"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/sparkl/storages"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/perf"
)

func doJSON(pb *pathbuilder.Pathbuilder, index *igraph.Index, bEngine storages.BundleEngine) {
	var err error

	// generate bundles
	var bundles map[string][]wisski.Entity
	{
		start := perf.Now()
		bundles, err = sparkl.LoadPathbuilder(pb, index, bEngine)
		if err != nil {
			log.Fatalf("Unable to load pathbuilder: %s", err)
		}
		bundleT := perf.Since(start)
		log.Printf("extracted bundles, took %s", bundleT)
	}

	json.NewEncoder(os.Stdout).Encode(bundles)
}
