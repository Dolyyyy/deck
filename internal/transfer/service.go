package transfer

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Dolyyyy/deck/internal/crypto"
	"github.com/Dolyyyy/deck/pkg/models"
)

var (
	ErrInvalidEnvelope = errors.New("invalid or unreadable export envelope")
)

// Service implements models.TransferService.
type Service struct {
	vault *crypto.Vault
}

// NewService creates a new transfer service.
func NewService(vault *crypto.Vault) *Service {
	if vault == nil {
		vault = crypto.NewVault("")
	}
	return &Service{vault: vault}
}

// ExportBase64 packages export data into a transportable Base64 string.
// If password is non-empty, payload is AES-256-GCM encrypted using Argon2id key derivation.
func (s *Service) ExportBase64(data *models.ExportData, password string) (string, error) {
	if data == nil {
		return "", errors.New("nil export data")
	}

	rawJSON, err := json.Marshal(data)
	if err != nil {
		return nilStr(), fmt.Errorf("failed to marshal export data: %w", err)
	}

	var envelope *models.ExportEnvelope

	if strings.TrimSpace(password) != "" {
		env, err := s.vault.Encrypt(rawJSON, password)
		if err != nil {
			return nilStr(), fmt.Errorf("encryption failed: %w", err)
		}
		envelope = env
	} else {
		envelope = &models.ExportEnvelope{
			Magic:     "DECK_EXPORT",
			Version:   "1.0",
			CreatedAt: time.Now().UTC(),
			Encrypted: false,
			Payload:   data,
		}
	}

	envJSON, err := json.Marshal(envelope)
	if err != nil {
		return nilStr(), fmt.Errorf("failed to marshal envelope: %w", err)
	}

	return base64.StdEncoding.EncodeToString(envJSON), nil
}

// ImportBase64 decodes and unpacks an export string (decrypting if necessary).
func (s *Service) ImportBase64(rawB64 string, password string) (*models.ExportData, error) {
	rawB64 = strings.TrimSpace(rawB64)
	if rawB64 == "" {
		return nil, errors.New("empty export string")
	}

	envBytes, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var envelope models.ExportEnvelope
	if err := json.Unmarshal(envBytes, &envelope); err != nil {
		return nil, fmt.Errorf("corrupted export envelope: %w", err)
	}

	if envelope.Magic != "DECK_EXPORT" {
		return nil, ErrInvalidEnvelope
	}

	if envelope.Encrypted {
		if strings.TrimSpace(password) == "" {
			return nil, errors.New("password required to decrypt this export")
		}

		decrypted, err := s.vault.Decrypt(&envelope, password)
		if err != nil {
			return nil, err
		}

		var data models.ExportData
		if err := json.Unmarshal(decrypted, &data); err != nil {
			return nil, fmt.Errorf("failed to parse decrypted data: %w", err)
		}
		return &data, nil
	}

	if envelope.Payload == nil {
		return nil, ErrInvalidEnvelope
	}

	return envelope.Payload, nil
}

func nilStr() string {
	return ""
}
