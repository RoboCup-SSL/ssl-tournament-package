// The API route table: single source for mux registration and the OpenAPI
// sync test.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// handlers carries the shared dependencies of every HTTP handler.
type handlers struct {
	dataStore *store.Store
}

// route pairs one HTTP method and path pattern with its handler.
type route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

// routes lists every JSON API route.
func (h *handlers) routes() []route {
	return []route{
		{"GET", "/api/tournaments", h.listTournaments},
		{"POST", "/api/tournaments", h.createTournament},
		{"GET", "/api/tournaments/{id}", h.getTournament},
		{"PATCH", "/api/tournaments/{id}", h.patchTournament},
		{"DELETE", "/api/tournaments/{id}", h.deleteTournament},
	}
}

// register adds every API route to mux.
func register(mux *http.ServeMux, h *handlers) {
	for _, apiRoute := range h.routes() {
		mux.HandleFunc(apiRoute.Method+" "+apiRoute.Pattern, apiRoute.Handler)
	}
}
