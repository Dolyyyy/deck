package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Dolyyyy/deck/pkg/models"
	"gopkg.in/yaml.v3"
)

// DeckConfig stores user-defined servers, directory aliases, tunnels, snippets, endpoints, and settings.
type DeckConfig struct {
	Servers   []*models.Server         `yaml:"servers"`
	Aliases   []*models.DirectoryAlias `yaml:"aliases"`
	Tunnels   []*models.Tunnel         `yaml:"tunnels"`
	Snippets  []*models.Snippet        `yaml:"snippets"`
	Endpoints []*models.Endpoint       `yaml:"endpoints"`
	Settings  models.Settings          `yaml:"settings"`
}

// Manager handles loading, saving, and merging configuration from SSH config and Deck config.
type Manager struct {
	configDir     string
	sshConfigPath string
	mu            sync.RWMutex
}

// NewManager creates a new configuration manager.
func NewManager(configDir string, sshConfigPath string) *Manager {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(home, ".config", "deck")
		} else {
			configDir = ".deck"
		}
	}
	return &Manager{
		configDir:     configDir,
		sshConfigPath: sshConfigPath,
	}
}

// ConfigFilePath returns the absolute path to deck.yaml.
func (m *Manager) ConfigFilePath() string {
	return filepath.Join(m.configDir, "deck.yaml")
}

// LoadDeckConfig reads ~/.config/deck/deck.yaml.
func (m *Manager) LoadDeckConfig() (*DeckConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := m.ConfigFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DeckConfig{
				Servers:   []*models.Server{},
				Aliases:   []*models.DirectoryAlias{},
				Tunnels:   []*models.Tunnel{},
				Snippets:  []*models.Snippet{},
				Endpoints: []*models.Endpoint{},
				Settings:  models.DefaultSettings(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read deck config: %w", err)
	}

	var cfg DeckConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse deck.yaml: %w", err)
	}

	if cfg.Settings.Theme == "" {
		cfg.Settings = models.DefaultSettings()
	}

	for _, s := range cfg.Servers {
		s.Source = "deck_yaml"
		if s.Status == "" {
			s.Status = models.StatusUnknown
		}
	}

	return &cfg, nil
}

// SaveDeckConfig writes to ~/.config/deck/deck.yaml.
func (m *Manager) SaveDeckConfig(cfg *DeckConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = os.MkdirAll(m.configDir, 0700)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(m.ConfigFilePath(), data, 0600)
}

// LoadAllServers loads both ~/.ssh/config and ~/.config/deck/deck.yaml servers, merging them cleanly.
func (m *Manager) LoadAllServers() ([]*models.Server, error) {
	sshServers, err := ParseSSHConfig(m.sshConfigPath)
	if err != nil {
		// Log and continue if ssh config fails
		sshServers = []*models.Server{}
	}

	deckCfg, err := m.LoadDeckConfig()
	if err != nil {
		deckCfg = &DeckConfig{}
	}

	serverMap := make(map[string]*models.Server)

	// First load SSH servers
	for _, s := range sshServers {
		serverMap[s.Name] = s
	}

	// Merge/override with custom deck servers (preserves custom tags, environments)
	for _, s := range deckCfg.Servers {
		if existing, ok := serverMap[s.Name]; ok {
			if len(s.Tags) > 0 {
				existing.Tags = s.Tags
			}
			if s.Environment != "" {
				existing.Environment = s.Environment
			}
			if s.Description != "" {
				existing.Description = s.Description
			}
		} else {
			serverMap[s.Name] = s
		}
	}

	result := make([]*models.Server, 0, len(serverMap))
	for _, s := range serverMap {
		result = append(result, s)
	}

	return result, nil
}

// AddOrUpdateServer adds or updates a server definition in ~/.config/deck/deck.yaml.
func (m *Manager) AddOrUpdateServer(server *models.Server) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{}
	}

	var updated bool
	for i, s := range cfg.Servers {
		if strings.EqualFold(s.Name, server.Name) {
			cfg.Servers[i] = server
			updated = true
			break
		}
	}
	if !updated {
		cfg.Servers = append(cfg.Servers, server)
	}

	return m.SaveDeckConfig(cfg)
}

// AddOrUpdateAlias adds or updates a directory alias in ~/.config/deck/deck.yaml.
func (m *Manager) AddOrUpdateAlias(alias *models.DirectoryAlias) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{}
	}

	var updated bool
	for i, a := range cfg.Aliases {
		if strings.EqualFold(a.Name, alias.Name) {
			cfg.Aliases[i] = alias
			updated = true
			break
		}
	}
	if !updated {
		cfg.Aliases = append(cfg.Aliases, alias)
	}

	return m.SaveDeckConfig(cfg)
}

// RemoveAlias removes a directory alias by name.
func (m *Manager) RemoveAlias(name string) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		return err
	}

	var remaining []*models.DirectoryAlias
	for _, a := range cfg.Aliases {
		if !strings.EqualFold(a.Name, name) {
			remaining = append(remaining, a)
		}
	}
	cfg.Aliases = remaining

	return m.SaveDeckConfig(cfg)
}

// AddOrUpdateTunnel adds or updates an SSH tunnel in ~/.config/deck/deck.yaml.
func (m *Manager) AddOrUpdateTunnel(tunnel *models.Tunnel) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{}
	}

	var updated bool
	for i, t := range cfg.Tunnels {
		if strings.EqualFold(t.Name, tunnel.Name) {
			cfg.Tunnels[i] = tunnel
			updated = true
			break
		}
	}
	if !updated {
		cfg.Tunnels = append(cfg.Tunnels, tunnel)
	}

	return m.SaveDeckConfig(cfg)
}

// RemoveTunnel removes an SSH tunnel by name.
func (m *Manager) RemoveTunnel(name string) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		return err
	}

	var remaining []*models.Tunnel
	for _, t := range cfg.Tunnels {
		if !strings.EqualFold(t.Name, name) {
			remaining = append(remaining, t)
		}
	}
	cfg.Tunnels = remaining

	return m.SaveDeckConfig(cfg)
}

// AddOrUpdateSnippet adds or updates a command snippet in ~/.config/deck/deck.yaml.
func (m *Manager) AddOrUpdateSnippet(snippet *models.Snippet) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{}
	}

	var updated bool
	for i, s := range cfg.Snippets {
		if strings.EqualFold(s.Name, snippet.Name) {
			cfg.Snippets[i] = snippet
			updated = true
			break
		}
	}
	if !updated {
		cfg.Snippets = append(cfg.Snippets, snippet)
	}

	return m.SaveDeckConfig(cfg)
}

// RemoveSnippet removes a command snippet by name.
func (m *Manager) RemoveSnippet(name string) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		return err
	}

	var remaining []*models.Snippet
	for _, s := range cfg.Snippets {
		if !strings.EqualFold(s.Name, name) {
			remaining = append(remaining, s)
		}
	}
	cfg.Snippets = remaining

	return m.SaveDeckConfig(cfg)
}

// AddOrUpdateEndpoint adds or updates an endpoint in ~/.config/deck/deck.yaml.
func (m *Manager) AddOrUpdateEndpoint(endpoint *models.Endpoint) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{}
	}

	var updated bool
	for i, e := range cfg.Endpoints {
		if strings.EqualFold(e.Name, endpoint.Name) {
			cfg.Endpoints[i] = endpoint
			updated = true
			break
		}
	}
	if !updated {
		cfg.Endpoints = append(cfg.Endpoints, endpoint)
	}

	return m.SaveDeckConfig(cfg)
}

// RemoveEndpoint removes an endpoint by name.
func (m *Manager) RemoveEndpoint(name string) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		return err
	}

	var remaining []*models.Endpoint
	for _, e := range cfg.Endpoints {
		if !strings.EqualFold(e.Name, name) {
			remaining = append(remaining, e)
		}
	}
	cfg.Endpoints = remaining

	return m.SaveDeckConfig(cfg)
}

// UpdateSettings updates user settings and saves to ~/.config/deck/deck.yaml.
func (m *Manager) UpdateSettings(settings models.Settings) error {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		cfg = &DeckConfig{
			Servers:   []*models.Server{},
			Aliases:   []*models.DirectoryAlias{},
			Tunnels:   []*models.Tunnel{},
			Snippets:  []*models.Snippet{},
			Endpoints: []*models.Endpoint{},
		}
	}
	cfg.Settings = settings
	return m.SaveDeckConfig(cfg)
}

// FindAlias finds a directory alias by exact or case-insensitive match.
func (m *Manager) FindAlias(query string) (*models.DirectoryAlias, error) {
	cfg, err := m.LoadDeckConfig()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(strings.TrimSpace(query))
	for _, a := range cfg.Aliases {
		if strings.ToLower(a.Name) == query {
			return a, nil
		}
	}

	// Fuzzy prefix match
	for _, a := range cfg.Aliases {
		if strings.HasPrefix(strings.ToLower(a.Name), query) {
			return a, nil
		}
	}

	return nil, fmt.Errorf("alias %q not found", query)
}

// ShellIntegrationScript generates the shell function for bash, zsh, or fish to enable seamless cd.
func ShellIntegrationScript(shell string) string {
	switch strings.ToLower(shell) {
	case "zsh", "bash":
		return `# deck shell integration
deck() {
    if [ "$1" = "cd" ] || [ "$1" = "jump" ]; then
        shift
        local target_dir
        target_dir=$(command deck path "$@")
        if [ -n "$target_dir" ] && [ -d "$target_dir" ]; then
            cd "$target_dir" || return
        fi
    else
        command deck "$@"
    fi
}
`
	case "fish":
		return `# deck fish integration
function deck
    if test "$argv[1]" = "cd" -o "$argv[1]" = "jump"
        set -l target_dir (command deck path $argv[2..-1])
        if test -n "$target_dir" -a -d "$target_dir"
            cd "$target_dir"
        end
    else
        command deck $argv
    end
end
`
	case "powershell", "pwsh":
		return `# deck PowerShell integration
function deck {
    param([Parameter(ValueFromRemainingArguments = $true)]$args)
    if ($args[0] -eq "cd" -or $args[0] -eq "jump") {
        $target = & (Get-Command deck -CommandType Application) path $args[1..($args.Length-1)]
        if ($target -and (Test-Path $target)) {
            Set-Location $target
        }
    } else {
        & (Get-Command deck -CommandType Application) $args
    }
}
`
	default:
		return ""
	}
}
