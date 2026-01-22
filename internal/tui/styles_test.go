package tui

import (
	"testing"
)

func TestStylesInitialized(t *testing.T) {
	// Verify styles are defined and usable
	s := DefaultStyles()

	// Test that styles can render without panic
	_ = s.LeftPane.Render("test")
	_ = s.RightPane.Render("test")
	_ = s.Header.Render("test")
	_ = s.Selected.Render("test")
	_ = s.GroupHeader.Render("test")
	_ = s.ScopeLocal.Render("LOCAL")
	_ = s.ScopeProject.Render("PROJECT")
}

func TestStylesDimensions(t *testing.T) {
	s := DefaultStyles()

	// Apply dimensions
	s = s.WithDimensions(120, 40)

	// Left pane should be roughly 1/3 width
	leftWidth := s.LeftPane.GetWidth()
	if leftWidth < 30 || leftWidth > 50 {
		t.Errorf("LeftPane width = %d, expected between 30-50 for 120 width terminal", leftWidth)
	}
}
