package tui

import (
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
