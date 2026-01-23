package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// KeyBindings defines all keyboard shortcuts.
type KeyBindings struct {
	Up        []string
	Down      []string
	PageUp    []string
	PageDown  []string
	Home      []string
	End       []string
	Enter     []string
	Quit      []string
	Local     []string
	Project   []string
	Toggle    []string
	Uninstall []string
	Escape    []string
	Filter    []string
	Refresh   []string
	Mouse     []string
}

// DefaultKeyBindings returns the default key bindings.
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Up:        []string{"up", "k"},
		Down:      []string{"down", "j"},
		PageUp:    []string{"pgup", "ctrl+u"},
		PageDown:  []string{"pgdown", "ctrl+d"},
		Home:      []string{"home", "g"},
		End:       []string{"end", "G"},
		Enter:     []string{"enter"},
		Quit:      []string{"q", "ctrl+c"},
		Local:     []string{"l"},
		Project:   []string{"p"},
		Toggle:    []string{"tab"},
		Uninstall: []string{"u"},
		Escape:    []string{"esc"},
		Filter:    []string{"/"},
		Refresh:   []string{"r"},
		Mouse:     []string{"m"},
	}
}

// matchesKey returns true if the key message matches any of the given key names.
func matchesKey(msg tea.KeyMsg, keys []string) bool {
	keyStr := msg.String()
	for _, k := range keys {
		if keyStr == k {
			return true
		}
	}
	return false
}
