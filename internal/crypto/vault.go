package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Dolyyyy/deck/pkg/models"
	"golang.org/x/crypto/argon2"
	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidPasswordOrCorrupted = errors.New("invalid password or corrupted data")
	ErrEmptyPassword              = errors.New("password cannot be empty")
	ErrNoData                     = errors.New("no data to encrypt")
)

const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 4
	argonKeyLength   = 32 // 256 bits for AES-256
	saltLength       = 16 // 128 bits
	nonceLength      = 12 // 96 bits for AES-GCM
)

// Vault handles cryptographic operations including Argon2id key derivation and AES-256-GCM.
type Vault struct {
	configDir string
}

// NewVault creates a new Vault instance.
func NewVault(configDir string) *Vault {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(home, ".config", "deck")
		} else {
			configDir = ".deck"
		}
	}
	return &Vault{configDir: configDir}
}

// DeriveKey produces a 256-bit AES key from a password and salt using Argon2id.
func (v *Vault) DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
}

// Encrypt encrypts plaintext bytes with AES-256-GCM using an Argon2id derived key.
func (v *Vault) Encrypt(plaintext []byte, password string) (*models.ExportEnvelope, error) {
	if len(plaintext) == 0 {
		return nil, ErrNoData
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate crypto salt: %w", err)
	}

	key := v.DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM instance: %w", err)
	}

	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate GCM nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &models.ExportEnvelope{
		Magic:      "DECK_EXPORT",
		Version:    "1.0",
		CreatedAt:  time.Now().UTC(),
		Encrypted:  true,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// Decrypt decrypts an ExportEnvelope using AES-256-GCM and the supplied password.
func (v *Vault) Decrypt(envelope *models.ExportEnvelope, password string) ([]byte, error) {
	if envelope == nil {
		return nil, errors.New("nil envelope")
	}
	if !envelope.Encrypted {
		return nil, errors.New("envelope is not encrypted")
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}

	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) != saltLength {
		return nil, ErrInvalidPasswordOrCorrupted
	}

	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != nonceLength {
		return nil, ErrInvalidPasswordOrCorrupted
	}

	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil || len(ciphertext) == 0 {
		return nil, ErrInvalidPasswordOrCorrupted
	}

	key := v.DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM instance: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidPasswordOrCorrupted
	}

	return plaintext, nil
}

type vaultMeta struct {
	Salt string `yaml:"salt"`
	Hash string `yaml:"hash"`
}

// SetMasterPassword stores a hashed master password to lock the Deck instance.
func (v *Vault) SetMasterPassword(password string) error {
	if password == "" {
		return ErrEmptyPassword
	}
	_ = os.MkdirAll(v.configDir, 0700)

	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}

	hash := v.DeriveKey(password, salt)

	meta := vaultMeta{
		Salt: base64.StdEncoding.EncodeToString(salt),
		Hash: base64.StdEncoding.EncodeToString(hash),
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	filePath := filepath.Join(v.configDir, "vault.meta")
	return os.WriteFile(filePath, data, 0600)
}

// IsPasswordProtected checks if a master password has been configured.
func (v *Vault) IsPasswordProtected() bool {
	filePath := filepath.Join(v.configDir, "vault.meta")
	_, err := os.Stat(filePath)
	return err == nil
}

// VerifyMasterPassword validates a given master password against the stored hash.
func (v *Vault) VerifyMasterPassword(password string) bool {
	filePath := filepath.Join(v.configDir, "vault.meta")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	var meta vaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return false
	}

	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return false
	}

	expectedHash, err := base64.StdEncoding.DecodeString(meta.Hash)
	if err != nil {
		return false
	}

	computedHash := v.DeriveKey(password, salt)
	return subtle.ConstantTimeCompare(expectedHash, computedHash) == 1
}
