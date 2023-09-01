package viewer

import (
	"html/template"
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
	mux         mux.Router
	Cache       *sparkl.Cache
	Pathbuilder *pathbuilder.Pathbuilder
	RenderFlags RenderFlags

	Footer template.HTML // html to include in footer of every page
	init   sync.Once
}

func (viewer *Viewer) Close() error {
	if viewer == nil {
		return nil
	}
	return viewer.Cache.Close()
}

type RenderFlags struct {
	PublicURL   string
	Predicates  sparkl.Predicates
	HTMLRender  bool
	ImageRender bool
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
		viewer.mux.HandleFunc("/legal", viewer.htmlLegal)
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
