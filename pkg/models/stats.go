package models

import (
	"strings"
	"time"
)

// ServerStats contains lightweight runtime performance metrics fetched from a remote server.
type ServerStats struct {
	Uptime      string    `json:"uptime" yaml:"uptime"`
	LoadAvg     string    `json:"load_avg" yaml:"load_avg"`
	MemoryUsed  string    `json:"memory_used" yaml:"memory_used"`
	MemoryTotal string    `json:"memory_total" yaml:"memory_total"`
	MemoryPct   int       `json:"memory_pct" yaml:"memory_pct"`
	DiskUsed    string    `json:"disk_used" yaml:"disk_used"`
	DiskTotal   string    `json:"disk_total" yaml:"disk_total"`
	DiskPct     int       `json:"disk_pct" yaml:"disk_pct"`
	OSInfo      string    `json:"os_info" yaml:"os_info"`
	FetchedAt   time.Time `json:"fetched_at" yaml:"fetched_at"`
	Error       string    `json:"error,omitempty" yaml:"error,omitempty"`
}

// PingResult represents the outcome of a server reachability test.
type PingResult struct {
	ServerName string
	Status     ServerStatus
	Latency    time.Duration
	Timestamp  time.Time
	Err        error
}

// Tunnel represents an SSH port-forwarding configuration.
type Tunnel struct {
	Name       string `json:"name" yaml:"name"`
	ServerName string `json:"server_name" yaml:"server_name"`
	LocalPort  int    `json:"local_port" yaml:"local_port"`
	RemoteHost string `json:"remote_host" yaml:"remote_host"`
	RemotePort int    `json:"remote_port" yaml:"remote_port"`
	Type       string `json:"type,omitempty" yaml:"type,omitempty"` // local, remote, dynamic
	Active     bool   `json:"active" yaml:"active"`
	PID        int    `json:"pid,omitempty" yaml:"pid,omitempty"`
	Error      string `json:"error,omitempty" yaml:"error,omitempty"`
}

// MatchesQuery returns true if the tunnel matches the search query.
func (t *Tunnel) MatchesQuery(q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(strings.TrimSpace(q))
	return strings.Contains(strings.ToLower(t.Name), q) ||
		strings.Contains(strings.ToLower(t.ServerName), q) ||
		strings.Contains(strings.ToLower(t.RemoteHost), q)
}
