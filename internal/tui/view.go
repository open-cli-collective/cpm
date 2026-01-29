package tui

import (
	"sort"
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
	if op, ok := m.pendingOps[plugin.ID]; ok {
		switch op.Type {
		case OpInstall:
			return styles.Pending.Render("[→ " + strings.ToUpper(string(op.Scope)) + "]")
		case OpUninstall:
			return styles.Pending.Render("[→ UNINSTALL]")
		case OpEnable:
			return styles.Pending.Render("[→ ENABLED]")
		case OpDisable:
			return styles.Pending.Render("[→ DISABLED]")
		}
	}

	// Show current scope
	var scopeText string
	switch plugin.InstalledScope {
	case claude.ScopeLocal:
		scopeText = "LOCAL"
	case claude.ScopeProject:
		scopeText = "PROJECT"
	case claude.ScopeUser:
		scopeText = "USER"
	default:
		return ""
	}

	// Append disabled status if applicable
	if !plugin.Enabled {
		scopeText += ", DISABLED"
	}

	// Apply style based on scope
	switch plugin.InstalledScope {
	case claude.ScopeLocal:
		return styles.ScopeLocal.Render("[" + scopeText + "]")
	case claude.ScopeProject:
		return styles.ScopeProject.Render("[" + scopeText + "]")
	case claude.ScopeUser:
		return styles.ScopeUser.Render("[" + scopeText + "]")
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

	// Basic info
	lines = append(lines, m.renderPluginInfo(plugin, styles)...)

	// Pending change
	lines = m.appendPendingChange(lines, plugin, styles)

	// Description
	if plugin.Description != "" {
		lines = append(lines, "")
		lines = append(lines, styles.DetailLabel.Render("Description:"))
		lines = append(lines, styles.DetailDescription.Render(plugin.Description))
	}

	// Components (what comes with this plugin)
	lines = m.appendComponents(lines, plugin, styles)

	// Show external plugin notice if applicable
	lines = m.appendExternalNotice(lines, plugin, styles)

	return strings.Join(lines, "\n")
}

// renderPluginInfo renders the basic plugin information fields.
func (m *Model) renderPluginInfo(plugin PluginState, styles Styles) []string {
	var lines []string

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

	// Author
	if plugin.AuthorName != "" {
		authorStr := plugin.AuthorName
		if plugin.AuthorEmail != "" {
			authorStr += " <" + plugin.AuthorEmail + ">"
		}
		lines = append(lines, styles.DetailLabel.Render("Author: ")+
			styles.DetailValue.Render(authorStr))
	}

	// Status
	status := m.getStatusText(plugin)
	lines = append(lines, styles.DetailLabel.Render("Status: ")+
		styles.DetailValue.Render(status))

	return lines
}

// getStatusText returns the status text for a plugin.
func (m *Model) getStatusText(plugin PluginState) string {
	if plugin.InstalledScope == claude.ScopeNone {
		return "Not installed"
	}
	status := "Installed (" + string(plugin.InstalledScope) + ")"
	if plugin.Enabled {
		status += " - Enabled"
	} else {
		status += " - Disabled"
	}
	return status
}

// appendPendingChange appends pending change information if applicable.
func (m *Model) appendPendingChange(lines []string, plugin PluginState, styles Styles) []string {
	op, ok := m.pendingOps[plugin.ID]
	if !ok {
		return lines
	}

	var pendingStr string
	switch op.Type {
	case OpInstall:
		pendingStr = "Will be installed to " + string(op.Scope)
	case OpUninstall:
		pendingStr = "Will be uninstalled"
	case OpEnable:
		pendingStr = "Will be enabled"
	case OpDisable:
		pendingStr = "Will be disabled"
	default:
		pendingStr = "Unknown operation"
	}

	return append(lines, styles.Pending.Render("Pending: "+pendingStr))
}

// appendComponents appends component information if the plugin has any.
func (m *Model) appendComponents(lines []string, plugin PluginState, styles Styles) []string {
	if plugin.Components == nil {
		return lines
	}

	hasComponents := len(plugin.Components.Skills) > 0 ||
		len(plugin.Components.Agents) > 0 ||
		len(plugin.Components.Commands) > 0 ||
		len(plugin.Components.Hooks) > 0 ||
		len(plugin.Components.MCPs) > 0

	if !hasComponents {
		return lines
	}

	lines = append(lines, "")
	lines = append(lines, styles.DetailLabel.Render("Includes:"))

	lines = appendComponentCategory(lines, "Skills", plugin.Components.Skills, styles)
	lines = appendComponentCategory(lines, "Agents", plugin.Components.Agents, styles)
	lines = appendComponentCategory(lines, "Commands", plugin.Components.Commands, styles)
	lines = appendComponentCategory(lines, "Hooks", plugin.Components.Hooks, styles)
	lines = appendComponentCategory(lines, "MCPs", plugin.Components.MCPs, styles)

	return lines
}

// appendComponentCategory appends a category of components if non-empty.
func appendComponentCategory(lines []string, category string, items []string, styles Styles) []string {
	if len(items) == 0 {
		return lines
	}
	lines = append(lines, styles.ComponentCategory.Render(category))
	for _, item := range items {
		lines = append(lines, styles.ComponentItem.Render("• "+item))
	}
	return lines
}

// appendExternalNotice appends the external plugin notice if applicable.
func (m *Model) appendExternalNotice(lines []string, plugin PluginState, styles Styles) []string {
	if !plugin.IsExternal || plugin.InstalledScope != claude.ScopeNone {
		return lines
	}
	lines = append(lines, "")
	lines = append(lines, styles.DetailLabel.Render("Source:"))
	lines = append(lines, styles.DetailDescription.Render("External plugin (hosted on GitHub)"))
	if plugin.ExternalURL != "" {
		lines = append(lines, styles.Help.Render(plugin.ExternalURL))
	}
	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Component details available after installation."))
	return lines
}

// renderHelp renders the help bar at the bottom.
func (m *Model) renderHelp(styles Styles) string {
	if m.filterActive {
		return styles.Help.Render("Type to filter • Enter: select • Esc: cancel")
	}

	// Show mouse state indicator
	mouseIndicator := "m: mouse off"
	if m.mouseEnabled {
		mouseIndicator = "m: mouse on"
	}

	if len(m.pendingOps) > 0 {
		return styles.Help.Render("↑↓: navigate • l/p/u: install/uninstall • Tab: toggle • Enter: apply • Esc: clear • /: filter • r: refresh • " + mouseIndicator + " • q: quit")
	}
	return styles.Help.Render("↑↓: navigate • l/p/u: install/uninstall • Tab: toggle • /: filter • r: refresh • " + mouseIndicator + " • q: quit")
}

// renderConfirmation renders the confirmation modal.
func (m *Model) renderConfirmation(styles Styles) string {
	if len(m.pendingOps) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, styles.Header.Render(" Apply Changes? "))
	lines = append(lines, "")

	// Count operations by type
	installs := 0
	uninstalls := 0
	enables := 0
	disables := 0

	// Display operations grouped by type
	var operations []Operation
	for _, op := range m.pendingOps {
		operations = append(operations, op)
	}

	// Sort operations by type for consistent display
	// Uninstalls, then installs, then enables, then disables
	sort.Slice(operations, func(i, j int) bool {
		typeOrder := map[OperationType]int{
			OpUninstall: 0,
			OpInstall:   1,
			OpEnable:    2,
			OpDisable:   3,
		}
		return typeOrder[operations[i].Type] < typeOrder[operations[j].Type]
	})

	for _, op := range operations {
		var action string
		switch op.Type {
		case OpInstall:
			action = styles.ScopeProject.Render("Install ("+string(op.Scope)+"): ") + op.PluginID
			installs++
		case OpUninstall:
			action = styles.Pending.Render("Uninstall: ") + op.PluginID
			uninstalls++
		case OpEnable:
			action = styles.ScopeProject.Render("Enable: ") + op.PluginID
			enables++
		case OpDisable:
			action = styles.Pending.Render("Disable: ") + op.PluginID
			disables++
		}
		lines = append(lines, "  "+action)
	}

	lines = append(lines, "")

	// Build summary line
	var summaryParts []string
	if installs > 0 {
		summaryParts = append(summaryParts, strconv.Itoa(installs)+" install(s)")
	}
	if uninstalls > 0 {
		summaryParts = append(summaryParts, strconv.Itoa(uninstalls)+" uninstall(s)")
	}
	if enables > 0 {
		summaryParts = append(summaryParts, strconv.Itoa(enables)+" enable(s)")
	}
	if disables > 0 {
		summaryParts = append(summaryParts, strconv.Itoa(disables)+" disable(s)")
	}

	summary := strings.Join(summaryParts, ", ")
	lines = append(lines, styles.DetailLabel.Render(summary))
	lines = append(lines, "")
	lines = append(lines, "Press Enter to confirm, Esc to cancel")

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Header.GetForeground()).
			Padding(1, 2).
			Render(strings.Join(lines, "\n")),
	)
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

		var action string
		var scopeStr string

		switch op.Type {
		case OpInstall:
			action = "Install"
			scopeStr = " (" + string(op.Scope) + ")"
		case OpUninstall:
			action = "Uninstall"
			scopeStr = ""
		case OpEnable:
			action = "Enable"
			scopeStr = ""
		case OpDisable:
			action = "Disable"
			scopeStr = ""
		default:
			action = "Unknown"
			scopeStr = ""
		}

		line := status + " " + action + scopeStr + ": " + op.PluginID
		lines = append(lines, "  "+line)
	}

	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Please wait..."))

	content := strings.Join(lines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Palette.Primary).
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
		BorderForeground(styles.Palette.Primary).
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
	lines = append(lines, "You have "+strconv.Itoa(len(m.pendingOps))+" pending change(s).")
	lines = append(lines, "")
	lines = append(lines, styles.Help.Render("Press q again to quit, Esc to cancel"))

	content := strings.Join(lines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Palette.Pending).
		Padding(1, 2).
		Width(40).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
