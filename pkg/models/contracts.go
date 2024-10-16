package models

import (
	"context"
)

// Pinger defines the contract for checking server reachability and latency.
type Pinger interface {
	Probe(ctx context.Context, server *Server) (*PingResult, error)
	ProbeAll(ctx context.Context, servers []*Server, concurrency int) <-chan *PingResult
}

// SSHExecutor handles interactive terminal handoff and background command execution.
type SSHExecutor interface {
	Connect(server *Server) error
	FetchStats(ctx context.Context, server *Server) (*ServerStats, error)
}

// ConfigProvider defines the contract for loading and persisting server inventories.
type ConfigProvider interface {
	Load() ([]*Server, error)
	Save(servers []*Server) error
}

// VaultService handles secure encryption, key derivation and payload decryption.
type VaultService interface {
	Encrypt(data []byte, password string) (*ExportEnvelope, error)
	Decrypt(envelope *ExportEnvelope, password string) ([]byte, error)
	IsPasswordProtected() bool
	SetMasterPassword(password string) error
	VerifyMasterPassword(password string) bool
}

// TransferService manages serialization, base64 encoding/decoding and selective imports.
type TransferService interface {
	ExportBase64(data *ExportData, password string) (string, error)
	ImportBase64(raw string, password string) (*ExportData, error)
}
