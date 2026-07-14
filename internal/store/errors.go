// Store error types and the translation of SQLite driver errors into them.
package store

import (
	"database/sql"
	"errors"
	"strings"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ConstraintKind classifies a database constraint violation.
type ConstraintKind int

const (
	ConstraintForeignKey ConstraintKind = iota
	ConstraintCheck
	ConstraintNotNull
	ConstraintDuplicate
)

// ConstraintError reports a database constraint violation with SQLite's text.
type ConstraintError struct {
	Kind   ConstraintKind
	Detail string
}

// Error returns SQLite's constraint message.
func (e *ConstraintError) Error() string { return e.Detail }

// translate classifies a driver error by its constraint message; SQLite's
// message texts are stable across versions, unlike its extended-code API.
func translate(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "FOREIGN KEY constraint failed"):
		return &ConstraintError{Kind: ConstraintForeignKey, Detail: message}
	case strings.Contains(message, "CHECK constraint failed"):
		return &ConstraintError{Kind: ConstraintCheck, Detail: message}
	case strings.Contains(message, "NOT NULL constraint failed"):
		return &ConstraintError{Kind: ConstraintNotNull, Detail: message}
	case strings.Contains(message, "UNIQUE constraint failed"),
		strings.Contains(message, "PRIMARY KEY constraint failed"):
		return &ConstraintError{Kind: ConstraintDuplicate, Detail: message}
	}
	return err
}

// notFoundIfZero maps a zero-row write result onto ErrNotFound.
func notFoundIfZero(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
