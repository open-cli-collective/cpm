package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// updateMain handles messages in main mode.
func (m *Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if ok {
		return m.handleKeyPress(keyMsg)
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

	case matchesKey(msg, keys.Local):
		m.selectForInstall(claude.ScopeLocal)

	case matchesKey(msg, keys.Project):
		m.selectForInstall(claude.ScopeProject)

	case matchesKey(msg, keys.Toggle):
		m.toggleScope()

	case matchesKey(msg, keys.Uninstall):
		m.selectForUninstall()

	case matchesKey(msg, keys.Escape):
		m.clearPending()
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

// selectForInstall marks the selected plugin for installation at the given scope.
func (m *Model) selectForInstall(scope claude.Scope) {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader {
		return
	}

	// If already installed at this scope, remove the pending change
	if plugin.InstalledScope == scope {
		delete(m.pending, plugin.ID)
		return
	}

	m.pending[plugin.ID] = scope
}

// toggleScope cycles through: none -> local -> project -> uninstall -> none
func (m *Model) toggleScope() {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader {
		return
	}

	current := m.getCurrentDesiredScope(plugin)

	var next claude.Scope
	switch current {
	case claude.ScopeNone:
		// Not installed and no pending -> install local
		next = claude.ScopeLocal
	case claude.ScopeLocal:
		// Local (or pending local) -> project
		next = claude.ScopeProject
	case claude.ScopeProject:
		// Project (or pending project) -> uninstall (if installed) or none
		if plugin.InstalledScope != claude.ScopeNone {
			// Mark for uninstall
			m.pending[plugin.ID] = claude.ScopeNone
			return
		}
		// Not installed, just clear pending
		delete(m.pending, plugin.ID)
		return
	}

	// If cycling back to original state, clear pending
	if next == plugin.InstalledScope {
		delete(m.pending, plugin.ID)
	} else {
		m.pending[plugin.ID] = next
	}
}

// selectForUninstall marks the selected plugin for uninstallation.
func (m *Model) selectForUninstall() {
	plugin := m.getSelectedPlugin()
	if plugin == nil || plugin.IsGroupHeader {
		return
	}

	// Can only uninstall if currently installed
	if plugin.InstalledScope == claude.ScopeNone {
		// If pending install, clear it
		delete(m.pending, plugin.ID)
		return
	}

	// Toggle uninstall
	if pending, ok := m.pending[plugin.ID]; ok && pending == claude.ScopeNone {
		// Already marked for uninstall, clear it
		delete(m.pending, plugin.ID)
	} else {
		// Mark for uninstall
		m.pending[plugin.ID] = claude.ScopeNone
	}
}

// clearPending clears the pending change for the selected plugin.
func (m *Model) clearPending() {
	plugin := m.getSelectedPlugin()
	if plugin == nil {
		return
	}
	delete(m.pending, plugin.ID)
}

// getSelectedPlugin returns the currently selected plugin, or nil if none.
func (m *Model) getSelectedPlugin() *PluginState {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.plugins) {
		return nil
	}
	return &m.plugins[m.selectedIdx]
}

// getCurrentDesiredScope returns the effective scope (pending or installed).
func (m *Model) getCurrentDesiredScope(plugin *PluginState) claude.Scope {
	if pending, ok := m.pending[plugin.ID]; ok {
		return pending
	}
	return plugin.InstalledScope
}
