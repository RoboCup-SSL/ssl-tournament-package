// HTTP handlers for /api/groups.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listGroups serves GET /api/groups.
func (h *handlers) listGroups(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id", "division_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	groups, apiError := api.ListGroups(h.dataStore, store.GroupFilter{
		TournamentID: filters["tournament_id"],
		DivisionID:   filters["division_id"]})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, groups)
}

// createGroup serves POST /api/groups.
func (h *handlers) createGroup(writer http.ResponseWriter, request *http.Request) {
	var patch api.GroupPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	group, apiError := api.CreateGroup(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, group)
}

// getGroup serves GET /api/groups/{id}.
func (h *handlers) getGroup(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	group, apiError := api.GetGroup(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, group)
}

// patchGroup serves PATCH /api/groups/{id}.
func (h *handlers) patchGroup(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.GroupPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	group, apiError := api.UpdateGroup(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, group)
}

// deleteGroup serves DELETE /api/groups/{id}.
func (h *handlers) deleteGroup(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteGroup(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
