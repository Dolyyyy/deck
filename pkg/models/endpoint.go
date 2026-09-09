package models

import (
	"strings"
	"time"
)

// EndpointStatus defines the health state of an HTTP/HTTPS endpoint.
type EndpointStatus string

const (
	EndpointStatusUp       EndpointStatus = "UP"
	EndpointStatusDegraded EndpointStatus = "DEGRADED"
	EndpointStatusDown     EndpointStatus = "DOWN"
	EndpointStatusChecking EndpointStatus = "CHECKING"
	EndpointStatusUnknown  EndpointStatus = "UNKNOWN"
)

// Endpoint represents an HTTP or HTTPS service to monitor for availability and SSL expiration.
type Endpoint struct {
	Name             string         `json:"name" yaml:"name"`
	URL              string         `json:"url" yaml:"url"`
	StatusCode       int            `json:"status_code,omitempty" yaml:"status_code,omitempty"`
	Latency          time.Duration  `json:"latency,omitempty" yaml:"latency,omitempty"`
	SSLDaysRemaining int            `json:"ssl_days_remaining,omitempty" yaml:"ssl_days_remaining,omitempty"`
	SSLExpiry        time.Time      `json:"ssl_expiry,omitempty" yaml:"ssl_expiry,omitempty"`
	Status           EndpointStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Error            string         `json:"error,omitempty" yaml:"error,omitempty"`
	LastChecked      time.Time      `json:"last_checked,omitempty" yaml:"last_checked,omitempty"`
}

// MatchesQuery returns true if the endpoint matches the search query.
func (e *Endpoint) MatchesQuery(q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(strings.TrimSpace(q))
	return strings.Contains(strings.ToLower(e.Name), q) ||
		strings.Contains(strings.ToLower(e.URL), q)
}
