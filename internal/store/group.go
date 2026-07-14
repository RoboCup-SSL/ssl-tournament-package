// Group accessors: the team_group row plus its member and ranking lists,
// written together in one transaction.
package store

import (
	"database/sql"
	"errors"
)

// TeamGroup mirrors one row of the team_group table.
type TeamGroup struct {
	ID                 int64   `json:"id"`
	TournamentID       int64   `json:"tournament_id"`
	DivisionID         *int64  `json:"division_id"`
	Name               string  `json:"name"`
	Notes              string  `json:"notes"`
	RankingConfirmedAt *string `json:"ranking_confirmed_at"`
}

// GroupRank is one entry of a group's confirmed ranking; ties are legal.
type GroupRank struct {
	TeamID int64 `json:"team_id"`
	Rank   int64 `json:"rank"`
}

// GroupFilter narrows ListGroupRows.
type GroupFilter struct {
	TournamentID *int64
	DivisionID   *int64
}

// CreateGroup inserts the group row with its member and ranking lists in one
// transaction, and fills in the group's ID.
func (s *Store) CreateGroup(group *TeamGroup, memberTeamIDs []int64, ranking []GroupRank) error {
	return s.inTx(func(transaction *sql.Tx) error {
		result, err := transaction.Exec(`INSERT INTO team_group
			(tournament_id, division_id, name, notes, ranking_confirmed_at)
			VALUES (?,?,?,?,?)`,
			group.TournamentID, group.DivisionID, group.Name, group.Notes,
			group.RankingConfirmedAt)
		if err != nil {
			return translate(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		group.ID = id
		if err := insertGroupMembers(transaction, id, memberTeamIDs); err != nil {
			return err
		}
		return insertGroupRanking(transaction, id, ranking)
	})
}

// insertGroupMembers inserts one member row per team id.
func insertGroupMembers(transaction *sql.Tx, groupID int64, memberTeamIDs []int64) error {
	for _, teamID := range memberTeamIDs {
		if _, err := transaction.Exec(`INSERT INTO team_group_member (group_id, team_id)
			VALUES (?,?)`, groupID, teamID); err != nil {
			return translate(err)
		}
	}
	return nil
}

// insertGroupRanking inserts one ranking row per entry.
func insertGroupRanking(transaction *sql.Tx, groupID int64, ranking []GroupRank) error {
	for _, entry := range ranking {
		if _, err := transaction.Exec(`INSERT INTO group_ranking (group_id, team_id, rank)
			VALUES (?,?,?)`, groupID, entry.TeamID, entry.Rank); err != nil {
			return translate(err)
		}
	}
	return nil
}

// GetGroup returns the group row with its member and ranking lists, or
// ErrNotFound.
func (s *Store) GetGroup(id int64) (*TeamGroup, []int64, []GroupRank, error) {
	group := TeamGroup{ID: id}
	err := s.db.QueryRow(`SELECT tournament_id, division_id, name, notes,
		ranking_confirmed_at FROM team_group WHERE id = ?`, id).
		Scan(&group.TournamentID, &group.DivisionID, &group.Name, &group.Notes,
			&group.RankingConfirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	memberTeamIDs, err := s.GroupMembers(id)
	if err != nil {
		return nil, nil, nil, err
	}
	ranking, err := s.GroupRanking(id)
	if err != nil {
		return nil, nil, nil, err
	}
	return &group, memberTeamIDs, ranking, nil
}

// GroupMembers returns the group's member team ids ordered by team id.
func (s *Store) GroupMembers(groupID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT team_id FROM team_group_member
		WHERE group_id = ? ORDER BY team_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memberTeamIDs := make([]int64, 0)
	for rows.Next() {
		var teamID int64
		if err := rows.Scan(&teamID); err != nil {
			return nil, err
		}
		memberTeamIDs = append(memberTeamIDs, teamID)
	}
	return memberTeamIDs, rows.Err()
}

// GroupRanking returns the group's ranking ordered by rank, then team id.
func (s *Store) GroupRanking(groupID int64) ([]GroupRank, error) {
	rows, err := s.db.Query(`SELECT team_id, rank FROM group_ranking
		WHERE group_id = ? ORDER BY rank, team_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ranking := make([]GroupRank, 0)
	for rows.Next() {
		var entry GroupRank
		if err := rows.Scan(&entry.TeamID, &entry.Rank); err != nil {
			return nil, err
		}
		ranking = append(ranking, entry)
	}
	return ranking, rows.Err()
}

// ListGroupRows returns group rows matching filter in id order.
func (s *Store) ListGroupRows(filter GroupFilter) ([]TeamGroup, error) {
	rows, err := s.db.Query(`SELECT id, tournament_id, division_id, name, notes,
		ranking_confirmed_at FROM team_group
		WHERE (?1 IS NULL OR tournament_id = ?1)
		  AND (?2 IS NULL OR division_id = ?2)
		ORDER BY id`, filter.TournamentID, filter.DivisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]TeamGroup, 0)
	for rows.Next() {
		var group TeamGroup
		if err := rows.Scan(&group.ID, &group.TournamentID, &group.DivisionID,
			&group.Name, &group.Notes, &group.RankingConfirmedAt); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// UpdateGroup writes the group row and, for each non-nil list, replaces that
// list wholesale — all in one transaction. Returns ErrNotFound for a missing
// row.
func (s *Store) UpdateGroup(group *TeamGroup, memberTeamIDs *[]int64, ranking *[]GroupRank) error {
	return s.inTx(func(transaction *sql.Tx) error {
		result, err := transaction.Exec(`UPDATE team_group SET tournament_id=?,
			division_id=?, name=?, notes=?, ranking_confirmed_at=? WHERE id=?`,
			group.TournamentID, group.DivisionID, group.Name, group.Notes,
			group.RankingConfirmedAt, group.ID)
		if err != nil {
			return translate(err)
		}
		if err := notFoundIfZero(result); err != nil {
			return err
		}
		if memberTeamIDs != nil {
			if _, err := transaction.Exec(`DELETE FROM team_group_member
				WHERE group_id = ?`, group.ID); err != nil {
				return translate(err)
			}
			if err := insertGroupMembers(transaction, group.ID, *memberTeamIDs); err != nil {
				return err
			}
		}
		if ranking != nil {
			if _, err := transaction.Exec(`DELETE FROM group_ranking
				WHERE group_id = ?`, group.ID); err != nil {
				return translate(err)
			}
			if err := insertGroupRanking(transaction, group.ID, *ranking); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteGroup deletes one group; its member and ranking rows cascade, and
// matches in the group are deleted via the schema's ON DELETE CASCADE.
func (s *Store) DeleteGroup(id int64) error {
	result, err := s.db.Exec(`DELETE FROM team_group WHERE id = ?`, id)
	if err != nil {
		return translate(err)
	}
	return notFoundIfZero(result)
}
