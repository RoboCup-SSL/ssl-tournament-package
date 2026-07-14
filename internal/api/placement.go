// Placement operations.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// PlacementPatch carries the client-writable placement fields.
type PlacementPatch struct {
	TournamentID   Opt[int64]  `json:"tournament_id"`
	DivisionID     Opt[int64]  `json:"division_id"`
	Rank           Opt[int64]  `json:"rank"`
	Label          Opt[string] `json:"label"`
	SourceKind     Opt[string] `json:"source_kind"`
	SourceGroupID  Opt[int64]  `json:"source_group_id"`
	SourceRank     Opt[int64]  `json:"source_rank"`
	SourceMatchID  Opt[int64]  `json:"source_match_id"`
	ResolvedTeamID Opt[int64]  `json:"resolved_team_id"`
}

// applyPlacementPatch copies the set patch fields onto placement.
func applyPlacementPatch(placement *store.Placement, patch PlacementPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &placement.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Rank, &placement.Rank, "rank"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Label, &placement.Label, "label"); applyError != nil {
		return applyError
	}
	applyNullable(patch.DivisionID, &placement.DivisionID)
	applyNullable(patch.SourceKind, &placement.SourceKind)
	applyNullable(patch.SourceGroupID, &placement.SourceGroupID)
	applyNullable(patch.SourceRank, &placement.SourceRank)
	applyNullable(patch.SourceMatchID, &placement.SourceMatchID)
	applyNullable(patch.ResolvedTeamID, &placement.ResolvedTeamID)
	return nil
}

// ListPlacements returns placements matching filter.
func ListPlacements(dataStore *store.Store, filter store.PlacementFilter) ([]store.Placement, *Error) {
	placements, err := dataStore.ListPlacements(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	return placements, nil
}

// CreatePlacement creates a placement from patch applied to an empty row.
func CreatePlacement(dataStore *store.Store, patch PlacementPatch) (*store.Placement, *Error) {
	var placement store.Placement
	if applyError := applyPlacementPatch(&placement, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreatePlacement(&placement); err != nil {
		return nil, fromStore(err)
	}
	return &placement, nil
}

// GetPlacement returns one placement by id.
func GetPlacement(dataStore *store.Store, id int64) (*store.Placement, *Error) {
	placement, err := dataStore.GetPlacement(id)
	if err != nil {
		return nil, notFoundOr("placement", id, err)
	}
	return placement, nil
}

// UpdatePlacement applies patch to the stored placement.
func UpdatePlacement(dataStore *store.Store, id int64, patch PlacementPatch) (*store.Placement, *Error) {
	placement, err := dataStore.GetPlacement(id)
	if err != nil {
		return nil, notFoundOr("placement", id, err)
	}
	if applyError := applyPlacementPatch(placement, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdatePlacement(placement); err != nil {
		return nil, notFoundOr("placement", id, err)
	}
	return placement, nil
}

// DeletePlacement deletes one placement by id.
func DeletePlacement(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeletePlacement(id); err != nil {
		return notFoundOr("placement", id, err)
	}
	return nil
}
