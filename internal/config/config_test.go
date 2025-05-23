package config

import (
	"os"
	"testing"

	"github.com/Dolyyyy/deck/pkg/models"
)

func TestParseSSHConfig(t *testing.T) {
	content := `
# My SSH Config
Host prod-web
    HostName 192.168.1.50
    User ubuntu
    Port 2222
    IdentityFile ~/.ssh/id_rsa
    ProxyJump bastion

Host staging-db
    HostName db.staging.internal
    User postgres
    Port 5432

Host *
    ServerAliveInterval 60
`
	tmpFile, err := os.CreateTemp("", "ssh_config_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	servers, err := ParseSSHConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseSSHConfig failed: %v", err)
	}

	if len(servers) != 2 {
		t.Fatalf("expected 2 valid servers (ignoring Host *), got %d", len(servers))
	}

	s1 := servers[0]
	if s1.Name != "prod-web" || s1.Hostname != "192.168.1.50" || s1.User != "ubuntu" || s1.Port != 2222 || s1.ProxyJump != "bastion" {
		t.Fatalf("unexpected s1 values: %+v", s1)
	}

	s2 := servers[1]
	if s2.Name != "staging-db" || s2.Hostname != "db.staging.internal" || s2.User != "postgres" || s2.Port != 5432 {
		t.Fatalf("unexpected s2 values: %+v", s2)
	}
}

func TestManager_Aliases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deck-cfg-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir, "")

	alias := &models.DirectoryAlias{
		Name:        "backend",
		Path:        "/var/www/backend",
		Description: "Main backend API service",
		Tags:        []string{"api", "golang"},
	}

	if err := mgr.AddOrUpdateAlias(alias); err != nil {
		t.Fatalf("failed to add alias: %v", err)
	}

	found, err := mgr.FindAlias("backend")
	if err != nil {
		t.Fatalf("FindAlias failed: %v", err)
	}
	if found.Path != alias.Path {
		t.Fatalf("got path %s, want %s", found.Path, alias.Path)
	}

	// Test prefix fuzzy match
	prefixFound, err := mgr.FindAlias("back")
	if err != nil {
		t.Fatalf("prefix FindAlias failed: %v", err)
	}
	if prefixFound.Name != "backend" {
		t.Fatalf("got name %s, want backend", prefixFound.Name)
	}

	// Test removal
	if err := mgr.RemoveAlias("backend"); err != nil {
		t.Fatalf("RemoveAlias failed: %v", err)
	}

	_, err = mgr.FindAlias("backend")
	if err == nil {
		t.Fatal("expected error finding removed alias, got nil")
	}
}
