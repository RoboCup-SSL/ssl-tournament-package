// Field-booking operations.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// BookingPatch carries the client-writable field-booking fields.
type BookingPatch struct {
	TournamentID Opt[int64]  `json:"tournament_id"`
	FieldID      Opt[int64]  `json:"field_id"`
	TeamID       Opt[int64]  `json:"team_id"`
	Kind         Opt[string] `json:"kind"`
	Label        Opt[string] `json:"label"`
	StartsAt     Opt[string] `json:"starts_at"`
	EndsAt       Opt[string] `json:"ends_at"`
	Notes        Opt[string] `json:"notes"`
}

// applyBookingPatch copies the set patch fields onto booking.
func applyBookingPatch(booking *store.FieldBooking, patch BookingPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &booking.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	applyNullable(patch.FieldID, &booking.FieldID)
	applyNullable(patch.TeamID, &booking.TeamID)
	if applyError := applyValue(patch.Kind, &booking.Kind, "kind"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Label, &booking.Label, "label"); applyError != nil {
		return applyError
	}
	applyNullable(patch.StartsAt, &booking.StartsAt)
	applyNullable(patch.EndsAt, &booking.EndsAt)
	return applyValue(patch.Notes, &booking.Notes, "notes")
}

// ListBookings returns bookings matching filter.
func ListBookings(dataStore *store.Store, filter store.BookingFilter) ([]store.FieldBooking, *Error) {
	bookings, err := dataStore.ListBookings(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	return bookings, nil
}

// CreateBooking creates a booking from patch applied to a row that carries
// the schema default kind.
func CreateBooking(dataStore *store.Store, patch BookingPatch) (*store.FieldBooking, *Error) {
	booking := store.FieldBooking{Kind: "booking"}
	if applyError := applyBookingPatch(&booking, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreateBooking(&booking); err != nil {
		return nil, fromStore(err)
	}
	return &booking, nil
}

// GetBooking returns one booking by id.
func GetBooking(dataStore *store.Store, id int64) (*store.FieldBooking, *Error) {
	booking, err := dataStore.GetBooking(id)
	if err != nil {
		return nil, notFoundOr("field booking", id, err)
	}
	return booking, nil
}

// UpdateBooking applies patch to the stored booking.
func UpdateBooking(dataStore *store.Store, id int64, patch BookingPatch) (*store.FieldBooking, *Error) {
	booking, err := dataStore.GetBooking(id)
	if err != nil {
		return nil, notFoundOr("field booking", id, err)
	}
	if applyError := applyBookingPatch(booking, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateBooking(booking); err != nil {
		return nil, notFoundOr("field booking", id, err)
	}
	return booking, nil
}

// DeleteBooking deletes one booking by id.
func DeleteBooking(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteBooking(id); err != nil {
		return notFoundOr("field booking", id, err)
	}
	return nil
}
