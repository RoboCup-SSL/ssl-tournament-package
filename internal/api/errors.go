// Package api defines the application's operations independent of transport;
// HTTP handlers (and a future MCP server) call these free functions.
package api

import (
	"errors"
	"fmt"
	"strings"

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

// Error is the transport-independent operation error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// Error renders the code and message.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// fromStore maps a store error onto an api Error.
func fromStore(err error) *Error {
	if errors.Is(err, store.ErrNotFound) {
		return &Error{Code: CodeNotFound, Message: "not found"}
	}
	var constraintError *store.ConstraintError
	if errors.As(err, &constraintError) {
		if constraintError.Kind == store.ConstraintForeignKey {
			return &Error{Code: CodeMissingReference, Message: constraintError.Detail}
		}
		return &Error{
			Code:    CodeInvalidValue,
			Message: constraintError.Detail,
			Field:   fieldFromDetail(constraintError.Detail),
		}
	}
	return &Error{Code: CodeInternal, Message: err.Error()}
}

// notFoundOr returns a named NOT_FOUND for ErrNotFound and defers everything
// else to fromStore.
func notFoundOr(entity string, id int64, err error) *Error {
	if errors.Is(err, store.ErrNotFound) {
		return &Error{Code: CodeNotFound, Message: fmt.Sprintf("%s %d not found", entity, id)}
	}
	return fromStore(err)
}

// fieldFromDetail extracts the column name from texts like
// "NOT NULL constraint failed: team.name"; empty when the text has no
// table.column suffix.
func fieldFromDetail(detail string) string {
	_, suffix, found := strings.Cut(detail, "constraint failed: ")
	if !found || strings.Contains(suffix, ",") {
		return ""
	}
	_, column, found := strings.Cut(suffix, ".")
	if !found || strings.Contains(column, " ") {
		return ""
	}
	return column
}
