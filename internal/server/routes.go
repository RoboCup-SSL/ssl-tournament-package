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
		{"GET", "/api/divisions", h.listDivisions},
		{"POST", "/api/divisions", h.createDivision},
		{"GET", "/api/divisions/{id}", h.getDivision},
		{"PATCH", "/api/divisions/{id}", h.patchDivision},
		{"DELETE", "/api/divisions/{id}", h.deleteDivision},
		{"GET", "/api/teams", h.listTeams},
		{"POST", "/api/teams", h.createTeam},
		{"GET", "/api/teams/{id}", h.getTeam},
		{"PATCH", "/api/teams/{id}", h.patchTeam},
		{"DELETE", "/api/teams/{id}", h.deleteTeam},
		{"GET", "/api/fields", h.listFields},
		{"POST", "/api/fields", h.createField},
		{"GET", "/api/fields/{id}", h.getField},
		{"PATCH", "/api/fields/{id}", h.patchField},
		{"DELETE", "/api/fields/{id}", h.deleteField},
		{"GET", "/api/field-bookings", h.listBookings},
		{"POST", "/api/field-bookings", h.createBooking},
		{"GET", "/api/field-bookings/{id}", h.getBooking},
		{"PATCH", "/api/field-bookings/{id}", h.patchBooking},
		{"DELETE", "/api/field-bookings/{id}", h.deleteBooking},
		{"GET", "/api/placements", h.listPlacements},
		{"POST", "/api/placements", h.createPlacement},
		{"GET", "/api/placements/{id}", h.getPlacement},
		{"PATCH", "/api/placements/{id}", h.patchPlacement},
		{"DELETE", "/api/placements/{id}", h.deletePlacement},
	}
}

// register adds every API route to mux.
func register(mux *http.ServeMux, h *handlers) {
	for _, apiRoute := range h.routes() {
		mux.HandleFunc(apiRoute.Method+" "+apiRoute.Pattern, apiRoute.Handler)
	}
}
