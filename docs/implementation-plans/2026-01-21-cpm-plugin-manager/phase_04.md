# cpm - Phase 4: Two-Pane Layout

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Split-pane UI with plugin list and details

**Architecture:** Two-pane layout using Lip Gloss for styling. Left pane (1/3 width) shows scrollable plugin list with marketplace headers. Right pane (2/3 width) shows selected plugin details. Navigation with j/k, arrows, Home/End, PgUp/PgDn.

**Tech Stack:** Lip Gloss (github.com/charmbracelet/lipgloss), Bubble Tea

**Scope:** Phase 4 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 3 complete with basic model and loading

---

## Task 1: Create Styles Package

**Files:**
- Create: `internal/tui/styles.go`

**Step 1: Write the test file**

Create file `internal/tui/styles_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/tui/...`
Expected: FAIL - Styles not defined

**Step 3: Write the implementation**

Create file `internal/tui/styles.go`:

```go
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors for the UI.
var (
	colorPrimary    = lipgloss.Color("#7D56F4")
	colorSecondary  = lipgloss.Color("#5A4FCF")
	colorText       = lipgloss.Color("#FAFAFA")
	colorMuted      = lipgloss.Color("#626262")
	colorLocal      = lipgloss.Color("#FF9F1C")
	colorProject    = lipgloss.Color("#2EC4B6")
	colorUser       = lipgloss.Color("#E71D36")
	colorPending    = lipgloss.Color("#FFBF69")
	colorBorder     = lipgloss.Color("#383838")
	colorBackground = lipgloss.Color("#1A1A1A")
)

// Styles holds all the styles used in the TUI.
type Styles struct {
	// Pane styles
	LeftPane  lipgloss.Style
	RightPane lipgloss.Style

	// List item styles
	Header       lipgloss.Style
	Selected     lipgloss.Style
	Normal       lipgloss.Style
	GroupHeader  lipgloss.Style
	Description  lipgloss.Style

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
			Foreground(colorPrimary).
			MarginTop(1),

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
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/styles.go internal/tui/styles_test.go
git commit -m "feat(tui): add lip gloss styles for two-pane layout"
```

---

## Task 2: Create Key Bindings

**Files:**
- Create: `internal/tui/keys.go`

**Step 1: Write the test file**

Create file `internal/tui/keys_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/tui/...`
Expected: FAIL - KeyBindings not defined

**Step 3: Write the implementation**

Create file `internal/tui/keys.go`:

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// KeyBindings defines all keyboard shortcuts.
type KeyBindings struct {
	Up       []string
	Down     []string
	PageUp   []string
	PageDown []string
	Home     []string
	End      []string
	Enter    []string
	Quit     []string
	Local    []string
	Project  []string
	Toggle   []string
	Uninstall []string
	Escape   []string
	Filter   []string
	Refresh  []string
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
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/keys.go internal/tui/keys_test.go
git commit -m "feat(tui): add key binding definitions"
```

---

## Task 3: Create View Rendering

**Files:**
- Create: `internal/tui/view.go`

**Step 1: Write the implementation**

Create file `internal/tui/view.go`:

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// renderMain renders the main two-pane view.
func (m *Model) renderMainView() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	styles := m.styles.WithDimensions(m.width, m.height)

	leftContent := m.renderList(styles)
	rightContent := m.renderDetails(styles)

	leftPane := styles.LeftPane.Render(leftContent)
	rightPane := styles.RightPane.Render(rightContent)

	main := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	help := m.renderHelp(styles)

	return lipgloss.JoinVertical(lipgloss.Left, main, help)
}

// renderList renders the left pane plugin list.
func (m *Model) renderList(styles Styles) string {
	if len(m.plugins) == 0 {
		return "No plugins found."
	}

	var lines []string
	visibleHeight := styles.LeftPane.GetHeight() - 2 // Account for padding

	// Calculate visible range
	start := m.listOffset
	end := start + visibleHeight
	if end > len(m.plugins) {
		end = len(m.plugins)
	}

	for i := start; i < end; i++ {
		plugin := m.plugins[i]
		line := m.renderListItem(plugin, i == m.selectedIdx, styles)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderListItem renders a single list item.
func (m *Model) renderListItem(plugin PluginState, selected bool, styles Styles) string {
	if plugin.IsGroupHeader {
		return styles.GroupHeader.Render("── " + plugin.Name + " ──")
	}

	// Build the line
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, ">")
	} else {
		parts = append(parts, " ")
	}

	// Plugin name
	name := plugin.Name
	if len(name) > 20 {
		name = name[:17] + "..."
	}
	parts = append(parts, name)

	// Scope indicator
	scope := m.getScopeIndicator(plugin, styles)
	if scope != "" {
		parts = append(parts, scope)
	}

	line := strings.Join(parts, " ")

	if selected {
		return styles.Selected.Render(line)
	}
	return styles.Normal.Render(line)
}

// getScopeIndicator returns the scope indicator for a plugin.
func (m *Model) getScopeIndicator(plugin PluginState, styles Styles) string {
	// Check for pending changes first
	if pending, ok := m.pending[plugin.ID]; ok {
		if pending == claude.ScopeNone {
			return styles.Pending.Render("[→ UNINSTALL]")
		}
		return styles.Pending.Render("[→ " + strings.ToUpper(string(pending)) + "]")
	}

	// Show current scope
	switch plugin.InstalledScope {
	case claude.ScopeLocal:
		return styles.ScopeLocal.Render("[LOCAL]")
	case claude.ScopeProject:
		return styles.ScopeProject.Render("[PROJECT]")
	case claude.ScopeUser:
		return styles.ScopeUser.Render("[USER]")
	default:
		return ""
	}
}

// renderDetails renders the right pane with plugin details.
func (m *Model) renderDetails(styles Styles) string {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.plugins) {
		return "No plugin selected."
	}

	plugin := m.plugins[m.selectedIdx]
	if plugin.IsGroupHeader {
		return styles.DetailTitle.Render("Marketplace: " + plugin.Name)
	}

	var lines []string

	// Title
	lines = append(lines, styles.DetailTitle.Render(plugin.Name))
	lines = append(lines, "")

	// Plugin ID
	lines = append(lines, styles.DetailLabel.Render("ID: ")+
		styles.DetailValue.Render(plugin.ID))

	// Marketplace
	lines = append(lines, styles.DetailLabel.Render("Marketplace: ")+
		styles.DetailValue.Render(plugin.Marketplace))

	// Version
	if plugin.Version != "" {
		lines = append(lines, styles.DetailLabel.Render("Version: ")+
			styles.DetailValue.Render(plugin.Version))
	}

	// Status
	status := "Not installed"
	if plugin.InstalledScope != claude.ScopeNone {
		status = "Installed (" + string(plugin.InstalledScope) + ")"
		if plugin.Enabled {
			status += " - Enabled"
		} else {
			status += " - Disabled"
		}
	}
	lines = append(lines, styles.DetailLabel.Render("Status: ")+
		styles.DetailValue.Render(status))

	// Pending change
	if pending, ok := m.pending[plugin.ID]; ok {
		var pendingStr string
		if pending == claude.ScopeNone {
			pendingStr = "Will be uninstalled"
		} else {
			pendingStr = "Will be installed to " + string(pending)
		}
		lines = append(lines, styles.Pending.Render("Pending: "+pendingStr))
	}

	// Description
	if plugin.Description != "" {
		lines = append(lines, "")
		lines = append(lines, styles.DetailLabel.Render("Description:"))
		lines = append(lines, styles.DetailDescription.Render(plugin.Description))
	}

	return strings.Join(lines, "\n")
}

// renderHelp renders the help bar at the bottom.
func (m *Model) renderHelp(styles Styles) string {
	if len(m.pending) > 0 {
		return styles.Help.Render("↑↓/jk: navigate • l/p: install local/project • u: uninstall • Tab: toggle • Enter: apply • Esc: clear • q: quit")
	}
	return styles.Help.Render("↑↓/jk: navigate • l/p: install local/project • u: uninstall • Tab: toggle • q: quit")
}
```

**Step 2: Run test**

Run: `go test -v ./internal/tui/...`
Expected: PASS (existing tests still pass)

**Step 3: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add two-pane view rendering"
```

---

## Task 4: Create Update Logic

**Files:**
- Create: `internal/tui/update.go`

**Step 1: Write the implementation**

Create file `internal/tui/update.go`:

```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateMain handles messages in main mode.
func (m *Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}
	return m, nil
}

// handleKeyPress processes keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys

	switch {
	case matchesKey(msg, keys.Quit):
		return m, tea.Quit

	case matchesKey(msg, keys.Up):
		m.moveUp()

	case matchesKey(msg, keys.Down):
		m.moveDown()

	case matchesKey(msg, keys.PageUp):
		m.pageUp()

	case matchesKey(msg, keys.PageDown):
		m.pageDown()

	case matchesKey(msg, keys.Home):
		m.moveToStart()

	case matchesKey(msg, keys.End):
		m.moveToEnd()
	}

	return m, nil
}

// moveUp moves selection up, skipping group headers.
func (m *Model) moveUp() {
	for i := m.selectedIdx - 1; i >= 0; i-- {
		if !m.plugins[i].IsGroupHeader {
			m.selectedIdx = i
			m.ensureVisible()
			return
		}
	}
}

// moveDown moves selection down, skipping group headers.
func (m *Model) moveDown() {
	for i := m.selectedIdx + 1; i < len(m.plugins); i++ {
		if !m.plugins[i].IsGroupHeader {
			m.selectedIdx = i
			m.ensureVisible()
			return
		}
	}
}

// pageUp moves up by a page.
func (m *Model) pageUp() {
	pageSize := m.getPageSize()
	for i := 0; i < pageSize; i++ {
		m.moveUp()
	}
}

// pageDown moves down by a page.
func (m *Model) pageDown() {
	pageSize := m.getPageSize()
	for i := 0; i < pageSize; i++ {
		m.moveDown()
	}
}

// moveToStart moves to the first selectable item.
func (m *Model) moveToStart() {
	for i := 0; i < len(m.plugins); i++ {
		if !m.plugins[i].IsGroupHeader {
			m.selectedIdx = i
			m.listOffset = 0
			return
		}
	}
}

// moveToEnd moves to the last selectable item.
func (m *Model) moveToEnd() {
	for i := len(m.plugins) - 1; i >= 0; i-- {
		if !m.plugins[i].IsGroupHeader {
			m.selectedIdx = i
			m.ensureVisible()
			return
		}
	}
}

// getPageSize returns the visible page size.
func (m *Model) getPageSize() int {
	if m.height == 0 {
		return 10
	}
	return m.height - 6 // Account for borders and help
}

// ensureVisible adjusts listOffset to keep selectedIdx visible.
func (m *Model) ensureVisible() {
	pageSize := m.getPageSize()

	// If selection is above visible area, scroll up
	if m.selectedIdx < m.listOffset {
		m.listOffset = m.selectedIdx
	}

	// If selection is below visible area, scroll down
	if m.selectedIdx >= m.listOffset+pageSize {
		m.listOffset = m.selectedIdx - pageSize + 1
	}

	// Clamp offset
	if m.listOffset < 0 {
		m.listOffset = 0
	}
	maxOffset := len(m.plugins) - pageSize
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.listOffset > maxOffset {
		m.listOffset = maxOffset
	}
}
```

**Step 2: Run test**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/update.go
git commit -m "feat(tui): add navigation update logic"
```

---

## Task 5: Integrate View and Update into Model

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Update model.go to use new components**

Update file `internal/tui/model.go` to integrate styles, keys, and use the new rendering:

```go
// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// Mode represents the current UI mode.
type Mode int

const (
	// ModeMain is the default two-pane view.
	ModeMain Mode = iota
	// ModeProgress shows operation progress.
	ModeProgress
	// ModeError shows error summary.
	ModeError
)

// PluginState holds the display state for a plugin.
type PluginState struct {
	ID             string
	Name           string
	Description    string
	Marketplace    string
	Version        string
	InstalledScope claude.Scope
	Enabled        bool
	IsGroupHeader  bool // True for marketplace group headers (non-selectable)
}

// PluginStateFromInstalled creates a PluginState from an installed plugin.
func PluginStateFromInstalled(p claude.InstalledPlugin) PluginState {
	// Parse name and marketplace from ID (format: name@marketplace)
	name, marketplace := parsePluginID(p.ID)
	return PluginState{
		ID:             p.ID,
		Name:           name,
		Marketplace:    marketplace,
		Version:        p.Version,
		InstalledScope: p.Scope,
		Enabled:        p.Enabled,
	}
}

// PluginStateFromAvailable creates a PluginState from an available plugin.
func PluginStateFromAvailable(p claude.AvailablePlugin) PluginState {
	return PluginState{
		ID:             p.PluginID,
		Name:           p.Name,
		Description:    p.Description,
		Marketplace:    p.MarketplaceName,
		Version:        p.Version,
		InstalledScope: claude.ScopeNone,
	}
}

// parsePluginID splits "name@marketplace" into (name, marketplace).
func parsePluginID(id string) (name, marketplace string) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '@' {
			return id[:i], id[i+1:]
		}
	}
	return id, ""
}

// Model is the main application model.
type Model struct {
	// Client for Claude CLI operations
	client claude.Client

	// Styles and keys
	styles Styles
	keys   KeyBindings

	// Plugin data
	plugins []PluginState

	// UI state
	selectedIdx int
	listOffset  int
	width       int
	height      int

	// Pending changes (plugin ID -> desired scope)
	pending map[string]claude.Scope

	// View mode
	mode         Mode
	progressMsgs []string
	errorMsgs    []string

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new Model with the given client.
func NewModel(client claude.Client) *Model {
	return &Model{
		client:  client,
		styles:  DefaultStyles(),
		keys:    DefaultKeyBindings(),
		pending: make(map[string]claude.Scope),
		loading: true,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return m.loadPlugins
}

// pluginsLoadedMsg is sent when plugins are loaded.
type pluginsLoadedMsg struct {
	plugins []PluginState
}

// pluginsErrorMsg is sent when loading fails.
type pluginsErrorMsg struct {
	err error
}

// loadPlugins fetches plugin data from the Claude CLI.
func (m *Model) loadPlugins() tea.Msg {
	list, err := m.client.ListPlugins(true)
	if err != nil {
		return pluginsErrorMsg{err: err}
	}

	plugins := mergePlugins(list)
	return pluginsLoadedMsg{plugins: plugins}
}

// mergePlugins combines installed and available plugins, grouped by marketplace.
func mergePlugins(list *claude.PluginList) []PluginState {
	// Build map of installed plugins by ID
	installedByID := make(map[string]claude.InstalledPlugin)
	for _, p := range list.Installed {
		installedByID[p.ID] = p
	}

	// Group by marketplace
	byMarketplace := make(map[string][]PluginState)

	// Add available plugins (which includes installed ones)
	for _, p := range list.Available {
		state := PluginStateFromAvailable(p)

		// Check if installed
		if installed, ok := installedByID[p.PluginID]; ok {
			state.InstalledScope = installed.Scope
			state.Enabled = installed.Enabled
			state.Version = installed.Version
		}

		byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
	}

	// Flatten with group headers
	var result []PluginState
	for marketplace, plugins := range byMarketplace {
		// Add group header
		result = append(result, PluginState{
			Name:          marketplace,
			IsGroupHeader: true,
			Marketplace:   marketplace,
		})
		result = append(result, plugins...)
	}

	return result
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case pluginsLoadedMsg:
		m.loading = false
		m.plugins = msg.plugins
		// Skip to first non-header item
		for i, p := range m.plugins {
			if !p.IsGroupHeader {
				m.selectedIdx = i
				break
			}
		}
		return m, nil

	case pluginsErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	// Handle mode-specific updates
	switch m.mode {
	case ModeMain:
		return m.updateMain(msg)
	}

	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.loading {
		return "Loading plugins..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	switch m.mode {
	case ModeMain:
		return m.renderMainView()
	}

	return ""
}
```

**Step 2: Run tests**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 3: Build and test visually**

```bash
mise run build
./cpm
```

Expected:
- Two-pane layout renders correctly
- Left pane shows plugin list with marketplace headers
- Right pane shows selected plugin details
- j/k and arrow keys navigate
- Group headers are skipped during navigation

**Step 4: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat(tui): integrate styles, keys, view into model"
```

---

## Phase 4 Complete

**Verification:**
- Two-pane layout renders correctly at various terminal sizes
- Left pane (1/3) shows plugin list with marketplace group headers
- Right pane (2/3) shows selected plugin details
- Navigation with j/k, arrows works
- Home/End and PgUp/PgDn work
- Group headers are non-selectable
- Detail pane updates with selection

**Files created/modified:**
- `internal/tui/styles.go`
- `internal/tui/styles_test.go`
- `internal/tui/keys.go`
- `internal/tui/keys_test.go`
- `internal/tui/view.go`
- `internal/tui/update.go`
- `internal/tui/model.go` (updated)
