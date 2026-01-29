// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// OperationType represents the type of operation to perform.
type OperationType int

const (
	OpInstall OperationType = iota
	OpUninstall
	OpMigrate // Move plugin from one scope to another
	OpUpdate
	OpEnable
	OpDisable
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
	// ModeDoc shows a document (README or CHANGELOG).
	ModeDoc
	// ModeConfig shows plugin configuration files.
	ModeConfig
)

// DocType represents the type of document being viewed.
type DocType int

const (
	DocReadme DocType = iota
	DocChangelog
)

// SortMode represents the current sort order for the plugin list.
type SortMode int

const (
	// SortByNameAsc sorts plugins by name A-Z (default).
	SortByNameAsc SortMode = iota
	// SortByNameDesc sorts plugins by name Z-A.
	SortByNameDesc
	// SortByScope sorts plugins by scope (installed first).
	SortByScope
	// SortByMarketplace sorts plugins by marketplace name.
	SortByMarketplace
)

// String returns the display name for the sort mode.
func (s SortMode) String() string {
	switch s {
	case SortByNameAsc:
		return "Name A-Z"
	case SortByNameDesc:
		return "Name Z-A"
	case SortByScope:
		return "Scope"
	case SortByMarketplace:
		return "Marketplace"
	default:
		return "Unknown"
	}
}

// PluginState holds the display state for a plugin.
// Fields are ordered for optimal memory alignment (strings/pointers first, bools last).
type PluginState struct {
	Components       *claude.PluginComponents
	Version          string // Installed version (or available version if not installed)
	AvailableVersion string // Latest available version from marketplace
	Description      string
	AuthorName       string
	AuthorEmail      string
	Marketplace      string
	ID               string
	InstallPath      string
	ExternalURL      string
	LastUpdated      string
	InstalledScope   claude.Scope
	Name             string
	InstallCount     int
	Enabled          bool
	IsGroupHeader    bool
	IsExternal       bool
	HasUpdate        bool // True if installed version < available version
}

// PluginStateFromInstalled creates a PluginState from an installed plugin.
// It reads the plugin manifest for description and scans for components.
func PluginStateFromInstalled(p claude.InstalledPlugin) PluginState {
	// Parse name and marketplace from ID (format: name@marketplace)
	name, marketplace := parsePluginID(p.ID)

	state := PluginState{
		ID:             p.ID,
		Name:           name,
		Marketplace:    marketplace,
		Version:        p.Version,
		InstalledScope: p.Scope,
		Enabled:        p.Enabled,
		InstallPath:    p.InstallPath,
		LastUpdated:    p.LastUpdated,
	}

	// Read manifest for description and author
	if p.InstallPath != "" {
		if manifest, err := claude.ReadPluginManifest(p.InstallPath); err == nil {
			state.Description = manifest.Description
			state.AuthorName = manifest.AuthorName
			state.AuthorEmail = manifest.AuthorEmail
		}
		// Scan for components
		state.Components = claude.ScanPluginComponents(p.InstallPath)
	}

	return state
}

// PluginStateFromAvailable creates a PluginState from an available plugin.
func PluginStateFromAvailable(p claude.AvailablePlugin) PluginState {
	name := p.Name
	// Fall back to plugin name from ID if name is empty (e.g., "foo@bar" -> "foo")
	if name == "" {
		name, _ = parsePluginID(p.PluginID)
	}

	state := PluginState{
		ID:             p.PluginID,
		Name:           name,
		Description:    p.Description,
		Marketplace:    p.MarketplaceName,
		Version:        p.Version,
		InstallCount:   p.InstallCount,
		InstalledScope: claude.ScopeNone,
	}

	// Try to resolve the marketplace source path to get additional info
	sourcePath := claude.ResolveMarketplaceSourcePath(p.MarketplaceName, p.Source)
	if sourcePath != "" {
		// Read manifest for author info
		if manifest, err := claude.ReadPluginManifest(sourcePath); err == nil {
			state.AuthorName = manifest.AuthorName
			state.AuthorEmail = manifest.AuthorEmail
			// Use manifest description if available and CLI description is empty
			if state.Description == "" && manifest.Description != "" {
				state.Description = manifest.Description
			}
		}
		// Scan for components
		state.Components = claude.ScanPluginComponents(sourcePath)
	} else {
		// Check if this is an external URL-based plugin
		if sourceObj, ok := p.Source.(map[string]any); ok {
			if sourceType, ok := sourceObj["source"].(string); ok && sourceType == "url" {
				state.IsExternal = true
				if url, ok := sourceObj["url"].(string); ok {
					state.ExternalURL = url
				}
			}
		}
	}

	return state
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

// MainState holds state for the main two-pane view.
type MainState struct {
	pendingOps      map[string]Operation
	bulkSelected    map[string]bool // Tracks plugins selected for bulk operations
	sortMode        SortMode
	showConfirm     bool
	showQuitConfirm bool
	mouseEnabled    bool
}

// FilterState holds state for filter mode.
type FilterState struct {
	text   string
	active bool
}

// DocState holds state for the document viewer mode (README or CHANGELOG).
type DocState struct {
	content string  // Rendered document content
	title   string  // Document title (plugin name + doc type)
	scroll  int     // Scroll position
	docType DocType // Type of document being viewed
}

// ConfigState holds state for the config viewer mode.
type ConfigState struct {
	content string // Rendered config content
	title   string // Title for config viewer
	scroll  int    // Scroll position
}

// ProgressState holds state for operation progress.
type ProgressState struct {
	operations []Operation
	errors     []string
	currentIdx int
	loading    bool
}

// Model is the main application model.
// Fields ordered for optimal memory alignment (pointers/slices first, bools last).
type Model struct {
	err         error
	client      claude.Client
	styles      Styles
	workingDir  string
	keys        KeyBindings
	config      ConfigState
	main        MainState
	plugins     []PluginState
	filter      FilterState
	filteredIdx []int
	doc         DocState
	progress    ProgressState
	mode        Mode
	height      int
	width       int
	selectedIdx int
	listOffset  int
}

// NewModel creates a new Model with the given client and working directory.
// Uses auto-detected theme.
func NewModel(client claude.Client, workingDir string) *Model {
	return NewModelWithTheme(client, workingDir, ThemeAuto)
}

// NewModelWithTheme creates a new Model with the specified theme.
func NewModelWithTheme(client claude.Client, workingDir string, theme Theme) *Model {
	return &Model{
		client:     client,
		workingDir: workingDir,
		styles:     DefaultStylesWithTheme(theme),
		keys:       DefaultKeyBindings(),
		main: MainState{
			pendingOps:   make(map[string]Operation),
			bulkSelected: make(map[string]bool),
			mouseEnabled: true,
		},
		progress: ProgressState{
			loading: true,
		},
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
	OriginalScope claude.Scope  // For uninstalls: the original scope to uninstall from
	Type          OperationType // Operation type: install, uninstall, enable, or disable
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

// mergePlugins combines installed and available plugins, grouped by marketplace.
// Only installed plugins relevant to workingDir are included.
func mergePlugins(list *claude.PluginList, workingDir string) []PluginState {
	projectEnabled := claude.GetProjectEnabledPlugins(workingDir)
	installedByID := buildInstalledByID(list.Installed, projectEnabled)
	seenInstalled := make(map[string]bool)
	byMarketplace := make(map[string][]PluginState)

	// Process available plugins
	for _, p := range list.Available {
		state := processAvailablePlugin(p, installedByID, seenInstalled)
		byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
	}

	// Process installed plugins not in available list
	for _, p := range list.Installed {
		state, ok := processInstalledPlugin(p, projectEnabled, seenInstalled)
		if ok {
			byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
		}
	}

	return sortAndGroupByMarketplace(byMarketplace)
}

// buildInstalledByID creates a map of installed plugins by ID.
// Only includes user-scoped plugins and those in project settings.
func buildInstalledByID(installed []claude.InstalledPlugin, projectEnabled map[string]claude.Scope) map[string]claude.InstalledPlugin {
	result := make(map[string]claude.InstalledPlugin)
	for _, p := range installed {
		if p.Scope == claude.ScopeUser {
			result[p.ID] = p
		} else if scope, ok := projectEnabled[p.ID]; ok {
			p.Scope = scope
			result[p.ID] = p
		}
	}
	return result
}

// processAvailablePlugin processes an available plugin and merges with installed data.
func processAvailablePlugin(p claude.AvailablePlugin, installedByID map[string]claude.InstalledPlugin, seenInstalled map[string]bool) PluginState {
	state := PluginStateFromAvailable(p)
	state.AvailableVersion = p.Version

	if installed, ok := installedByID[p.PluginID]; ok {
		mergeInstalledInfo(&state, installed, p.Version)
		seenInstalled[p.PluginID] = true
	}

	return state
}

// mergeInstalledInfo merges installed plugin info into a PluginState.
func mergeInstalledInfo(state *PluginState, installed claude.InstalledPlugin, availableVersion string) {
	state.InstalledScope = installed.Scope
	state.Enabled = installed.Enabled
	state.Version = installed.Version
	state.InstallPath = installed.InstallPath

	// Check if update is available
	if availableVersion != "" && installed.Version != "" && availableVersion != installed.Version {
		state.HasUpdate = true
	}

	// Read manifest and scan components
	if installed.InstallPath != "" {
		if manifest, err := claude.ReadPluginManifest(installed.InstallPath); err == nil {
			state.AuthorName = manifest.AuthorName
			state.AuthorEmail = manifest.AuthorEmail
		}
		state.Components = claude.ScanPluginComponents(installed.InstallPath)
	}
}

// processInstalledPlugin processes an installed plugin not in the available list.
// Returns the state and true if it should be included, false otherwise.
func processInstalledPlugin(p claude.InstalledPlugin, projectEnabled map[string]claude.Scope, seenInstalled map[string]bool) (PluginState, bool) {
	// Check relevance
	isRelevant := p.Scope == claude.ScopeUser
	if scope, ok := projectEnabled[p.ID]; ok {
		isRelevant = true
		p.Scope = scope
	}

	if !isRelevant || seenInstalled[p.ID] {
		return PluginState{}, false
	}

	seenInstalled[p.ID] = true
	return PluginStateFromInstalled(p), true
}

// sortAndGroupByMarketplace sorts and groups plugins by marketplace with headers.
func sortAndGroupByMarketplace(byMarketplace map[string][]PluginState) []PluginState {
	// Sort marketplace names
	marketplaces := make([]string, 0, len(byMarketplace))
	for marketplace := range byMarketplace {
		marketplaces = append(marketplaces, marketplace)
	}
	sort.Strings(marketplaces)

	// Build result with headers
	var result []PluginState
	for _, marketplace := range marketplaces {
		plugins := byMarketplace[marketplace]
		sort.Slice(plugins, func(i, j int) bool {
			return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
		})
		result = append(result, PluginState{
			Name:          marketplace,
			IsGroupHeader: true,
			Marketplace:   marketplace,
		})
		result = append(result, plugins...)
	}

	return result
}

// toggleEnablement toggles the enabled/disabled state of the selected plugin.
// Only works for installed plugins. Blocked if plugin has pending install/uninstall.
func (m *Model) toggleEnablement() {
	plugin := m.getSelectedPlugin()
	if plugin == nil {
		return
	}

	// Can only enable/disable installed plugins
	if plugin.InstalledScope == claude.ScopeNone {
		return
	}

	// Block if plugin has pending install/uninstall operation
	if existingOp, ok := m.main.pendingOps[plugin.ID]; ok {
		if existingOp.Type == OpInstall || existingOp.Type == OpUninstall {
			// Don't allow enable/disable when install/uninstall is pending
			return
		}
	}

	// Determine operation type based on current enabled state
	var opType OperationType
	if plugin.Enabled {
		opType = OpDisable
	} else {
		opType = OpEnable
	}

	// If already pending the same operation, clear it (toggle off)
	if existingOp, ok := m.main.pendingOps[plugin.ID]; ok {
		if existingOp.Type == opType {
			m.clearPending(plugin.ID)
			return
		}
	}

	// Create enable/disable operation
	m.main.pendingOps[plugin.ID] = Operation{
		PluginID: plugin.ID,
		Scope:    plugin.InstalledScope, // Use current installed scope
		Type:     opType,
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case pluginsLoadedMsg:
		m.progress.loading = false
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
		m.progress.loading = false
		m.err = msg.err
		return m, nil
	}

	// Handle confirmation dialog
	if m.main.showConfirm {
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
	case ModeDoc:
		return m.updateDoc(msg)
	case ModeConfig:
		return m.updateConfig(msg)
	}

	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.progress.loading {
		return "Loading plugins..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	if m.main.showQuitConfirm {
		return m.renderQuitConfirmation(m.styles)
	}

	if m.main.showConfirm {
		return m.renderConfirmation(m.styles)
	}

	switch m.mode {
	case ModeMain:
		return m.renderMainView()
	case ModeProgress:
		return m.renderProgress(m.styles)
	case ModeSummary:
		return m.renderErrorSummary(m.styles)
	case ModeDoc:
		return m.renderDoc(m.styles)
	case ModeConfig:
		return m.renderConfig(m.styles)
	}

	return ""
}
