# cpm - Phase 6: Execution Flow

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Apply pending changes with progress feedback and error handling

**Architecture:** Enter key triggers confirmation modal. Upon confirmation, operations execute sequentially via tea.Cmd. Progress modal shows per-operation status. Errors are collected and shown in summary. Data refreshes after completion.

**Tech Stack:** Bubble Tea (async commands)

**Scope:** Phase 6 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 5 complete with selection and pending changes

---

## Task 1: Add Execution Messages

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Add message types for execution flow**

Add to `internal/tui/model.go` after the existing message types:

```go
// Operation represents a pending change to execute.
type Operation struct {
	PluginID string
	Scope    claude.Scope
	IsInstall bool // true for install, false for uninstall
}

// confirmationMsg is sent to confirm pending changes.
type confirmationMsg struct{}

// operationStartMsg is sent when an operation starts.
type operationStartMsg struct {
	op Operation
}

// operationDoneMsg is sent when an operation completes.
type operationDoneMsg struct {
	op  Operation
	err error
}

// executionCompleteMsg is sent when all operations are done.
type executionCompleteMsg struct {
	errors []string
}
```

**Step 2: Add fields to Model**

Add these fields to the Model struct in `internal/tui/model.go`:

```go
// Model is the main application model.
type Model struct {
	// ... existing fields ...

	// Execution state
	operations     []Operation
	currentOpIdx   int
	operationErrors []string
	showConfirm    bool
}
```

**Step 3: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat(tui): add execution message types and state fields"
```

---

## Task 2: Add Confirmation Modal View

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Add confirmation modal rendering**

Add to `internal/tui/view.go`:

```go
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
```

**Step 2: Update View function**

Update the `View` function to show confirmation when active:

```go
// View implements tea.Model.
func (m *Model) View() string {
	if m.loading {
		return "Loading plugins..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	if m.showConfirm {
		return m.renderConfirmation(m.styles)
	}

	switch m.mode {
	case ModeMain:
		return m.renderMainView()
	case ModeProgress:
		return m.renderProgress(m.styles)
	case ModeError:
		return m.renderErrorSummary(m.styles)
	}

	return ""
}
```

**Step 3: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add confirmation modal view"
```

---

## Task 3: Add Progress Modal View

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Add progress modal rendering**

Add to `internal/tui/view.go`:

```go
// renderProgress renders the progress modal.
func (m *Model) renderProgress(styles Styles) string {
	var lines []string
	lines = append(lines, styles.Header.Render(" Applying Changes "))
	lines = append(lines, "")

	for i, op := range m.operations {
		var status string
		if i < m.currentOpIdx {
			// Completed
			if i < len(m.operationErrors) && m.operationErrors[i] != "" {
				status = "✗ Failed: " + m.operationErrors[i]
			} else {
				status = "✓ Done"
			}
		} else if i == m.currentOpIdx {
			// In progress
			status = "⟳ Running..."
		} else {
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
```

**Step 2: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add progress modal view"
```

---

## Task 4: Add Error Summary View

**Files:**
- Modify: `internal/tui/view.go`

**Step 1: Add error summary rendering**

Add to `internal/tui/view.go`:

```go
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
```

**Step 2: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat(tui): add error summary modal view"
```

---

## Task 5: Add Execution Update Logic

**Files:**
- Modify: `internal/tui/update.go`

**Step 1: Add Enter key handler and execution logic**

Add to `internal/tui/update.go`:

```go
// In handleKeyPress, add the Enter handler:
case matchesKey(msg, keys.Enter):
	if len(m.pending) > 0 {
		m.showConfirm = true
	}
```

**Step 2: Add confirmation mode handling**

Add a new function to handle confirmation mode:

```go
// updateConfirmation handles messages in confirmation mode.
func (m *Model) updateConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case matchesKey(msg, m.keys.Enter):
			// Start execution
			m.showConfirm = false
			return m.startExecution()
		case matchesKey(msg, m.keys.Escape), matchesKey(msg, m.keys.Quit):
			// Cancel
			m.showConfirm = false
		}
	}
	return m, nil
}

// startExecution begins executing pending operations.
func (m *Model) startExecution() (tea.Model, tea.Cmd) {
	// Build operation list
	m.operations = nil
	for pluginID, scope := range m.pending {
		isInstall := scope != claude.ScopeNone
		m.operations = append(m.operations, Operation{
			PluginID:  pluginID,
			Scope:     scope,
			IsInstall: isInstall,
		})
	}

	m.currentOpIdx = 0
	m.operationErrors = make([]string, len(m.operations))
	m.mode = ModeProgress

	// Start first operation
	if len(m.operations) > 0 {
		return m, m.executeOperation(m.operations[0])
	}

	return m, nil
}

// executeOperation returns a command that executes a single operation.
func (m *Model) executeOperation(op Operation) tea.Cmd {
	return func() tea.Msg {
		var err error
		if op.IsInstall {
			err = m.client.InstallPlugin(op.PluginID, op.Scope)
		} else {
			err = m.client.UninstallPlugin(op.PluginID, op.Scope)
		}

		return operationDoneMsg{op: op, err: err}
	}
}
```

**Step 3: Add progress mode handling**

Add a new function to handle progress mode:

```go
// updateProgress handles messages in progress mode.
func (m *Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case operationDoneMsg:
		// Record result
		if msg.err != nil {
			m.operationErrors[m.currentOpIdx] = msg.err.Error()
		}

		m.currentOpIdx++

		// Execute next operation or finish
		if m.currentOpIdx < len(m.operations) {
			return m, m.executeOperation(m.operations[m.currentOpIdx])
		}

		// All done - refresh and show summary
		m.mode = ModeError
		m.pending = make(map[string]claude.Scope)
		// Clear filter state to avoid stale filter on refreshed data
		m.filterActive = false
		m.filterText = ""
		m.filteredIdx = nil
		return m, m.loadPlugins
	}
	return m, nil
}
```

**Step 4: Add error mode handling**

Add a new function to handle error mode:

```go
// updateError handles messages in error mode.
func (m *Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case matchesKey(msg, m.keys.Enter), matchesKey(msg, m.keys.Escape):
			m.mode = ModeMain
			m.operations = nil
			m.operationErrors = nil
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
```

**Step 5: Update main Update function**

Update the `Update` function in `model.go` to route to mode-specific handlers:

```go
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
	case ModeError:
		return m.updateError(msg)
	}

	return m, nil
}
```

**Step 6: Run tests**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/tui/update.go internal/tui/model.go
git commit -m "feat(tui): add execution flow with progress and error handling"
```

---

## Task 6: Integration Test

**Step 1: Build and run**

```bash
mise run build
./cpm
```

**Step 2: Test the full flow**

1. Select an uninstalled plugin
2. Press `l` to mark for local install
3. Press Enter -> Confirmation modal appears
4. Press Enter again -> Progress modal shows
5. Wait for completion -> Error summary shows
6. Press Enter -> Returns to main view with refreshed data

**Step 3: Test error handling**

1. Mark a plugin for installation
2. If an error occurs, it should be shown in the summary
3. Successful operations should complete even if others fail

**Step 4: Run full test suite**

```bash
mise run ci
```

Expected: All checks pass

---

## Phase 6 Complete

**Verification:**
- Enter with pending changes shows confirmation modal
- Confirmation lists all pending operations
- Enter in confirmation starts execution
- Esc in confirmation cancels
- Progress modal shows per-operation status
- Operations execute sequentially
- Errors are collected and shown
- Error summary shows success/failure counts
- Enter or Esc dismisses error summary
- Plugin list refreshes after execution
- Pending changes cleared after execution

**Files modified:**
- `internal/tui/model.go` (added execution state and messages)
- `internal/tui/view.go` (added confirmation, progress, error modals)
- `internal/tui/update.go` (added execution logic and mode handlers)
