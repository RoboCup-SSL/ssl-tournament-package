// Package api defines the application's operations independent of transport;
// HTTP handlers (and a future MCP server) call these free functions.
package api

import (
	"errors"
	"fmt"
	"log"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// Error codes carried by every failed operation.
const (
	CodeInvalidJSON      = "INVALID_JSON"
	CodeUnknownField     = "UNKNOWN_FIELD"
	CodeInvalidValue     = "INVALID_VALUE"
	CodeNotFound         = "NOT_FOUND"
	CodeMissingReference = "MISSING_REFERENCE"
	CodeInternal         = "INTERNAL"
)

// Error is the transport-independent operation error. Code and Field are the
// stable machine-readable parts; Message is an API-authored English rendering
// of them, never text from the database driver.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Error renders the code and message.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// fromStore maps a store error onto an api Error. Unclassified errors are
// logged and reported as a bare internal error.
func fromStore(err error) *Error {
	if errors.Is(err, store.ErrNotFound) {
		return &Error{Code: CodeNotFound, Message: "not found"}
	}
	var constraintError *store.ConstraintError
	if errors.As(err, &constraintError) {
		return fromConstraint(constraintError)
	}
	log.Printf("internal error: %v", err)
	return &Error{Code: CodeInternal, Message: "internal error"}
}

// fromConstraint authors a stable message for a constraint violation.
func fromConstraint(violation *store.ConstraintError) *Error {
	column := violation.Column
	switch violation.Kind {
	case store.ConstraintForeignKey:
		return &Error{Code: CodeMissingReference, Message: "referenced resource does not exist"}
	case store.ConstraintCheck:
		if column == "" {
			return &Error{Code: CodeInvalidValue, Message: "value is not allowed"}
		}
		return &Error{Code: CodeInvalidValue, Message: "invalid value for " + column, Field: column}
	case store.ConstraintNotNull:
		if column == "" {
			return &Error{Code: CodeInvalidValue, Message: "required value is missing"}
		}
		return &Error{Code: CodeInvalidValue, Message: column + " must not be null", Field: column}
	case store.ConstraintDuplicate:
		if column == "" {
			return &Error{Code: CodeInvalidValue, Message: "duplicate value"}
		}
		return &Error{Code: CodeInvalidValue, Message: "duplicate value for " + column, Field: column}
	}
	log.Printf("unclassified constraint violation: %v", violation)
	return &Error{Code: CodeInternal, Message: "internal error"}
}

// notFoundOr returns a named NOT_FOUND for ErrNotFound and defers everything
// else to fromStore.
func notFoundOr(entity string, id int64, err error) *Error {
	if errors.Is(err, store.ErrNotFound) {
		return &Error{Code: CodeNotFound, Message: fmt.Sprintf("%s %d not found", entity, id)}
	}
	return fromStore(err)
}
