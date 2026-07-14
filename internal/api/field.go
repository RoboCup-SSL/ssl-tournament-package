// Field operations.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// FieldPatch carries the client-writable field fields.
type FieldPatch struct {
	TournamentID Opt[int64]  `json:"tournament_id"`
	Name         Opt[string] `json:"name"`
}

// applyFieldPatch copies the set patch fields onto field.
func applyFieldPatch(field *store.Field, patch FieldPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &field.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	return applyValue(patch.Name, &field.Name, "name")
}

// ListFields returns fields matching filter.
func ListFields(dataStore *store.Store, filter store.FieldFilter) ([]store.Field, *Error) {
	fields, err := dataStore.ListFields(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	return fields, nil
}

// CreateField creates a field from patch applied to an empty row.
func CreateField(dataStore *store.Store, patch FieldPatch) (*store.Field, *Error) {
	var field store.Field
	if applyError := applyFieldPatch(&field, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.CreateField(&field); err != nil {
		return nil, fromStore(err)
	}
	return &field, nil
}

// GetField returns one field by id.
func GetField(dataStore *store.Store, id int64) (*store.Field, *Error) {
	field, err := dataStore.GetField(id)
	if err != nil {
		return nil, notFoundOr("field", id, err)
	}
	return field, nil
}

// UpdateField applies patch to the stored field.
func UpdateField(dataStore *store.Store, id int64, patch FieldPatch) (*store.Field, *Error) {
	field, err := dataStore.GetField(id)
	if err != nil {
		return nil, notFoundOr("field", id, err)
	}
	if applyError := applyFieldPatch(field, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateField(field); err != nil {
		return nil, notFoundOr("field", id, err)
	}
	return field, nil
}

// DeleteField deletes one field by id.
func DeleteField(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteField(id); err != nil {
		return notFoundOr("field", id, err)
	}
	return nil
}
