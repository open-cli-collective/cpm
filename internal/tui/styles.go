package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents the color scheme to use.
type Theme int

const (
	// ThemeAuto detects the terminal background color.
	ThemeAuto Theme = iota
	// ThemeDark uses colors optimized for dark backgrounds.
	ThemeDark
	// ThemeLight uses colors optimized for light backgrounds.
	ThemeLight
)

// ColorPalette holds the colors used for styling.
type ColorPalette struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Text      lipgloss.Color
	Muted     lipgloss.Color
	Local     lipgloss.Color
	Project   lipgloss.Color
	User      lipgloss.Color
	Pending   lipgloss.Color
	Border    lipgloss.Color
}

// darkPalette is optimized for dark terminal backgrounds.
var darkPalette = ColorPalette{
	Primary:   lipgloss.Color("#7D56F4"),
	Secondary: lipgloss.Color("#5A4FCF"),
	Text:      lipgloss.Color("#FAFAFA"),
	Muted:     lipgloss.Color("#626262"),
	Local:     lipgloss.Color("#FF9F1C"),
	Project:   lipgloss.Color("#2EC4B6"),
	User:      lipgloss.Color("#E71D36"),
	Pending:   lipgloss.Color("#FFBF69"),
	Border:    lipgloss.Color("#383838"),
}

// lightPalette is optimized for light terminal backgrounds.
var lightPalette = ColorPalette{
	Primary:   lipgloss.Color("#5B3DC8"),
	Secondary: lipgloss.Color("#4A3AA8"),
	Text:      lipgloss.Color("#1A1A1A"),
	Muted:     lipgloss.Color("#6B6B6B"),
	Local:     lipgloss.Color("#D97706"),
	Project:   lipgloss.Color("#0F9488"),
	User:      lipgloss.Color("#BE123C"),
	Pending:   lipgloss.Color("#B45309"),
	Border:    lipgloss.Color("#D1D5DB"),
}

// GetPalette returns the color palette for the given theme.
// If ThemeAuto, it detects the terminal background.
func GetPalette(theme Theme) ColorPalette {
	switch theme {
	case ThemeDark:
		return darkPalette
	case ThemeLight:
		return lightPalette
	default:
		// Auto-detect based on terminal background
		if lipgloss.HasDarkBackground() {
			return darkPalette
		}
		return lightPalette
	}
}


// Styles holds all the styles used in the TUI.
type Styles struct {
	// Palette holds the color palette for modal borders and other direct color usage
	Palette ColorPalette

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

	// Component list styles
	ComponentCategory lipgloss.Style
	ComponentItem     lipgloss.Style

	// Footer/help
	Help lipgloss.Style
}

// DefaultStyles returns the default styles with auto-detected theme.
func DefaultStyles() Styles {
	return DefaultStylesWithTheme(ThemeAuto)
}

// DefaultStylesWithTheme returns styles using the specified theme.
func DefaultStylesWithTheme(theme Theme) Styles {
	p := GetPalette(theme)

	return Styles{
		Palette: p,

		LeftPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Border).
			Padding(0, 1),

		RightPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Border).
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(p.Primary).
			Padding(0, 1),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(p.Secondary),

		Normal: lipgloss.NewStyle().
			Foreground(p.Text),

		GroupHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Primary),

		Description: lipgloss.NewStyle().
			Foreground(p.Muted).
			Italic(true),

		ScopeLocal: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Local),

		ScopeProject: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Project),

		ScopeUser: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.User),

		Pending: lipgloss.NewStyle().
			Foreground(p.Pending),

		DetailTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Primary).
			MarginBottom(1),

		DetailLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Muted),

		DetailValue: lipgloss.NewStyle().
			Foreground(p.Text),

		DetailDescription: lipgloss.NewStyle().
			Foreground(p.Text).
			MarginTop(1),

		ComponentCategory: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Secondary).
			PaddingLeft(2),

		ComponentItem: lipgloss.NewStyle().
			Foreground(p.Text).
			PaddingLeft(4),

		Help: lipgloss.NewStyle().
			Foreground(p.Muted),
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
