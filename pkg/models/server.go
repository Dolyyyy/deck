package models

import (
	"fmt"
	"strings"
	"time"
)

// ServerStatus represents the connectivity state of a server.
type ServerStatus string

const (
	StatusOnline   ServerStatus = "online"
	StatusOffline  ServerStatus = "offline"
	StatusChecking ServerStatus = "checking"
	StatusUnknown  ServerStatus = "unknown"
)

// Server represents an individual SSH host or target in deck.
type Server struct {
	Name         string            `json:"name" yaml:"name"`
	Hostname     string            `json:"hostname" yaml:"hostname"`
	User         string            `json:"user,omitempty" yaml:"user,omitempty"`
	Port         int               `json:"port,omitempty" yaml:"port,omitempty"`
	IdentityFile string            `json:"identity_file,omitempty" yaml:"identity_file,omitempty"`
	ProxyJump    string            `json:"proxy_jump,omitempty" yaml:"proxy_jump,omitempty"`
	Tags         []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Environment  string            `json:"environment,omitempty" yaml:"environment,omitempty"`
	Description  string            `json:"description,omitempty" yaml:"description,omitempty"`
	Source       string            `json:"source" yaml:"source"` // "ssh_config" or "deck_yaml"
	Status       ServerStatus      `json:"status" yaml:"status"`
	Latency      time.Duration     `json:"latency" yaml:"latency"`
	LastChecked  time.Time         `json:"last_checked" yaml:"last_checked"`
	Stats        *ServerStats      `json:"stats,omitempty" yaml:"stats,omitempty"`
	ExtraOptions map[string]string `json:"extra_options,omitempty" yaml:"extra_options,omitempty"`
}

// EffectivePort returns the port or default 22.
func (s *Server) EffectivePort() int {
	if s.Port <= 0 {
		return 22
	}
	return s.Port
}

// EffectiveUser returns the configured user or empty string.
func (s *Server) EffectiveUser() string {
	return strings.TrimSpace(s.User)
}

// DisplayAddress returns user@host:port or host:port.
func (s *Server) DisplayAddress() string {
	target := s.Hostname
	if target == "" {
		target = s.Name
	}
	if s.User != "" {
		target = fmt.Sprintf("%s@%s", s.User, target)
	}
	if s.EffectivePort() != 22 {
		target = fmt.Sprintf("%s:%d", target, s.EffectivePort())
	}
	return target
}

// MatchesQuery returns true if the server matches the search query.
func (s *Server) MatchesQuery(q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(s.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Hostname), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.User), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Environment), q) {
		return true
	}
	for _, tag := range s.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}
