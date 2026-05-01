package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// DiscoverHubURL tries to find the Hub URL from a YouTrack base URL.
func DiscoverHubURL(youtrackURL string) string {
	youtrackURL = strings.TrimSuffix(youtrackURL, "/")
	// YouTrack Cloud: https://domain.youtrack.cloud -> https://domain.youtrack.cloud/hub
	if strings.Contains(youtrackURL, ".youtrack.cloud") {
		return youtrackURL + "/hub"
	}
	// YouTrack Server with /youtrack path
	if strings.HasSuffix(youtrackURL, "/youtrack") {
		return strings.TrimSuffix(youtrackURL, "/youtrack") + "/hub"
	}
	// Default: append /hub
	return youtrackURL + "/hub"
}

// Service represents a Hub service.
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Application string `json:"applicationName"`
	HomeURL     string `json:"homeUrl"`
}

// ListServices queries Hub for all registered services.
func ListServices(ctx context.Context, hubURL, token string) ([]Service, error) {
	hubURL = strings.TrimSuffix(hubURL, "/")
	req, err := http.NewRequestWithContext(ctx, "GET", hubURL+"/api/rest/services?fields=id,name,applicationName,homeUrl", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub services: HTTP %d", resp.StatusCode)
	}

	var services []Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}
	return services, nil
}

// FindYouTrackServiceID looks for the YouTrack service in Hub.
func FindYouTrackServiceID(services []Service) string {
	for _, s := range services {
		if strings.EqualFold(s.Name, "YouTrack") || strings.EqualFold(s.Application, "YouTrack") {
			return s.ID
		}
	}
	return ""
}
