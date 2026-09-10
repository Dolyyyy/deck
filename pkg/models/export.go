package models

import "time"

// ExportEnvelope represents the top-level serialized container for exports.
type ExportEnvelope struct {
	Magic      string            `json:"magic"`                // "DECK_EXPORT"
	Version    string            `json:"version"`              // "1.0"
	CreatedAt  time.Time         `json:"created_at"`           // Timestamp
	Encrypted  bool              `json:"encrypted"`            // True if payload is AES-GCM encrypted
	Salt       string            `json:"salt,omitempty"`       // Argon2id salt (base64)
	Nonce      string            `json:"nonce,omitempty"`      // AES-GCM nonce (base64)
	Ciphertext string            `json:"ciphertext,omitempty"` // Encrypted payload (base64)
	Payload    *ExportData       `json:"payload,omitempty"`    // Unencrypted payload if Encrypted is false
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ExportData contains the actual exported resources.
type ExportData struct {
	Servers   []Server          `json:"servers" yaml:"servers"`
	Tunnels   []Tunnel          `json:"tunnels,omitempty" yaml:"tunnels,omitempty"`
	Aliases   []DirectoryAlias  `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Snippets  []Snippet         `json:"snippets,omitempty" yaml:"snippets,omitempty"`
	Endpoints []Endpoint        `json:"endpoints,omitempty" yaml:"endpoints,omitempty"`
	Tags      []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Meta      map[string]string `json:"meta,omitempty" yaml:"meta,omitempty"`
}
