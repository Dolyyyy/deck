package models

import "strings"

// Snippet represents a reusable terminal command or runbook script.
type Snippet struct {
	Name        string   `json:"name" yaml:"name"`
	Command     string   `json:"command" yaml:"command"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	TargetHost  string   `json:"target_host,omitempty" yaml:"target_host,omitempty"` // empty for local, or SSH host
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// MatchesQuery returns true if the snippet matches the search query.
func (s *Snippet) MatchesQuery(q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if strings.Contains(strings.ToLower(s.Name), q) ||
		strings.Contains(strings.ToLower(s.Command), q) ||
		strings.Contains(strings.ToLower(s.Description), q) ||
		strings.Contains(strings.ToLower(s.TargetHost), q) {
		return true
	}
	for _, t := range s.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}
