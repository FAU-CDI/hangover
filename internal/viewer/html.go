package viewer

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	_ "embed"

	"github.com/FAU-CDI/drincw/pathbuilder"
	"github.com/FAU-CDI/hangover/internal/assets"
	"github.com/FAU-CDI/hangover/internal/triplestore/impl"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/htmlx"
	"github.com/gorilla/mux"
)

// cspell:words pathbuilder

var contextTemplateFuncs = template.FuncMap{
	"renderhtml": func(html string, globals contextGlobal) template.HTML {
		return template.HTML(htmlx.ReplaceLinks(html, globals.ReplaceURL))
	},
	"datum2string": func(datum impl.Datum) string {
		return string(datum)
	},
	"combine": func(pairs ...any) (map[string]any, error) {
		if len(pairs)%2 != 0 {
			return nil, errors.New("pairs must be of even length")
		}
		result := make(map[string]any, len(pairs)/2)
		for i, v := range pairs {
			if i%2 == 1 {
				result[pairs[(i-1)].(string)] = v
			}
		}
		return result, nil
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

var indexTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"index.html",
	indexHTML,
	contextTemplateFuncs,
)

//go:embed templates/pathbuilder.html
var pathbuilderHTML string

var pbTemplate *template.Template = assets.Assetshangover.MustParseShared(
	"pathbuilder.html",
	pathbuilderHTML,
	contextTemplateFuncs,
)

type contextGlobal struct {
	InterceptedPrefixes []string // urls that are redirected to this server
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
	global.RenderFlags = viewer.RenderFlags

	if viewer.RenderFlags.PublicURL == "" {
		return
	}

	for _, public := range viewer.RenderFlags.PublicURIS() {
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

func (viewer *Viewer) htmlIndex(w http.ResponseWriter, r *http.Request) {
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

type htmlBundleContext struct {
	Bundle  *pathbuilder.Bundle
	URIS    []impl.Label
	Globals contextGlobal
}

func (viewer *Viewer) htmlBundle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	bundle, entities, ok := viewer.getEntityURIs(vars["bundle"])
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := bundleTemplate.Execute(w, htmlBundleContext{
		Globals: viewer.contextGlobal(),
		Bundle:  bundle,
		URIS:    entities,
	})
	if err != nil {
		panic(err)
	}
}

type htmlPathbuilderContext struct {
	Pathbuilder *pathbuilder.Pathbuilder
	Globals     contextGlobal
}

func (viewer *Viewer) htmlPathbuilder(w http.ResponseWriter, r *http.Request) {
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

func (viewer *Viewer) htmlEntityResolve(w http.ResponseWriter, r *http.Request) {
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
	publics := viewer.RenderFlags.PublicURIS()
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
	Bundle  *pathbuilder.Bundle
	Entity  *wisski.Entity
	Aliases []impl.Label
	Globals contextGlobal
}

func (viewer *Viewer) htmlEntity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	bundle, entity, ok := viewer.findEntity(vars["bundle"], impl.Label(vars["uri"]))
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := entityTemplate.Execute(w, htmlEntityContext{
		Globals: viewer.contextGlobal(),

		Bundle:  bundle,
		Entity:  entity,
		Aliases: viewer.Cache.Aliases(entity.URI),
	})
	if err != nil {
		log.Println(err)
	}
}
