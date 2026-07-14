// Team accessors.
package store

import (
	"database/sql"
	"errors"
)

// Team mirrors one row of the team table; pointer fields are nullable columns.
type Team struct {
	ID           int64   `json:"id"`
	TournamentID int64   `json:"tournament_id"`
	DivisionID   *int64  `json:"division_id"`
	Name         string  `json:"name"`
	Country      string  `json:"country"`
	Contact      string  `json:"contact"`
	Notes        string  `json:"notes"`
	WithdrawnAt  *string `json:"withdrawn_at"`
	CreatedAt    string  `json:"created_at"`
}

// TeamFilter narrows ListTeams.
type TeamFilter struct {
	TournamentID *int64
	DivisionID   *int64
}

// CreateTeam inserts team and fills in its ID and CreatedAt.
func (s *Store) CreateTeam(team *Team) error {
	result, err := s.db.Exec(`INSERT INTO team
		(tournament_id, division_id, name, country, contact, notes, withdrawn_at)
		VALUES (?,?,?,?,?,?,?)`,
		team.TournamentID, team.DivisionID, team.Name, team.Country, team.Contact,
		team.Notes, team.WithdrawnAt)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = id
	return s.db.QueryRow(`SELECT created_at FROM team WHERE id = ?`, id).
		Scan(&team.CreatedAt)
}

// GetTeam returns the team with the given id, or ErrNotFound.
func (s *Store) GetTeam(id int64) (*Team, error) {
	team := Team{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, division_id, name, country, contact,
		notes, withdrawn_at, created_at FROM team WHERE id = ?`, id).
		Scan(&team.TournamentID, &team.DivisionID, &team.Name, &team.Country,
			&team.Contact, &team.Notes, &team.WithdrawnAt, &team.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// ListTeams returns teams matching filter in id order.
func (s *Store) ListTeams(filter TeamFilter) ([]Team, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, division_id, name, country,
		contact, notes, withdrawn_at, created_at FROM team
		WHERE (?1 IS NULL OR tournament_id = ?1)
		  AND (?2 IS NULL OR division_id = ?2)
		ORDER BY id`, filter.TournamentID, filter.DivisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	teams := make([]Team, 0)
	for rows.Next() {
		var team Team
		if err := rows.Scan(&team.ID, &team.TournamentID, &team.DivisionID,
			&team.Name, &team.Country, &team.Contact, &team.Notes,
			&team.WithdrawnAt, &team.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, rows.Err()
}

// UpdateTeam writes every column of team, or ErrNotFound.
func (s *Store) UpdateTeam(team *Team) error {
	result, err := s.db.Exec(`UPDATE team SET tournament_id=?, division_id=?,
		name=?, country=?, contact=?, notes=?, withdrawn_at=? WHERE id=?`,
		team.TournamentID, team.DivisionID, team.Name, team.Country, team.Contact,
		team.Notes, team.WithdrawnAt, team.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeleteTeam deletes one team; references to it become NULL via the schema's
// ON DELETE SET NULL (withdrawing is usually the better operation).
func (s *Store) DeleteTeam(id int64) error {
	result, err := s.db.Exec(`DELETE FROM team WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
