// Match accessors: the match row plus its slot_source rows, written together
// in one transaction.
package store

import (
	"database/sql"
	"errors"
)

// Match mirrors one row of the match table; pointer fields are nullable
// columns.
type Match struct {
	ID                     int64   `json:"id"`
	TournamentID           int64   `json:"tournament_id"`
	DivisionID             *int64  `json:"division_id"`
	GroupID                *int64  `json:"group_id"`
	Label                  string  `json:"label"`
	FieldID                *int64  `json:"field_id"`
	ScheduledAt            *string `json:"scheduled_at"`
	DurationMinutes        *int64  `json:"duration_minutes"`
	Status                 string  `json:"status"`
	RefereeTeamID          *int64  `json:"referee_team_id"`
	AssistantRefereeTeamID *int64  `json:"assistant_referee_team_id"`
	ATeamID                *int64  `json:"a_team_id"`
	AScore                 *int64  `json:"a_score"`
	AFouls                 *int64  `json:"a_fouls"`
	AYellowCards           *int64  `json:"a_yellow_cards"`
	ARedCards              *int64  `json:"a_red_cards"`
	BTeamID                *int64  `json:"b_team_id"`
	BScore                 *int64  `json:"b_score"`
	BFouls                 *int64  `json:"b_fouls"`
	BYellowCards           *int64  `json:"b_yellow_cards"`
	BRedCards              *int64  `json:"b_red_cards"`
	WinnerTeamID           *int64  `json:"winner_team_id"`
	Notes                  string  `json:"notes"`
	CreatedAt              string  `json:"created_at"`
}

// SlotSource says where one match slot's team will come from.
type SlotSource struct {
	Kind    string `json:"kind"`
	GroupID *int64 `json:"group_id"`
	Rank    *int64 `json:"rank"`
	MatchID *int64 `json:"match_id"`
}

// SlotSourceUpdate says what to do with one slot's stored source: leave it
// (Replace false), delete it (Replace true, Source nil), or replace it.
type SlotSourceUpdate struct {
	Replace bool
	Source  *SlotSource
}

// MatchFilter narrows ListMatchRows.
type MatchFilter struct {
	TournamentID *int64
	DivisionID   *int64
	GroupID      *int64
	FieldID      *int64
}

// CreateMatch inserts the match row and any non-nil slot sources in one
// transaction, and fills in the match's ID and CreatedAt.
func (s *Store) CreateMatch(match *Match, sourceA, sourceB *SlotSource) error {
	return s.inTx(func(transaction *sql.Tx) error {
		result, err := transaction.Exec(`INSERT INTO match
			(tournament_id, division_id, group_id, label, field_id, scheduled_at,
			 duration_minutes, status, referee_team_id, assistant_referee_team_id,
			 a_team_id, a_score, a_fouls, a_yellow_cards, a_red_cards,
			 b_team_id, b_score, b_fouls, b_yellow_cards, b_red_cards,
			 winner_team_id, notes)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			match.TournamentID, match.DivisionID, match.GroupID, match.Label,
			match.FieldID, match.ScheduledAt, match.DurationMinutes, match.Status,
			match.RefereeTeamID, match.AssistantRefereeTeamID,
			match.ATeamID, match.AScore, match.AFouls, match.AYellowCards,
			match.ARedCards, match.BTeamID, match.BScore, match.BFouls,
			match.BYellowCards, match.BRedCards, match.WinnerTeamID, match.Notes)
		if err != nil {
			return translate(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		match.ID = id
		if err := transaction.QueryRow(`SELECT created_at FROM match WHERE id = ?`, id).
			Scan(&match.CreatedAt); err != nil {
			return err
		}
		if err := insertSlotSource(transaction, id, "a", sourceA); err != nil {
			return err
		}
		return insertSlotSource(transaction, id, "b", sourceB)
	})
}

// insertSlotSource inserts one slot_source row when source is non-nil.
func insertSlotSource(transaction *sql.Tx, matchID int64, slot string, source *SlotSource) error {
	if source == nil {
		return nil
	}
	_, err := transaction.Exec(`INSERT INTO slot_source
		(match_id, slot, kind, ref_group_id, ref_rank, ref_match_id)
		VALUES (?,?,?,?,?,?)`,
		matchID, slot, source.Kind, source.GroupID, source.Rank, source.MatchID)
	return translate(err)
}

// GetMatch returns the match row and its slot sources (nil when unset), or
// ErrNotFound.
func (s *Store) GetMatch(id int64) (*Match, *SlotSource, *SlotSource, error) {
	match := Match{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, division_id, group_id, label,
		field_id, scheduled_at, duration_minutes, status, referee_team_id,
		assistant_referee_team_id, a_team_id, a_score, a_fouls, a_yellow_cards,
		a_red_cards, b_team_id, b_score, b_fouls, b_yellow_cards, b_red_cards,
		winner_team_id, notes, created_at FROM match WHERE id = ?`, id).
		Scan(&match.TournamentID, &match.DivisionID, &match.GroupID, &match.Label,
			&match.FieldID, &match.ScheduledAt, &match.DurationMinutes, &match.Status,
			&match.RefereeTeamID, &match.AssistantRefereeTeamID,
			&match.ATeamID, &match.AScore, &match.AFouls, &match.AYellowCards,
			&match.ARedCards, &match.BTeamID, &match.BScore, &match.BFouls,
			&match.BYellowCards, &match.BRedCards, &match.WinnerTeamID,
			&match.Notes, &match.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	sourceA, err := s.MatchSlotSource(id, "a")
	if err != nil {
		return nil, nil, nil, err
	}
	sourceB, err := s.MatchSlotSource(id, "b")
	if err != nil {
		return nil, nil, nil, err
	}
	return &match, sourceA, sourceB, nil
}

// MatchSlotSource returns the slot_source row for one slot, or nil when the
// slot has no source.
func (s *Store) MatchSlotSource(matchID int64, slot string) (*SlotSource, error) {
	var source SlotSource
	err := s.db.QueryRow(`SELECT kind, ref_group_id, ref_rank, ref_match_id
		FROM slot_source WHERE match_id = ? AND slot = ?`, matchID, slot).
		Scan(&source.Kind, &source.GroupID, &source.Rank, &source.MatchID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// ListMatchRows returns match rows matching filter in id order.
func (s *Store) ListMatchRows(filter MatchFilter) ([]Match, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, division_id, group_id,
		label, field_id, scheduled_at, duration_minutes, status, referee_team_id,
		assistant_referee_team_id, a_team_id, a_score, a_fouls, a_yellow_cards,
		a_red_cards, b_team_id, b_score, b_fouls, b_yellow_cards, b_red_cards,
		winner_team_id, notes, created_at FROM match
		WHERE (?1 IS NULL OR tournament_id = ?1)
		  AND (?2 IS NULL OR division_id = ?2)
		  AND (?3 IS NULL OR group_id = ?3)
		  AND (?4 IS NULL OR field_id = ?4)
		ORDER BY id`,
		filter.TournamentID, filter.DivisionID, filter.GroupID, filter.FieldID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]Match, 0)
	for rows.Next() {
		var match Match
		if err := rows.Scan(&match.ID, &match.TournamentID, &match.DivisionID,
			&match.GroupID, &match.Label, &match.FieldID, &match.ScheduledAt,
			&match.DurationMinutes, &match.Status, &match.RefereeTeamID,
			&match.AssistantRefereeTeamID, &match.ATeamID, &match.AScore,
			&match.AFouls, &match.AYellowCards, &match.ARedCards, &match.BTeamID,
			&match.BScore, &match.BFouls, &match.BYellowCards, &match.BRedCards,
			&match.WinnerTeamID, &match.Notes, &match.CreatedAt); err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, rows.Err()
}

// UpdateMatch writes the match row and applies the slot-source updates, all
// in one transaction. Returns ErrNotFound for a missing row.
func (s *Store) UpdateMatch(match *Match, sourceA, sourceB SlotSourceUpdate) error {
	return s.inTx(func(transaction *sql.Tx) error {
		result, err := transaction.Exec(`UPDATE match SET tournament_id=?,
			division_id=?, group_id=?, label=?, field_id=?, scheduled_at=?,
			duration_minutes=?, status=?, referee_team_id=?,
			assistant_referee_team_id=?, a_team_id=?, a_score=?, a_fouls=?,
			a_yellow_cards=?, a_red_cards=?, b_team_id=?, b_score=?, b_fouls=?,
			b_yellow_cards=?, b_red_cards=?, winner_team_id=?, notes=? WHERE id=?`,
			match.TournamentID, match.DivisionID, match.GroupID, match.Label,
			match.FieldID, match.ScheduledAt, match.DurationMinutes, match.Status,
			match.RefereeTeamID, match.AssistantRefereeTeamID,
			match.ATeamID, match.AScore, match.AFouls, match.AYellowCards,
			match.ARedCards, match.BTeamID, match.BScore, match.BFouls,
			match.BYellowCards, match.BRedCards, match.WinnerTeamID, match.Notes,
			match.ID)
		if err != nil {
			return translate(err)
		}
		if err := notFoundIfZero(result); err != nil {
			return err
		}
		updates := []struct {
			slot   string
			update SlotSourceUpdate
		}{{"a", sourceA}, {"b", sourceB}}
		for _, slotUpdate := range updates {
			if !slotUpdate.update.Replace {
				continue
			}
			if _, err := transaction.Exec(`DELETE FROM slot_source
				WHERE match_id = ? AND slot = ?`, match.ID, slotUpdate.slot); err != nil {
				return translate(err)
			}
			if err := insertSlotSource(transaction, match.ID, slotUpdate.slot,
				slotUpdate.update.Source); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteMatch deletes one match; its slot_source rows cascade and references
// to it become NULL.
func (s *Store) DeleteMatch(id int64) error {
	result, err := s.db.Exec(`DELETE FROM match WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
