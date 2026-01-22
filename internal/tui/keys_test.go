package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyBindings(t *testing.T) {
	keys := DefaultKeyBindings()

	// Test that key bindings are defined
	tests := []struct {
		name string
		keys []string
	}{
		{"Up", keys.Up},
		{"Down", keys.Down},
		{"Quit", keys.Quit},
		{"Enter", keys.Enter},
	}

	for _, tt := range tests {
		if len(tt.keys) == 0 {
			t.Errorf("%s has no key bindings", tt.name)
		}
	}
}

func TestMatchesKey(t *testing.T) {
	keys := DefaultKeyBindings()

	// Create a mock key message for "j"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	if !matchesKey(msg, keys.Down) {
		t.Error("'j' should match Down binding")
	}
	if matchesKey(msg, keys.Up) {
		t.Error("'j' should not match Up binding")
	}
}
