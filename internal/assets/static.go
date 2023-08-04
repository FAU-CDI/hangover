// Package assets implements serving of fully static resources
package assets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var staticFS embed.FS

// AssetHandler handles serving static files under the /assets/ route
var AssetHandler http.Handler

func init() {
	// take the filesystem
	fs, err := fs.Sub(staticFS, "dist")
	if err != nil {
		panic("AssetHandler: Unable to init")
	}

	// and serve it
	AssetHandler = http.StripPrefix("/assets/", http.FileServer(http.FS(fs)))
}
