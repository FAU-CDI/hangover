package viewer

import (
	"encoding/json"
	"net/http"

	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/anglo-korean/rdf"
	"github.com/gorilla/mux"
)

func (viewer *Viewer) jsonProgress(w http.ResponseWriter, r *http.Request) {
	progress := viewer.Status.Progress()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(progress)
}

func (viewer *Viewer) jsonPerf(w http.ResponseWriter, r *http.Request) {
	perf := viewer.Perf()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(perf)
}

func (viewer *Viewer) jsonIndex(w http.ResponseWriter, r *http.Request) {
	if viewer.jsonFallback(w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(viewer.Cache.BundleNames())
}

func (viewer *Viewer) jsonBundle(w http.ResponseWriter, r *http.Request) {
	if viewer.jsonFallback(w, r) {
		return
	}

	vars := mux.Vars(r)

	_, uris, ok := viewer.getEntityURIs(vars["bundle"])
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(uris)
}

func (viewer *Viewer) jsonEntity(w http.ResponseWriter, r *http.Request) {
	if viewer.jsonFallback(w, r) {
		return
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Setup the json response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// render the entity
	json.NewEncoder(w).Encode(entity)
}

func (viewer *Viewer) jsonNTriples(w http.ResponseWriter, r *http.Request) {
	if viewer.jsonFallback(w, r) {
		return
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Setup the json response
	w.Header().Set("Content-Type", "application/n-triples")
	w.Header().Set("Content-Disposition", `attachment; filename="entity.nt"`)
	w.WriteHeader(http.StatusOK)

	// render the entity
	err := entity.WriteAllTriples(w, true, rdf.NTriples)
	if err != nil {
		viewer.Status.LogError("entity.nt", err, "uri", vars["uri"])
	}
}

func (viewer *Viewer) jsonTurtle(w http.ResponseWriter, r *http.Request) {
	if viewer.jsonFallback(w, r) {
		return
	}

	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Setup the json response
	w.Header().Set("Content-Type", "text/turtle")
	w.Header().Set("Content-Disposition", `attachment; filename="entity.ttl"`)
	w.WriteHeader(http.StatusOK)

	// render the entity
	err := entity.WriteAllTriples(w, true, rdf.Turtle)
	if err != nil {
		viewer.Status.LogError("entity.ttl", err, "uri", vars["uri"])
	}
}
