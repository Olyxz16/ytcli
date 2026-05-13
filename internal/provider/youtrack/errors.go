package youtrack

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents a structured error from the YouTrack API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsAuthError returns true for 401 Unauthorized.
func IsAuthError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsNotFoundError returns true for 404 Not Found.
func IsNotFoundError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsValidationError returns true for 400 Bad Request.
func IsValidationError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusBadRequest
	}
	return false
}

// IsForbiddenError returns true for 403 Forbidden.
func IsForbiddenError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}

func newAPIError(statusCode int, body string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    strings.TrimSpace(body),
	}
}
