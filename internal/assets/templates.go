package assets

import (
	"embed"
	"html/template"
)

//go:embed "templates/*.html"
var templates embed.FS

var (
	shared *template.Template = template.Must(
		template.New("").Funcs(template.FuncMap{
			"renderhtml": func(args ...any) any { panic("not implemented") },
			"combine":    func(args ...any) any { panic("not implemented") },
		}).ParseFS(templates, "templates/*.html"),
	)
)

// NewSharedTemplate creates a new template with the given name.
// It will be able to make use of shared templates as well as functions.
func NewSharedTemplate(name string, funcMap template.FuncMap) *template.Template {
	new := template.New(name)
	new.Funcs(funcMap)
	for _, template := range shared.Templates() {
		if template != nil && template.Tree != nil {
			new.AddParseTree(template.Tree.Name, template.Tree.Copy())
		}
	}
	return new
}
