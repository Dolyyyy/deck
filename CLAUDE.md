# Claude & Assistant Quick Reference for Deck

## Common Commands
* **Run application directly**: `go run .` or `go run main.go`
* **Build binary**: `make build` or `go build -o bin/deck ./cmd/deck`
* **Run all tests**: `go test -v -race ./...`
* **Run package test**: `go test -v ./internal/crypto/...`
* **Check cross-compilation**: `make cross-build`
* **Format & tidy**: `go fmt ./... && go mod tidy`

## Key Locations
* `pkg/models/` — Data contracts and structs (`Server`, `DirectoryAlias`, `Stats`, `ExportData`)
* `internal/config/` — `~/.ssh/config` parser and `deck.yaml` manager
* `internal/tui/` — Bubbletea application, views, and Lipgloss styles
* `internal/crypto/` — AES-256-GCM and Argon2id encryption vault
* `internal/ssh/` — Native OpenSSH execution and remote metrics parser
