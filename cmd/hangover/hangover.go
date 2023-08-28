// Command hangover implements a WissKI Viewer
package main

// cspell:words WissKI

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/FAU-CDI/hangover"
	"github.com/FAU-CDI/hangover/internal/glass"
	"github.com/FAU-CDI/hangover/internal/sparkl"
	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/FAU-CDI/hangover/internal/wisski"
	"github.com/FAU-CDI/hangover/pkg/perf"
)

func main() {
	if debugServer != "" {
		go listenDebug()
	}

	if len(nArgs) == 0 || len(nArgs) > 2 {
		log.Print("Usage: hangover [-help] [...flags] [/path/to/pathbuilder /path/to/nquads | /path/to/export]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var listener net.Listener
	var err error

	// start listening, so that even during loading we are not performing that badly
	if export == "" && !benchMode {
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("listening on", addr)
	}

	// find the paths
	nq, pb, idx, err := hangover.FindSource(true, nArgs...)
	if err != nil {
		panic(err)
	}

	// start tracking performance for everything
	loadStart := perf.Now()

	var drincw glass.Glass
	if idx == "" {
		log.Printf("pathbuilder=%q nquads=%q", pb, nq)
		sparkl.ParsePredicateString(&flags.Predicates.SameAs, sameAs)
		sparkl.ParsePredicateString(&flags.Predicates.InverseOf, inverseOf)

		drincw, err = glass.Create(pb, nq, cache, flags, os.Stderr)
	} else {
		log.Printf("index=%q", idx)
		drincw, err = glass.Import(idx, os.Stderr)
	}
	if err != nil {
		return
	}

	// create an export if requested
	if export != "" {
		log.Printf("exporting to %s", export)
		glass.Export(export, drincw, os.Stderr)
		return
	}

	// otherwise create a viewer
	var handler viewer.Viewer
	defer handler.Close()

	var handlerPerf perf.Diff
	{
		start := perf.Now()

		handler = viewer.Viewer{
			Cache:       drincw.Cache,
			Pathbuilder: &drincw.Pathbuilder,
			RenderFlags: flags,
		}
		handler.Prepare()
		handlerPerf = perf.Since(start)
		log.Printf("built handler, took %s", handlerPerf)
	}

	loadPerformance := perf.Since(loadStart)

	log.Printf("loading overall took %s", loadPerformance)
	log.Println(perf.Now())

	if benchMode {
		return
	}

	http.Serve(listener, &handler)
}

var nArgs []string

var addr string = ":3000"

var flags viewer.RenderFlags
var sameAs string = string(wisski.SameAs)
var inverseOf string = string(wisski.InverseOf)

var cache string
var export string
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

	flag.StringVar(&addr, "addr", addr, "Instead of dumping data as json, start up a server at the given address")
	flag.BoolVar(&flags.ImageRender, "images", flags.ImageRender, "Enable rendering of images")
	flag.BoolVar(&flags.HTMLRender, "html", flags.HTMLRender, "Enable rendering of html")
	flag.StringVar(&flags.PublicURL, "public", flags.PublicURL, "Public URL of the wisski the data comes from")
	flag.StringVar(&sameAs, "sameas", sameAs, "SameAs Properties")
	flag.StringVar(&inverseOf, "inverseof", inverseOf, "InverseOf Properties")
	flag.StringVar(&cache, "cache", cache, "During indexing, cache data in the given directory as opposed to memory")
	flag.StringVar(&debugServer, "debug-listen", debugServer, "start a profiling server on the given address")
	flag.StringVar(&export, "export", export, "export completed index to path and exit")
	flag.BoolVar(&benchMode, "bench", benchMode, "benchmarking mode: only load for statistics and exit")

	flag.Parse()
	nArgs = flag.Args()
}
