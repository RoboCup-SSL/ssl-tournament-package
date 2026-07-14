// Tests for the store-error to api.Error mapping.
package api

import (
	"errors"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

func TestFromStore(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"not found", store.ErrNotFound, CodeNotFound},
		{"foreign key", &store.ConstraintError{Kind: store.ConstraintForeignKey, Detail: "FOREIGN KEY constraint failed"}, CodeMissingReference},
		{"check", &store.ConstraintError{Kind: store.ConstraintCheck, Detail: "CHECK constraint failed: kind"}, CodeInvalidValue},
		{"not null", &store.ConstraintError{Kind: store.ConstraintNotNull, Detail: "NOT NULL constraint failed: team.name"}, CodeInvalidValue},
		{"duplicate", &store.ConstraintError{Kind: store.ConstraintDuplicate, Detail: "UNIQUE constraint failed: group_ranking.group_id, group_ranking.team_id"}, CodeInvalidValue},
		{"other", errors.New("disk exploded"), CodeInternal},
	}
	for _, testCase := range cases {
		if mapped := fromStore(testCase.err); mapped.Code != testCase.code {
			t.Errorf("%s: got %s, want %s", testCase.name, mapped.Code, testCase.code)
		}
	}
	notNull := fromStore(&store.ConstraintError{Kind: store.ConstraintNotNull,
		Detail: "NOT NULL constraint failed: team.name"})
	if notNull.Field != "name" {
		t.Errorf("field extraction: got %q, want name", notNull.Field)
	}
}

func TestNotFoundOr(t *testing.T) {
	mapped := notFoundOr("team", 5, store.ErrNotFound)
	if mapped.Code != CodeNotFound || mapped.Message != "team 5 not found" {
		t.Errorf("got %+v", mapped)
	}
	other := notFoundOr("team", 5, &store.ConstraintError{Kind: store.ConstraintForeignKey, Detail: "x"})
	if other.Code != CodeMissingReference {
		t.Errorf("non-ErrNotFound must fall through to fromStore: %+v", other)
	}
}
