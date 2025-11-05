package transfer

import (
	"os"
	"testing"

	"github.com/Dolyyyy/deck/internal/crypto"
	"github.com/Dolyyyy/deck/pkg/models"
)

func TestService_ExportImportPlain(t *testing.T) {
	svc := NewService(nil)

	originalData := &models.ExportData{
		Servers: []models.Server{
			{Name: "web-prod", Hostname: "10.0.0.1", User: "root", Port: 22},
			{Name: "db-prod", Hostname: "10.0.0.2", User: "postgres", Port: 5432},
		},
		Aliases: []models.DirectoryAlias{
			{Name: "backend", Path: "/var/www/backend"},
		},
		Tags: []string{"prod", "infra"},
	}

	b64, err := svc.ExportBase64(originalData, "")
	if err != nil {
		t.Fatalf("ExportBase64 failed: %v", err)
	}

	imported, err := svc.ImportBase64(b64, "")
	if err != nil {
		t.Fatalf("ImportBase64 failed: %v", err)
	}

	if len(imported.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(imported.Servers))
	}
	if len(imported.Aliases) != 1 || imported.Aliases[0].Name != "backend" {
		t.Fatalf("unexpected aliases: %+v", imported.Aliases)
	}
}

func TestService_ExportImportEncrypted(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deck-transfer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	v := crypto.NewVault(tmpDir)
	svc := NewService(v)

	originalData := &models.ExportData{
		Servers: []models.Server{
			{Name: "secret-bastion", Hostname: "vpn.corp.com", User: "secops"},
		},
	}
	pass := "StrongPassword123!"

	b64, err := svc.ExportBase64(originalData, pass)
	if err != nil {
		t.Fatalf("ExportBase64 failed: %v", err)
	}

	// Try without password -> should fail
	_, err = svc.ImportBase64(b64, "")
	if err == nil {
		t.Fatal("expected error importing encrypted payload without password")
	}

	// Try with wrong password -> should fail
	_, err = svc.ImportBase64(b64, "WrongPassword")
	if err == nil {
		t.Fatal("expected error importing with wrong password")
	}

	// Try with correct password -> should succeed
	imported, err := svc.ImportBase64(b64, pass)
	if err != nil {
		t.Fatalf("ImportBase64 with correct password failed: %v", err)
	}

	if len(imported.Servers) != 1 || imported.Servers[0].Name != "secret-bastion" {
		t.Fatalf("unexpected imported servers: %+v", imported.Servers)
	}
}
