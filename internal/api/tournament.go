// Tournament operations — the per-entity pattern every resource follows.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// TournamentPatch carries the client-writable tournament fields.
type TournamentPatch struct {
	Name                Opt[string] `json:"name"`
	Location            Opt[string] `json:"location"`
	StartsOn            Opt[string] `json:"starts_on"`
	EndsOn              Opt[string] `json:"ends_on"`
	VenueOpens          Opt[string] `json:"venue_opens"`
	VenueCloses         Opt[string] `json:"venue_closes"`
	DefaultMatchMinutes Opt[int64]  `json:"default_match_minutes"`
	DefaultGapMinutes   Opt[int64]  `json:"default_gap_minutes"`
	TimeZone            Opt[string] `json:"time_zone"`
}

// applyTournamentPatch copies the set patch fields onto tournament.
func applyTournamentPatch(tournament *store.Tournament, patch TournamentPatch) *Error {
	if applyError := applyValue(patch.Name, &tournament.Name, "name"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Location, &tournament.Location, "location"); applyError != nil {
		return applyError
	}
	if applyError := applyNullableString(patch.StartsOn, &tournament.StartsOn, "starts_on", validDate); applyError != nil {
		return applyError
	}
	if applyError := applyNullableString(patch.EndsOn, &tournament.EndsOn, "ends_on", validDate); applyError != nil {
		return applyError
	}
	if applyError := applyNullableString(patch.VenueOpens, &tournament.VenueOpens, "venue_opens", validClock); applyError != nil {
		return applyError
	}
	if applyError := applyNullableString(patch.VenueCloses, &tournament.VenueCloses, "venue_closes", validClock); applyError != nil {
		return applyError
	}
	if applyError := applyNullableString(patch.TimeZone, &tournament.TimeZone, "time_zone", validZone); applyError != nil {
		return applyError
	}
	applyNullable(patch.DefaultMatchMinutes, &tournament.DefaultMatchMinutes)
	applyNullable(patch.DefaultGapMinutes, &tournament.DefaultGapMinutes)
	return nil
}

// ListTournaments returns every tournament.
func ListTournaments(dataStore *store.Store) ([]store.Tournament, *Error) {
	tournaments, err := dataStore.ListTournaments()
	if err != nil {
		return nil, fromStore(err)
	}
	return tournaments, nil
}

// CreateTournament creates a tournament from patch applied to an empty row.
func CreateTournament(dataStore *store.Store, patch TournamentPatch) (*store.Tournament, *Error) {
	var tournament store.Tournament
	if applyError := applyTournamentPatch(&tournament, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreateTournament(&tournament); err != nil {
		return nil, fromStore(err)
	}
	return &tournament, nil
}

// GetTournament returns one tournament by id.
func GetTournament(dataStore *store.Store, id int64) (*store.Tournament, *Error) {
	tournament, err := dataStore.GetTournament(id)
	if err != nil {
		return nil, notFoundOr("tournament", id, err)
	}
	return tournament, nil
}

// UpdateTournament applies patch to the stored tournament.
func UpdateTournament(dataStore *store.Store, id int64, patch TournamentPatch) (*store.Tournament, *Error) {
	tournament, err := dataStore.GetTournament(id)
	if err != nil {
		return nil, notFoundOr("tournament", id, err)
	}
	if applyError := applyTournamentPatch(tournament, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateTournament(tournament); err != nil {
		return nil, notFoundOr("tournament", id, err)
	}
	return tournament, nil
}

// DeleteTournament deletes one tournament by id.
func DeleteTournament(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteTournament(id); err != nil {
		return notFoundOr("tournament", id, err)
	}
	return nil
}
