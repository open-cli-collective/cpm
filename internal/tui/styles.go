package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors for the UI.
var (
	colorPrimary   = lipgloss.Color("#7D56F4")
	colorSecondary = lipgloss.Color("#5A4FCF")
	colorText      = lipgloss.Color("#FAFAFA")
	colorMuted     = lipgloss.Color("#626262")
	colorLocal     = lipgloss.Color("#FF9F1C")
	colorProject   = lipgloss.Color("#2EC4B6")
	colorUser      = lipgloss.Color("#E71D36")
	colorPending   = lipgloss.Color("#FFBF69")
	colorBorder    = lipgloss.Color("#383838")
)

// Styles holds all the styles used in the TUI.
type Styles struct {
	// Pane styles
	LeftPane  lipgloss.Style
	RightPane lipgloss.Style

	// List item styles
	Header      lipgloss.Style
	Selected    lipgloss.Style
	Normal      lipgloss.Style
	GroupHeader lipgloss.Style
	Description lipgloss.Style

	// Scope indicator styles
	ScopeLocal   lipgloss.Style
	ScopeProject lipgloss.Style
	ScopeUser    lipgloss.Style
	Pending      lipgloss.Style

	// Detail pane styles
	DetailTitle       lipgloss.Style
	DetailLabel       lipgloss.Style
	DetailValue       lipgloss.Style
	DetailDescription lipgloss.Style

	// Footer/help
	Help lipgloss.Style
}

// DefaultStyles returns the default styles.
func DefaultStyles() Styles {
	return Styles{
		LeftPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),

		RightPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorSecondary),

		Normal: lipgloss.NewStyle().
			Foreground(colorText),

		GroupHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),

		Description: lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true),

		ScopeLocal: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorLocal),

		ScopeProject: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorProject),

		ScopeUser: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorUser),

		Pending: lipgloss.NewStyle().
			Foreground(colorPending),

		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1),

		DetailLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted),

		DetailValue: lipgloss.NewStyle().
			Foreground(colorText),

		DetailDescription: lipgloss.NewStyle().
			Foreground(colorText).
			MarginTop(1),

		Help: lipgloss.NewStyle().
			Foreground(colorMuted),
	}
}

// WithDimensions returns a new Styles with pane dimensions set.
func (s Styles) WithDimensions(width, height int) Styles {
	// Calculate pane widths (1/3 left, 2/3 right, minus borders and padding)
	leftWidth := width/3 - 4
	rightWidth := width - leftWidth - 8

	// Calculate heights (minus borders, header, footer)
	paneHeight := height - 4

	s.LeftPane = s.LeftPane.
		Width(leftWidth).
		Height(paneHeight)

	s.RightPane = s.RightPane.
		Width(rightWidth).
		Height(paneHeight)

	return s
}
