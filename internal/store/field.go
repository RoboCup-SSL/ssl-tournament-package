// Field accessors.
package store

import (
	"database/sql"
	"errors"
)

// Field mirrors one row of the field table.
type Field struct {
	ID           int64  `json:"id"`
	TournamentID int64  `json:"tournament_id"`
	Name         string `json:"name"`
}

// FieldFilter narrows ListFields.
type FieldFilter struct {
	TournamentID *int64
}

// CreateField inserts field and fills in its ID.
func (s *Store) CreateField(field *Field) error {
	result, err := s.db.Exec(`INSERT INTO field (tournament_id, name) VALUES (?,?)`,
		field.TournamentID, field.Name)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	field.ID = id
	return nil
}

// GetField returns the field with the given id, or ErrNotFound.
func (s *Store) GetField(id int64) (*Field, error) {
	field := Field{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, name FROM field WHERE id = ?`, id).
		Scan(&field.TournamentID, &field.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &field, nil
}

// ListFields returns fields matching filter in id order.
func (s *Store) ListFields(filter FieldFilter) ([]Field, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, name FROM field
		WHERE (?1 IS NULL OR tournament_id = ?1) ORDER BY id`, filter.TournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := make([]Field, 0)
	for rows.Next() {
		var field Field
		if err := rows.Scan(&field.ID, &field.TournamentID, &field.Name); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, rows.Err()
}

// UpdateField writes every column of field, or ErrNotFound.
func (s *Store) UpdateField(field *Field) error {
	result, err := s.db.Exec(`UPDATE field SET tournament_id=?, name=? WHERE id=?`,
		field.TournamentID, field.Name, field.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeleteField deletes one field; match and booking pointers to it become NULL.
func (s *Store) DeleteField(id int64) error {
	result, err := s.db.Exec(`DELETE FROM field WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
