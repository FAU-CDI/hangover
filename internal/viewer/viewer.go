package viewer

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/assets"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/gorilla/mux"
)

// cspell:words WissKI

// Viewer implements an [http.Handler] that displays WissKI Entities.
type Viewer struct {
	Cache       *sparkl.Cache
	Pathbuilder *pathbuilder.Pathbuilder

	RenderFlags RenderFlags

	init sync.Once
	mux  mux.Router
}

func (viewer *Viewer) Close() error {
	if viewer == nil {
		return nil
	}
	return viewer.Cache.Close()
}

type RenderFlags struct {
	HTMLRender  bool // should we render "text_long" as actual html?
	ImageRender bool // should we render "image" as actual images

	PublicURL string // should we replace links from the provided wisski?

	Predicates sparkl.Predicates
}

func (rf RenderFlags) PublicURIS() (public []string) {
	// add all the public urls
	for _, raw := range strings.Split(rf.PublicURL, ",") {
		url, err := url.Parse(raw)
		if err != nil {
			log.Printf("Unable to parse url %q: %s", raw, err)
			continue
		}

		url.Scheme = "http"
		public = append(public, url.String())

		url.Scheme = "https"
		public = append(public, url.String())
	}
	return public
}

func (viewer *Viewer) Prepare() {
	viewer.init.Do(func() {
		viewer.mux.HandleFunc("/", viewer.htmlIndex)
		viewer.mux.HandleFunc("/pathbuilder", viewer.htmlPathbuilder)
		viewer.mux.HandleFunc("/bundle/{bundle}", viewer.htmlBundle)
		viewer.mux.HandleFunc("/entity/{bundle}", viewer.htmlEntity).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/wisski/get", viewer.htmlEntityResolve).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/wisski/navigate/{id}/view", viewer.sendToResolver)

		viewer.mux.HandleFunc("/api/v1", viewer.jsonIndex)
		viewer.mux.HandleFunc("/api/v1/bundle/{bundle}", viewer.jsonBundle)
		viewer.mux.HandleFunc("/api/v1/entity/{bundle}", viewer.jsonEntity).Queries("uri", "{uri:.+}")

		viewer.mux.PathPrefix("/assets/").Handler(assets.AssetHandler)
	})

}

func (viewer *Viewer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	viewer.Prepare()
	viewer.mux.ServeHTTP(w, r)
}
