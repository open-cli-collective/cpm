package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/open-cli-collective/cpm/internal/claude"
	"github.com/sahilm/fuzzy"
)

// wheelScrollSpeed defines how many items to scroll per wheel event
const wheelScrollSpeed = 3

// updateMain handles messages in main mode.
func (m *Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle quit confirmation
		if m.main.showQuitConfirm {
			switch {
			case matchesKey(msg, m.keys.Quit):
				return m, tea.Quit
			case matchesKey(msg, m.keys.Escape):
				m.main.showQuitConfirm = false
				return m, nil
			}
		}

		// Handle filter mode
		if m.filter.active {
			return m.updateFilter(msg)
		}
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

// handleKeyPress processes keyboard input.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keys := m.keys

	// Handle keys that return commands first
	if matchesKey(msg, keys.Quit) {
		return m.handleQuitKey()
	}
	if matchesKey(msg, keys.Refresh) {
		return m.handleRefreshKey()
	}
	if matchesKey(msg, keys.Mouse) {
		return m.handleMouseToggle()
	}

	// Handle keys that modify state
	m.handleRegularKeyPress(msg, keys)
	return m, nil
}

// handleMouseToggle toggles mouse capture on/off.
func (m *Model) handleMouseToggle() (tea.Model, tea.Cmd) {
	m.main.mouseEnabled = !m.main.mouseEnabled
	if m.main.mouseEnabled {
		return m, tea.EnableMouseCellMotion
	}
	return m, tea.DisableMouse
}

// handleRegularKeyPress handles all non-command keys that modify state.
func (m *Model) handleRegularKeyPress(msg tea.KeyMsg, keys KeyBindings) {
	switch {
	case matchesKey(msg, keys.Filter):
		m.handleFilterKey()
	case matchesKey(msg, keys.Readme):
		m.openReadme()
	case matchesKey(msg, keys.Up), matchesKey(msg, keys.Down),
		matchesKey(msg, keys.PageUp), matchesKey(msg, keys.PageDown),
		matchesKey(msg, keys.Home), matchesKey(msg, keys.End):
		m.handleNavigationKeys(msg, keys)
	case matchesKey(msg, keys.Local), matchesKey(msg, keys.Project),
		matchesKey(msg, keys.Toggle), matchesKey(msg, keys.Uninstall),
		matchesKey(msg, keys.Enable):
		m.handleOperationKeys(msg, keys)
	case matchesKey(msg, keys.Sort):
		m.cycleSortMode()
	case matchesKey(msg, keys.Enter):
		if len(m.main.pendingOps) > 0 {
			m.main.showConfirm = true
		}
	case matchesKey(msg, keys.Escape):
		plugin := m.getSelectedPlugin()
		if plugin != nil {
			m.clearPending(plugin.ID)
		}
	}
}

// handleNavigationKeys handles all navigation-related key presses.
func (m *Model) handleNavigationKeys(msg tea.KeyMsg, keys KeyBindings) {
	switch {
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
}

// handleOperationKeys handles all operation-related key presses (install, uninstall, toggle).
func (m *Model) handleOperationKeys(msg tea.KeyMsg, keys KeyBindings) {
	switch {
	case matchesKey(msg, keys.Local):
		m.selectForInstall(claude.ScopeLocal)
	case matchesKey(msg, keys.Project):
		m.selectForInstall(claude.ScopeProject)
	case matchesKey(msg, keys.Toggle):
		m.toggleScope()
	case matchesKey(msg, keys.Uninstall):
		m.selectForUninstall()
	case matchesKey(msg, keys.Enable):
		m.toggleEnablement()
	}
}

// handleQuitKey handles the quit key, showing confirmation if there are pending changes.
func (m *Model) handleQuitKey() (tea.Model, tea.Cmd) {
	if len(m.main.pendingOps) > 0 && !m.main.showQuitConfirm {
		m.main.showQuitConfirm = true
		return m, nil
	}
	return m, tea.Quit
}

// handleRefreshKey handles the refresh key.
func (m *Model) handleRefreshKey() (tea.Model, tea.Cmd) {
	m.progress.loading = true
	return m, m.loadPlugins
}

// handleFilterKey activates filter mode.
func (m *Model) handleFilterKey() {
	m.filter.active = true
	m.filter.text = ""
	m.filteredIdx = nil
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

// selectForInstall marks the selected plugin for installation at the given scope.
// If the plugin is already installed at a different scope, creates a migration operation.
func (m *Model) selectForInstall(scope claude.Scope) {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader {
		return
	}

	// If already pending for the same scope/operation, clear it (toggle off)
	if existingOp, ok := m.main.pendingOps[plugin.ID]; ok {
		if (existingOp.Type == OpInstall || existingOp.Type == OpMigrate) && existingOp.Scope == scope {
			m.clearPending(plugin.ID)
			return
		}
	}

	// Check if plugin is already installed at a different scope
	if plugin.InstalledScope != claude.ScopeNone && plugin.InstalledScope != scope {
		// Create migrate operation (move from current scope to new scope)
		m.main.pendingOps[plugin.ID] = Operation{
			PluginID:      plugin.ID,
			Scope:         scope,
			OriginalScope: plugin.InstalledScope,
			Type:          OpMigrate,
		}
		return
	}

	// Create install operation
	m.main.pendingOps[plugin.ID] = Operation{
		PluginID: plugin.ID,
		Scope:    scope,
		Type:     OpInstall,
	}
}

// toggleScope cycles through: none -> local -> project -> uninstall -> none
// For installed plugins, this becomes: migrate to local -> migrate to project -> uninstall -> none
func (m *Model) toggleScope() {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader {
		return
	}

	nextOp := m.computeNextToggleOp(plugin)
	if nextOp == nil {
		m.clearPending(plugin.ID)
		return
	}

	m.main.pendingOps[plugin.ID] = *nextOp
}

// computeNextToggleOp determines the next operation in the toggle cycle.
// Returns nil if the operation should be cleared.
func (m *Model) computeNextToggleOp(plugin *PluginState) *Operation {
	existingOp, hasPending := m.main.pendingOps[plugin.ID]

	if !hasPending {
		return m.firstToggleOp(plugin, claude.ScopeLocal)
	}

	// Cycle based on current pending operation
	switch {
	case (existingOp.Type == OpInstall || existingOp.Type == OpMigrate) && existingOp.Scope == claude.ScopeLocal:
		return m.firstToggleOp(plugin, claude.ScopeProject)
	case (existingOp.Type == OpInstall || existingOp.Type == OpMigrate) && existingOp.Scope == claude.ScopeProject:
		if plugin.InstalledScope != claude.ScopeNone {
			return &Operation{
				PluginID:      plugin.ID,
				Scope:         claude.ScopeNone,
				OriginalScope: plugin.InstalledScope,
				Type:          OpUninstall,
			}
		}
		return nil // Not installed, clear
	case existingOp.Type == OpUninstall:
		return nil // Clear
	default:
		return m.firstToggleOp(plugin, claude.ScopeLocal)
	}
}

// firstToggleOp returns the appropriate operation for installing/migrating to a scope.
func (m *Model) firstToggleOp(plugin *PluginState, scope claude.Scope) *Operation {
	if plugin.InstalledScope != claude.ScopeNone && plugin.InstalledScope != scope {
		return &Operation{
			PluginID:      plugin.ID,
			Scope:         scope,
			OriginalScope: plugin.InstalledScope,
			Type:          OpMigrate,
		}
	}
	return &Operation{
		PluginID: plugin.ID,
		Scope:    scope,
		Type:     OpInstall,
	}
}

// selectForUninstall marks the selected plugin for uninstallation.
func (m *Model) selectForUninstall() {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader || plugin.InstalledScope == claude.ScopeNone {
		return // Can't uninstall if not installed or if group header
	}

	// If already pending uninstall, clear it (toggle off)
	if existingOp, ok := m.main.pendingOps[plugin.ID]; ok {
		if existingOp.Type == OpUninstall {
			m.clearPending(plugin.ID)
			return
		}
	}

	// Create uninstall operation
	m.main.pendingOps[plugin.ID] = Operation{
		PluginID:      plugin.ID,
		Scope:         claude.ScopeNone,
		OriginalScope: plugin.InstalledScope,
		Type:          OpUninstall,
	}
}

// clearPending clears the pending change for the selected plugin.
func (m *Model) clearPending(pluginID string) {
	delete(m.main.pendingOps, pluginID)
}

// getSelectedPlugin returns the currently selected plugin, or nil if none.
func (m *Model) getSelectedPlugin() *PluginState {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.plugins) {
		return nil
	}
	return &m.plugins[m.selectedIdx]
}

// updateConfirmation handles messages in confirmation mode.
func (m *Model) updateConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case matchesKey(keyMsg, m.keys.Enter):
			// Start execution
			m.main.showConfirm = false
			return m.startExecution()
		case matchesKey(keyMsg, m.keys.Escape), matchesKey(keyMsg, m.keys.Quit):
			// Cancel
			m.main.showConfirm = false
		}
	}
	return m, nil
}

// startExecution begins executing pending operations.
func (m *Model) startExecution() (tea.Model, tea.Cmd) {
	// Build operation list from pendingOps
	m.progress.operations = nil
	for _, op := range m.main.pendingOps {
		m.progress.operations = append(m.progress.operations, op)
	}

	// Sort operations: uninstalls first, then migrations, then installs, then enable/disable
	sort.Slice(m.progress.operations, func(i, j int) bool {
		typeOrder := map[OperationType]int{
			OpUninstall: 0,
			OpMigrate:   1,
			OpInstall:   2,
			OpEnable:    3,
			OpDisable:   4,
		}
		orderI := typeOrder[m.progress.operations[i].Type]
		orderJ := typeOrder[m.progress.operations[j].Type]

		// If same order, maintain stable sort (don't swap)
		if orderI == orderJ {
			return false
		}
		return orderI < orderJ
	})

	m.progress.currentIdx = 0
	m.mode = ModeProgress
	m.progress.errors = make([]string, len(m.progress.operations))

	if len(m.progress.operations) == 0 {
		return m, nil
	}

	return m, m.executeOperation(m.progress.operations[0])
}

// executeOperation returns a command that executes a single operation.
func (m *Model) executeOperation(op Operation) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch op.Type {
		case OpInstall:
			err = m.client.InstallPlugin(op.PluginID, op.Scope)
		case OpUninstall:
			// For uninstalls, use the original scope to uninstall from the specific scope
			err = m.client.UninstallPlugin(op.PluginID, op.OriginalScope)
		case OpMigrate:
			// Migration = uninstall from old scope + install to new scope
			err = m.client.UninstallPlugin(op.PluginID, op.OriginalScope)
			if err == nil {
				err = m.client.InstallPlugin(op.PluginID, op.Scope)
			}
		case OpEnable:
			err = m.client.EnablePlugin(op.PluginID, op.Scope)
		case OpDisable:
			err = m.client.DisablePlugin(op.PluginID, op.Scope)
		default:
			err = fmt.Errorf("unknown operation type: %d", op.Type)
		}

		return operationDoneMsg{op: op, err: err}
	}
}

// updateProgress handles messages in progress mode.
func (m *Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if opMsg, ok := msg.(operationDoneMsg); ok {
		// Record result
		if opMsg.err != nil {
			m.progress.errors[m.progress.currentIdx] = opMsg.err.Error()
		}

		m.progress.currentIdx++

		// Execute next operation or finish
		if m.progress.currentIdx < len(m.progress.operations) {
			return m, m.executeOperation(m.progress.operations[m.progress.currentIdx])
		}

		// All done - refresh and show summary
		m.mode = ModeSummary
		m.main.pendingOps = make(map[string]Operation)
		return m, m.loadPlugins
	}
	return m, nil
}

// updateError handles messages in error mode.
func (m *Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case matchesKey(msg, m.keys.Enter), matchesKey(msg, m.keys.Escape):
			m.mode = ModeMain
			m.progress.operations = nil
			m.progress.errors = nil
		case matchesKey(msg, m.keys.Quit):
			return m, tea.Quit
		}

	case pluginsLoadedMsg:
		m.plugins = msg.plugins
		// Re-select first non-header
		for i, p := range m.plugins {
			if !p.IsGroupHeader {
				m.selectedIdx = i
				break
			}
		}
	}
	return m, nil
}

// updateFilter handles filter mode input.
func (m *Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.exitFilter()
	case tea.KeyEnter:
		m.selectFilterMatch()
	case tea.KeyBackspace:
		m.backspaceFilter()
	case tea.KeyRunes:
		m.filter.text += string(msg.Runes)
		m.applyFilter()
	case tea.KeyUp:
		m.navigateFilterUp()
	case tea.KeyDown:
		m.navigateFilterDown()
	}

	return m, nil
}

// exitFilter exits filter mode and clears filter state.
func (m *Model) exitFilter() {
	m.filter.active = false
	m.filter.text = ""
	m.filteredIdx = nil
	m.listOffset = 0
}

// selectFilterMatch selects the first match and exits filter mode.
func (m *Model) selectFilterMatch() {
	m.filter.active = false
	// Keep filtered results, select first match if any
	if len(m.filteredIdx) > 0 {
		m.selectedIdx = m.filteredIdx[0]
	}
	m.filter.text = ""
	m.filteredIdx = nil
}

// backspaceFilter removes the last character from filter text.
func (m *Model) backspaceFilter() {
	if len(m.filter.text) > 0 {
		m.filter.text = m.filter.text[:len(m.filter.text)-1]
		m.applyFilter()
	}
}

// navigateFilterUp navigates up within filtered results.
func (m *Model) navigateFilterUp() {
	if len(m.filteredIdx) == 0 {
		return
	}

	// Find current selection in filtered list
	currentPos := -1
	for i, idx := range m.filteredIdx {
		if idx == m.selectedIdx {
			currentPos = i
			break
		}
	}

	if currentPos > 0 {
		m.selectedIdx = m.filteredIdx[currentPos-1]
		m.ensureVisible()
	}
}

// navigateFilterDown navigates down within filtered results.
func (m *Model) navigateFilterDown() {
	if len(m.filteredIdx) == 0 {
		return
	}

	// Find current selection in filtered list
	currentPos := -1
	for i, idx := range m.filteredIdx {
		if idx == m.selectedIdx {
			currentPos = i
			break
		}
	}

	if currentPos >= 0 && currentPos < len(m.filteredIdx)-1 {
		m.selectedIdx = m.filteredIdx[currentPos+1]
		m.ensureVisible()
	}
}

// pluginSearchData implements fuzzy.Source for plugin searching.
type pluginSearchData struct {
	plugins []PluginState
	indices []int // indices into original plugins slice (excluding group headers)
}

func (d pluginSearchData) String(i int) string {
	p := d.plugins[d.indices[i]]
	// Combine name, description, and ID for matching
	return strings.ToLower(p.Name + " " + p.Description + " " + p.ID)
}

func (d pluginSearchData) Len() int {
	return len(d.indices)
}

// applyFilter updates filteredIdx based on filterText using fuzzy matching.
func (m *Model) applyFilter() {
	if m.filter.text == "" {
		m.filteredIdx = nil
		return
	}

	// Build search data (non-header plugins only)
	data := pluginSearchData{plugins: m.plugins}
	for i, p := range m.plugins {
		if !p.IsGroupHeader {
			data.indices = append(data.indices, i)
		}
	}

	if len(data.indices) == 0 {
		m.filteredIdx = nil
		return
	}

	// Perform fuzzy search
	matches := fuzzy.FindFrom(strings.ToLower(m.filter.text), data)

	// Convert matches to original plugin indices (already sorted by score)
	m.filteredIdx = make([]int, len(matches))
	for i, match := range matches {
		m.filteredIdx[i] = data.indices[match.Index]
	}

	m.listOffset = 0
	if len(m.filteredIdx) > 0 {
		m.selectedIdx = m.filteredIdx[0]
	}
}

// handleMouse processes mouse input.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonLeft:
		m.handleMouseClick(msg)
	case tea.MouseButtonWheelUp:
		for i := 0; i < wheelScrollSpeed; i++ {
			m.moveUp()
		}
	case tea.MouseButtonWheelDown:
		for i := 0; i < wheelScrollSpeed; i++ {
			m.moveDown()
		}
	}

	return m, nil
}

// handleMouseClick handles a mouse click on the left pane.
func (m *Model) handleMouseClick(msg tea.MouseMsg) {
	// Left pane width is roughly 1/3 of total width, minus 4 for padding/borders, plus 2 for border
	// = width/3 - 2 (net adjustment for borders and padding)
	leftPaneWidth := m.width/3 - 2
	if msg.X >= leftPaneWidth {
		return
	}

	// Calculate vertical offset: account for filter bar (1 line if active) + pane border (1 line)
	verticalOffset := 1 // Default: 1 for top border
	if m.filter.active {
		verticalOffset++ // Add 1 for filter input bar
	}
	// Calculate row index relative to visible area (not absolute plugin index)
	// getActualIndex will add listOffset, so don't add it here
	row := msg.Y - verticalOffset
	plugins := m.getVisiblePlugins()
	if row >= 0 && row < len(plugins)-m.listOffset {
		actualIdx := m.getActualIndex(row)
		if actualIdx >= 0 && !m.plugins[actualIdx].IsGroupHeader {
			m.selectedIdx = actualIdx
		}
	}
}

// cycleSortMode cycles through the available sort modes and applies sorting.
func (m *Model) cycleSortMode() {
	// Cycle: NameAsc -> NameDesc -> Scope -> Marketplace -> NameAsc
	switch m.main.sortMode {
	case SortByNameAsc:
		m.main.sortMode = SortByNameDesc
	case SortByNameDesc:
		m.main.sortMode = SortByScope
	case SortByScope:
		m.main.sortMode = SortByMarketplace
	case SortByMarketplace:
		m.main.sortMode = SortByNameAsc
	default:
		m.main.sortMode = SortByNameAsc
	}
	m.sortPlugins()
}

// sortPlugins sorts the plugin list according to the current sort mode.
func (m *Model) sortPlugins() {
	selectedID := m.getSelectedPluginID()
	plugins := m.extractNonHeaderPlugins()
	applySortMode(plugins, m.main.sortMode)
	m.plugins = rebuildWithGroupHeaders(plugins, m.main.sortMode)
	m.restoreSelection(selectedID)
}

// getSelectedPluginID returns the ID of the currently selected plugin, or empty string.
func (m *Model) getSelectedPluginID() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.plugins) && !m.plugins[m.selectedIdx].IsGroupHeader {
		return m.plugins[m.selectedIdx].ID
	}
	return ""
}

// extractNonHeaderPlugins returns all plugins excluding group headers.
func (m *Model) extractNonHeaderPlugins() []PluginState {
	var plugins []PluginState
	for _, p := range m.plugins {
		if !p.IsGroupHeader {
			plugins = append(plugins, p)
		}
	}
	return plugins
}

// applySortMode sorts the plugins slice according to the sort mode.
func applySortMode(plugins []PluginState, sortMode SortMode) {
	switch sortMode {
	case SortByNameAsc:
		sort.Slice(plugins, func(i, j int) bool {
			return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
		})
	case SortByNameDesc:
		sort.Slice(plugins, func(i, j int) bool {
			return strings.ToLower(plugins[i].Name) > strings.ToLower(plugins[j].Name)
		})
	case SortByScope:
		sortByScope(plugins)
	case SortByMarketplace:
		sortByMarketplace(plugins)
	}
}

// sortByScope sorts plugins by scope (installed first).
func sortByScope(plugins []PluginState) {
	scopeOrder := map[claude.Scope]int{
		claude.ScopeLocal:   0,
		claude.ScopeProject: 1,
		claude.ScopeUser:    2,
		claude.ScopeNone:    3,
	}
	sort.Slice(plugins, func(i, j int) bool {
		orderI := scopeOrder[plugins[i].InstalledScope]
		orderJ := scopeOrder[plugins[j].InstalledScope]
		if orderI != orderJ {
			return orderI < orderJ
		}
		return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
	})
}

// sortByMarketplace sorts plugins by marketplace name.
func sortByMarketplace(plugins []PluginState) {
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Marketplace != plugins[j].Marketplace {
			return plugins[i].Marketplace < plugins[j].Marketplace
		}
		return strings.ToLower(plugins[i].Name) < strings.ToLower(plugins[j].Name)
	})
}

// restoreSelection restores the selection to the plugin with the given ID.
func (m *Model) restoreSelection(selectedID string) {
	if selectedID != "" {
		for i, p := range m.plugins {
			if p.ID == selectedID {
				m.selectedIdx = i
				m.ensureVisible()
				return
			}
		}
	}
	m.selectFirstNonHeader()
}

// selectFirstNonHeader selects the first non-header plugin.
func (m *Model) selectFirstNonHeader() {
	for i, p := range m.plugins {
		if !p.IsGroupHeader {
			m.selectedIdx = i
			break
		}
	}
	m.listOffset = 0
}

// rebuildWithGroupHeaders rebuilds the plugin list with group headers based on sort mode.
func rebuildWithGroupHeaders(plugins []PluginState, sortMode SortMode) []PluginState {
	var result []PluginState

	switch sortMode {
	case SortByMarketplace:
		// Group by marketplace
		byGroup := make(map[string][]PluginState)
		var groups []string
		for _, p := range plugins {
			group := p.Marketplace
			if _, ok := byGroup[group]; !ok {
				groups = append(groups, group)
			}
			byGroup[group] = append(byGroup[group], p)
		}
		for _, group := range groups {
			result = append(result, PluginState{
				Name:          group,
				IsGroupHeader: true,
				Marketplace:   group,
			})
			result = append(result, byGroup[group]...)
		}

	case SortByScope:
		// Group by scope
		scopeOrder := []claude.Scope{claude.ScopeLocal, claude.ScopeProject, claude.ScopeUser, claude.ScopeNone}
		scopeNames := map[claude.Scope]string{
			claude.ScopeLocal:   "Local",
			claude.ScopeProject: "Project",
			claude.ScopeUser:    "User",
			claude.ScopeNone:    "Not Installed",
		}
		byScope := make(map[claude.Scope][]PluginState)
		for _, p := range plugins {
			byScope[p.InstalledScope] = append(byScope[p.InstalledScope], p)
		}
		for _, scope := range scopeOrder {
			if len(byScope[scope]) > 0 {
				result = append(result, PluginState{
					Name:          scopeNames[scope],
					IsGroupHeader: true,
				})
				result = append(result, byScope[scope]...)
			}
		}

	default:
		// For name sorts, group by marketplace to maintain structure
		byGroup := make(map[string][]PluginState)
		var groups []string
		for _, p := range plugins {
			group := p.Marketplace
			if _, ok := byGroup[group]; !ok {
				groups = append(groups, group)
			}
			byGroup[group] = append(byGroup[group], p)
		}
		sort.Strings(groups) // Sort groups alphabetically
		for _, group := range groups {
			result = append(result, PluginState{
				Name:          group,
				IsGroupHeader: true,
				Marketplace:   group,
			})
			// Sort within group by current sort mode
			groupPlugins := byGroup[group]
			if sortMode == SortByNameDesc {
				sort.Slice(groupPlugins, func(i, j int) bool {
					return strings.ToLower(groupPlugins[i].Name) > strings.ToLower(groupPlugins[j].Name)
				})
			} else {
				sort.Slice(groupPlugins, func(i, j int) bool {
					return strings.ToLower(groupPlugins[i].Name) < strings.ToLower(groupPlugins[j].Name)
				})
			}
			result = append(result, groupPlugins...)
		}
	}

	return result
}

// openReadme loads and renders the README for the selected plugin.
func (m *Model) openReadme() {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.InstallPath == "" {
		return
	}

	// Try to read README.md from the plugin's install path
	readmePath := filepath.Join(plugin.InstallPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		// Try lowercase
		readmePath = filepath.Join(plugin.InstallPath, "readme.md")
		content, err = os.ReadFile(readmePath)
		if err != nil {
			m.readme.content = "No README found for this plugin."
			m.readme.title = plugin.Name
			m.readme.scroll = 0
			m.mode = ModeReadme
			return
		}
	}

	// Render markdown with glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width-4),
	)
	if err != nil {
		m.readme.content = string(content)
	} else {
		rendered, err := renderer.Render(string(content))
		if err != nil {
			m.readme.content = string(content)
		} else {
			m.readme.content = rendered
		}
	}

	m.readme.title = plugin.Name
	m.readme.scroll = 0
	m.mode = ModeReadme
}

// updateReadme handles input in README view mode.
func (m *Model) updateReadme(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.handleReadmeKeyInput(msg)
	case tea.MouseMsg:
		m.handleReadmeMouseInput(msg)
	}
	return m, nil
}

// handleReadmeKeyInput processes keyboard input in README view mode.
func (m *Model) handleReadmeKeyInput(msg tea.KeyMsg) {
	keys := m.keys
	switch {
	case matchesKey(msg, keys.Escape), matchesKey(msg, keys.Quit), msg.String() == "?":
		m.closeReadmeView()
	case matchesKey(msg, keys.Up), msg.String() == "k":
		if m.readme.scroll > 0 {
			m.readme.scroll--
		}
	case matchesKey(msg, keys.Down), msg.String() == "j":
		m.readme.scroll++
	case matchesKey(msg, keys.PageUp):
		m.readme.scroll -= 10
		if m.readme.scroll < 0 {
			m.readme.scroll = 0
		}
	case matchesKey(msg, keys.PageDown):
		m.readme.scroll += 10
	case matchesKey(msg, keys.Home):
		m.readme.scroll = 0
	}
}

// handleReadmeMouseInput processes mouse input in README view mode.
func (m *Model) handleReadmeMouseInput(msg tea.MouseMsg) {
	if msg.Action != tea.MouseActionPress {
		return
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.readme.scroll -= wheelScrollSpeed
		if m.readme.scroll < 0 {
			m.readme.scroll = 0
		}
	case tea.MouseButtonWheelDown:
		m.readme.scroll += wheelScrollSpeed
	}
}

// closeReadmeView exits README view and returns to main view.
func (m *Model) closeReadmeView() {
	m.mode = ModeMain
	m.readme.content = ""
	m.readme.title = ""
	m.readme.scroll = 0
}
