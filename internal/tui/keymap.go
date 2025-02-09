package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keyboard shortcuts for deck TUI.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Inspect  key.Binding
	Filter   key.Binding
	Clear    key.Binding
	Tab      key.Binding
	Refresh  key.Binding
	Edit     key.Binding
	Export   key.Binding
	Import   key.Binding
	Add      key.Binding
	Delete   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default keyboard configuration.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect / jump"),
		),
		Inspect: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "inspect stats"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter / search"),
		),
		Clear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear / close"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "p"),
			key.WithHelp("r/p", "re-ping all"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit connection"),
		),
		Export: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "export base64"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import base64"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add alias/server"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete item"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
