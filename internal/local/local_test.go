package local

import (
	"testing"

	"github.com/Olyxz16/tkt/internal/config"
)

func TestGetSchemaDefault(t *testing.T) {
	schema := GetSchema(nil)
	if len(schema.States) == 0 {
		t.Error("expected default states")
	}
	if schema.DefaultState != "Open" {
		t.Errorf("default state = %q", schema.DefaultState)
	}
	if schema.DefaultPriority != "Normal" {
		t.Errorf("default priority = %q", schema.DefaultPriority)
	}
}

func TestGetSchemaCustom(t *testing.T) {
	cfg := &config.LocalConfig{
		Local: config.LocalSchema{
			States:     []string{"Todo", "Done"},
			Priorities: []string{"High", "Low"},
		},
	}
	schema := GetSchema(cfg)
	if len(schema.States) != 2 {
		t.Errorf("states len = %d", len(schema.States))
	}
}

func TestValidateState(t *testing.T) {
	schema := DefaultSchema
	if err := ValidateState(schema, "Open"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateState(schema, "Missing"); err == nil {
		t.Error("expected error for invalid state")
	}
}

func TestValidatePriority(t *testing.T) {
	schema := DefaultSchema
	if err := ValidatePriority(schema, "Critical"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidatePriority(schema, "Missing"); err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestIsDoneState(t *testing.T) {
	schema := DefaultSchema
	if !IsDoneState(schema, "Done") {
		t.Error("expected Done to be a done state")
	}
	if !IsDoneState(schema, "Closed") {
		t.Error("expected Closed to be a done state")
	}
	if IsDoneState(schema, "Open") {
		t.Error("expected Open not to be a done state")
	}
}
