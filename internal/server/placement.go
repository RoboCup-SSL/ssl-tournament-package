// HTTP handlers for /api/placements.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listPlacements serves GET /api/placements.
func (h *handlers) listPlacements(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id", "division_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	placements, apiError := api.ListPlacements(h.dataStore, store.PlacementFilter{
		TournamentID: filters["tournament_id"],
		DivisionID:   filters["division_id"],
	})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, placements)
}

// createPlacement serves POST /api/placements.
func (h *handlers) createPlacement(writer http.ResponseWriter, request *http.Request) {
	var patch api.PlacementPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	placement, apiError := api.CreatePlacement(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, placement)
}

// getPlacement serves GET /api/placements/{id}.
func (h *handlers) getPlacement(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	placement, apiError := api.GetPlacement(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, placement)
}

// patchPlacement serves PATCH /api/placements/{id}.
func (h *handlers) patchPlacement(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.PlacementPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	placement, apiError := api.UpdatePlacement(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, placement)
}

// deletePlacement serves DELETE /api/placements/{id}.
func (h *handlers) deletePlacement(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeletePlacement(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
