package models

import (
	"os"
	"path/filepath"
	"strings"
)

// DirectoryAlias represents a bookmarked local folder shortcut.
type DirectoryAlias struct {
	Name        string   `json:"name" yaml:"name"`
	Path        string   `json:"path" yaml:"path"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ExpandedPath expands ~ and environment variables in the alias path.
func (a *DirectoryAlias) ExpandedPath() string {
	p := a.Path
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	return os.ExpandEnv(p)
}

// Exists checks if the bookmarked directory exists on the filesystem.
func (a *DirectoryAlias) Exists() bool {
	info, err := os.Stat(a.ExpandedPath())
	if err != nil {
		return false
	}
	return info.IsDir()
}

// MatchesQuery returns true if the alias matches the search query.
func (a *DirectoryAlias) MatchesQuery(q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(a.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(a.Path), q) {
		return true
	}
	if strings.Contains(strings.ToLower(a.Description), q) {
		return true
	}
	for _, tag := range a.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}
