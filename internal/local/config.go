package local

import (
	"fmt"

	"github.com/Olyxz16/tkt/internal/config"
)

// DefaultSchema provides sensible defaults for a local-only project.
var DefaultSchema = config.LocalSchema{
	States: []string{"Open", "In Progress", "Review", "Done", "Closed"},
	Priorities: []string{"Critical", "Major", "Normal", "Minor"},
	DefaultState: "Open",
	DefaultPriority: "Normal",
	DoneStates: []string{"Done", "Closed"},
}

// GetSchema returns the effective schema for a project.
// Falls back to defaults if not configured.
func GetSchema(cfg *config.LocalConfig) config.LocalSchema {
	if cfg == nil || len(cfg.Local.States) == 0 {
		return DefaultSchema
	}
	return cfg.Local
}

// ValidateState checks if a state is allowed in the schema.
func ValidateState(schema config.LocalSchema, state string) error {
	for _, s := range schema.States {
		if s == state {
			return nil
		}
	}
	return fmt.Errorf("state %q not in schema (%v)", state, schema.States)
}

// ValidatePriority checks if a priority is allowed in the schema.
func ValidatePriority(schema config.LocalSchema, priority string) error {
	for _, p := range schema.Priorities {
		if p == priority {
			return nil
		}
	}
	return fmt.Errorf("priority %q not in schema (%v)", priority, schema.Priorities)
}

// IsDoneState returns true if the state is considered "done".
func IsDoneState(schema config.LocalSchema, state string) bool {
	for _, s := range schema.DoneStates {
		if s == state {
			return true
		}
	}
	return false
}
