// HTTP handlers for /api/divisions.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listDivisions serves GET /api/divisions.
func (h *handlers) listDivisions(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	divisions, apiError := api.ListDivisions(h.dataStore, store.DivisionFilter{TournamentID: filters["tournament_id"]})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, divisions)
}

// createDivision serves POST /api/divisions.
func (h *handlers) createDivision(writer http.ResponseWriter, request *http.Request) {
	var patch api.DivisionPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	division, apiError := api.CreateDivision(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, division)
}

// getDivision serves GET /api/divisions/{id}.
func (h *handlers) getDivision(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	division, apiError := api.GetDivision(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, division)
}

// patchDivision serves PATCH /api/divisions/{id}.
func (h *handlers) patchDivision(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.DivisionPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	division, apiError := api.UpdateDivision(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, division)
}

// deleteDivision serves DELETE /api/divisions/{id}.
func (h *handlers) deleteDivision(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteDivision(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
