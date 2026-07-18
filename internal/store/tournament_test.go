package store

import (
	"errors"
	"testing"
)

// TestTournamentCRUD exercises the full create/get/list/delete cycle.
func TestTournamentCRUD(t *testing.T) {
	s := openTestStore(t)

	mins := int64(60)
	opens := "08:00"
	tr := &Tournament{Name: "RC2026", Location: "Incheon",
		DefaultMatchMinutes: &mins, VenueOpens: &opens}
	if err := s.CreateTournament(tr); err != nil {
		t.Fatal(err)
	}
	if tr.ID == 0 {
		t.Fatal("ID not filled after create")
	}
	if tr.CreatedAt == "" {
		t.Fatal("CreatedAt not filled after create")
	}

	got, err := s.GetTournament(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "RC2026" || got.Location != "Incheon" {
		t.Fatalf("got %q/%q, want RC2026/Incheon", got.Name, got.Location)
	}
	if got.DefaultMatchMinutes == nil || *got.DefaultMatchMinutes != 60 {
		t.Fatalf("DefaultMatchMinutes = %v, want 60", got.DefaultMatchMinutes)
	}
	if got.VenueOpens == nil || *got.VenueOpens != "08:00" {
		t.Fatalf("VenueOpens = %v, want 08:00", got.VenueOpens)
	}
	if got.StartsOn != nil {
		t.Fatalf("StartsOn = %v, want nil", got.StartsOn)
	}

	list, err := s.ListTournaments()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != tr.ID {
		t.Fatalf("list = %+v, want the one created tournament", list)
	}

	if err := s.DeleteTournament(tr.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTournament(tr.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteTournament(tr.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: err = %v, want ErrNotFound", err)
	}
}

// TestTournamentTimeZoneRoundTrip verifies time_zone is persisted and can be cleared.
func TestTournamentTimeZoneRoundTrip(t *testing.T) {
	store := openTestStore(t)
	zone := "Asia/Seoul"
	tournament := &Tournament{Name: "TZ", TimeZone: &zone}
	if err := store.CreateTournament(tournament); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.GetTournament(tournament.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TimeZone == nil || *got.TimeZone != "Asia/Seoul" {
		t.Fatalf("time_zone = %v, want Asia/Seoul", got.TimeZone)
	}
	got.TimeZone = nil
	if err := store.UpdateTournament(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	again, _ := store.GetTournament(tournament.ID)
	if again.TimeZone != nil {
		t.Fatalf("time_zone = %v, want nil after clear", again.TimeZone)
	}
}
