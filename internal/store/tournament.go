// Tournament accessors — the CRUD pattern every entity accessor follows.
package store

import (
	"database/sql"
	"errors"
)

// Tournament mirrors one row of the tournament table; pointer fields are
// nullable columns.
type Tournament struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	Location            string  `json:"location"`
	StartsOn            *string `json:"starts_on"`
	EndsOn              *string `json:"ends_on"`
	VenueOpens          *string `json:"venue_opens"`
	VenueCloses         *string `json:"venue_closes"`
	DefaultMatchMinutes *int64  `json:"default_match_minutes"`
	DefaultGapMinutes   *int64  `json:"default_gap_minutes"`
	CreatedAt           string  `json:"created_at"`
}

// CreateTournament inserts tournament and fills in its ID and CreatedAt.
func (s *Store) CreateTournament(tournament *Tournament) error {
	result, err := s.db.Exec(`INSERT INTO tournament
		(name, location, starts_on, ends_on, venue_opens, venue_closes,
		 default_match_minutes, default_gap_minutes)
		VALUES (?,?,?,?,?,?,?,?)`,
		tournament.Name, tournament.Location, tournament.StartsOn, tournament.EndsOn,
		tournament.VenueOpens, tournament.VenueCloses,
		tournament.DefaultMatchMinutes, tournament.DefaultGapMinutes)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	tournament.ID = id
	return s.db.QueryRow(`SELECT created_at FROM tournament WHERE id = ?`, id).
		Scan(&tournament.CreatedAt)
}

// GetTournament returns the tournament with the given id, or ErrNotFound.
func (s *Store) GetTournament(id int64) (*Tournament, error) {
	tournament := Tournament{ID: id}
	err := s.db.QueryRow(`SELECT name, location, starts_on, ends_on, venue_opens,
		venue_closes, default_match_minutes, default_gap_minutes, created_at
		FROM tournament WHERE id = ?`, id).
		Scan(&tournament.Name, &tournament.Location, &tournament.StartsOn,
			&tournament.EndsOn, &tournament.VenueOpens, &tournament.VenueCloses,
			&tournament.DefaultMatchMinutes, &tournament.DefaultGapMinutes,
			&tournament.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tournament, nil
}

// ListTournaments returns all tournaments in id order.
func (s *Store) ListTournaments() ([]Tournament, error) {
	rows, err := s.db.Query(`SELECT id, name, location, starts_on, ends_on,
		venue_opens, venue_closes, default_match_minutes, default_gap_minutes,
		created_at FROM tournament ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tournaments := make([]Tournament, 0)
	for rows.Next() {
		var tournament Tournament
		if err := rows.Scan(&tournament.ID, &tournament.Name, &tournament.Location,
			&tournament.StartsOn, &tournament.EndsOn, &tournament.VenueOpens,
			&tournament.VenueCloses, &tournament.DefaultMatchMinutes,
			&tournament.DefaultGapMinutes, &tournament.CreatedAt); err != nil {
			return nil, err
		}
		tournaments = append(tournaments, tournament)
	}
	return tournaments, rows.Err()
}

// UpdateTournament writes every column of tournament, or ErrNotFound.
func (s *Store) UpdateTournament(tournament *Tournament) error {
	result, err := s.db.Exec(`UPDATE tournament SET name=?, location=?,
		starts_on=?, ends_on=?, venue_opens=?, venue_closes=?,
		default_match_minutes=?, default_gap_minutes=? WHERE id=?`,
		tournament.Name, tournament.Location, tournament.StartsOn, tournament.EndsOn,
		tournament.VenueOpens, tournament.VenueCloses,
		tournament.DefaultMatchMinutes, tournament.DefaultGapMinutes, tournament.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeleteTournament deletes the tournament and, via cascades, everything it
// owns. Returns ErrNotFound if no such row exists.
func (s *Store) DeleteTournament(id int64) error {
	result, err := s.db.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
