//spellchecker:words viewer
package viewer

//spellchecker:words errors html template http strconv strings embed github drincw pathbuilder pbxml hangover internal assets stats triplestore impl wisski htmlx gorilla golang
import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	_ "embed"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/drincw/pathbuilder/pbxml"
	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/assets"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/htmlx"
	"github.com/gorilla/mux"
	"golang.org/x/net/html"
)

//spellchecker:words pathbuilder

var errEvenLength = errors.New("pairs must be of even length")

var contextTemplateFuncs = template.FuncMap{
	"renderhtml": func(html string, globals contextGlobal) (template.HTML, error) {
		render, err := htmlx.ReplaceLinks(html, globals.ReplaceURL)
		if err != nil {
			return "", fmt.Errorf("failed to replace links: %w", err)
		}
		return template.HTML(render), nil // #nosec G203
	},
	"combine": func(pairs ...any) (map[string]any, error) {
		if len(pairs)%2 != 0 {
			return nil, errEvenLength
		}
		result := make(map[string]any, len(pairs)/2)
		for i, v := range pairs {
			if i%2 == 1 {
				result[pairs[(i-1)].(string)] = v
			}
		}
		return result, nil
	},
	"debug": func(value any) string {
		// log out a value for debugging
		fmt.Printf("%#v", value)
		return ""
	},
}

//go:embed templates/bundle.html
var bundleHTML string

var bundleTemplate = assets.Assetshangover.MustParseShared(
	"bundle.html",
	bundleHTML,
	contextTemplateFuncs,
)

//go:embed templates/entity.html
var entityHTML string

var entityTemplate = assets.Assetshangover.MustParseShared(
	"entity.html",
	entityHTML,
	contextTemplateFuncs,
)

//go:embed templates/index.html
var indexHTML string

//go:embed templates/about.html
var aboutHTML string

var indexTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"index.html",
	indexHTML,
	contextTemplateFuncs,
)

var aboutTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"about.html",
	aboutHTML,
	contextTemplateFuncs,
)

//go:embed templates/perf.html
var perfHTML string

var perfTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"perf.html",
	perfHTML,
	contextTemplateFuncs,
)

//go:embed templates/pathbuilder.html
var pathbuilderHTML string

var pbTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"pathbuilder.html",
	pathbuilderHTML,
	contextTemplateFuncs,
)

//go:embed templates/tipsy.html
var tipsy string

var tipsyTemplate *template.Template = assets.Assetstipsy.MustParseShared(
	"tipsy.html",
	tipsy,
	contextTemplateFuncs,
)

type contextGlobal struct {
	InterceptedPrefixes []string // urls that are redirected to this server
	Footer              template.HTML
	DisableForm         bool
	RenderFlags
}

func (cg contextGlobal) ReplaceURL(u string) string {
	for _, prefix := range cg.InterceptedPrefixes {
		if strings.HasPrefix(u, prefix) {
			url, err := url.Parse(u)
			if err != nil {
				continue
			}
			url.Scheme = ""
			url.Host = ""
			url.OmitHost = true
			return url.String()
		}
	}
	return u
}

func (viewer *Viewer) contextGlobal() (global contextGlobal) {
	global.Footer = viewer.Footer
	global.RenderFlags = viewer.RenderFlags
	global.DisableForm = !viewer.Stats.Done()

	if viewer.RenderFlags.PublicURL == "" {
		return
	}

	for _, public := range viewer.RenderFlags.PublicURLs(viewer.logPublicURI) {
		prefix, err := url.JoinPath(public, "wisski")
		if err != nil {
			continue
		}
		global.InterceptedPrefixes = append(global.InterceptedPrefixes, prefix)
	}

	return
}

type htmlIndexContext struct {
	Bundles []*pathbuilder.Bundle
	Globals contextGlobal
}

type htmlLegalContext struct {
	Globals  contextGlobal
	License  string
	Backend  string
	Frontend string
}

func (viewer *Viewer) htmlIndex(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	bundles, ok := viewer.getBundles()
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	w.WriteHeader(http.StatusOK)
	err := indexTemplate.Execute(w, htmlIndexContext{
		Globals: viewer.contextGlobal(),
		Bundles: bundles,
	})
	if err != nil {
		panic(err)
	}
}

type htmlPerfContext struct {
	Globals contextGlobal
	Perf    Perf
	Stages  []stats.StageStats
}

func (viewer *Viewer) htmlPerf(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	err := perfTemplate.Execute(w, htmlPerfContext{
		Globals: viewer.contextGlobal(),
		Perf:    viewer.Perf(),
	})
	if err != nil {
		panic(err)
	}
}

func (viewer *Viewer) htmlLegal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := aboutTemplate.Execute(w, htmlLegalContext{
		Globals:  viewer.contextGlobal(),
		License:  hangover.License,
		Backend:  hangover.Notices,
		Frontend: assets.Disclaimer,
	})
	if err != nil {
		panic(err)
	}
}

type htmlBundleContext struct {
	Bundle *pathbuilder.Bundle

	Total int

	FirstLink template.URL
	PrevLink  template.URL

	PageStart int
	PageEnd   int

	NextLink template.URL
	LastLink template.URL

	URIS    []impl.Label
	Globals contextGlobal
}

const (
	defaultBundleLimit = 100
	maxBundleLimit     = 1000
	defaultBundleSkip  = 0
)

func (viewer *Viewer) htmlBundle(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	vars := mux.Vars(r)
	limit, skip := vars["limit"], vars["skip"]

	limiti, err := strconv.Atoi(limit)
	if err != nil || limiti <= 0 || limiti > maxBundleLimit {
		limiti = defaultBundleLimit
	}

	skipi, err := strconv.Atoi(skip)
	if err != nil || skipi < 0 {
		skipi = defaultBundleSkip
	}

	viewer.htmlBundleWithLimit(w, r, vars["bundle"], limiti, skipi)
}

func (viewer *Viewer) htmlBundleWithLimit(w http.ResponseWriter, r *http.Request, bundleName string, limit, skip int) {
	bundle, entities, ok := viewer.getEntityURIs(bundleName)
	if !ok {
		http.NotFound(w, r)
		return
	}

	total := len(entities)
	// slice the entities received
	if skip < total {
		entities = entities[skip:]
	} else {
		http.NotFound(w, r)
		return
	}
	if limit < len(entities) {
		entities = entities[:limit]
	}

	// prepare the context
	context := htmlBundleContext{
		Globals: viewer.contextGlobal(),

		Total: total,

		Bundle: bundle,
		URIS:   entities,
	}

	context.PageStart = skip + 1
	context.PageEnd = context.PageStart + len(entities) - 1

	// generate all the page links
	pageLink := func(skip int) template.URL {
		if skip < 0 {
			skip = 0
		}
		return template.URL("/bundle/" + url.PathEscape(bundleName) + "?limit=" + strconv.Itoa(limit) + "&" + "skip=" + strconv.Itoa(skip)) // #nosec G203
	}

	// add the previous link if there are previous pages
	prev := skip - limit
	if prev > 0 {
		context.PrevLink = pageLink(prev)
		context.FirstLink = pageLink(0)
	}
	// add the next link if there are more
	next := skip + limit
	last := total - (total % limit) - 1
	if next < total {
		context.NextLink = pageLink(next)
		context.LastLink = pageLink(last)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := bundleTemplate.Execute(w, context)
	if err != nil {
		panic(err)
	}
}

type htmlPathbuilderContext struct {
	Pathbuilder *pathbuilder.Pathbuilder
	Globals     contextGlobal
}

func (viewer *Viewer) htmlPathbuilder(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	w.Header().Set("Content-Type", "text/html")

	w.WriteHeader(http.StatusOK)
	err := pbTemplate.Execute(w, htmlPathbuilderContext{
		Globals:     viewer.contextGlobal(),
		Pathbuilder: viewer.Pathbuilder,
	})
	if err != nil {
		panic(err)
	}
}

type htmlTipsyContext struct {
	Globals contextGlobal

	URL      template.HTMLAttr
	Data     template.HTMLAttr
	Filename template.HTMLAttr
}

func makeDataAttr(name string, value string) template.HTMLAttr {
	return template.HTMLAttr(` data-` + name + `="` + html.EscapeString(value) + `"`) // #nosec G203
}

func (viewer *Viewer) htmlTipsy(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	w.Header().Set("Content-Type", "text/html")

	xml, err := pbxml.Marshal(*viewer.Pathbuilder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = tipsyTemplate.Execute(w, htmlTipsyContext{
		Globals: viewer.contextGlobal(),

		URL:      makeDataAttr("url", viewer.RenderFlags.Tipsy()),
		Data:     makeDataAttr("data", string(xml)),
		Filename: makeDataAttr("filename", "hangover.xml"),
	})
	if err != nil {
		panic(err)
	}
}

func (viewer *Viewer) htmlEntityResolve(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	vars := mux.Vars(r)
	uri := impl.Label(strings.TrimSpace(vars["uri"]))

	bundle, ok := viewer.Cache.Bundle(uri)
	if !ok {
		http.NotFound(w, r)
		return
	}

	canon := viewer.Cache.Canonical(uri)

	// redirect to the entity
	target := "/entity/" + bundle + "?uri=" + url.PathEscape(string(canon))
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

func (viewer *Viewer) sendToResolver(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}
	publics := viewer.RenderFlags.PublicURLs(viewer.logPublicURI)
	uris := make([]impl.Label, 0, len(publics))
	for _, public := range publics {
		uri, err := url.JoinPath(public, r.URL.Path)
		if err != nil {
			continue
		}
		uris = append(uris, impl.Label(uri))
	}

	uri, _, ok := viewer.Cache.FirstBundle(uris...)
	if !ok {
		http.NotFound(w, r)
		return
	}

	target := "/wisski/get?uri=" + url.PathEscape(string(uri))
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

type htmlEntityContext struct {
	Bundle        *pathbuilder.Bundle
	Entity        *wisski.Entity
	DownloadLinks struct {
		Triples template.URL
		Turtle  template.URL
	}
	Aliases []impl.Label
	Globals contextGlobal
}

func (viewer *Viewer) htmlEntity(w http.ResponseWriter, r *http.Request) {
	if viewer.htmlFallback(w, r) {
		return
	}

	vars := mux.Vars(r)

	bundle, entity, ok := viewer.findEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return
	}

	var context htmlEntityContext

	context.Globals = viewer.contextGlobal()
	context.Bundle = bundle
	context.Entity = entity
	context.Aliases = viewer.Cache.Aliases(entity.URI)

	suffix := url.PathEscape(vars["bundle"]) + "?uri=" + url.QueryEscape(vars["uri"])

	context.DownloadLinks.Triples = template.URL("/api/v1/ntriples/" + suffix) // #nosec G203
	context.DownloadLinks.Turtle = template.URL("/api/v1/turtle/" + suffix)    // #nosec G203

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := entityTemplate.Execute(w, context)
	if err != nil {
		viewer.Stats.LogError("render entity", err)
	}
}
