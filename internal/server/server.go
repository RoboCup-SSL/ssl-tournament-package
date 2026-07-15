// Package server runs the HTTP API and embedded web UI shared by both binaries.
package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// NewMux returns the HTTP routes: /healthz (reports database reachability),
// /api/version, and the embedded web UI at /.
func NewMux(version string, dataStore *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := dataStore.Ping(); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/version", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(writer).Encode(map[string]string{"version": version})
	})
	register(mux, &handlers{dataStore: dataStore})
	registerDocs(mux)
	mux.Handle("/", frontend.Handler())
	return mux
}

// Run starts the HTTP server on host:port and blocks until it exits.
func Run(host, port, version string, dataStore *store.Store) error {
	addr := host + ":" + port
	log.Printf("ssl-tournament %s serving on http://%s", version, addr)
	return http.ListenAndServe(addr, NewMux(version, dataStore))
}
