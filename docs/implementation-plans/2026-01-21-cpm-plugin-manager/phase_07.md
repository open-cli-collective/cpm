# cpm - Phase 7: UX Enhancements

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Polish features for delightful user experience

**Architecture:** Add filter/search mode, refresh functionality, quit confirmation when pending changes exist, and mouse support. Keep existing architecture, extend with new state fields and handlers.

**Tech Stack:** Bubble Tea (mouse events, text input)

**Scope:** Phase 7 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 6 complete with full execution flow

---

## Task 1: Add Filter Mode State

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Add filter state fields to Model**

Add these fields to the Model struct:

```go
// Model is the main application model.
type Model struct {
	// ... existing fields ...

	// Filter state
	filterText   string
	filterActive bool
	filteredIdx  []int // indices into plugins that match filter
}
```

**Step 2: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat(tui): add filter state fields"
```

---

## Task 2: Add Filter Input View

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Add filter input rendering**

Add to `internal/tui/view.go`:

```go
// renderFilterInput renders the filter input bar.
func (m *Model) renderFilterInput(styles Styles) string {
	if !m.filterActive {
		return ""
	}

	input := "/" + m.filterText + "█"
	return styles.Header.Render(input)
}
```

**Step 2: Update renderMainView to include filter**

Update `renderMainView` to show filter input:

```go
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
```

**Step 3: Update renderList to use filtered results**

Update `renderList` to only show filtered plugins when filter is active:

```go
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
		isSelected := m.getActualIndex(i) == m.selectedIdx
		line := m.renderListItem(plugin, isSelected, styles)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
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
```

**Step 4: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add filter input and filtered list rendering"
```

---

## Task 3: Add Filter Update Logic

**Files:**
- Modify: `internal/tui/update.go`

**Step 1: Add filter handlers**

Add to `internal/tui/update.go`:

```go
// updateFilter handles filter mode input.
func (m *Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterActive = false
		m.filterText = ""
		m.filteredIdx = nil
		m.listOffset = 0

	case tea.KeyEnter:
		m.filterActive = false
		// Keep filtered results, select first match if any
		if len(m.filteredIdx) > 0 {
			m.selectedIdx = m.filteredIdx[0]
		}
		m.filterText = ""
		m.filteredIdx = nil

	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}

	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.applyFilter()
	}

	return m, nil
}

// applyFilter updates filteredIdx based on filterText.
func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filteredIdx = nil
		return
	}

	filter := strings.ToLower(m.filterText)
	m.filteredIdx = nil

	for i, p := range m.plugins {
		if p.IsGroupHeader {
			continue
		}
		name := strings.ToLower(p.Name)
		desc := strings.ToLower(p.Description)
		id := strings.ToLower(p.ID)

		if strings.Contains(name, filter) || strings.Contains(desc, filter) || strings.Contains(id, filter) {
			m.filteredIdx = append(m.filteredIdx, i)
		}
	}

	m.listOffset = 0
	if len(m.filteredIdx) > 0 {
		m.selectedIdx = m.filteredIdx[0]
	}
}
```

**Step 2: Update handleKeyPress to handle filter key**

Add to `handleKeyPress`:

```go
case matchesKey(msg, keys.Filter):
	m.filterActive = true
	m.filterText = ""
	m.filteredIdx = nil
```

**Step 3: Update updateMain to route to filter**

Update `updateMain`:

```go
// updateMain handles messages in main mode.
func (m *Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter mode
		if m.filterActive {
			return m.updateFilter(msg)
		}
		return m.handleKeyPress(msg)
	}
	return m, nil
}
```

**Step 4: Commit**

```bash
git add internal/tui/update.go
git commit -m "feat(tui): add filter mode with search functionality"
```

---

## Task 4: Add Refresh Functionality

**Files:**
- Modify: `internal/tui/update.go`

**Step 1: Add refresh handler**

Add to `handleKeyPress` in `internal/tui/update.go`:

```go
case matchesKey(msg, keys.Refresh):
	m.loading = true
	return m, m.loadPlugins
```

**Step 2: Commit**

```bash
git add internal/tui/update.go
git commit -m "feat(tui): add refresh with r key"
```

---

## Task 5: Add Quit Confirmation

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/update.go`

**Step 1: Add quit confirmation state**

Add to Model struct in `model.go`:

```go
// Quit confirmation
showQuitConfirm bool
```

**Step 2: Add quit confirmation view**

Add to `view.go`:

```go
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
```

**Step 3: Update View to show quit confirmation**

Update `View` in `model.go`:

```go
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

	// ... rest of View
}
```

**Step 4: Add quit confirmation logic**

Update `handleKeyPress` in `update.go`:

```go
case matchesKey(msg, keys.Quit):
	if len(m.pending) > 0 && !m.showQuitConfirm {
		m.showQuitConfirm = true
		return m, nil
	}
	return m, tea.Quit
```

**Step 5: Add quit confirmation handler**

Add to `updateMain`:

```go
// updateMain handles messages in main mode.
func (m *Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle quit confirmation
		if m.showQuitConfirm {
			switch {
			case matchesKey(msg, m.keys.Quit):
				return m, tea.Quit
			case matchesKey(msg, m.keys.Escape):
				m.showQuitConfirm = false
				return m, nil
			}
		}

		// Handle filter mode
		if m.filterActive {
			return m.updateFilter(msg)
		}
		return m.handleKeyPress(msg)
	}
	return m, nil
}
```

**Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/view.go internal/tui/update.go
git commit -m "feat(tui): add quit confirmation when pending changes exist"
```

---

## Task 6: Add Mouse Support

**Files:**
- Modify: `cmd/cpm/main.go`
- Modify: `internal/tui/update.go`

**Step 1: Enable mouse in main.go**

Update the tea.NewProgram call in `cmd/cpm/main.go`:

```go
// Run the TUI
p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
```

**Step 2: Add mouse handlers**

Add to `updateMain` in `update.go`:

```go
case tea.MouseMsg:
	return m.handleMouse(msg)
```

Add the mouse handler function:

```go
// handleMouse processes mouse input.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseLeft:
		// Calculate which item was clicked
		// Left pane is roughly 1/3 of width
		leftPaneWidth := m.width / 3
		if msg.X < leftPaneWidth {
			// Clicked in left pane
			// Calculate vertical offset: account for filter bar (1 line if active) + pane border (1 line)
			verticalOffset := 1 // Default: 1 for top border
			if m.filterActive {
				verticalOffset += 1 // Add 1 for filter input bar
			}
			row := msg.Y - verticalOffset + m.listOffset
			plugins := m.getVisiblePlugins()
			if row >= 0 && row < len(plugins) {
				actualIdx := m.getActualIndex(row)
				if actualIdx >= 0 && !m.plugins[actualIdx].IsGroupHeader {
					m.selectedIdx = actualIdx
				}
			}
		}

	case tea.MouseWheelUp:
		m.moveUp()
		m.moveUp()
		m.moveUp()

	case tea.MouseWheelDown:
		m.moveDown()
		m.moveDown()
		m.moveDown()
	}

	return m, nil
}
```

**Step 3: Commit**

```bash
git add cmd/cpm/main.go internal/tui/update.go
git commit -m "feat(tui): add mouse support for selection and scrolling"
```

---

## Task 7: Add Clear All Pending

**Files:**
- Modify: `internal/tui/update.go`

**Step 1: Update Escape handler**

Update the Escape handler in `handleKeyPress` to clear all pending when not on a selected item:

```go
case matchesKey(msg, keys.Escape):
	if m.filterActive {
		// Already handled by filter mode
		return m, nil
	}
	// Clear pending for selected plugin, or all if shift is held
	// Since we can't detect shift easily, clear selected only
	m.clearPending()
```

**Step 2: Add Shift+Escape for clear all (alternative: double-tap Esc)**

For simplicity, we can use a double-tap pattern or a separate key. Let's add a "clear all" with Shift+Esc or Ctrl+Esc:

Actually, let's keep it simple - Esc clears current selection. If user wants to clear all, they can press Esc multiple times or use the quit confirmation.

**Step 3: Commit (if any changes made)**

```bash
git add internal/tui/update.go
git commit -m "feat(tui): clarify escape behavior for clearing pending"
```

---

## Task 8: Update Help Text

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Update renderHelp to include filter**

Update `renderHelp` in `view.go`:

```go
// renderHelp renders the help bar at the bottom.
func (m *Model) renderHelp(styles Styles) string {
	if m.filterActive {
		return styles.Help.Render("Type to filter • Enter: select • Esc: cancel")
	}

	if len(m.pending) > 0 {
		return styles.Help.Render("↑↓: navigate • l/p: local/project • u: uninstall • Tab: toggle • Enter: apply • /: filter • r: refresh • q: quit")
	}
	return styles.Help.Render("↑↓: navigate • l/p: local/project • u: uninstall • Tab: toggle • /: filter • r: refresh • q: quit")
}
```

**Step 2: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): update help text with filter and refresh"
```

---

## Task 9: Integration Test

**Step 1: Build and run**

```bash
mise run build
./cpm
```

**Step 2: Test filter functionality**

1. Press `/` -> Filter input appears
2. Type a plugin name -> List filters in real-time
3. Press Enter -> Filter closes, first match selected
4. Press Esc -> Filter clears

**Step 3: Test refresh**

1. Press `r` -> Shows loading, then refreshes data

**Step 4: Test quit confirmation**

1. Mark a plugin for install
2. Press `q` -> Confirmation modal appears
3. Press Esc -> Returns to main view
4. Press `q` again -> Shows confirmation
5. Press `q` again -> Quits

**Step 5: Test mouse**

1. Click on a plugin in the list -> Selects it
2. Scroll wheel up/down -> Navigates list

**Step 6: Run full test suite**

```bash
mise run ci
```

Expected: All checks pass

---

## Phase 7 Complete

**Verification:**
- `/` activates filter mode
- Typing filters plugins in real-time
- Enter in filter selects first match
- Esc in filter clears and exits filter
- `r` refreshes plugin data
- `q` with pending changes shows confirmation
- `q` in confirmation quits
- Mouse click selects items
- Mouse wheel scrolls list
- Help text updates based on mode

**Files modified:**
- `internal/tui/model.go` (filter and quit state)
- `internal/tui/view.go` (filter input, help text, quit confirmation)
- `internal/tui/update.go` (filter, refresh, mouse, quit handlers)
- `cmd/cpm/main.go` (mouse support enabled)
