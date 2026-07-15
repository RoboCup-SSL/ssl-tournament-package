// Tests for constraint-error translation using real schema violations.
package store

import (
	"errors"
	"testing"
)

// constraintFor runs statement and returns the translated ConstraintError.
func constraintFor(t *testing.T, dataStore *Store, statement string) *ConstraintError {
	t.Helper()
	_, err := dataStore.db.Exec(statement)
	translated := translate(err)
	var constraintError *ConstraintError
	if !errors.As(translated, &constraintError) {
		t.Fatalf("expected ConstraintError, got %v", translated)
	}
	return constraintError
}

func TestTranslateConstraintKinds(t *testing.T) {
	dataStore := openTestStore(t)
	if _, err := dataStore.db.Exec(
		`INSERT INTO tournament (id, name) VALUES (1, 'T')`); err != nil {
		t.Fatal(err)
	}
	foreignKey := constraintFor(t, dataStore,
		`INSERT INTO division (tournament_id, name) VALUES (99, 'A')`)
	if foreignKey.Kind != ConstraintForeignKey || foreignKey.Column != "" {
		t.Errorf("foreign key: got %+v", foreignKey)
	}
	check := constraintFor(t, dataStore,
		`INSERT INTO field_booking (tournament_id, kind) VALUES (1, 'party')`)
	if check.Kind != ConstraintCheck || check.Column != "kind" {
		t.Errorf("check: got %+v", check)
	}
	notNull := constraintFor(t, dataStore,
		`INSERT INTO division (tournament_id, name) VALUES (1, NULL)`)
	if notNull.Kind != ConstraintNotNull || notNull.Column != "name" {
		t.Errorf("not null: got %+v", notNull)
	}
	duplicateSingle := constraintFor(t, dataStore,
		`INSERT INTO tournament (id, name) VALUES (1, 'dup')`)
	if duplicateSingle.Kind != ConstraintDuplicate || duplicateSingle.Column != "id" {
		t.Errorf("single-column duplicate: got %+v", duplicateSingle)
	}
	if _, err := dataStore.db.Exec(
		`INSERT INTO team (id, tournament_id, name) VALUES (1, 1, 'X')`); err != nil {
		t.Fatal(err)
	}
	if _, err := dataStore.db.Exec(
		`INSERT INTO team_group (id, tournament_id, name) VALUES (1, 1, 'G')`); err != nil {
		t.Fatal(err)
	}
	if _, err := dataStore.db.Exec(
		`INSERT INTO group_ranking (group_id, team_id, rank) VALUES (1, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	duplicateComposite := constraintFor(t, dataStore,
		`INSERT INTO group_ranking (group_id, team_id, rank) VALUES (1, 1, 2)`)
	if duplicateComposite.Kind != ConstraintDuplicate || duplicateComposite.Column != "" {
		t.Errorf("multi-column duplicate: got %+v", duplicateComposite)
	}
	if translated := translate(nil); translated != nil {
		t.Errorf("translate(nil) = %v", translated)
	}
}

func TestUpdateTournament(t *testing.T) {
	dataStore := openTestStore(t)
	tournament := Tournament{Name: "RoboCup"}
	if err := dataStore.CreateTournament(&tournament); err != nil {
		t.Fatal(err)
	}
	tournament.Name = "RoboCup 2026"
	location := tournament.Location
	tournament.Location = location + "Incheon"
	if err := dataStore.UpdateTournament(&tournament); err != nil {
		t.Fatal(err)
	}
	stored, err := dataStore.GetTournament(tournament.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "RoboCup 2026" || stored.Location != "Incheon" {
		t.Errorf("update not persisted: %+v", stored)
	}
	missing := Tournament{ID: 999, Name: "ghost"}
	if err := dataStore.UpdateTournament(&missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListTournamentsAscending(t *testing.T) {
	dataStore := openTestStore(t)
	for _, name := range []string{"first", "second"} {
		if err := dataStore.CreateTournament(&Tournament{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	tournaments, err := dataStore.ListTournaments()
	if err != nil {
		t.Fatal(err)
	}
	if len(tournaments) != 2 || tournaments[0].Name != "first" {
		t.Errorf("expected ascending id order, got %+v", tournaments)
	}
}
