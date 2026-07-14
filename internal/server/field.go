// HTTP handlers for /api/fields.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listFields serves GET /api/fields.
func (h *handlers) listFields(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	fields, apiError := api.ListFields(h.dataStore, store.FieldFilter{TournamentID: filters["tournament_id"]})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, fields)
}

// createField serves POST /api/fields.
func (h *handlers) createField(writer http.ResponseWriter, request *http.Request) {
	var patch api.FieldPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	field, apiError := api.CreateField(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, field)
}

// getField serves GET /api/fields/{id}.
func (h *handlers) getField(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	field, apiError := api.GetField(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, field)
}

// patchField serves PATCH /api/fields/{id}.
func (h *handlers) patchField(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.FieldPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	field, apiError := api.UpdateField(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, field)
}

// deleteField serves DELETE /api/fields/{id}.
func (h *handlers) deleteField(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteField(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
