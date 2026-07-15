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

// ConstraintError reports a database constraint violation. Column names the
// violated column when the driver message identifies exactly one, else empty.
// Detail keeps the raw driver text for logs; it must never reach API clients.
type ConstraintError struct {
	Kind   ConstraintKind
	Column string
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
		return &ConstraintError{Kind: ConstraintCheck, Column: checkColumn(message), Detail: message}
	case strings.Contains(message, "NOT NULL constraint failed"):
		return &ConstraintError{Kind: ConstraintNotNull, Column: qualifiedColumn(message), Detail: message}
	case strings.Contains(message, "UNIQUE constraint failed"),
		strings.Contains(message, "PRIMARY KEY constraint failed"):
		return &ConstraintError{Kind: ConstraintDuplicate, Column: qualifiedColumn(message), Detail: message}
	}
	return err
}

// constraintSubject returns the text after the last "constraint failed: "
// (the driver wraps SQLite's message in its own "constraint failed: " prefix)
// with the trailing " (extended-code)" stripped; empty when the message
// carries no subject.
func constraintSubject(message string) string {
	marker := "constraint failed: "
	index := strings.LastIndex(message, marker)
	if index == -1 {
		return ""
	}
	subject := message[index+len(marker):]
	if open := strings.LastIndex(subject, " ("); open != -1 && strings.HasSuffix(subject, ")") {
		if isDigits(subject[open+2 : len(subject)-1]) {
			subject = subject[:open]
		}
	}
	return subject
}

// qualifiedColumn extracts the column from a single "table.column" subject,
// as in NOT NULL and UNIQUE messages; empty for multi-column lists.
func qualifiedColumn(message string) string {
	subject := constraintSubject(message)
	if strings.Contains(subject, ",") {
		return ""
	}
	_, column, found := strings.Cut(subject, ".")
	if !found || strings.Contains(column, " ") {
		return ""
	}
	return column
}

// checkColumn extracts the leading column identifier from a CHECK expression
// subject like "kind IN ('booking','blocked')"; empty when the expression
// does not start with a plain identifier.
func checkColumn(message string) string {
	identifier, _, _ := strings.Cut(constraintSubject(message), " ")
	if !isIdentifier(identifier) {
		return ""
	}
	return identifier
}

// isDigits reports whether text is non-empty and all ASCII digits.
func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// isIdentifier reports whether text is a plain SQL identifier: a letter or
// underscore followed by letters, digits, or underscores.
func isIdentifier(text string) bool {
	for position, character := range text {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character == '_':
		case character >= '0' && character <= '9' && position > 0:
		default:
			return false
		}
	}
	return text != ""
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
