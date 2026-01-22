# cpm - Phase 2: Claude CLI Client

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Interface for interacting with Claude Code CLI

**Architecture:** Client package that shells out to `claude plugin` commands and parses JSON responses. Uses Go's exec package for command execution and encoding/json for parsing.

**Tech Stack:** Go standard library (os/exec, encoding/json)

**Scope:** Phase 2 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 1 complete with directory structure in place

**Claude CLI verified:** JSON output structure confirmed with `claude plugin list --json --available`

---

## Task 1: Create Type Definitions

**Files:**
- Create: `internal/claude/types.go`
- Delete: `internal/claude/.gitkeep`

**Step 1: Write the test file**

Create file `internal/claude/types_test.go`:

```go
package claude

import (
	"encoding/json"
	"testing"
)

func TestScopeConstants(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeNone, ""},
		{ScopeUser, "user"},
		{ScopeProject, "project"},
		{ScopeLocal, "local"},
	}

	for _, tt := range tests {
		if string(tt.scope) != tt.want {
			t.Errorf("Scope %v = %q, want %q", tt.scope, tt.scope, tt.want)
		}
	}
}

func TestInstalledPluginJSON(t *testing.T) {
	jsonData := `{
		"id": "context7@claude-plugins-official",
		"version": "e30768372b41",
		"scope": "user",
		"enabled": true,
		"installPath": "/Users/test/.claude/plugins/cache/claude-plugins-official/context7/e30768372b41",
		"installedAt": "2026-01-16T02:46:08.054Z",
		"lastUpdated": "2026-01-22T01:25:12.553Z"
	}`

	var plugin InstalledPlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.ID != "context7@claude-plugins-official" {
		t.Errorf("ID = %q, want %q", plugin.ID, "context7@claude-plugins-official")
	}
	if plugin.Scope != ScopeUser {
		t.Errorf("Scope = %q, want %q", plugin.Scope, ScopeUser)
	}
	if !plugin.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestInstalledPluginWithProjectPath(t *testing.T) {
	jsonData := `{
		"id": "ed3d-basic-agents@ed3d-plugins",
		"version": "1.0.0",
		"scope": "project",
		"enabled": false,
		"installPath": "/Users/test/.claude/plugins/cache/ed3d-plugins/ed3d-basic-agents/1.0.0",
		"installedAt": "2026-01-16T02:46:08.684Z",
		"lastUpdated": "2026-01-16T02:46:08.684Z",
		"projectPath": "/Users/test/Code/myproject"
	}`

	var plugin InstalledPlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.Scope != ScopeProject {
		t.Errorf("Scope = %q, want %q", plugin.Scope, ScopeProject)
	}
	if plugin.ProjectPath != "/Users/test/Code/myproject" {
		t.Errorf("ProjectPath = %q, want %q", plugin.ProjectPath, "/Users/test/Code/myproject")
	}
}

func TestAvailablePluginJSON(t *testing.T) {
	jsonData := `{
		"pluginId": "github@claude-plugins-official",
		"name": "github",
		"description": "Official GitHub MCP server for repository management.",
		"marketplaceName": "claude-plugins-official",
		"source": "./external_plugins/github",
		"installCount": 47711
	}`

	var plugin AvailablePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.PluginID != "github@claude-plugins-official" {
		t.Errorf("PluginID = %q, want %q", plugin.PluginID, "github@claude-plugins-official")
	}
	if plugin.Name != "github" {
		t.Errorf("Name = %q, want %q", plugin.Name, "github")
	}
	if plugin.MarketplaceName != "claude-plugins-official" {
		t.Errorf("MarketplaceName = %q, want %q", plugin.MarketplaceName, "claude-plugins-official")
	}
	// Source field should be captured (can be string or object in CLI output)
	if plugin.Source == nil {
		t.Error("Source should not be nil")
	}
}

func TestPluginListJSON(t *testing.T) {
	jsonData := `{
		"installed": [
			{
				"id": "context7@claude-plugins-official",
				"version": "e30768372b41",
				"scope": "user",
				"enabled": true,
				"installPath": "/test/path",
				"installedAt": "2026-01-16T02:46:08.054Z",
				"lastUpdated": "2026-01-22T01:25:12.553Z"
			}
		],
		"available": [
			{
				"pluginId": "github@claude-plugins-official",
				"name": "github",
				"description": "GitHub integration",
				"marketplaceName": "claude-plugins-official",
				"source": "./external_plugins/github",
				"installCount": 47711
			}
		]
	}`

	var list PluginList
	if err := json.Unmarshal([]byte(jsonData), &list); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(list.Installed) != 1 {
		t.Errorf("len(Installed) = %d, want 1", len(list.Installed))
	}
	if len(list.Available) != 1 {
		t.Errorf("len(Available) = %d, want 1", len(list.Available))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/claude/...`
Expected: FAIL - types not defined

**Step 3: Write the implementation**

Create file `internal/claude/types.go`:

```go
// Package claude provides a client for interacting with the Claude Code CLI.
package claude

// Scope represents the installation scope of a plugin.
type Scope string

const (
	// ScopeNone indicates no scope (used for uninstall operations).
	ScopeNone Scope = ""
	// ScopeUser is the global user scope (~/.claude/settings.json).
	ScopeUser Scope = "user"
	// ScopeProject is the shared project scope (.claude/settings.json).
	ScopeProject Scope = "project"
	// ScopeLocal is the local project scope (.claude/settings.local.json).
	ScopeLocal Scope = "local"
)

// InstalledPlugin represents a plugin that is currently installed.
type InstalledPlugin struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Scope       Scope  `json:"scope"`
	Enabled     bool   `json:"enabled"`
	InstallPath string `json:"installPath"`
	InstalledAt string `json:"installedAt"`
	LastUpdated string `json:"lastUpdated"`
	ProjectPath string `json:"projectPath,omitempty"`
}

// AvailablePlugin represents a plugin available from a marketplace.
type AvailablePlugin struct {
	PluginID        string `json:"pluginId"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	MarketplaceName string `json:"marketplaceName"`
	Version         string `json:"version,omitempty"`
	InstallCount    int    `json:"installCount,omitempty"`
	Source          any    `json:"source,omitempty"` // Can be string or object depending on plugin type
}

// PluginList is the response from `claude plugin list --json --available`.
type PluginList struct {
	Installed []InstalledPlugin `json:"installed"`
	Available []AvailablePlugin `json:"available"`
}
```

**Step 4: Remove .gitkeep**

```bash
rm internal/claude/.gitkeep
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/claude/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/claude/types.go internal/claude/types_test.go
git rm internal/claude/.gitkeep
git commit -m "feat(claude): add type definitions for plugin data structures"
```

---

## Task 2: Create Client Interface and Constructor

**Files:**
- Create: `internal/claude/client.go`

**Step 1: Write the test file**

Create file `internal/claude/client_test.go`:

```go
package claude

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Verify it's a *realClient
	_, ok := client.(*realClient)
	if !ok {
		t.Error("NewClient() did not return *realClient")
	}
}

func TestNewClientWithPath(t *testing.T) {
	client := NewClientWithPath("/usr/local/bin/claude")
	if client == nil {
		t.Fatal("NewClientWithPath() returned nil")
	}

	rc, ok := client.(*realClient)
	if !ok {
		t.Fatal("NewClientWithPath() did not return *realClient")
	}
	if rc.claudePath != "/usr/local/bin/claude" {
		t.Errorf("claudePath = %q, want %q", rc.claudePath, "/usr/local/bin/claude")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/claude/...`
Expected: FAIL - Client interface and NewClient not defined

**Step 3: Write the implementation**

Create file `internal/claude/client.go`:

```go
package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Client defines the interface for interacting with the Claude CLI.
type Client interface {
	// ListPlugins returns installed and optionally available plugins.
	ListPlugins(includeAvailable bool) (*PluginList, error)

	// InstallPlugin installs a plugin at the specified scope.
	InstallPlugin(pluginID string, scope Scope) error

	// UninstallPlugin removes a plugin from the specified scope.
	UninstallPlugin(pluginID string, scope Scope) error
}

// realClient implements Client by shelling out to the claude CLI.
type realClient struct {
	claudePath string
}

// NewClient creates a new Client using "claude" from PATH.
func NewClient() Client {
	return &realClient{claudePath: "claude"}
}

// NewClientWithPath creates a new Client using the specified claude binary path.
func NewClientWithPath(path string) Client {
	return &realClient{claudePath: path}
}

// ListPlugins implements Client.ListPlugins.
func (c *realClient) ListPlugins(includeAvailable bool) (*PluginList, error) {
	args := []string{"plugin", "list", "--json"}
	if includeAvailable {
		args = append(args, "--available")
	}

	cmd := exec.Command(c.claudePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude plugin list failed: %w: %s", err, stderr.String())
	}

	var list PluginList
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("failed to parse plugin list: %w", err)
	}

	return &list, nil
}

// InstallPlugin implements Client.InstallPlugin.
func (c *realClient) InstallPlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "install", pluginID}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}

	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin install failed: %w: %s", err, stderr.String())
	}

	return nil
}

// UninstallPlugin implements Client.UninstallPlugin.
func (c *realClient) UninstallPlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "uninstall", pluginID}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}

	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin uninstall failed: %w: %s", err, stderr.String())
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/claude/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/claude/client.go internal/claude/client_test.go
git commit -m "feat(claude): add client interface and implementation"
```

---

## Task 3: Add Integration Test (Optional, Manual Verification)

Since the client shells out to the actual `claude` CLI, we should verify it works with a manual integration test.

**Step 1: Create a simple integration test script**

This is for manual verification, not automated tests:

```bash
# Run from project root
go run ./cmd/cpm

# If claude is installed, you can test the client:
# go test -v -run TestIntegration ./internal/claude/... -integration
```

For now, we'll rely on the JSON parsing tests and manual verification.

**Step 2: Run the full test suite**

Run: `mise run test`
Expected: All tests pass

**Step 3: Run linting**

Run: `mise run lint`
Expected: No errors

---

## Phase 2 Complete

**Verification:**
- `go test -v ./internal/claude/...` passes
- `mise run lint` passes
- Types correctly parse actual Claude CLI JSON output

**Files created:**
- `internal/claude/types.go`
- `internal/claude/types_test.go`
- `internal/claude/client.go`
- `internal/claude/client_test.go`

**Types defined:**
- `Scope` (string enum: "", "user", "project", "local")
- `InstalledPlugin` (parsed from claude plugin list output)
- `AvailablePlugin` (parsed from claude plugin list --available output)
- `PluginList` (container for installed + available)
- `Client` interface with `ListPlugins`, `InstallPlugin`, `UninstallPlugin`
