// Command hangover implements a WissKI Viewer
package main

// cspell:words WissKI

import (
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"

	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/glass"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/stats"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/tkw1536/pkglib/perf"
)

var handler = viewer.NewViewer(os.Stderr)

func main() {
	if debugServer != "" {
		go listenDebug()
	}

	if len(nArgs) == 0 || len(nArgs) > 2 {
		handler.Stats.Log("Usage: hangover [-help] [...flags] [/path/to/pathbuilder /path/to/nquads | /path/to/export]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var listener net.Listener
	var err error

	// prepare the handler
	handler.RenderFlags = flags
	handler.Footer = template.HTML(footerHTML)

	// create a channel to wait for being done listening
	done := make(chan struct{})

	// start listening, so that even during loading we are not performing that badly
	if !benchMode {
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			handler.Stats.LogError("listen", err)
			os.Exit(1)
		}
		handler.Stats.Log("listen", "addr", addr)
	}

	if !benchMode {
		go func() {
			defer close(done)
			http.Serve(listener, handler)
		}()
	} else {
		close(done)
	}

	// find the paths
	nq, pb, err := hangover.FindSource(nArgs...)
	if err != nil {
		handler.Stats.LogFatal("find source", err)
	}

	// create a new glass
	var drincw glass.Glass
	handler.Stats.Log("loading files", "pathbuilder", pb, "nquads", nq)
	drincw, err = glass.Create(pb, nq, cache, flags, handler.Stats)
	if err != nil {
		handler.Stats.LogFatal("unable to load or make index", err)
	}

	// otherwise create a viewer
	defer handler.Close()

	handler.Stats.DoStage(stats.StageHandler, func() error {
		handler.Prepare(drincw.Cache, &drincw.Pathbuilder)
		return nil
	})

	handler.Stats.Log("finished", "took", handler.Stats.Diff(), "now", perf.Now())

	<-done
}

var nArgs []string

var addr string = ":3000"

var flags viewer.RenderFlags
var sameAs string = string(wisski.DefaultSameAsProperties)
var inverseOf string = string(wisski.InverseOf)

var footerHTML string = "powered by <a href='https://github.com/FAU-CDI/hangover' target='_blank' rel='noopener noreferer'>hangover</a>. "

var cache string
var debugServer string
var benchMode bool

func init() {
	var legalFlag bool = false
	flag.BoolVar(&legalFlag, "legal", legalFlag, "Display legal notices and exit")
	defer func() {
		if legalFlag {
			fmt.Print(hangover.LegalText())
			os.Exit(0)
		}
	}()

	flag.StringVar(&addr, "addr", addr, "Start up a server at the given address")
	flag.BoolVar(&flags.ImageRender, "images", flags.ImageRender, "Enable rendering of images")
	flag.BoolVar(&flags.HTMLRender, "html", flags.HTMLRender, "Enable rendering of html")
	flag.StringVar(&flags.PublicURL, "public", flags.PublicURL, "Public URL of the wisski the data comes from")
	flag.StringVar(&sameAs, "sameas", sameAs, "SameAs Properties")
	flag.StringVar(&inverseOf, "inverseof", inverseOf, "InverseOf Properties")
	flag.StringVar(&cache, "cache", cache, "During indexing, cache data in the given directory as opposed to memory")
	flag.StringVar(&debugServer, "debug-listen", debugServer, "start a profiling server on the given address")
	flag.StringVar(&footerHTML, "footer", footerHTML, "html to include in footer of every page")
	flag.BoolVar(&flags.StrictCSP, "strict-csp", flags.StrictCSP, "include a strict csp header in every page")
	flag.BoolVar(&benchMode, "bench", benchMode, "benchmarking mode: only load for statistics and exit")
	flag.StringVar(&flags.TipsyURL, "tipsy", flags.TipsyURL, "embed a tipsy at the given url. Must start with 'http://' or 'https://'")

	flag.Parse()
	nArgs = flag.Args()

	// setup predicates
	flags.Predicates.SameAs = sparkl.ParsePredicateString(sameAs)
	flags.Predicates.InverseOf = sparkl.ParsePredicateString(inverseOf)
}
