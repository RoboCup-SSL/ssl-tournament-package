// HTTP plumbing shared by every handler: strict JSON decoding, query and path
// parsing, and the error envelope.
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/api"
)

var statusByCode = map[string]int{
	api.CodeInvalidJSON:      http.StatusBadRequest,
	api.CodeUnknownField:     http.StatusBadRequest,
	api.CodeInvalidValue:     http.StatusBadRequest,
	api.CodeNotFound:         http.StatusNotFound,
	api.CodeMissingReference: http.StatusConflict,
	api.CodeInternal:         http.StatusInternalServerError,
}

// writeJSON writes value as a JSON response with the given status.
func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	json.NewEncoder(writer).Encode(value)
}

// writeError writes the error envelope with the status matching its code.
func writeError(writer http.ResponseWriter, apiError *api.Error) {
	status, known := statusByCode[apiError.Code]
	if !known {
		status = http.StatusInternalServerError
	}
	writeJSON(writer, status, map[string]*api.Error{"error": apiError})
}

// decode strictly parses the JSON request body into target.
func decode(request *http.Request, target any) *api.Error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		return nil
	}
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return &api.Error{Code: api.CodeInvalidValue, Field: typeError.Field,
			Message: "invalid value for field " + typeError.Field}
	}
	if quoted, found := strings.CutPrefix(err.Error(), "json: unknown field "); found {
		field := strings.Trim(quoted, `"`)
		return &api.Error{Code: api.CodeUnknownField, Field: field,
			Message: "unknown field " + field}
	}
	return &api.Error{Code: api.CodeInvalidJSON, Message: err.Error()}
}

// pathID parses the {id} path segment as an integer.
func pathID(request *http.Request) (int64, *api.Error) {
	id, err := strconv.ParseInt(request.PathValue("id"), 10, 64)
	if err != nil {
		return 0, &api.Error{Code: api.CodeInvalidValue, Field: "id",
			Message: "id must be an integer"}
	}
	return id, nil
}

// queryFilters parses the allowed integer query parameters and rejects
// unknown ones.
func queryFilters(request *http.Request, allowed ...string) (map[string]*int64, *api.Error) {
	values := request.URL.Query()
	for name := range values {
		if !slices.Contains(allowed, name) {
			return nil, &api.Error{Code: api.CodeUnknownField, Field: name,
				Message: "unknown query parameter " + name}
		}
	}
	filters := make(map[string]*int64)
	for _, name := range allowed {
		raw := values.Get(name)
		if raw == "" {
			continue
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, &api.Error{Code: api.CodeInvalidValue, Field: name,
				Message: name + " must be an integer"}
		}
		filters[name] = &parsed
	}
	return filters, nil
}
