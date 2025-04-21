//spellchecker:words viewer
package viewer

//spellchecker:words bytes html template http strings sync time github drincw pathbuilder hangover internal assets sparkl stats gorilla pkglib text
import (
	"bytes"
	"fmt"
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
	"github.com/tkw1536/pkglib/text"
)

//spellchecker:words Wiss KI

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

func (viewer *Viewer) logPublicURI(uri string, err error) {
	viewer.Stats.LogError("unable to parse public url", err, "uri", uri)
}

// NewViewer creates a new viewer that logs to the given output.
func NewViewer(writer io.Writer, debug bool) *Viewer {
	return &Viewer{
		Stats: stats.NewStats(writer, debug),
	}
}

func (viewer *Viewer) Close() error {
	if viewer == nil {
		return nil
	}
	if err := viewer.Cache.Close(); err != nil {
		return fmt.Errorf("failed to close cache: %w", err)
	}
	return nil
}

type RenderFlags struct {
	PublicURL   string
	TipsyURL    string
	Predicates  sparkl.Predicates
	StrictCSP   bool // use strict content-security-policy for images and media by only allowing content from public uris
	HTMLRender  bool
	ImageRender bool
}

func (rf RenderFlags) PublicURLs(onError func(string, error)) (public []string) {
	// add all the public urls
	for _, raw := range text.Splitter(",\n")(rf.PublicURL) {
		url, err := url.Parse(strings.TrimSpace(raw))
		if err != nil {
			if onError != nil {
				onError(raw, err)
			}
			continue
		}

		url.Scheme = "http"
		public = append(public, url.String())

		url.Scheme = "https"
		public = append(public, url.String())
	}
	return public
}

func (rf RenderFlags) Tipsy() string {
	if strings.HasPrefix(rf.TipsyURL, "http://") || strings.HasPrefix(rf.TipsyURL, "https://") {
		return rf.TipsyURL
	}
	return ""
}

// CSPHeader returns the CSPHeader to be included in viewer responses.
func (rf RenderFlags) CSPHeader(onURIError func(string, error)) string {
	// don't allow anything by default
	header := "default-src 'none'; object-src 'self'; connect-src 'self'; script-src 'self'; font-src 'self'; "

	if rf.HTMLRender {
		// when rendering html, we explicitly want to allow inline styles.
		header += "style-src 'self' 'unsafe-inline'; "
	} else {
		// by default only allow self styles
		header += "style-src 'self'; "
	}
	if tipsy := rf.Tipsy(); tipsy != "" {
		header += "frame-src " + tipsy + "; "
	}

	// determine the source for media and images
	// by default it is everything, but in the strict case we use only public uris
	source := "*"

	if rf.StrictCSP {
		source = strings.Join(rf.PublicURLs(onURIError), " ")
	}

	if rf.ImageRender || rf.HTMLRender {
		header += "img-src " + source + ";"
	}
	if rf.HTMLRender {
		header += "media-src " + source + ";"
	}
	return header
}

// handlerError wraps the given handler to log the error into the debug log if non-nil.
func (viewer *Viewer) handlerError(handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			viewer.Stats.LogDebug("error handling request", "url", r.URL.String(), "method", r.Method, "err", err)
		}
	}
}

func (viewer *Viewer) setupMux() {
	viewer.init.Do(func() {
		viewer.mux.HandleFunc("/", viewer.htmlIndex)
		viewer.mux.HandleFunc("/about", viewer.htmlLegal)
		viewer.mux.HandleFunc("/pathbuilder", viewer.htmlPathbuilder)

		if viewer.RenderFlags.Tipsy() != "" {
			viewer.mux.HandleFunc("/tipsy", viewer.htmlTipsy)
		}
		viewer.mux.HandleFunc("/perf", viewer.htmlPerf)

		viewer.mux.HandleFunc("/bundle/{bundle}", viewer.htmlBundle).Queries("limit", "{limit:\\d+}", "skip", "{skip:\\d+}")
		viewer.mux.HandleFunc("/bundle/{bundle}", viewer.htmlBundle)

		viewer.mux.HandleFunc("/entity/{bundle}", viewer.htmlEntity).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/wisski/get", viewer.htmlEntityResolve).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/wisski/navigate/{id}/view", viewer.sendToResolver)

		viewer.mux.HandleFunc("/api/v1", viewer.handlerError(viewer.jsonIndex))
		viewer.mux.HandleFunc("/api/v1/progress", viewer.handlerError(viewer.jsonProgress))
		viewer.mux.HandleFunc("/api/v1/perf", viewer.handlerError(viewer.jsonPerf))
		viewer.mux.HandleFunc("/api/v1/bundle/{bundle}", viewer.handlerError(viewer.jsonBundle))
		viewer.mux.HandleFunc("/api/v1/entity/{bundle}", viewer.handlerError(viewer.jsonEntity)).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/api/v1/ntriples/{bundle}", viewer.handlerError(viewer.jsonNTriples)).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/api/v1/turtle/{bundle}", viewer.handlerError(viewer.jsonTurtle)).Queries("uri", "{uri:.+}")

		viewer.mux.HandleFunc("/api/v1/svg/{bundle}", viewer.handlerError(viewer.jsonSVG)).Queries("uri", "{uri:.+}")
		viewer.mux.HandleFunc("/api/v1/dot/{bundle}", viewer.handlerError(viewer.jsonDOT)).Queries("uri", "{uri:.+}")

		viewer.mux.PathPrefix("/assets/").Handler(assets.AssetHandler)

		viewer.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/svg+xml")
			http.ServeContent(w, r, "favicon.ico", time.Time{}, bytes.NewReader(hangover.IconSVG))
		})

		viewer.cspHeader = viewer.RenderFlags.CSPHeader(viewer.logPublicURI)
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
