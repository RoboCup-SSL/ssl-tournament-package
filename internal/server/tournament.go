// HTTP handlers for /api/tournaments.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
)

// listTournaments serves GET /api/tournaments.
func (h *handlers) listTournaments(writer http.ResponseWriter, request *http.Request) {
	if _, filterError := queryFilters(request); filterError != nil {
		writeError(writer, filterError)
		return
	}
	tournaments, apiError := api.ListTournaments(h.dataStore)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, tournaments)
}

// createTournament serves POST /api/tournaments.
func (h *handlers) createTournament(writer http.ResponseWriter, request *http.Request) {
	var patch api.TournamentPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	tournament, apiError := api.CreateTournament(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, tournament)
}

// getTournament serves GET /api/tournaments/{id}.
func (h *handlers) getTournament(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	tournament, apiError := api.GetTournament(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, tournament)
}

// patchTournament serves PATCH /api/tournaments/{id}.
func (h *handlers) patchTournament(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.TournamentPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	tournament, apiError := api.UpdateTournament(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, tournament)
}

// deleteTournament serves DELETE /api/tournaments/{id}.
func (h *handlers) deleteTournament(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteTournament(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
