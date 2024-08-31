package headache

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
)

// Settings represent bound settings
type settings struct {
	addr binding.String

	nquads      binding.String
	pathbuilder binding.String

	public binding.String
	images binding.Bool
	html   binding.Bool

	sameAs    binding.String
	inverseOf binding.String

	tipsy binding.String
}

// Addr returns the address to listen on
func (settings *settings) Addr() string {
	addr, _ := settings.addr.Get()
	return addr
}

func (settings *settings) Pathbuilder() (pb string) {
	pb, _ = settings.pathbuilder.Get()
	return
}

func (settings *settings) Nquads() (nq string) {
	nq, _ = settings.nquads.Get()
	return
}

// Flags returns the flags to use for the viewer
func (settings *settings) Flags() (flags viewer.RenderFlags) {
	sa, _ := settings.sameAs.Get()
	io, _ := settings.inverseOf.Get()

	flags.Predicates.SameAs = sparkl.ParsePredicateString(sa)
	flags.Predicates.InverseOf = sparkl.ParsePredicateString(io)

	flags.ImageRender, _ = settings.images.Get()
	flags.HTMLRender, _ = settings.html.Get()
	flags.PublicURL, _ = settings.public.Get()

	flags.TipsyURL, _ = settings.tipsy.Get()

	return flags
}

// newSettings creates new bindings and sets them to their default values.
func newSettings() (s settings) {
	s.addr = binding.NewString()
	s.addr.Set("127.0.0.1:8000")

	s.nquads = binding.NewString()
	s.pathbuilder = binding.NewString()

	s.public = binding.NewString()
	s.images = binding.NewBool()
	s.html = binding.NewBool()

	s.sameAs = binding.NewString()
	s.sameAs.Set(string(wisski.DefaultSameAsProperties))

	s.inverseOf = binding.NewString()
	s.inverseOf.Set(string(wisski.InverseOf))

	s.tipsy = binding.NewString()
	s.tipsy.Set("https://tipsy.guys.wtf")

	return
}
