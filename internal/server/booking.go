// HTTP handlers for /api/field-bookings.
package server

import (
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// listBookings serves GET /api/field-bookings.
func (h *handlers) listBookings(writer http.ResponseWriter, request *http.Request) {
	filters, filterError := queryFilters(request, "tournament_id", "field_id", "team_id")
	if filterError != nil {
		writeError(writer, filterError)
		return
	}
	bookings, apiError := api.ListBookings(h.dataStore, store.BookingFilter{
		TournamentID: filters["tournament_id"],
		FieldID:      filters["field_id"],
		TeamID:       filters["team_id"],
	})
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, bookings)
}

// createBooking serves POST /api/field-bookings.
func (h *handlers) createBooking(writer http.ResponseWriter, request *http.Request) {
	var patch api.BookingPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	booking, apiError := api.CreateBooking(h.dataStore, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusCreated, booking)
}

// getBooking serves GET /api/field-bookings/{id}.
func (h *handlers) getBooking(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	booking, apiError := api.GetBooking(h.dataStore, id)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, booking)
}

// patchBooking serves PATCH /api/field-bookings/{id}.
func (h *handlers) patchBooking(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	var patch api.BookingPatch
	if decodeError := decode(request, &patch); decodeError != nil {
		writeError(writer, decodeError)
		return
	}
	booking, apiError := api.UpdateBooking(h.dataStore, id, patch)
	if apiError != nil {
		writeError(writer, apiError)
		return
	}
	writeJSON(writer, http.StatusOK, booking)
}

// deleteBooking serves DELETE /api/field-bookings/{id}.
func (h *handlers) deleteBooking(writer http.ResponseWriter, request *http.Request) {
	id, idError := pathID(request)
	if idError != nil {
		writeError(writer, idError)
		return
	}
	if apiError := api.DeleteBooking(h.dataStore, id); apiError != nil {
		writeError(writer, apiError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
