package tui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyBindings defines all keyboard shortcuts.
type KeyBindings struct {
	Up         []string
	Down       []string
	PageUp     []string
	PageDown   []string
	Home       []string
	End        []string
	Enter      []string
	Quit       []string
	Local      []string
	Project    []string
	Toggle     []string
	Uninstall  []string
	Update     []string // Update plugin to latest version
	Enable     []string // Toggle plugin enabled/disabled state
	Escape     []string
	Filter     []string
	Refresh    []string
	Mouse      []string
	Sort       []string // Cycle through sort options
	Readme     []string // View plugin README
	Changelog  []string // View plugin CHANGELOG
	Config     []string // Open plugin config viewer
	BulkToggle []string // Toggle bulk selection on current plugin
	BulkAll    []string // Select all plugins
	BulkNone   []string // Deselect all plugins
	Scope      []string // Open multi-scope dialog
}

// DefaultKeyBindings returns the default key bindings.
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Up:         []string{"up", "k"},
		Down:       []string{"down", "j"},
		PageUp:     []string{"pgup", "ctrl+u"},
		PageDown:   []string{"pgdown", "ctrl+d"},
		Home:       []string{"home", "g"},
		End:        []string{"end", "G"},
		Enter:      []string{"enter"},
		Quit:       []string{"q", "ctrl+c"},
		Local:      []string{"l"},
		Project:    []string{"p"},
		Toggle:     []string{"tab"},
		Uninstall:  []string{"u"},
		Update:     []string{"U"}, // Shift+u for update
		Enable:     []string{"e"},
		Escape:     []string{"esc"},
		Filter:     []string{"/"},
		Refresh:    []string{"r"},
		Mouse:      []string{"m"},
		Sort:       []string{"s"},
		Readme:     []string{"?"},
		Changelog:  []string{"C"},
		Config:     []string{"c"},
		BulkToggle: []string{" "}, // Space to toggle selection
		BulkAll:    []string{"a"}, // Select all
		BulkNone:   []string{"A"}, // Deselect all (shift+a)
		Scope:      []string{"S"}, // Shift+s for scope dialog
	}
}

// matchesKey returns true if the key message matches any of the given key names.
func matchesKey(msg tea.KeyMsg, keys []string) bool {
	return slices.Contains(keys, msg.String())
}
