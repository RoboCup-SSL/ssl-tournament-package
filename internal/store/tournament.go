// Tournament accessors — the CRUD pattern every entity accessor follows.
package store

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// Tournament mirrors one row of the tournament table; pointer fields are
// nullable columns.
type Tournament struct {
	ID                  int64
	Name                string
	Location            string
	StartsOn            *string
	EndsOn              *string
	VenueOpens          *string
	VenueCloses         *string
	DefaultMatchMinutes *int64
	DefaultGapMinutes   *int64
	CreatedAt           string
}

// CreateTournament inserts t and fills in its ID and CreatedAt.
func (s *Store) CreateTournament(t *Tournament) error {
	res, err := s.db.Exec(`INSERT INTO tournament
		(name, location, starts_on, ends_on, venue_opens, venue_closes,
		 default_match_minutes, default_gap_minutes)
		VALUES (?,?,?,?,?,?,?,?)`,
		t.Name, t.Location, t.StartsOn, t.EndsOn, t.VenueOpens, t.VenueCloses,
		t.DefaultMatchMinutes, t.DefaultGapMinutes)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return s.db.QueryRow(`SELECT created_at FROM tournament WHERE id = ?`, id).
		Scan(&t.CreatedAt)
}

// GetTournament returns the tournament with the given id, or ErrNotFound.
func (s *Store) GetTournament(id int64) (*Tournament, error) {
	t := Tournament{ID: id}
	err := s.db.QueryRow(`SELECT name, location, starts_on, ends_on, venue_opens,
		venue_closes, default_match_minutes, default_gap_minutes, created_at
		FROM tournament WHERE id = ?`, id).
		Scan(&t.Name, &t.Location, &t.StartsOn, &t.EndsOn, &t.VenueOpens,
			&t.VenueCloses, &t.DefaultMatchMinutes, &t.DefaultGapMinutes, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTournaments returns all tournaments, newest first.
func (s *Store) ListTournaments() ([]Tournament, error) {
	rows, err := s.db.Query(`SELECT id, name, location, starts_on, ends_on,
		venue_opens, venue_closes, default_match_minutes, default_gap_minutes,
		created_at FROM tournament ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tournament
	for rows.Next() {
		var t Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.Location, &t.StartsOn, &t.EndsOn,
			&t.VenueOpens, &t.VenueCloses, &t.DefaultMatchMinutes,
			&t.DefaultGapMinutes, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTournament deletes the tournament and, via cascades, everything it
// owns. Returns ErrNotFound if no such row exists.
func (s *Store) DeleteTournament(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
