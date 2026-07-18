// Tests for FieldBooking patch validation.
package api

import "testing"

func TestBookingScheduleValidation(t *testing.T) {
	store := newTestStore(t)
	tournament, err := CreateTournament(store, TournamentPatch{Name: set("Cup")})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	booking, err := CreateBooking(store, BookingPatch{
		TournamentID: Opt[int64]{Set: true, Value: tournament.ID},
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}

	if _, e := UpdateBooking(store, booking.ID, BookingPatch{StartsAt: set("2026-07-15T14:00:00+09:00")}); e == nil || e.Code != CodeInvalidValue || e.Field != "starts_at" {
		t.Fatalf("offset should be rejected on starts_at, got %+v", e)
	}
	if _, e := UpdateBooking(store, booking.ID, BookingPatch{EndsAt: set("2026-07-15T14:00:00+09:00")}); e == nil || e.Code != CodeInvalidValue || e.Field != "ends_at" {
		t.Fatalf("offset should be rejected on ends_at, got %+v", e)
	}

	ok, e := UpdateBooking(store, booking.ID, BookingPatch{
		StartsAt: set("2026-07-15T14:00"), EndsAt: set("2026-07-15T15:00"),
	})
	if e != nil {
		t.Fatalf("good update: %v", e)
	}
	if ok.StartsAt == nil || *ok.StartsAt != "2026-07-15T14:00" {
		t.Fatalf("starts_at not stored canonical: %+v", ok)
	}
	if ok.EndsAt == nil || *ok.EndsAt != "2026-07-15T15:00" {
		t.Fatalf("ends_at not stored canonical: %+v", ok)
	}
}
