// Team operations.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// TeamPatch carries the client-writable team fields.
type TeamPatch struct {
	TournamentID Opt[int64]  `json:"tournament_id"`
	DivisionID   Opt[int64]  `json:"division_id"`
	Name         Opt[string] `json:"name"`
	Country      Opt[string] `json:"country"`
	Contact      Opt[string] `json:"contact"`
	Notes        Opt[string] `json:"notes"`
	WithdrawnAt  Opt[string] `json:"withdrawn_at"`
}

// applyTeamPatch copies the set patch fields onto team.
func applyTeamPatch(team *store.Team, patch TeamPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &team.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	applyNullable(patch.DivisionID, &team.DivisionID)
	if applyError := applyValue(patch.Name, &team.Name, "name"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Country, &team.Country, "country"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Contact, &team.Contact, "contact"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Notes, &team.Notes, "notes"); applyError != nil {
		return applyError
	}
	if e := applyNullableString(patch.WithdrawnAt, &team.WithdrawnAt, "withdrawn_at", validNaiveTime); e != nil {
		return e
	}
	return nil
}

// ListTeams returns teams matching filter.
func ListTeams(dataStore *store.Store, filter store.TeamFilter) ([]store.Team, *Error) {
	teams, err := dataStore.ListTeams(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	return teams, nil
}

// CreateTeam creates a team from patch applied to an empty row.
func CreateTeam(dataStore *store.Store, patch TeamPatch) (*store.Team, *Error) {
	var team store.Team
	if applyError := applyTeamPatch(&team, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreateTeam(&team); err != nil {
		return nil, fromStore(err)
	}
	return &team, nil
}

// GetTeam returns one team by id.
func GetTeam(dataStore *store.Store, id int64) (*store.Team, *Error) {
	team, err := dataStore.GetTeam(id)
	if err != nil {
		return nil, notFoundOr("team", id, err)
	}
	return team, nil
}

// UpdateTeam applies patch to the stored team.
func UpdateTeam(dataStore *store.Store, id int64, patch TeamPatch) (*store.Team, *Error) {
	team, err := dataStore.GetTeam(id)
	if err != nil {
		return nil, notFoundOr("team", id, err)
	}
	if applyError := applyTeamPatch(team, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateTeam(team); err != nil {
		return nil, notFoundOr("team", id, err)
	}
	return team, nil
}

// DeleteTeam deletes one team by id.
func DeleteTeam(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteTeam(id); err != nil {
		return notFoundOr("team", id, err)
	}
	return nil
}
