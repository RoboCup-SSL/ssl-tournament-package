// Package server runs the HTTP API and embedded web UI shared by both binaries.
package server

import (
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// NewMux returns the HTTP routes: /healthz (reports database reachability)
// and the embedded web UI at /.
func NewMux(dataStore *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := dataStore.Ping(); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	register(mux, &handlers{dataStore: dataStore})
	mux.Handle("/", frontend.Handler())
	return mux
}

// Run starts the HTTP server on host:port and blocks until it exits.
func Run(host, port, version string, dataStore *store.Store) error {
	addr := host + ":" + port
	log.Printf("ssl-tournament %s serving on http://%s", version, addr)
	return http.ListenAndServe(addr, NewMux(dataStore))
}
