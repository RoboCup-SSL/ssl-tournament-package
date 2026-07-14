// Tests for the Opt tri-state and the patch-apply helpers.
package api

import (
	"encoding/json"
	"testing"
)

type optProbe struct {
	Name       Opt[string] `json:"name"`
	DivisionID Opt[int64]  `json:"division_id"`
}

func TestOptTriState(t *testing.T) {
	var probe optProbe
	if err := json.Unmarshal([]byte(`{"name": "RTT", "division_id": null}`), &probe); err != nil {
		t.Fatal(err)
	}
	if !probe.Name.Set || probe.Name.Null || probe.Name.Value != "RTT" {
		t.Errorf("value field decoded wrong: %+v", probe.Name)
	}
	if !probe.DivisionID.Set || !probe.DivisionID.Null {
		t.Errorf("null field decoded wrong: %+v", probe.DivisionID)
	}
	var absent optProbe
	if err := json.Unmarshal([]byte(`{}`), &absent); err != nil {
		t.Fatal(err)
	}
	if absent.Name.Set || absent.DivisionID.Set {
		t.Errorf("absent fields must stay unset: %+v", absent)
	}
}

func TestApplyHelpers(t *testing.T) {
	name := "old"
	if applyError := applyValue(Opt[string]{Set: true, Value: "new"}, &name, "name"); applyError != nil {
		t.Fatal(applyError)
	}
	if name != "new" {
		t.Errorf("applyValue did not overwrite: %q", name)
	}
	if applyError := applyValue(Opt[string]{}, &name, "name"); applyError != nil || name != "new" {
		t.Errorf("unset applyValue must be a no-op: %q %v", name, applyError)
	}
	applyError := applyValue(Opt[string]{Set: true, Null: true}, &name, "name")
	if applyError == nil || applyError.Code != CodeInvalidValue || applyError.Field != "name" {
		t.Errorf("null on NOT NULL field must be INVALID_VALUE: %+v", applyError)
	}
	var divisionID *int64
	applyNullable(Opt[int64]{Set: true, Value: 3}, &divisionID)
	if divisionID == nil || *divisionID != 3 {
		t.Errorf("applyNullable set failed: %v", divisionID)
	}
	applyNullable(Opt[int64]{Set: true, Null: true}, &divisionID)
	if divisionID != nil {
		t.Errorf("applyNullable null must clear: %v", divisionID)
	}
	previous := int64(7)
	divisionID = &previous
	applyNullable(Opt[int64]{}, &divisionID)
	if divisionID == nil || *divisionID != 7 {
		t.Errorf("unset applyNullable must be a no-op: %v", divisionID)
	}
	if replacement := listValue(Opt[[]int64]{}); replacement != nil {
		t.Errorf("unset list must be nil: %v", replacement)
	}
	if replacement := listValue(Opt[[]int64]{Set: true, Null: true}); replacement == nil || len(*replacement) != 0 {
		t.Errorf("null list must replace with empty: %v", replacement)
	}
	if replacement := listValue(Opt[[]int64]{Set: true, Value: []int64{1, 2}}); replacement == nil || len(*replacement) != 2 {
		t.Errorf("value list must pass through: %v", replacement)
	}
}
