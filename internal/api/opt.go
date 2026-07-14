// Opt models one PATCH field with three states: absent, null, or a value.
package api

import "encoding/json"

// Opt distinguishes an absent JSON key (Set false) from an explicit null
// (Set and Null) from a value (Set with Value).
type Opt[T any] struct {
	Set   bool
	Null  bool
	Value T
}

// UnmarshalJSON records presence; encoding/json only calls it when the key
// exists, which is what makes absent distinguishable from null.
func (o *Opt[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(data, &o.Value)
}

// applyValue overwrites target when field is set; null is rejected because
// the underlying column is NOT NULL.
func applyValue[T any](field Opt[T], target *T, name string) *Error {
	if !field.Set {
		return nil
	}
	if field.Null {
		return &Error{Code: CodeInvalidValue, Field: name, Message: name + " cannot be null"}
	}
	*target = field.Value
	return nil
}

// applyNullable overwrites target when field is set; null clears the column.
func applyNullable[T any](field Opt[T], target **T) {
	if !field.Set {
		return
	}
	if field.Null {
		*target = nil
		return
	}
	value := field.Value
	*target = &value
}

// listValue converts a list patch field into the store's replace pointer:
// nil means untouched, empty means clear.
func listValue[T any](field Opt[[]T]) *[]T {
	if !field.Set {
		return nil
	}
	if field.Null || field.Value == nil {
		empty := []T{}
		return &empty
	}
	return &field.Value
}
