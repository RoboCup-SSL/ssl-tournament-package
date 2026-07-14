// Match operations: the match resource embeds its slot sources as a_source
// and b_source.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// Match is the wire shape of a match: the row plus both slot sources.
type Match struct {
	store.Match
	ASource *store.SlotSource `json:"a_source"`
	BSource *store.SlotSource `json:"b_source"`
}

// MatchPatch carries the client-writable match fields; a_source and b_source
// replace wholesale.
type MatchPatch struct {
	TournamentID           Opt[int64]            `json:"tournament_id"`
	DivisionID             Opt[int64]            `json:"division_id"`
	GroupID                Opt[int64]            `json:"group_id"`
	Label                  Opt[string]           `json:"label"`
	FieldID                Opt[int64]            `json:"field_id"`
	ScheduledAt            Opt[string]           `json:"scheduled_at"`
	DurationMinutes        Opt[int64]            `json:"duration_minutes"`
	Status                 Opt[string]           `json:"status"`
	RefereeTeamID          Opt[int64]            `json:"referee_team_id"`
	AssistantRefereeTeamID Opt[int64]            `json:"assistant_referee_team_id"`
	ATeamID                Opt[int64]            `json:"a_team_id"`
	AScore                 Opt[int64]            `json:"a_score"`
	AFouls                 Opt[int64]            `json:"a_fouls"`
	AYellowCards           Opt[int64]            `json:"a_yellow_cards"`
	ARedCards              Opt[int64]            `json:"a_red_cards"`
	BTeamID                Opt[int64]            `json:"b_team_id"`
	BScore                 Opt[int64]            `json:"b_score"`
	BFouls                 Opt[int64]            `json:"b_fouls"`
	BYellowCards           Opt[int64]            `json:"b_yellow_cards"`
	BRedCards              Opt[int64]            `json:"b_red_cards"`
	WinnerTeamID           Opt[int64]            `json:"winner_team_id"`
	Notes                  Opt[string]           `json:"notes"`
	ASource                Opt[store.SlotSource] `json:"a_source"`
	BSource                Opt[store.SlotSource] `json:"b_source"`
}

// applyMatchPatch copies the set scalar patch fields onto match.
func applyMatchPatch(match *store.Match, patch MatchPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &match.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	applyNullable(patch.DivisionID, &match.DivisionID)
	applyNullable(patch.GroupID, &match.GroupID)
	if applyError := applyValue(patch.Label, &match.Label, "label"); applyError != nil {
		return applyError
	}
	applyNullable(patch.FieldID, &match.FieldID)
	applyNullable(patch.ScheduledAt, &match.ScheduledAt)
	applyNullable(patch.DurationMinutes, &match.DurationMinutes)
	if applyError := applyValue(patch.Status, &match.Status, "status"); applyError != nil {
		return applyError
	}
	applyNullable(patch.RefereeTeamID, &match.RefereeTeamID)
	applyNullable(patch.AssistantRefereeTeamID, &match.AssistantRefereeTeamID)
	applyNullable(patch.ATeamID, &match.ATeamID)
	applyNullable(patch.AScore, &match.AScore)
	applyNullable(patch.AFouls, &match.AFouls)
	applyNullable(patch.AYellowCards, &match.AYellowCards)
	applyNullable(patch.ARedCards, &match.ARedCards)
	applyNullable(patch.BTeamID, &match.BTeamID)
	applyNullable(patch.BScore, &match.BScore)
	applyNullable(patch.BFouls, &match.BFouls)
	applyNullable(patch.BYellowCards, &match.BYellowCards)
	applyNullable(patch.BRedCards, &match.BRedCards)
	applyNullable(patch.WinnerTeamID, &match.WinnerTeamID)
	return applyValue(patch.Notes, &match.Notes, "notes")
}

// slotSourceValue converts a source patch field into a create-time pointer.
func slotSourceValue(field Opt[store.SlotSource]) *store.SlotSource {
	if !field.Set || field.Null {
		return nil
	}
	source := field.Value
	return &source
}

// slotSourceUpdate converts a source patch field into the store's tri-state
// update.
func slotSourceUpdate(field Opt[store.SlotSource]) store.SlotSourceUpdate {
	if !field.Set {
		return store.SlotSourceUpdate{}
	}
	return store.SlotSourceUpdate{Replace: true, Source: slotSourceValue(field)}
}

// matchView assembles the wire shape.
func matchView(match *store.Match, sourceA, sourceB *store.SlotSource) *Match {
	return &Match{Match: *match, ASource: sourceA, BSource: sourceB}
}

// ListMatches returns matches matching filter, each with its slot sources.
func ListMatches(dataStore *store.Store, filter store.MatchFilter) ([]Match, *Error) {
	matchRows, err := dataStore.ListMatchRows(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	matches := make([]Match, 0, len(matchRows))
	for index := range matchRows {
		sourceA, err := dataStore.MatchSlotSource(matchRows[index].ID, "a")
		if err != nil {
			return nil, fromStore(err)
		}
		sourceB, err := dataStore.MatchSlotSource(matchRows[index].ID, "b")
		if err != nil {
			return nil, fromStore(err)
		}
		matches = append(matches, *matchView(&matchRows[index], sourceA, sourceB))
	}
	return matches, nil
}

// CreateMatch creates a match (row plus sources) from patch; the status
// default matches the schema's.
func CreateMatch(dataStore *store.Store, patch MatchPatch) (*Match, *Error) {
	match := store.Match{Status: "scheduled"}
	if applyError := applyMatchPatch(&match, patch); applyError != nil {
		return nil, applyError
	}
	sourceA := slotSourceValue(patch.ASource)
	sourceB := slotSourceValue(patch.BSource)
	if err := dataStore.CreateMatch(&match, sourceA, sourceB); err != nil {
		return nil, fromStore(err)
	}
	return matchView(&match, sourceA, sourceB), nil
}

// GetMatch returns one match by id, with its slot sources.
func GetMatch(dataStore *store.Store, id int64) (*Match, *Error) {
	match, sourceA, sourceB, err := dataStore.GetMatch(id)
	if err != nil {
		return nil, notFoundOr("match", id, err)
	}
	return matchView(match, sourceA, sourceB), nil
}

// UpdateMatch applies patch to the stored match; set source fields replace
// wholesale, absent ones stay untouched.
func UpdateMatch(dataStore *store.Store, id int64, patch MatchPatch) (*Match, *Error) {
	match, _, _, err := dataStore.GetMatch(id)
	if err != nil {
		return nil, notFoundOr("match", id, err)
	}
	if applyError := applyMatchPatch(match, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateMatch(match, slotSourceUpdate(patch.ASource),
		slotSourceUpdate(patch.BSource)); err != nil {
		return nil, notFoundOr("match", id, err)
	}
	return GetMatch(dataStore, id)
}

// DeleteMatch deletes one match by id.
func DeleteMatch(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteMatch(id); err != nil {
		return notFoundOr("match", id, err)
	}
	return nil
}
