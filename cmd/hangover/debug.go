//spellchecker:words main
package main

//spellchecker:words http pprof time github hangover internal viewer gorilla
import (
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/FAU-CDI/hangover/internal/viewer"
	"github.com/gorilla/mux"
)

func listenDebug(handler *viewer.Viewer) {
	router := mux.NewRouter()
	router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	router.Handle("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index)) // special handling for Gorilla mux

	handler.Stats.Log("debug server listening", "addr", debugServer)

	server := http.Server{
		Addr:              debugServer,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	err := server.ListenAndServe()

	handler.Stats.LogFatal("pprof server listen", err)
}
