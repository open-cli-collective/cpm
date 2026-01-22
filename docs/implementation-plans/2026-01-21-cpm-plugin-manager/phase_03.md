# cpm - Phase 3: Core TUI Model & Data Loading

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Basic Bubble Tea application that loads and displays plugin data

**Architecture:** Bubble Tea application with flat model architecture. Entry point handles --version/--help flags and claude CLI check. Model fetches plugin data on Init and displays loading state.

**Tech Stack:** Bubble Tea (github.com/charmbracelet/bubbletea)

**Scope:** Phase 3 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 2 complete with claude client in place

---

## Task 1: Create Version Package

**Files:**
- Create: `internal/version/version.go`
- Delete: `internal/version/.gitkeep`

**Step 1: Write the test file**

Create file `internal/version/version_test.go`:

```go
package version

import "testing"

func TestString(t *testing.T) {
	// Test default values
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if Date == "" {
		Date = "unknown"
	}

	s := String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// Should contain version
	if !contains(s, Version) {
		t.Errorf("String() = %q, should contain version %q", s, Version)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/version/...`
Expected: FAIL - package not defined

**Step 3: Write the implementation**

Create file `internal/version/version.go`:

```go
// Package version provides build-time version information.
package version

import "fmt"

// These variables are set at build time via ldflags.
var (
	// Version is the semantic version (e.g., "1.0.0").
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "unknown"
	// Date is the build date in RFC3339 format.
	Date = "unknown"
)

// String returns a formatted version string.
func String() string {
	return fmt.Sprintf("cpm %s (commit: %s, built: %s)", Version, Commit, Date)
}
```

**Step 4: Remove .gitkeep**

```bash
rm internal/version/.gitkeep
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/version/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/version/version.go internal/version/version_test.go
git rm internal/version/.gitkeep
git commit -m "feat(version): add version package with build-time injection"
```

---

## Task 2: Add Bubble Tea Dependencies

**Files:**
- Modify: `go.mod`
- Create: `go.sum`

**Step 1: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go mod tidy
```

**Step 2: Verify operationally**

Run: `go mod verify`
Expected: "all modules verified"

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add bubble tea dependencies"
```

---

## Task 3: Create TUI Model

**Files:**
- Create: `internal/tui/model.go`
- Delete: `internal/tui/.gitkeep`

**Step 1: Write the test file**

Create file `internal/tui/model_test.go`:

```go
package tui

import (
	"testing"

	"github.com/open-cli-collective/cpm/internal/claude"
)

func TestNewModel(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)

	if m.client == nil {
		t.Error("client is nil")
	}
	if !m.loading {
		t.Error("loading should be true initially")
	}
	if m.err != nil {
		t.Error("err should be nil initially")
	}
}

func TestPluginStateFromInstalled(t *testing.T) {
	installed := claude.InstalledPlugin{
		ID:      "test@marketplace",
		Version: "1.0.0",
		Scope:   claude.ScopeProject,
		Enabled: true,
	}

	state := PluginStateFromInstalled(installed)

	if state.ID != "test@marketplace" {
		t.Errorf("ID = %q, want %q", state.ID, "test@marketplace")
	}
	if state.InstalledScope != claude.ScopeProject {
		t.Errorf("InstalledScope = %q, want %q", state.InstalledScope, claude.ScopeProject)
	}
	if !state.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestPluginStateFromAvailable(t *testing.T) {
	available := claude.AvailablePlugin{
		PluginID:        "test@marketplace",
		Name:            "test",
		Description:     "A test plugin",
		MarketplaceName: "marketplace",
	}

	state := PluginStateFromAvailable(available)

	if state.ID != "test@marketplace" {
		t.Errorf("ID = %q, want %q", state.ID, "test@marketplace")
	}
	if state.Name != "test" {
		t.Errorf("Name = %q, want %q", state.Name, "test")
	}
	if state.InstalledScope != claude.ScopeNone {
		t.Errorf("InstalledScope = %q, want empty", state.InstalledScope)
	}
}

// mockClient implements claude.Client for testing
type mockClient struct {
	plugins *claude.PluginList
	err     error
}

func (m *mockClient) ListPlugins(includeAvailable bool) (*claude.PluginList, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.plugins != nil {
		return m.plugins, nil
	}
	return &claude.PluginList{}, nil
}

func (m *mockClient) InstallPlugin(pluginID string, scope claude.Scope) error {
	return m.err
}

func (m *mockClient) UninstallPlugin(pluginID string, scope claude.Scope) error {
	return m.err
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/tui/...`
Expected: FAIL - Model not defined

**Step 3: Write the implementation**

Create file `internal/tui/model.go`:

```go
// Package tui implements the terminal user interface using Bubble Tea.
package tui

import (
	"sort"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)

// Mode represents the current UI mode.
type Mode int

const (
	// ModeMain is the default two-pane view.
	ModeMain Mode = iota
	// ModeProgress shows operation progress.
	ModeProgress
	// ModeError shows error summary.
	ModeError
)

// PluginState holds the display state for a plugin.
type PluginState struct {
	ID             string
	Name           string
	Description    string
	Marketplace    string
	Version        string
	InstalledScope claude.Scope
	Enabled        bool
	IsGroupHeader  bool // True for marketplace group headers (non-selectable)
}

// PluginStateFromInstalled creates a PluginState from an installed plugin.
func PluginStateFromInstalled(p claude.InstalledPlugin) PluginState {
	// Parse name and marketplace from ID (format: name@marketplace)
	name, marketplace := parsePluginID(p.ID)
	return PluginState{
		ID:             p.ID,
		Name:           name,
		Marketplace:    marketplace,
		Version:        p.Version,
		InstalledScope: p.Scope,
		Enabled:        p.Enabled,
	}
}

// PluginStateFromAvailable creates a PluginState from an available plugin.
func PluginStateFromAvailable(p claude.AvailablePlugin) PluginState {
	return PluginState{
		ID:             p.PluginID,
		Name:           p.Name,
		Description:    p.Description,
		Marketplace:    p.MarketplaceName,
		Version:        p.Version,
		InstalledScope: claude.ScopeNone,
	}
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

// Model is the main application model.
type Model struct {
	// Client for Claude CLI operations
	client claude.Client

	// Plugin data
	plugins []PluginState

	// UI state
	selectedIdx int
	width       int
	height      int

	// Pending changes (plugin ID -> desired scope)
	pending map[string]claude.Scope

	// View mode
	mode         Mode
	progressMsgs []string
	errorMsgs    []string

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new Model with the given client.
func NewModel(client claude.Client) *Model {
	return &Model{
		client:  client,
		pending: make(map[string]claude.Scope),
		loading: true,
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

// loadPlugins fetches plugin data from the Claude CLI.
func (m *Model) loadPlugins() tea.Msg {
	list, err := m.client.ListPlugins(true)
	if err != nil {
		return pluginsErrorMsg{err: err}
	}

	plugins := mergePlugins(list)
	return pluginsLoadedMsg{plugins: plugins}
}

// mergePlugins combines installed and available plugins, grouped by marketplace.
func mergePlugins(list *claude.PluginList) []PluginState {
	// Build map of installed plugins by ID
	installedByID := make(map[string]claude.InstalledPlugin)
	for _, p := range list.Installed {
		installedByID[p.ID] = p
	}

	// Group by marketplace
	byMarketplace := make(map[string][]PluginState)

	// Add available plugins (which includes installed ones)
	for _, p := range list.Available {
		state := PluginStateFromAvailable(p)

		// Check if installed
		if installed, ok := installedByID[p.PluginID]; ok {
			state.InstalledScope = installed.Scope
			state.Enabled = installed.Enabled
			state.Version = installed.Version
		}

		byMarketplace[state.Marketplace] = append(byMarketplace[state.Marketplace], state)
	}

	// Sort marketplace names for deterministic ordering
	marketplaces := make([]string, 0, len(byMarketplace))
	for marketplace := range byMarketplace {
		marketplaces = append(marketplaces, marketplace)
	}
	sort.Strings(marketplaces)

	// Flatten with group headers in sorted order
	var result []PluginState
	for _, marketplace := range marketplaces {
		plugins := byMarketplace[marketplace]
		// Sort plugins within marketplace by name for deterministic ordering
		sort.Slice(plugins, func(i, j int) bool {
			return plugins[i].Name < plugins[j].Name
		})
		// Add group header
		result = append(result, PluginState{
			Name:          marketplace,
			IsGroupHeader: true,
			Marketplace:   marketplace,
		})
		result = append(result, plugins...)
	}

	return result
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

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

	case pluginsErrorMsg:
		m.loading = false
		m.err = msg.err
	}

	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.loading {
		return "Loading plugins..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	return m.renderMain()
}

// renderMain renders the main view (placeholder for now).
func (m *Model) renderMain() string {
	if len(m.plugins) == 0 {
		return "No plugins found.\n\nPress q to quit."
	}

	// Count non-header plugins
	count := 0
	for _, p := range m.plugins {
		if !p.IsGroupHeader {
			count++
		}
	}

	return "cpm - Claude Plugin Manager\n\n" +
		"Found " + strconv.Itoa(count) + " plugins.\n\n" +
		"Press q to quit."
}
```

**Step 4: Remove .gitkeep**

```bash
rm internal/tui/.gitkeep
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/tui/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/tui/model.go internal/tui/model_test.go
git rm internal/tui/.gitkeep
git commit -m "feat(tui): add core model with plugin loading"
```

---

## Task 4: Update Main Entry Point

**Files:**
- Modify: `cmd/cpm/main.go`

**Step 1: Read current file**

The current main.go is a placeholder that just prints "cpm - Claude Plugin Manager".

**Step 2: Write the updated implementation**

Replace file `cmd/cpm/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
	"github.com/open-cli-collective/cpm/internal/tui"
	"github.com/open-cli-collective/cpm/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Handle --version and --help
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version.String())
			return nil
		case "--help", "-h":
			printUsage()
			return nil
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}

	// Check for claude CLI
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Please install Claude Code first")
	}

	// Create client and model
	client := claude.NewClient()
	model := tui.NewModel(client)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func printUsage() {
	fmt.Println("cpm - Claude Plugin Manager")
	fmt.Println()
	fmt.Println("A TUI for managing Claude Code plugins with clear scope visibility.")
	fmt.Println()
	fmt.Println("Usage: cpm [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help     Show this help message")
	fmt.Println("  -v, --version  Show version information")
}
```

**Step 3: Verify operationally**

Run: `go build -o cpm ./cmd/cpm && ./cpm --version`
Expected: Prints version string like "cpm dev (commit: unknown, built: unknown)"

Run: `go build -o cpm ./cmd/cpm && ./cpm --help`
Expected: Prints usage information

**Step 4: Commit**

```bash
git add cmd/cpm/main.go
git commit -m "feat(main): add entry point with version/help flags and claude check"
```

---

## Task 5: Integration Test

**Step 1: Build and run**

```bash
mise run build
./cpm
```

Expected:
- Shows "Loading plugins..." briefly
- Then shows "cpm - Claude Plugin Manager" with plugin count
- Press q to quit cleanly

**Step 2: Test error case (if possible)**

If you temporarily rename or hide the claude binary:
```bash
PATH=/usr/bin ./cpm
```
Expected: Error message about claude CLI not found

**Step 3: Clean up**

```bash
rm -f cpm
```

---

## Phase 3 Complete

**Verification:**
- `mise run build` succeeds
- `./cpm --version` shows version info
- `./cpm --help` shows usage
- `./cpm` shows loading state, then plugin count
- Ctrl+C or q quits cleanly

**Files created/modified:**
- `internal/version/version.go`
- `internal/version/version_test.go`
- `internal/tui/model.go`
- `internal/tui/model_test.go`
- `cmd/cpm/main.go` (updated)
- `go.mod` (updated with dependencies)
- `go.sum` (created)

**Functionality:**
- --version flag prints version info
- --help flag prints usage
- Checks for claude CLI in PATH
- Loads plugin data via claude client
- Displays loading state and plugin count
- Quit with q or Ctrl+C
