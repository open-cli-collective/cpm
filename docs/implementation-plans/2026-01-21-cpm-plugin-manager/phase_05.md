# cpm - Phase 5: Selection & Pending Changes

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Plugin selection state management and visual feedback

**Architecture:** Pending changes stored in map[string]Scope. Key handlers for l/p/Tab/u/Esc modify pending map. Visual indicators show current state and pending changes.

**Tech Stack:** Bubble Tea, Lip Gloss

**Scope:** Phase 5 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 4 complete with two-pane layout and navigation

---

## Task 1: Add Selection Key Handlers

**Files:**
- Modify: `internal/tui/update.go`

**Step 1: Update update.go with selection handlers**

Add the following handlers to the `handleKeyPress` function in `internal/tui/update.go`:

```go
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
```

**Step 2: Add import for claude package**

Ensure the import section includes:
```go
import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)
```

**Step 3: Run tests**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/tui/update.go
git commit -m "feat(tui): add selection key handlers for l/p/Tab/u/Esc"
```

---

## Task 2: Add Selection Tests

**Files:**
- Modify: `internal/tui/model_test.go`

**Step 1: Add tests for selection behavior**

Add to `internal/tui/model_test.go`:

```go
func TestSelectForInstallLocal(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeLocal)

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeLocal {
		t.Errorf("pending = %v, want local scope", m.pending)
	}
}

func TestSelectForInstallProject(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeProject)

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeProject {
		t.Errorf("pending = %v, want project scope", m.pending)
	}
}

func TestSelectForInstallClearsIfSameScope(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0
	m.pending["test@marketplace"] = claude.ScopeProject

	// Selecting local should clear pending since it's the same as installed
	m.selectForInstall(claude.ScopeLocal)

	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("pending should be cleared when selecting same scope as installed")
	}
}

func TestSelectForUninstall(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	m.selectForUninstall()

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeNone {
		t.Errorf("pending = %v, want ScopeNone for uninstall", m.pending)
	}
}

func TestSelectForUninstallToggle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	// First uninstall
	m.selectForUninstall()
	if _, ok := m.pending["test@marketplace"]; !ok {
		t.Error("first uninstall should mark pending")
	}

	// Toggle back
	m.selectForUninstall()
	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("second uninstall should clear pending")
	}
}

func TestClearPending(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0
	m.pending["test@marketplace"] = claude.ScopeLocal

	m.clearPending()

	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("clearPending should remove pending change")
	}
}

func TestToggleScopeCycle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	// None -> Local
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeLocal {
		t.Errorf("after first toggle = %v, want local", scope)
	}

	// Local -> Project
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeProject {
		t.Errorf("after second toggle = %v, want project", scope)
	}

	// Project -> None (not installed, clears pending)
	m.toggleScope()
	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("after third toggle, pending should be cleared")
	}
}

func TestToggleScopeInstalledPlugin(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	// Local installed -> Project pending
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeProject {
		t.Errorf("after first toggle = %v, want project", scope)
	}

	// Project pending -> Uninstall pending
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeNone {
		t.Errorf("after second toggle = %v, want ScopeNone (uninstall)", scope)
	}
}

func TestSkipsGroupHeaders(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{Name: "marketplace", IsGroupHeader: true},
	}
	m.selectedIdx = 0

	// Should not panic or add to pending
	m.selectForInstall(claude.ScopeLocal)
	m.selectForUninstall()
	m.toggleScope()

	if len(m.pending) != 0 {
		t.Error("operations on group headers should not modify pending")
	}
}
```

**Step 2: Run tests**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/model_test.go
git commit -m "test(tui): add tests for selection behavior"
```

---

## Task 3: Verify Visual Indicators

The visual indicators are already implemented in `view.go` (Task 3 of Phase 4). Let's verify they work correctly.

**Step 1: Build and test visually**

```bash
mise run build
./cpm
```

**Step 2: Test the following scenarios**

1. Select an uninstalled plugin, press `l` -> Should show `[→ LOCAL]`
2. Press `p` -> Should show `[→ PROJECT]`
3. Press `Esc` -> Should clear the indicator
4. Select an installed plugin (e.g., with `[LOCAL]`), press `u` -> Should show `[→ UNINSTALL]`
5. Press `Tab` repeatedly -> Should cycle through states
6. Navigate to a different plugin -> Previous selection should retain its pending state

**Step 3: Verify help text updates**

- When no pending changes: Help shows basic navigation
- When pending changes exist: Help shows "Enter: apply"

**Step 4: Run all tests**

```bash
mise run test
mise run lint
```

Expected: All tests pass, no lint errors

---

## Phase 5 Complete

**Verification:**
- Press `l` on uninstalled plugin -> shows `[→ LOCAL]`
- Press `p` on uninstalled plugin -> shows `[→ PROJECT]`
- Press `u` on installed plugin -> shows `[→ UNINSTALL]`
- Press `Tab` cycles through states correctly
- Press `Esc` clears pending for selected plugin
- Visual indicators show in list view
- Detail pane shows pending status
- Help bar updates when changes are pending

**Files modified:**
- `internal/tui/update.go` (added selection handlers)
- `internal/tui/model_test.go` (added selection tests)

**State transitions:**
- Uninstalled: none → local → project → none
- Installed at local: local → project → uninstall → local
- Installed at project: project → uninstall → project
