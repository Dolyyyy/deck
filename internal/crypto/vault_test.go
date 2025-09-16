package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestVault_EncryptDecrypt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deck-vault-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	v := NewVault(tmpDir)
	secretData := []byte("{\"servers\":[{\"name\":\"bastion-prod\",\"hostname\":\"10.0.0.1\"}]}")
	password := "CorrectHorseBatteryStaple123!"

	// 1. Test successful encryption & decryption
	envelope, err := v.Encrypt(secretData, password)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	if envelope == nil || !envelope.Encrypted {
		t.Fatal("expected encrypted envelope")
	}

	decrypted, err := v.Decrypt(envelope, password)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(secretData, decrypted) {
		t.Fatalf("decrypted data does not match original: got %s, want %s", string(decrypted), string(secretData))
	}

	// 2. Test wrong password
	_, err = v.Decrypt(envelope, "WrongPassword!")
	if err == nil {
		t.Fatal("expected error with wrong password, got nil")
	}

	// 3. Test empty password error
	_, err = v.Encrypt(secretData, "")
	if err != ErrEmptyPassword {
		t.Fatalf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestVault_MasterPassword(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deck-master-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	v := NewVault(tmpDir)

	if v.IsPasswordProtected() {
		t.Fatal("vault should not be password protected initially")
	}

	masterPass := "SuperSecretAdminPass123"
	if err := v.SetMasterPassword(masterPass); err != nil {
		t.Fatalf("failed to set master password: %v", err)
	}

	if !v.IsPasswordProtected() {
		t.Fatal("vault should now be password protected")
	}

	if !v.VerifyMasterPassword(masterPass) {
		t.Fatal("expected correct master password to verify")
	}

	if v.VerifyMasterPassword("WrongPassword") {
		t.Fatal("expected wrong master password to fail verification")
	}
}

func TestVault_ConfigDirFallback(t *testing.T) {
	v := NewVault("")
	if v.configDir == "" {
		t.Fatal("expected default config directory")
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "deck")
	if v.configDir != expected {
		t.Fatalf("got %s, want %s", v.configDir, expected)
	}
}
