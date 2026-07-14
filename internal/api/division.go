// Division operations.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// DivisionPatch carries the client-writable division fields.
type DivisionPatch struct {
	TournamentID Opt[int64]  `json:"tournament_id"`
	Name         Opt[string] `json:"name"`
	Notes        Opt[string] `json:"notes"`
}

// applyDivisionPatch copies the set patch fields onto division.
func applyDivisionPatch(division *store.Division, patch DivisionPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &division.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Name, &division.Name, "name"); applyError != nil {
		return applyError
	}
	return applyValue(patch.Notes, &division.Notes, "notes")
}

// ListDivisions returns divisions matching filter.
func ListDivisions(dataStore *store.Store, filter store.DivisionFilter) ([]store.Division, *Error) {
	divisions, err := dataStore.ListDivisions(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	return divisions, nil
}

// CreateDivision creates a division from patch applied to an empty row.
func CreateDivision(dataStore *store.Store, patch DivisionPatch) (*store.Division, *Error) {
	var division store.Division
	if applyError := applyDivisionPatch(&division, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreateDivision(&division); err != nil {
		return nil, fromStore(err)
	}
	return &division, nil
}

// GetDivision returns one division by id.
func GetDivision(dataStore *store.Store, id int64) (*store.Division, *Error) {
	division, err := dataStore.GetDivision(id)
	if err != nil {
		return nil, notFoundOr("division", id, err)
	}
	return division, nil
}

// UpdateDivision applies patch to the stored division.
func UpdateDivision(dataStore *store.Store, id int64, patch DivisionPatch) (*store.Division, *Error) {
	division, err := dataStore.GetDivision(id)
	if err != nil {
		return nil, notFoundOr("division", id, err)
	}
	if applyError := applyDivisionPatch(division, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateDivision(division); err != nil {
		return nil, notFoundOr("division", id, err)
	}
	return division, nil
}

// DeleteDivision deletes one division by id.
func DeleteDivision(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteDivision(id); err != nil {
		return notFoundOr("division", id, err)
	}
	return nil
}
