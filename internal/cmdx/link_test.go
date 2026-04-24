package cmdx

import (
	"testing"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"30m", 30},
		{"2h", 120},
		{"1d", 480},
		{"1d 2h 30m", 630},
		{"2h30m", 150},
		{"45", 45},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if err != nil {
				t.Fatalf("parseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDurationInvalid(t *testing.T) {
	_, err := parseDuration("abc")
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
