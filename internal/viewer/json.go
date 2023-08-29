package viewer

import (
	"encoding/json"
	"net/http"

	"github.com/FAU-CDI/hangover/internal/imap"
	"github.com/gorilla/mux"
)

func (viewer *Viewer) jsonIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(viewer.Cache.BundleNames())
}

func (viewer *Viewer) jsonBundle(w http.ResponseWriter, r *http.Request) {
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
	vars := mux.Vars(r)

	entity, ok := viewer.getEntity(vars["bundle"], imap.Label(vars["uri"]))
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
