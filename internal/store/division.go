// Division accessors.
package store

import (
	"database/sql"
	"errors"
)

// Division mirrors one row of the division table.
type Division struct {
	ID           int64  `json:"id"`
	TournamentID int64  `json:"tournament_id"`
	Name         string `json:"name"`
	Notes        string `json:"notes"`
}

// DivisionFilter narrows ListDivisions.
type DivisionFilter struct {
	TournamentID *int64
}

// CreateDivision inserts division and fills in its ID.
func (s *Store) CreateDivision(division *Division) error {
	result, err := s.db.Exec(`INSERT INTO division (tournament_id, name, notes)
		VALUES (?,?,?)`, division.TournamentID, division.Name, division.Notes)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	division.ID = id
	return nil
}

// GetDivision returns the division with the given id, or ErrNotFound.
func (s *Store) GetDivision(id int64) (*Division, error) {
	division := Division{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, name, notes FROM division
		WHERE id = ?`, id).
		Scan(&division.TournamentID, &division.Name, &division.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &division, nil
}

// ListDivisions returns divisions matching filter in id order.
func (s *Store) ListDivisions(filter DivisionFilter) ([]Division, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, name, notes FROM division
		WHERE (?1 IS NULL OR tournament_id = ?1) ORDER BY id`, filter.TournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	divisions := make([]Division, 0)
	for rows.Next() {
		var division Division
		if err := rows.Scan(&division.ID, &division.TournamentID, &division.Name,
			&division.Notes); err != nil {
			return nil, err
		}
		divisions = append(divisions, division)
	}
	return divisions, rows.Err()
}

// UpdateDivision writes every column of division, or ErrNotFound.
func (s *Store) UpdateDivision(division *Division) error {
	result, err := s.db.Exec(`UPDATE division SET tournament_id=?, name=?, notes=?
		WHERE id=?`, division.TournamentID, division.Name, division.Notes, division.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeleteDivision deletes one division; team/group/match pointers to it become
// NULL via the schema's ON DELETE SET NULL.
func (s *Store) DeleteDivision(id int64) error {
	result, err := s.db.Exec(`DELETE FROM division WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
