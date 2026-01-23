package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// renderMainView renders the main two-pane view.
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

	// Add filter input if active
	if m.filterActive {
		filter := m.renderFilterInput(styles)
		return lipgloss.JoinVertical(lipgloss.Left, filter, main, help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, main, help)
}

// renderList renders the left pane plugin list.
func (m *Model) renderList(styles Styles) string {
	plugins := m.getVisiblePlugins()
	if len(plugins) == 0 {
		if m.filterActive && m.filterText != "" {
			return "No matches for: " + m.filterText
		}
		return "No plugins found."
	}

	var lines []string
	visibleHeight := styles.LeftPane.GetHeight() - 2

	// Calculate visible range
	start := m.listOffset
	end := start + visibleHeight
	if end > len(plugins) {
		end = len(plugins)
	}

	for i := start; i < end; i++ {
		plugin := plugins[i]
		// When filtering, getActualIndex converts filtered index to original.
		// When not filtering, i is already the actual index.
		var isSelected bool
		if m.filterActive && m.filterText != "" {
			// i is index into filtered list, need to get original index
			if i < len(m.filteredIdx) {
				isSelected = m.filteredIdx[i] == m.selectedIdx
			}
		} else {
			isSelected = i == m.selectedIdx
		}
		line := m.renderListItem(plugin, isSelected, styles)
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
	if m.filterActive {
		return styles.Help.Render("Type to filter • Enter: select • Esc: cancel")
	}

	if len(m.pending) > 0 {
		return styles.Help.Render("↑↓: navigate • l/p: local/project • u: uninstall • Tab: toggle • Enter: apply • Esc: clear • /: filter • r: refresh • q: quit")
	}
	return styles.Help.Render("↑↓: navigate • l/p: local/project • u: uninstall • Tab: toggle • /: filter • r: refresh • q: quit")
}

// renderConfirmation renders the confirmation modal.
func (m *Model) renderConfirmation(styles Styles) string {
	if len(m.pending) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, styles.Header.Render(" Apply Changes? "))
	lines = append(lines, "")

	// List pending operations
	installs := 0
	uninstalls := 0
	for pluginID, scope := range m.pending {
		var action string
		if scope == claude.ScopeNone {
			action = styles.Pending.Render("Uninstall: ") + pluginID
			uninstalls++
		} else {
			action = styles.ScopeProject.Render("Install ("+string(scope)+"): ") + pluginID
			installs++
		}
		lines = append(lines, "  "+action)
	}

	lines = append(lines, "")
	summary := ""
	if installs > 0 {
		summary += strconv.Itoa(installs) + " install(s)"
	}
	if uninstalls > 0 {
		if summary != "" {
			summary += ", "
		}
		summary += strconv.Itoa(uninstalls) + " uninstall(s)"
	}
	lines = append(lines, styles.DetailLabel.Render("Total: ")+summary)
	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Press Enter to confirm, Esc to cancel"))

	content := strings.Join(lines, "\n")

	// Center the modal
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(50).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// renderProgress renders the progress modal.
func (m *Model) renderProgress(styles Styles) string {
	var lines []string
	lines = append(lines, styles.Header.Render(" Applying Changes "))
	lines = append(lines, "")

	for i, op := range m.operations {
		var status string
		switch {
		case i < m.currentOpIdx:
			// Completed
			if i < len(m.operationErrors) && m.operationErrors[i] != "" {
				status = "✗ Failed: " + m.operationErrors[i]
			} else {
				status = "✓ Done"
			}
		case i == m.currentOpIdx:
			// In progress
			status = "⟳ Running..."
		default:
			// Pending
			status = "○ Pending"
		}

		action := "Install"
		if !op.IsInstall {
			action = "Uninstall"
		}
		scopeStr := ""
		if op.IsInstall {
			scopeStr = " (" + string(op.Scope) + ")"
		}

		line := status + " " + action + scopeStr + ": " + op.PluginID
		lines = append(lines, "  "+line)
	}

	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Please wait..."))

	content := strings.Join(lines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(60).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// renderErrorSummary renders the error summary modal.
func (m *Model) renderErrorSummary(styles Styles) string {
	var lines []string

	// Count errors
	errorCount := 0
	for _, e := range m.operationErrors {
		if e != "" {
			errorCount++
		}
	}

	if errorCount == 0 {
		lines = append(lines, styles.Header.Render(" All Changes Applied "))
	} else {
		lines = append(lines, styles.Header.Render(" Completed With Errors "))
	}
	lines = append(lines, "")

	successCount := len(m.operations) - errorCount
	lines = append(lines, styles.ScopeProject.Render(strconv.Itoa(successCount)+" succeeded"))
	if errorCount > 0 {
		lines = append(lines, styles.Pending.Render(strconv.Itoa(errorCount)+" failed"))
		lines = append(lines, "")
		lines = append(lines, styles.DetailLabel.Render("Errors:"))
		for i, op := range m.operations {
			if i < len(m.operationErrors) && m.operationErrors[i] != "" {
				lines = append(lines, "  • "+op.PluginID+": "+m.operationErrors[i])
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Press Enter or Esc to continue"))

	content := strings.Join(lines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(60).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// renderFilterInput renders the filter input bar.
func (m *Model) renderFilterInput(styles Styles) string {
	if !m.filterActive {
		return ""
	}

	input := "/" + m.filterText + "█"
	return styles.Header.Render(input)
}

// getVisiblePlugins returns plugins to display (filtered or all).
func (m *Model) getVisiblePlugins() []PluginState {
	if !m.filterActive || m.filterText == "" {
		return m.plugins
	}

	if len(m.filteredIdx) == 0 {
		return nil
	}

	result := make([]PluginState, len(m.filteredIdx))
	for i, idx := range m.filteredIdx {
		result[i] = m.plugins[idx]
	}
	return result
}

// getActualIndex converts a filtered index to the actual plugin index.
func (m *Model) getActualIndex(filteredIndex int) int {
	if !m.filterActive || m.filterText == "" {
		return filteredIndex + m.listOffset
	}
	if filteredIndex+m.listOffset < len(m.filteredIdx) {
		return m.filteredIdx[filteredIndex+m.listOffset]
	}
	return -1
}

// renderQuitConfirmation renders the quit confirmation modal.
func (m *Model) renderQuitConfirmation(styles Styles) string {
	var lines []string
	lines = append(lines, styles.Header.Render(" Quit Without Applying? "))
	lines = append(lines, "")
	lines = append(lines, "You have "+strconv.Itoa(len(m.pending))+" pending change(s).")
	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Press q again to quit, Esc to cancel"))

	content := strings.Join(lines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPending).
		Padding(1, 2).
		Width(40).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
