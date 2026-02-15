package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// scopeOrder defines the canonical display order for scopes.
var scopeOrder = []claude.Scope{claude.ScopeUser, claude.ScopeProject, claude.ScopeLocal}

// scopeLabel returns the uppercase label for a scope.
func scopeLabel(s claude.Scope) string {
	return strings.ToUpper(string(s))
}

// formatScopeSet renders multiple scopes as a comma-separated string in canonical order.
// Example: "USER, LOCAL" for a plugin installed at user and local scopes.
func formatScopeSet(scopes map[claude.Scope]bool) string {
	var labels []string
	for _, s := range scopeOrder {
		if _, ok := scopes[s]; ok {
			labels = append(labels, scopeLabel(s))
		}
	}
	return strings.Join(labels, ", ")
}

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
	if m.filter.active {
		filter := m.renderFilterInput(styles)
		return lipgloss.JoinVertical(lipgloss.Left, filter, main, help)
	}

	return lipgloss.JoinVertical(lipgloss.Left, main, help)
}

// renderList renders the left pane plugin list.
func (m *Model) renderList(styles Styles) string {
	plugins := m.getVisiblePlugins()
	if len(plugins) == 0 {
		if m.filter.active && m.filter.text != "" {
			return "No matches for: " + m.filter.text
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
		if m.filter.active && m.filter.text != "" {
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

	// Bulk selection checkbox
	if m.main.bulkSelected[plugin.ID] {
		parts = append(parts, "[x]")
	} else {
		parts = append(parts, "[ ]")
	}

	// Cursor indicator
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
	if op, ok := m.main.pendingOps[plugin.ID]; ok {
		return renderPendingIndicator(op, plugin, styles)
	}

	if !plugin.IsInstalled() {
		return ""
	}

	scopeText := formatScopeSet(plugin.InstalledScopes)

	// Append disabled status if applicable
	if !plugin.Enabled {
		scopeText += " (disabled)"
	}

	// For single-scope, style by scope type; for multi-scope, use first scope's style
	var style lipgloss.Style
	if plugin.IsSingleScope() {
		switch plugin.SingleScope() {
		case claude.ScopeLocal:
			style = styles.ScopeLocal
		case claude.ScopeProject:
			style = styles.ScopeProject
		case claude.ScopeUser:
			style = styles.ScopeUser
		default:
			return ""
		}
	} else {
		// Multi-scope: use project style as neutral choice
		style = styles.ScopeProject
	}

	result := style.Render("[" + scopeText + "]")

	if plugin.HasUpdate {
		result += styles.Pending.Render(" ↑")
	}

	return result
}

// applyScopeDelta returns a new scope set with uninstalls removed and installs added.
func applyScopeDelta(base map[claude.Scope]bool, uninstall, install []claude.Scope) map[claude.Scope]bool {
	result := copyMap(base)
	for _, s := range uninstall {
		delete(result, s)
	}
	for _, s := range install {
		result[s] = true
	}
	return result
}

// renderPendingIndicator renders the pending operation indicator for a plugin.
func renderPendingIndicator(op Operation, plugin PluginState, styles Styles) string {
	pending := func(s string) string { return styles.Pending.Render(s) }

	switch op.Type {
	case OpInstall:
		if plugin.IsInstalled() {
			newScopes := applyScopeDelta(plugin.InstalledScopes, nil, op.Scopes)
			return pending("[-> " + formatScopeSet(newScopes) + "]")
		}
		return pending("[-> " + scopeLabel(op.Scopes[0]) + "]")
	case OpUninstall:
		if len(op.OriginalScopes) > 1 || (plugin.IsInstalled() && !plugin.IsSingleScope()) {
			remaining := applyScopeDelta(plugin.InstalledScopes, op.Scopes, nil)
			if len(remaining) > 0 {
				return pending("[" + formatScopeSet(plugin.InstalledScopes) + " -> " + formatScopeSet(remaining) + "]")
			}
		}
		return pending("[-> UNINSTALL]")
	case OpMigrate:
		return pending("[" + scopeLabel(firstScope(op.OriginalScopes)) + " -> " + scopeLabel(op.Scopes[0]) + "]")
	case OpUpdate:
		return pending("[-> UPDATE]")
	case OpEnable:
		return pending("[-> ENABLED]")
	case OpDisable:
		return pending("[-> DISABLED]")
	case OpScopeChange:
		newScopes := applyScopeDelta(plugin.InstalledScopes, op.UninstallScopes, op.Scopes)
		return pending("[" + formatScopeSet(plugin.InstalledScopes) + " -> " + formatScopeSet(newScopes) + "]")
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
		versionStr := plugin.Version
		if plugin.HasUpdate && plugin.AvailableVersion != "" {
			versionStr += " → " + plugin.AvailableVersion + " available"
		}
		lines = append(lines, styles.DetailLabel.Render("Version: ")+
			styles.DetailValue.Render(versionStr))
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

	// Install count (only for plugins with available info)
	if plugin.InstallCount > 0 {
		lines = append(lines, styles.DetailLabel.Render("Installs: ")+
			styles.DetailValue.Render(formatInstallCount(plugin.InstallCount)))
	}

	// Last updated (only for installed plugins)
	if plugin.LastUpdated != "" {
		lines = append(lines, styles.DetailLabel.Render("Last updated: ")+
			styles.DetailValue.Render(formatTimestamp(plugin.LastUpdated)))
	}

	// Status
	status := m.getStatusText(plugin)
	lines = append(lines, styles.DetailLabel.Render("Status: ")+
		styles.DetailValue.Render(status))

	return lines
}

// getStatusText returns the status text for a plugin.
func (m *Model) getStatusText(plugin PluginState) string {
	if !plugin.IsInstalled() {
		return "Not installed"
	}
	status := "Installed (" + formatScopeSet(plugin.InstalledScopes) + ")"
	if plugin.Enabled {
		status += " - Enabled"
	} else {
		status += " - Disabled"
	}
	return status
}

// joinScopes returns a comma-separated string of scope names.
func joinScopes(scopes []claude.Scope) string {
	parts := make([]string, len(scopes))
	for i, s := range scopes {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}

// appendPendingChange appends pending change information if applicable.
func (m *Model) appendPendingChange(lines []string, plugin PluginState, styles Styles) []string {
	op, ok := m.main.pendingOps[plugin.ID]
	if !ok {
		return lines
	}

	var pendingStr string
	switch op.Type {
	case OpInstall:
		pendingStr = op.Type.meta().Pending + " " + joinScopes(op.Scopes)
	case OpUninstall:
		pendingStr = m.formatUninstallPending(op, plugin)
	case OpMigrate:
		pendingStr = op.Type.meta().Pending + " " + string(firstScope(op.OriginalScopes)) + " to " + string(op.Scopes[0])
	case OpScopeChange:
		pendingStr = "Will add " + joinScopes(op.Scopes) + " and remove " + joinScopes(op.UninstallScopes)
	default:
		pendingStr = op.Type.meta().Pending
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

// formatUninstallPending returns the pending string for an uninstall operation.
func (m *Model) formatUninstallPending(op Operation, plugin PluginState) string {
	if len(op.Scopes) > 0 && plugin.IsInstalled() && !plugin.IsSingleScope() {
		remaining := applyScopeDelta(plugin.InstalledScopes, op.Scopes, nil)
		if len(remaining) > 0 {
			return "Will be uninstalled from " + joinScopes(op.Scopes) + " (keeping " + formatScopeSet(remaining) + ")"
		}
	}
	return "Will be uninstalled"
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
	if !plugin.IsExternal || plugin.IsInstalled() {
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
	if m.filter.active {
		return styles.Help.Render("Type to filter • Enter: select • Esc: cancel")
	}

	// Show mouse state indicator
	mouseIndicator := "m: mouse off"
	if m.main.mouseEnabled {
		mouseIndicator = "m: mouse on"
	}

	// Show current sort mode
	sortInfo := "s: " + m.main.sortMode.String()

	// Show selection count if any
	selectionInfo := ""
	if len(m.main.bulkSelected) > 0 {
		selectionInfo = fmt.Sprintf(" • %d selected", len(m.main.bulkSelected))
	}

	baseHelp := "↑↓: navigate • Space: select • a/A: all/none • l/p/u/U: install/uninstall/update • Tab: toggle • " + sortInfo + " • c: config"
	if len(m.main.pendingOps) > 0 {
		return styles.Help.Render(baseHelp + " • Enter: apply • Esc: clear • /: filter • ?: readme • C: changelog • " + mouseIndicator + selectionInfo + " • q: quit")
	}
	return styles.Help.Render(baseHelp + " • /: filter • ?: readme • C: changelog • " + mouseIndicator + selectionInfo + " • q: quit")
}

// renderDoc renders the document view (README or CHANGELOG).
func (m *Model) renderDoc(styles Styles) string {
	if m.doc.content == "" {
		return "No content."
	}

	// Split content into lines and handle scrolling
	lines := strings.Split(m.doc.content, "\n")
	visibleHeight := m.height - 4 // Account for header and help bar

	// Clamp scroll position
	maxScroll := len(lines) - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.doc.scroll > maxScroll {
		m.doc.scroll = maxScroll
	}

	// Get visible lines
	endIdx := m.doc.scroll + visibleHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	visibleLines := lines[m.doc.scroll:endIdx]
	content := strings.Join(visibleLines, "\n")

	// Build header
	header := styles.Header.Render(" " + m.doc.title + " ")

	// Build content pane
	contentPane := lipgloss.NewStyle().
		Width(m.width - 2).
		Height(visibleHeight).
		Render(content)

	// Build help bar
	help := styles.Help.Render("↑↓/jk: scroll • PgUp/PgDn: page • g: top • q/Esc/?: close")

	return lipgloss.JoinVertical(lipgloss.Left, header, contentPane, help)
}

// renderConfirmation renders the confirmation modal.
func (m *Model) renderConfirmation(styles Styles) string {
	if len(m.main.pendingOps) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, styles.Header.Render(" Apply Changes? "))
	lines = append(lines, "")

	// Collect and sort operations by type
	var operations []Operation
	for _, op := range m.main.pendingOps {
		operations = append(operations, op)
	}

	// Sort operations by type for consistent display
	// Uninstalls, then migrations, then updates, then installs, then enables, then disables
	sort.Slice(operations, func(i, j int) bool {
		typeOrder := map[OperationType]int{
			OpUninstall: 0,
			OpMigrate:   1,
			OpUpdate:    2,
			OpInstall:   3,
			OpEnable:    4,
			OpDisable:   5,
		}
		return typeOrder[operations[i].Type] < typeOrder[operations[j].Type]
	})

	for _, op := range operations {
		lines = append(lines, "  "+formatOperationLine(op, styles))
	}

	lines = append(lines, "")

	// Build summary line
	summary := buildOperationSummary(operations)
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

// formatOperationLine formats a single operation for display in the confirmation modal.
func formatOperationLine(op Operation, styles Styles) string {
	verb, scopeDetail := formatOperationAction(op)

	// Destructive operations use pending (warning) style
	style := styles.ScopeProject
	if op.Type == OpUninstall || op.Type == OpDisable {
		style = styles.Pending
	}

	return style.Render(verb+scopeDetail+": ") + op.PluginID
}

// buildOperationSummary builds a summary string counting operations by type.
func buildOperationSummary(operations []Operation) string {
	counts := make(map[OperationType]int)
	for _, op := range operations {
		counts[op.Type]++
	}

	order := []OperationType{OpInstall, OpUninstall, OpMigrate, OpUpdate, OpEnable, OpDisable, OpScopeChange}

	var parts []string
	for _, opType := range order {
		if c := counts[opType]; c > 0 {
			parts = append(parts, strconv.Itoa(c)+" "+opType.meta().Noun)
		}
	}
	return strings.Join(parts, ", ")
}

// formatOperationAction returns the action label and scope detail for a progress line.
func formatOperationAction(op Operation) (string, string) {
	verb := op.Type.meta().Verb
	switch op.Type {
	case OpInstall:
		return verb, " (" + joinScopes(op.Scopes) + ")"
	case OpMigrate:
		return verb, " (" + string(firstScope(op.OriginalScopes)) + " -> " + string(op.Scopes[0]) + ")"
	case OpScopeChange:
		return verb, " (+" + joinScopes(op.Scopes) + " -" + joinScopes(op.UninstallScopes) + ")"
	default:
		return verb, ""
	}
}

// renderProgress renders the progress modal.
func (m *Model) renderProgress(styles Styles) string {
	var lines []string
	lines = append(lines, styles.Header.Render(" Applying Changes "))
	lines = append(lines, "")

	for i, op := range m.progress.operations {
		var status string
		switch {
		case i < m.progress.currentIdx:
			if i < len(m.progress.errors) && m.progress.errors[i] != "" {
				status = "✗ Failed: " + m.progress.errors[i]
			} else {
				status = "✓ Done"
			}
		case i == m.progress.currentIdx:
			status = "⟳ Running..."
		default:
			status = "○ Pending"
		}

		action, scopeStr := formatOperationAction(op)
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
	for _, e := range m.progress.errors {
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

	successCount := len(m.progress.operations) - errorCount
	lines = append(lines, styles.ScopeProject.Render(strconv.Itoa(successCount)+" succeeded"))
	if errorCount > 0 {
		lines = append(lines, styles.Pending.Render(strconv.Itoa(errorCount)+" failed"))
		lines = append(lines, "")
		lines = append(lines, styles.DetailLabel.Render("Errors:"))
		for i, op := range m.progress.operations {
			if i < len(m.progress.errors) && m.progress.errors[i] != "" {
				lines = append(lines, "  • "+op.PluginID+": "+m.progress.errors[i])
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
	if !m.filter.active {
		return ""
	}

	input := "/" + m.filter.text + "█"
	return styles.Header.Render(input)
}

// getVisiblePlugins returns plugins to display (filtered or all).
func (m *Model) getVisiblePlugins() []PluginState {
	if !m.filter.active || m.filter.text == "" {
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
	if !m.filter.active || m.filter.text == "" {
		return filteredIndex + m.listOffset
	}
	if filteredIndex+m.listOffset < len(m.filteredIdx) {
		return m.filteredIdx[filteredIndex+m.listOffset]
	}
	return -1
}

// formatInstallCount formats an install count for display.
func formatInstallCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// formatTimestamp formats an ISO timestamp for display.
func formatTimestamp(timestamp string) string {
	// Try to parse ISO 8601 format
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Fallback to just showing the raw timestamp
		return timestamp
	}
	return t.Format("Jan 2, 2006")
}

// renderQuitConfirmation renders the quit confirmation modal.
func (m *Model) renderQuitConfirmation(styles Styles) string {
	var lines []string
	lines = append(lines, styles.Header.Render(" Quit Without Applying? "))
	lines = append(lines, "")
	lines = append(lines, "You have "+strconv.Itoa(len(m.main.pendingOps))+" pending change(s).")
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

// scopeDialogLabels shows the scope name and settings file path.
var scopeDialogLabels = [3]struct {
	name string
	path string
}{
	{"User", "~/.claude/settings.json"},
	{"Project", ".claude/settings.json"},
	{"Local", ".claude/settings.local.json"},
}

// renderScopeDialog renders the scope selection dialog as a centered overlay.
func (m *Model) renderScopeDialog(styles Styles) string {
	dialog := &m.main.scopeDialog

	var lines []string
	lines = append(lines, styles.Header.Render(" Scopes for "+dialog.pluginID+" "))
	lines = append(lines, "")

	for i := 0; i < 3; i++ {
		checkbox := "[ ]"
		if dialog.scopes[i] {
			checkbox = "[x]"
		}

		cursor := "  "
		if dialog.cursor == i {
			cursor = "> "
		}

		label := scopeDialogLabels[i]
		line := cursor + checkbox + " " + label.name
		// Pad to align paths
		for len(line) < 25 {
			line += " "
		}
		line += "(" + label.path + ")"

		if dialog.cursor == i {
			lines = append(lines, styles.Selected.Render(line))
		} else {
			lines = append(lines, "  "+line)
		}
	}

	lines = append(lines, "")
	lines = append(lines, "  Press space to toggle, Enter to confirm, Esc to cancel")

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Palette.Project).
			Padding(1, 2).
			Render(strings.Join(lines, "\n")),
	)
}

// renderConfig renders the config viewer.
func (m *Model) renderConfig(styles Styles) string {
	// Header
	header := styles.Header.Render(" " + m.config.title + " ")

	// Content area
	contentHeight := m.height - 4 // Account for header and help bar
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Split content into lines and apply scroll
	lines := strings.Split(m.config.content, "\n")
	startLine := m.config.scroll
	if startLine >= len(lines) {
		startLine = len(lines) - 1
		if startLine < 0 {
			startLine = 0
		}
	}
	endLine := startLine + contentHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[startLine:endLine]
	content := strings.Join(visibleLines, "\n")

	// Apply style to content
	contentStyle := lipgloss.NewStyle().
		Width(m.width-4).
		Height(contentHeight).
		Padding(1, 2)

	contentBox := contentStyle.Render(content)

	// Help bar
	scrollInfo := ""
	if len(lines) > contentHeight {
		scrollInfo = " (" + strconv.Itoa(startLine+1) + "-" + strconv.Itoa(endLine) + "/" + strconv.Itoa(len(lines)) + ")"
	}
	help := styles.Help.Render("↑↓/PgUp/PgDn: scroll • c/Esc: close" + scrollInfo + " • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, contentBox, help)
}
