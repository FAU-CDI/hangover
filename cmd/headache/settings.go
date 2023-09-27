package main

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// Settings represent bound settings
type Settings struct {
	addr binding.String

	nquads      binding.String
	pathbuilder binding.String

	images binding.Bool
	html   binding.Bool

	sameAs    binding.String
	inverseOf binding.String
}

// Addr returns the address to listen on
func (settings *Settings) Addr() string {
	addr, _ := settings.addr.Get()
	return addr
}

// Flags returns the flags to use for the viewer
func (settings *Settings) Flags() (flags viewer.RenderFlags) {
	sa, _ := settings.sameAs.Get()
	io, _ := settings.inverseOf.Get()
	sparkl.ParsePredicateString(&flags.Predicates.SameAs, sa)
	sparkl.ParsePredicateString(&flags.Predicates.InverseOf, io)

	flags.ImageRender, _ = settings.images.Get()
	flags.HTMLRender, _ = settings.html.Get()

	return flags
}

// Reset creates new bindings and sets them to their default values.
func NewSettings() (settings Settings) {
	settings.addr = binding.NewString()
	settings.addr.Set("127.0.0.1:8000")

	settings.nquads = binding.NewString()
	settings.pathbuilder = binding.NewString()

	settings.images = binding.NewBool()
	settings.html = binding.NewBool()

	settings.sameAs = binding.NewString()
	settings.sameAs.Set(string(wisski.SameAs))

	settings.inverseOf = binding.NewString()
	settings.inverseOf.Set(string(wisski.InverseOf))

	return
}
