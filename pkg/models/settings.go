package models

// Settings holds configurable user preferences for Deck.
type Settings struct {
	AutoSyncShellAliases bool   `yaml:"auto_sync_shell_aliases" json:"auto_sync_shell_aliases"`
	Theme                string `yaml:"theme" json:"theme"`
	AutoPingOnLaunch     bool   `yaml:"auto_ping_on_launch" json:"auto_ping_on_launch"`
	PingInterval         int    `yaml:"ping_interval" json:"ping_interval"` // In seconds (0 = disabled)
	DefaultSSHUser       string `yaml:"default_ssh_user" json:"default_ssh_user"`
	ConfirmOnDelete      bool   `yaml:"confirm_on_delete" json:"confirm_on_delete"`
	CheckUpdatesOnStart  bool   `yaml:"check_updates_on_start" json:"check_updates_on_start"`
}

// DefaultSettings returns Deck default settings.
func DefaultSettings() Settings {
	return Settings{
		AutoSyncShellAliases: false,
		Theme:                "Catppuccin Mocha",
		AutoPingOnLaunch:     true,
		PingInterval:         30,
		DefaultSSHUser:       "root",
		ConfirmOnDelete:      true,
		CheckUpdatesOnStart:  true,
	}
}
