// Field-booking accessors (practice slots and blocked-field windows).
package store

import (
	"database/sql"
	"errors"
)

// FieldBooking mirrors one row of the field_booking table.
type FieldBooking struct {
	ID           int64   `json:"id"`
	TournamentID int64   `json:"tournament_id"`
	FieldID      *int64  `json:"field_id"`
	TeamID       *int64  `json:"team_id"`
	Kind         string  `json:"kind"`
	Label        string  `json:"label"`
	StartsAt     *string `json:"starts_at"`
	EndsAt       *string `json:"ends_at"`
	Notes        string  `json:"notes"`
}

// BookingFilter narrows ListBookings.
type BookingFilter struct {
	TournamentID *int64
	FieldID      *int64
	TeamID       *int64
}

// CreateBooking inserts booking and fills in its ID.
func (s *Store) CreateBooking(booking *FieldBooking) error {
	result, err := s.db.Exec(`INSERT INTO field_booking
		(tournament_id, field_id, team_id, kind, label, starts_at, ends_at, notes)
		VALUES (?,?,?,?,?,?,?,?)`,
		booking.TournamentID, booking.FieldID, booking.TeamID, booking.Kind,
		booking.Label, booking.StartsAt, booking.EndsAt, booking.Notes)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	booking.ID = id
	return nil
}

// GetBooking returns the booking with the given id, or ErrNotFound.
func (s *Store) GetBooking(id int64) (*FieldBooking, error) {
	booking := FieldBooking{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, field_id, team_id, kind, label,
		starts_at, ends_at, notes FROM field_booking WHERE id = ?`, id).
		Scan(&booking.TournamentID, &booking.FieldID, &booking.TeamID, &booking.Kind,
			&booking.Label, &booking.StartsAt, &booking.EndsAt, &booking.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// ListBookings returns bookings matching filter in id order.
func (s *Store) ListBookings(filter BookingFilter) ([]FieldBooking, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, field_id, team_id, kind,
		label, starts_at, ends_at, notes FROM field_booking
		WHERE (?1 IS NULL OR tournament_id = ?1)
		  AND (?2 IS NULL OR field_id = ?2)
		  AND (?3 IS NULL OR team_id = ?3)
		ORDER BY id`, filter.TournamentID, filter.FieldID, filter.TeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bookings := make([]FieldBooking, 0)
	for rows.Next() {
		var booking FieldBooking
		if err := rows.Scan(&booking.ID, &booking.TournamentID, &booking.FieldID,
			&booking.TeamID, &booking.Kind, &booking.Label, &booking.StartsAt,
			&booking.EndsAt, &booking.Notes); err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

// UpdateBooking writes every column of booking, or ErrNotFound.
func (s *Store) UpdateBooking(booking *FieldBooking) error {
	result, err := s.db.Exec(`UPDATE field_booking SET tournament_id=?, field_id=?,
		team_id=?, kind=?, label=?, starts_at=?, ends_at=?, notes=? WHERE id=?`,
		booking.TournamentID, booking.FieldID, booking.TeamID, booking.Kind,
		booking.Label, booking.StartsAt, booking.EndsAt, booking.Notes, booking.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeleteBooking deletes one booking by id.
func (s *Store) DeleteBooking(id int64) error {
	result, err := s.db.Exec(`DELETE FROM field_booking WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
