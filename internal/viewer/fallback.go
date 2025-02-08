package viewer

// fallback implements an api that implements the current status

import (
	"encoding/json"
	"html/template"
	"net/http"

	_ "embed"

	"github.com/FAU-CDI/hangover/internal/assets"
	"github.com/FAU-CDI/hangover/internal/stats"
)

//go:embed templates/loading.html
var loadingHTML string

var loadTemplate *template.Template = assets.Assetshangover_fallback.MustParseShared(
	"loading.html",
	loadingHTML,
	contextTemplateFuncs,
)

type htmlLoadingContext struct {
	Globals  contextGlobal
	Progress stats.Progress
}

func (viewer *Viewer) htmlFallback(w http.ResponseWriter, _ *http.Request) (sent bool) {
	progress := viewer.Stats.Progress()
	if progress.Done {
		return false
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Retry-After", viewerRetrySeconds)
	err := loadTemplate.Execute(w, htmlLoadingContext{
		Globals:  viewer.contextGlobal(),
		Progress: progress,
	})
	if err != nil {
		viewer.RenderFlags.Stats.LogError("render fallback", err)
	}

	return true
}

const (
	viewerNotReady     = "data is still being loaded and the server is not ready"
	viewerRetrySeconds = "60"
)

// ProgressMessage is returned by the viewer when the progress is not available
type ProgressMessage struct {
	Message  string
	Progress stats.Progress
}

func (viewer *Viewer) jsonFallback(w http.ResponseWriter, _ *http.Request) (sent bool) {
	progress := viewer.Stats.Progress()
	if progress.Done {
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", viewerRetrySeconds)
	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(ProgressMessage{
		Message:  viewerNotReady,
		Progress: progress,
	})

	return
}
