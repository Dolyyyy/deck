package models

import (
	"testing"
)

func TestServer_EffectivePort(t *testing.T) {
	s1 := &Server{Name: "srv1", Port: 0}
	if s1.EffectivePort() != 22 {
		t.Fatalf("expected default port 22, got %d", s1.EffectivePort())
	}

	s2 := &Server{Name: "srv2", Port: 2222}
	if s2.EffectivePort() != 2222 {
		t.Fatalf("expected port 2222, got %d", s2.EffectivePort())
	}
}

func TestServer_DisplayAddress(t *testing.T) {
	s := &Server{
		Name:     "bastion",
		Hostname: "192.168.1.10",
		User:     "ubuntu",
		Port:     22,
	}
	if s.DisplayAddress() != "ubuntu@192.168.1.10" {
		t.Fatalf("unexpected display address: %s", s.DisplayAddress())
	}

	s.Port = 2202
	if s.DisplayAddress() != "ubuntu@192.168.1.10:2202" {
		t.Fatalf("unexpected display address with custom port: %s", s.DisplayAddress())
	}
}

func TestServer_MatchesQuery(t *testing.T) {
	s := &Server{
		Name:        "prod-db-master",
		Hostname:    "10.0.0.5",
		User:        "postgres",
		Environment: "production",
		Tags:        []string{"database", "primary", "critical"},
	}

	tests := []struct {
		query    string
		expected bool
	}{
		{"", true},
		{"prod", true},
		{"master", true},
		{"10.0.0", true},
		{"postgres", true},
		{"database", true},
		{"critical", true},
		{"staging", false},
		{"redis", false},
	}

	for _, tc := range tests {
		got := s.MatchesQuery(tc.query)
		if got != tc.expected {
			t.Errorf("MatchesQuery(%q) = %v; want %v", tc.query, got, tc.expected)
		}
	}
}
