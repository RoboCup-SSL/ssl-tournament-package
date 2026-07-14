// HTTP handlers for /api/teams.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listTeams serves GET /api/teams.
func (h *handlers) listTeams(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id", "division_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	teams, apiError := api.ListTeams(h.dataStore, store.TeamFilter{TournamentID: filters["tournament_id"], DivisionID: filters["division_id"]})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, teams)
}

// createTeam serves POST /api/teams.
func (h *handlers) createTeam(writer http.ResponseWriter, request *http.Request) {
	var patch api.TeamPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	team, apiError := api.CreateTeam(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, team)
}

// getTeam serves GET /api/teams/{id}.
func (h *handlers) getTeam(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	team, apiError := api.GetTeam(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, team)
}

// patchTeam serves PATCH /api/teams/{id}.
func (h *handlers) patchTeam(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.TeamPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	team, apiError := api.UpdateTeam(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, team)
}

// deleteTeam serves DELETE /api/teams/{id}.
func (h *handlers) deleteTeam(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteTeam(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
