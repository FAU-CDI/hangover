package viewer

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/anglo-korean/rdf"
	"github.com/gorilla/mux"
)

func (viewer *Viewer) jsonProgress(w http.ResponseWriter, r *http.Request) error {
	progress := viewer.Stats.Progress()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(progress); err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonPerf(w http.ResponseWriter, r *http.Request) error {
	perf := viewer.Perf()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(perf); err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonIndex(w http.ResponseWriter, r *http.Request) error {
	if viewer.jsonFallback(w, r) {
		return nil
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(viewer.Cache.BundleNames()); err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonBundle(w http.ResponseWriter, r *http.Request) error {
	if viewer.jsonFallback(w, r) {
		return nil
	}

	vars := mux.Vars(r)

	_, uris, ok := viewer.getEntityURIs(vars["bundle"])
	if !ok {
		http.NotFound(w, r)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(uris); err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonEntity(w http.ResponseWriter, r *http.Request) error {
	if viewer.jsonFallback(w, r) {
		return nil
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	// Setup the json response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// render the entity
	if err := json.NewEncoder(w).Encode(entity); err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonNTriples(w http.ResponseWriter, r *http.Request) error {
	if viewer.jsonFallback(w, r) {
		return nil
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	// Setup the json response
	w.Header().Set("Content-Type", "application/n-triples")
	w.Header().Set("Content-Disposition", `attachment; filename="entity.nt"`)
	w.WriteHeader(http.StatusOK)

	// render the entity
	err := entity.WriteAllTriples(w, true, rdf.NTriples)
	if err != nil {
		viewer.Stats.LogError("entity.nt", err, "uri", vars["uri"])
		return fmt.Errorf("failed to write all triples: %w", err)
	}
	return nil
}

func (viewer *Viewer) jsonTurtle(w http.ResponseWriter, r *http.Request) error {
	if viewer.jsonFallback(w, r) {
		return nil
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	// Setup the json response
	w.Header().Set("Content-Type", "text/turtle")
	w.Header().Set("Content-Disposition", `attachment; filename="entity.ttl"`)
	w.WriteHeader(http.StatusOK)

	// render the entity
	err := entity.WriteAllTriples(w, true, rdf.Turtle)
	if err != nil {
		viewer.Stats.LogError("entity.ttl", err, "uri", vars["uri"])
		return fmt.Errorf("failed to write all triples: %w", err)
	}
	return nil
}
