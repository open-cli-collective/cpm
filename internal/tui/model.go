// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	"sort"
	"strconv"

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

	// Plugin data
	plugins []PluginState

	// UI state
	selectedIdx int
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
		// Sort plugins within marketplace by name for deterministic ordering
		sort.Slice(plugins, func(i, j int) bool {
			return plugins[i].Name < plugins[j].Name
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
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

	case pluginsErrorMsg:
		m.loading = false
		m.err = msg.err
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

	return m.renderMain()
}

// renderMain renders the main view (placeholder for now).
func (m *Model) renderMain() string {
	if len(m.plugins) == 0 {
		return "No plugins found.\n\nPress q to quit."
	}

	// Count non-header plugins
	count := 0
	for _, p := range m.plugins {
		if !p.IsGroupHeader {
			count++
		}
	}

	return "cpm - Claude Plugin Manager\n\n" +
		"Found " + strconv.Itoa(count) + " plugins.\n\n" +
		"Press q to quit."
}
