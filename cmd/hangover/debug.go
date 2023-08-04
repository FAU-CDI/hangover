package main

import (
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
)

func listenDebug() {
	router := mux.NewRouter()
	router.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	router.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	router.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	router.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	router.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	router.Handle("/debug/pprof/{cmd}", http.HandlerFunc(pprof.Index)) // special handling for Gorilla mux

	log.Printf("debug server listening on %s", debugServer)
	err := http.ListenAndServe(debugServer, router)
	log.Printf("pprof server listen failed: %v", err)
}
