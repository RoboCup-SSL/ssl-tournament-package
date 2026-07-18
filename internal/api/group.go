// Group operations: the group resource embeds its member and ranking lists.
package api

import "github.com/RoboCup-SSL/ssl-tournament-package/internal/store"

// Group is the wire shape of a group: the row plus its two lists.
type Group struct {
	store.TeamGroup
	MemberTeamIDs []int64           `json:"member_team_ids"`
	Ranking       []store.GroupRank `json:"ranking"`
}

// GroupPatch carries the client-writable group fields; the list fields
// replace wholesale.
type GroupPatch struct {
	TournamentID       Opt[int64]             `json:"tournament_id"`
	DivisionID         Opt[int64]             `json:"division_id"`
	Name               Opt[string]            `json:"name"`
	Notes              Opt[string]            `json:"notes"`
	RankingConfirmedAt Opt[string]            `json:"ranking_confirmed_at"`
	MemberTeamIDs      Opt[[]int64]           `json:"member_team_ids"`
	Ranking            Opt[[]store.GroupRank] `json:"ranking"`
}

// applyGroupPatch copies the set scalar patch fields onto group.
func applyGroupPatch(group *store.TeamGroup, patch GroupPatch) *Error {
	if applyError := applyValue(patch.TournamentID, &group.TournamentID, "tournament_id"); applyError != nil {
		return applyError
	}
	applyNullable(patch.DivisionID, &group.DivisionID)
	if applyError := applyValue(patch.Name, &group.Name, "name"); applyError != nil {
		return applyError
	}
	if applyError := applyValue(patch.Notes, &group.Notes, "notes"); applyError != nil {
		return applyError
	}
	if e := applyNullableString(patch.RankingConfirmedAt, &group.RankingConfirmedAt, "ranking_confirmed_at", validNaiveTime); e != nil {
		return e
	}
	return nil
}

// groupView assembles the wire shape with non-nil lists.
func groupView(group *store.TeamGroup, memberTeamIDs []int64, ranking []store.GroupRank) *Group {
	if memberTeamIDs == nil {
		memberTeamIDs = []int64{}
	}
	if ranking == nil {
		ranking = []store.GroupRank{}
	}
	return &Group{TeamGroup: *group, MemberTeamIDs: memberTeamIDs, Ranking: ranking}
}

// ListGroups returns groups matching filter, each with its lists.
func ListGroups(dataStore *store.Store, filter store.GroupFilter) ([]Group, *Error) {
	groupRows, err := dataStore.ListGroupRows(filter)
	if err != nil {
		return nil, fromStore(err)
	}
	groups := make([]Group, 0, len(groupRows))
	for index := range groupRows {
		memberTeamIDs, err := dataStore.GroupMembers(groupRows[index].ID)
		if err != nil {
			return nil, fromStore(err)
		}
		ranking, err := dataStore.GroupRanking(groupRows[index].ID)
		if err != nil {
			return nil, fromStore(err)
		}
		groups = append(groups, *groupView(&groupRows[index], memberTeamIDs, ranking))
	}
	return groups, nil
}

// CreateGroup creates a group (row plus lists) from patch.
func CreateGroup(dataStore *store.Store, patch GroupPatch) (*Group, *Error) {
	var group store.TeamGroup
	if applyError := applyGroupPatch(&group, patch); applyError != nil {
		return nil, applyError
	}
	memberTeamIDs := listValue(patch.MemberTeamIDs)
	ranking := listValue(patch.Ranking)
	if memberTeamIDs == nil {
		memberTeamIDs = &[]int64{}
	}
	if ranking == nil {
		ranking = &[]store.GroupRank{}
	}
	if err := dataStore.CreateGroup(&group, *memberTeamIDs, *ranking); err != nil {
		return nil, fromStore(err)
	}
	return groupView(&group, *memberTeamIDs, *ranking), nil
}

// GetGroup returns one group by id, with its lists.
func GetGroup(dataStore *store.Store, id int64) (*Group, *Error) {
	group, memberTeamIDs, ranking, err := dataStore.GetGroup(id)
	if err != nil {
		return nil, notFoundOr("group", id, err)
	}
	return groupView(group, memberTeamIDs, ranking), nil
}

// UpdateGroup applies patch to the stored group; set list fields replace
// wholesale, absent ones stay untouched.
func UpdateGroup(dataStore *store.Store, id int64, patch GroupPatch) (*Group, *Error) {
	group, _, _, err := dataStore.GetGroup(id)
	if err != nil {
		return nil, notFoundOr("group", id, err)
	}
	if applyError := applyGroupPatch(group, patch); applyError != nil {
		return nil, applyError
	}
	if err := dataStore.UpdateGroup(group, listValue(patch.MemberTeamIDs), listValue(patch.Ranking)); err != nil {
		return nil, notFoundOr("group", id, err)
	}
	return GetGroup(dataStore, id)
}

// DeleteGroup deletes one group by id.
func DeleteGroup(dataStore *store.Store, id int64) *Error {
	if err := dataStore.DeleteGroup(id); err != nil {
		return notFoundOr("group", id, err)
	}
	return nil
}
