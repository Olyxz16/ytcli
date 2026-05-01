package oauth

import (
	"testing"
)

func TestDiscoverHubURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://ytcli.youtrack.cloud", "https://ytcli.youtrack.cloud/hub"},
		{"https://ytcli.youtrack.cloud/", "https://ytcli.youtrack.cloud/hub"},
		{"https://company.com/youtrack", "https://company.com/hub"},
		{"https://company.com/youtrack/", "https://company.com/hub"},
		{"https://server.local", "https://server.local/hub"},
	}

	for _, tt := range tests {
		got := DiscoverHubURL(tt.input)
		if got != tt.expected {
			t.Errorf("DiscoverHubURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFindYouTrackServiceID(t *testing.T) {
	services := []Service{
		{ID: "s1", Name: "Hub", Application: "Hub"},
		{ID: "s2", Name: "YouTrack", Application: "YouTrack"},
	}
	id := FindYouTrackServiceID(services)
	if id != "s2" {
		t.Errorf("got %q, want s2", id)
	}
}

func TestFindYouTrackServiceIDNotFound(t *testing.T) {
	services := []Service{{ID: "s1", Name: "Hub"}}
	id := FindYouTrackServiceID(services)
	if id != "" {
		t.Errorf("got %q, want empty", id)
	}
}
