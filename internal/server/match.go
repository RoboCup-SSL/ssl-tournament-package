// HTTP handlers for /api/matches.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listMatches serves GET /api/matches.
func (h *handlers) listMatches(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id", "division_id", "group_id", "field_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	matches, apiError := api.ListMatches(h.dataStore, store.MatchFilter{
		TournamentID: filters["tournament_id"],
		DivisionID:   filters["division_id"],
		GroupID:      filters["group_id"],
		FieldID:      filters["field_id"]})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, matches)
}

// createMatch serves POST /api/matches.
func (h *handlers) createMatch(writer http.ResponseWriter, request *http.Request) {
	var patch api.MatchPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	match, apiError := api.CreateMatch(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, match)
}

// getMatch serves GET /api/matches/{id}.
func (h *handlers) getMatch(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	match, apiError := api.GetMatch(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, match)
}

// patchMatch serves PATCH /api/matches/{id}.
func (h *handlers) patchMatch(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.MatchPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	match, apiError := api.UpdateMatch(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, match)
}

// deleteMatch serves DELETE /api/matches/{id}.
func (h *handlers) deleteMatch(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteMatch(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
