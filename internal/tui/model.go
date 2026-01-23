// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	"sort"
	"strings"

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
	// ModeSummary shows completion summary (both successes and errors).
	ModeSummary
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
	name := p.Name
	// Fall back to plugin name from ID if name is empty (e.g., "foo@bar" -> "foo")
	if name == "" {
		name, _ = parsePluginID(p.PluginID)
	}
	return PluginState{
		ID:             p.PluginID,
		Name:           name,
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
type Model struct { //nolint:govet
	// Client for Claude CLI operations
	client claude.Client

	// Working directory for filtering project-scoped plugins
	workingDir string

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
	mode Mode

	// Loading state
	loading bool
	err     error

	// Execution state
	operations      []Operation
	currentOpIdx    int
	operationErrors []string
	showConfirm     bool

	// Filter state
	filterText   string
	filterActive bool
	filteredIdx  []int // indices into plugins that match filter

	// Quit confirmation
	showQuitConfirm bool
}

// NewModel creates a new Model with the given client and working directory.
func NewModel(client claude.Client, workingDir string) *Model {
	return &Model{
		client:     client,
		workingDir: workingDir,
		styles:     DefaultStyles(),
		keys:       DefaultKeyBindings(),
		pending:    make(map[string]claude.Scope),
		loading:    true,
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

// Operation represents a pending change to execute.
type Operation struct {
	PluginID      string
	Scope         claude.Scope
	OriginalScope claude.Scope // For uninstalls: the original scope to uninstall from
	IsInstall     bool         // true for install, false for uninstall
}

// operationDoneMsg is sent when an operation completes.
type operationDoneMsg struct {
	err error
	op  Operation
}

// loadPlugins fetches plugin data from the Claude CLI.
func (m *Model) loadPlugins() tea.Msg {
	list, err := m.client.ListPlugins(true)
	if err != nil {
		return pluginsErrorMsg{err: err}
	}

	plugins := mergePlugins(list, m.workingDir)
	return pluginsLoadedMsg{plugins: plugins}
}

// isRelevantInstall checks if an installed plugin is relevant to the current working directory.
// User-scoped plugins are always relevant; project/local-scoped plugins must match the working directory.
func isRelevantInstall(p claude.InstalledPlugin, workingDir string) bool {
	if p.Scope == claude.ScopeUser {
		return true // User-scoped plugins apply everywhere
	}
	// Project and local scoped plugins must match the working directory.
	// Use prefix matching to handle git worktrees (workingDir may be inside projectPath).
	return strings.HasPrefix(workingDir, p.ProjectPath)
}

// mergePlugins combines installed and available plugins, grouped by marketplace.
// Only installed plugins relevant to workingDir are included.
func mergePlugins(list *claude.PluginList, workingDir string) []PluginState {
	// Build map of installed plugins by ID, filtered to relevant installs
	installedByID := make(map[string]claude.InstalledPlugin)
	for _, p := range list.Installed {
		if !isRelevantInstall(p, workingDir) {
			continue
		}
		installedByID[p.ID] = p
	}

	// Track which installed plugins we've seen via available list
	seenInstalled := make(map[string]bool)

	// Group by marketplace
	byMarketplace := make(map[string][]PluginState)

	// Add available plugins (which includes installed ones)
	for _, p := range list.Available {
		state := PluginStateFromAvailable(p)

		// Check if installed (in the filtered set)
		if installed, ok := installedByID[p.PluginID]; ok {
			state.InstalledScope = installed.Scope
			state.Enabled = installed.Enabled
			state.Version = installed.Version
			seenInstalled[p.PluginID] = true
		}

		byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
	}

	// Add installed plugins that weren't in the available list (already filtered)
	for _, p := range list.Installed {
		if !isRelevantInstall(p, workingDir) {
			continue
		}
		if seenInstalled[p.ID] {
			continue // Already added (via available list or earlier in installed list)
		}
		seenInstalled[p.ID] = true // Mark as seen to prevent duplicates
		state := PluginStateFromInstalled(p)
		byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
	}

	// Sort marketplace names for deterministic ordering
	marketplaces := make([]string, 0, len(byMarketplace))
	for marketplace := range byMarketplace {
		marketplaces = append(marketplaces, marketplace)
	}
	sort.Strings(marketplaces)

	// Flatten with group headers in sorted order
	var result []PluginState
	for _, marketplace := range marketplaces {
		plugins := byMarketplace[marketplace]
		// Sort plugins within marketplace by name (case-insensitive) for deterministic ordering
		sort.Slice(plugins, func(i, j int) bool {
			return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
		})
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

	// Handle confirmation dialog
	if m.showConfirm {
		return m.updateConfirmation(msg)
	}

	// Handle mode-specific updates
	switch m.mode {
	case ModeMain:
		return m.updateMain(msg)
	case ModeProgress:
		return m.updateProgress(msg)
	case ModeSummary:
		return m.updateError(msg)
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

	if m.showQuitConfirm {
		return m.renderQuitConfirmation(m.styles)
	}

	if m.showConfirm {
		return m.renderConfirmation(m.styles)
	}

	switch m.mode {
	case ModeMain:
		return m.renderMainView()
	case ModeProgress:
		return m.renderProgress(m.styles)
	case ModeSummary:
		return m.renderErrorSummary(m.styles)
	}

	return ""
}
