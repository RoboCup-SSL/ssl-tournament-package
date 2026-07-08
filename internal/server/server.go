// Package server runs the HTTP API and embedded web UI shared by both binaries.
package server

import (
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
)

// Run starts the HTTP server on host:port and blocks until it exits.
func Run(host, port, version string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", frontend.Handler())

	addr := host + ":" + port
	log.Printf("ssl-tournament %s serving on http://%s", version, addr)
	return http.ListenAndServe(addr, mux)
}
