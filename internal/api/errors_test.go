// Tests for the store-error to api.Error mapping.
package api

import (
	"errors"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

func TestFromStore(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		code    string
		message string
		field   string
	}{
		{"not found", store.ErrNotFound,
			CodeNotFound, "not found", ""},
		{"foreign key", &store.ConstraintError{Kind: store.ConstraintForeignKey, Detail: "FOREIGN KEY constraint failed (787)"},
			CodeMissingReference, "referenced resource does not exist", ""},
		{"check with column", &store.ConstraintError{Kind: store.ConstraintCheck, Column: "kind", Detail: "CHECK constraint failed: kind IN ('booking','blocked') (275)"},
			CodeInvalidValue, "invalid value for kind", "kind"},
		{"check without column", &store.ConstraintError{Kind: store.ConstraintCheck},
			CodeInvalidValue, "value is not allowed", ""},
		{"not null with column", &store.ConstraintError{Kind: store.ConstraintNotNull, Column: "name", Detail: "NOT NULL constraint failed: team.name (1299)"},
			CodeInvalidValue, "name must not be null", "name"},
		{"not null without column", &store.ConstraintError{Kind: store.ConstraintNotNull},
			CodeInvalidValue, "required value is missing", ""},
		{"duplicate with column", &store.ConstraintError{Kind: store.ConstraintDuplicate, Column: "id", Detail: "UNIQUE constraint failed: tournament.id (1555)"},
			CodeInvalidValue, "duplicate value for id", "id"},
		{"duplicate without column", &store.ConstraintError{Kind: store.ConstraintDuplicate, Detail: "UNIQUE constraint failed: group_ranking.group_id, group_ranking.team_id (2067)"},
			CodeInvalidValue, "duplicate value", ""},
		{"other", errors.New("disk exploded"),
			CodeInternal, "internal error", ""},
	}
	for _, testCase := range cases {
		mapped := fromStore(testCase.err)
		if mapped.Code != testCase.code || mapped.Message != testCase.message || mapped.Field != testCase.field {
			t.Errorf("%s: got %+v, want {%s %q %q}",
				testCase.name, mapped, testCase.code, testCase.message, testCase.field)
		}
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
