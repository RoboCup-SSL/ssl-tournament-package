// Package server wires up the HTTP API + embedded web UI. Both the plain
// ssl-tournament binary and the ssl-tournament-service binary run it, so the
// serving behavior is identical no matter how it was started.
package server

import (
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
)

// Run starts the HTTP server and blocks. version is used only for logging.
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
