package viewer

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/assets"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/gorilla/mux"
)

// cspell:words WissKI

// Viewer implements an [http.Handler] that displays WissKI Entities.
type Viewer struct {
	Stats *stats.Stats // Stats holds the current stats of the viewer

	mux       mux.Router
	cspHeader string

	Cache       *sparkl.Cache
	Pathbuilder *pathbuilder.Pathbuilder
	RenderFlags RenderFlags

	Footer template.HTML // html to include in footer of every page
	init   sync.Once
}

// NewViewer creates a new viewer that logs to the given output
func NewViewer(writer io.Writer) *Viewer {
	return &Viewer{
		Stats: stats.NewStats(writer),
	}
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
	StrictCSP   bool // use strict content-security-policy for images and media by only allowing content from public uris
	HTMLRender  bool
	ImageRender bool

	// Stats holds the status to use for logging
	Stats *stats.Stats
}

func (rf RenderFlags) PublicURIS() (public []string) {
	// add all the public urls
	for _, raw := range strings.Split(rf.PublicURL, ",") {
		url, err := url.Parse(raw)
		if err != nil {
			rf.Stats.LogError("parse url", err)
			continue
		}

		url.Scheme = "http"
		public = append(public, url.String())

		url.Scheme = "https"
		public = append(public, url.String())
	}
	return public
}

// CSPHeader returns the CSPHeader to be included in viewer responses
func (rf RenderFlags) CSPHeader() string {
	// don't allow anything by default
	header := "default-src 'none'; connect-src 'self'; script-src 'self'; font-src 'self'; "

	if rf.HTMLRender {
		// when rendering html, we explicitly want to allow inline styles.
		header += "style-src 'self' 'unsafe-inline'; "
	} else {
		// by default only allow self styles
		header += "style-src 'self'; "
	}

	// determine the source for media and images
	// by default it is everything, but in the strict case we use only public uris
	source := "*"

	if rf.StrictCSP {
		source = strings.Join(rf.PublicURIS(), " ")
	}

	if rf.ImageRender || rf.HTMLRender {
		header += "img-src " + source + ";"
	}
	if rf.HTMLRender {
		header += "media-src " + source + ";"
	}
	return header
}

func (viewer *Viewer) setupMux() {
	viewer.init.Do(func() {
		viewer.mux.HandleFunc("/", viewer.htmlIndex)
		viewer.mux.HandleFunc("/about", viewer.htmlLegal)
		viewer.mux.HandleFunc("/pathbuilder", viewer.htmlPathbuilder)
		viewer.mux.HandleFunc("/perf", viewer.htmlPerf)

		viewer.mux.HandleFunc("/bundle/{bundle}", viewer.htmlBundle).Queries("limit", "{limit:\\d+}", "skip", "{skip:\\d+}")
		viewer.mux.HandleFunc("/bundle/{bundle}", viewer.htmlBundle)

		viewer.mux.HandleFunc("/entity/{bundle}", viewer.htmlEntity).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/wisski/get", viewer.htmlEntityResolve).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/wisski/navigate/{id}/view", viewer.sendToResolver)

		viewer.mux.HandleFunc("/api/v1", viewer.jsonIndex)
		viewer.mux.HandleFunc("/api/v1/progress", viewer.jsonProgress)
		viewer.mux.HandleFunc("/api/v1/perf", viewer.jsonPerf)
		viewer.mux.HandleFunc("/api/v1/bundle/{bundle}", viewer.jsonBundle)
		viewer.mux.HandleFunc("/api/v1/entity/{bundle}", viewer.jsonEntity).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/api/v1/ntriples/{bundle}", viewer.jsonNTriples).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/api/v1/turtle/{bundle}", viewer.jsonTurtle).Queries("uri", "{uri:.+}")

		viewer.mux.PathPrefix("/assets/").Handler(assets.AssetHandler)

		viewer.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			http.ServeContent(w, r, "favicon.ico", time.Time{}, bytes.NewReader(hangover.IconSVG))
		})

		viewer.cspHeader = viewer.RenderFlags.CSPHeader()
	})
}
func (viewer *Viewer) Prepare(cache *sparkl.Cache, pb *pathbuilder.Pathbuilder) {
	if !viewer.Stats.Done() {
		viewer.Cache = cache
		viewer.Pathbuilder = pb
		viewer.Stats.Close()
	}

	viewer.setupMux()
}

func (viewer *Viewer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	viewer.setupMux()

	w.Header().Set("Content-Security-Policy", viewer.cspHeader)
	viewer.mux.ServeHTTP(w, r)
}
