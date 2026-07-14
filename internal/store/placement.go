// Placement accessors (final standings entries).
package store

import (
	"database/sql"
	"errors"
)

// Placement mirrors one row of the placement table.
type Placement struct {
	ID             int64   `json:"id"`
	TournamentID   int64   `json:"tournament_id"`
	DivisionID     *int64  `json:"division_id"`
	Rank           int64   `json:"rank"`
	Label          string  `json:"label"`
	SourceKind     *string `json:"source_kind"`
	SourceGroupID  *int64  `json:"source_group_id"`
	SourceRank     *int64  `json:"source_rank"`
	SourceMatchID  *int64  `json:"source_match_id"`
	ResolvedTeamID *int64  `json:"resolved_team_id"`
}

// PlacementFilter narrows ListPlacements.
type PlacementFilter struct {
	TournamentID *int64
	DivisionID   *int64
}

// CreatePlacement inserts placement and fills in its ID.
func (s *Store) CreatePlacement(placement *Placement) error {
	result, err := s.db.Exec(`INSERT INTO placement
		(tournament_id, division_id, rank, label, source_kind, source_group_id,
		 source_rank, source_match_id, resolved_team_id)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		placement.TournamentID, placement.DivisionID, placement.Rank, placement.Label,
		placement.SourceKind, placement.SourceGroupID, placement.SourceRank,
		placement.SourceMatchID, placement.ResolvedTeamID)
	if err != nil {
		return translate(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	placement.ID = id
	return nil
}

// GetPlacement returns the placement with the given id, or ErrNotFound.
func (s *Store) GetPlacement(id int64) (*Placement, error) {
	placement := Placement{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, division_id, rank, label,
		source_kind, source_group_id, source_rank, source_match_id, resolved_team_id
		FROM placement WHERE id = ?`, id).
		Scan(&placement.TournamentID, &placement.DivisionID, &placement.Rank,
			&placement.Label, &placement.SourceKind, &placement.SourceGroupID,
			&placement.SourceRank, &placement.SourceMatchID, &placement.ResolvedTeamID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &placement, nil
}

// ListPlacements returns placements matching filter in id order.
func (s *Store) ListPlacements(filter PlacementFilter) ([]Placement, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, division_id, rank, label,
		source_kind, source_group_id, source_rank, source_match_id, resolved_team_id
		FROM placement
		WHERE (?1 IS NULL OR tournament_id = ?1)
		  AND (?2 IS NULL OR division_id = ?2)
		ORDER BY id`, filter.TournamentID, filter.DivisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	placements := make([]Placement, 0)
	for rows.Next() {
		var placement Placement
		if err := rows.Scan(&placement.ID, &placement.TournamentID,
			&placement.DivisionID, &placement.Rank, &placement.Label,
			&placement.SourceKind, &placement.SourceGroupID, &placement.SourceRank,
			&placement.SourceMatchID, &placement.ResolvedTeamID); err != nil {
			return nil, err
		}
		placements = append(placements, placement)
	}
	return placements, rows.Err()
}

// UpdatePlacement writes every column of placement, or ErrNotFound.
func (s *Store) UpdatePlacement(placement *Placement) error {
	result, err := s.db.Exec(`UPDATE placement SET tournament_id=?, division_id=?,
		rank=?, label=?, source_kind=?, source_group_id=?, source_rank=?,
		source_match_id=?, resolved_team_id=? WHERE id=?`,
		placement.TournamentID, placement.DivisionID, placement.Rank, placement.Label,
		placement.SourceKind, placement.SourceGroupID, placement.SourceRank,
		placement.SourceMatchID, placement.ResolvedTeamID, placement.ID)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}

// DeletePlacement deletes one placement by id.
func (s *Store) DeletePlacement(id int64) error {
	result, err := s.db.Exec(`DELETE FROM placement WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
