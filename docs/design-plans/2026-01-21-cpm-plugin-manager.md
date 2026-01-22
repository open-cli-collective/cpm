# cpm - Claude Plugin Manager Design

## Definition of Done

The primary deliverable is a terminal user interface (TUI) application (`cpm`) that manages Claude Code plugins with clear visibility into installation scopes (local vs project). Success is achieved when users can: view all available and installed plugins organized by marketplace, select plugins for installation/uninstallation at either scope, apply pending changes with progress feedback and error handling, and filter/search the plugin list. The application must be released through automated CI/CD pipelines with binaries distributed via GitHub releases, Homebrew, Chocolatey, and WinGet.

## Summary

cpm (Claude Plugin Manager) is a terminal user interface application that addresses a gap in Claude Code's built-in plugin management by providing clear visibility into plugin installation scopes. While Claude Code can install plugins at both local (user-wide) and project (repository-specific) scopes, its built-in tooling doesn't clearly indicate where plugins are installed or allow easy scope management. cpm solves this by presenting a two-pane TUI where users can see all plugins from configured marketplaces, view their current installation status with scope indicators, and batch operations to install, uninstall, or change scopes.

The implementation uses a flat model architecture built on the Bubble Tea framework, with Lip Gloss for styling and minimal dependencies. The application shells out to the existing `claude` CLI for all operations, parsing its JSON output rather than reimplementing Claude's plugin logic. The project includes comprehensive tooling (mise for tool management, lefthook for pre-commit hooks, golangci-lint for code quality) and a complete CI/CD pipeline using release-please for automated release PR generation and GoReleaser for multi-platform binary distribution.

## Glossary

- **Bubble Tea**: A Go framework for building terminal user interfaces using The Elm Architecture (model-update-view pattern)
- **Lip Gloss**: A Go library for terminal styling, layout, and colors that integrates with Bubble Tea
- **Bubbles**: A collection of pre-built Bubble Tea components (lists, viewports, spinners, etc.)
- **TUI (Terminal User Interface)**: An application that runs in a terminal/console with interactive text-based UI, as opposed to a graphical UI or simple command-line arguments
- **Flat model architecture**: A Bubble Tea pattern where all application state lives in a single Model struct, as opposed to nested models with separate state management for components
- **Mise**: A polyglot tool version manager (successor to asdf) that manages programming language runtimes and CLI tools
- **Lefthook**: A fast Git hooks manager written in Go that runs pre-commit checks like linting and formatting
- **golangci-lint**: A fast Go linters aggregator that runs multiple static analysis tools in parallel
- **gofumpt**: A stricter formatter for Go code (extends gofmt with additional style rules)
- **GoReleaser**: A release automation tool for Go projects that builds cross-platform binaries and creates release artifacts
- **release-please**: A GitHub Action that automates release PR creation and changelog generation based on conventional commits
- **Scope (in Claude context)**: The installation boundary for a plugin—either "local" (user-wide, stored in `~/.claude`) or "project" (repository-specific, stored in `.claude/` directory)
- **Marketplace (in Claude context)**: A source repository of plugins, configured in Claude Code's settings
- **Cobra**: A popular Go CLI framework (noted as explicitly NOT used in this design)
- **Homebrew**: A package manager for macOS and Linux
- **Chocolatey**: A package manager for Windows
- **WinGet**: Microsoft's official package manager for Windows
- **Conventional Commits**: A commit message format specification (type(scope): description) used by release-please for automated versioning

## Architecture

### Overview

cpm is a terminal user interface (TUI) application for managing Claude Code plugins. It provides clear visibility into plugin installation scopes (local vs project) that the built-in Claude Code plugin management lacks.

The application uses a **flat model architecture** with a single Model struct containing all state. This was chosen over nested models because the state is interconnected (selected plugin affects both panes) and the app has medium complexity.

### System Boundary

```
┌─────────────────────────────────────────────────────────────────┐
│                         cpm TUI                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────────┐  │
│  │ Left Pane   │  │ Right Pane   │  │ Modals                │  │
│  │ (List)      │  │ (Details)    │  │ (Progress/Error)      │  │
│  └─────────────┘  └──────────────┘  └───────────────────────┘  │
│                           │                                     │
│                    ┌──────┴──────┐                              │
│                    │ Claude      │                              │
│                    │ Client      │                              │
│                    └──────┬──────┘                              │
└───────────────────────────┼─────────────────────────────────────┘
                            │ exec
                    ┌───────┴───────┐
                    │ claude CLI    │
                    │ (external)    │
                    └───────────────┘
```

### Core Components

**Entry Point** (`cmd/cpm/main.go`):
- Parses `--version` and `--help` flags (no Cobra)
- Checks for `claude` CLI in PATH, exits with error if missing
- Launches Bubble Tea program

**TUI Package** (`internal/tui/`):
- `model.go` - Single flat Model struct with all state
- `view.go` - View rendering for main, progress, and error modes
- `update.go` - Message handling and state transitions
- `keys.go` - Keybinding definitions using Bubbles key package
- `styles.go` - Lip Gloss style definitions

**Claude Client** (`internal/claude/`):
- `types.go` - Marketplace, Plugin, InstalledPlugin, Scope types
- `client.go` - Shells out to `claude plugin` commands, parses JSON output

**Version** (`internal/version/`):
- Build-time injected Version, Commit, BuildDate variables

### Data Flow

1. **Startup**: Check for `claude` CLI → Launch TUI → Fetch plugin data via `claude plugin list --json --available`
2. **Interaction**: User navigates list → Updates selectedIdx → Right pane re-renders with selected plugin details
3. **Selection**: User presses l/p/Tab/u → Updates pending changes map → Visual indicators update
4. **Execution**: User presses Enter → Modal appears → Sequential `claude plugin install/uninstall` calls → Results shown → Data refreshed

### Contracts

**Claude Client Interface:**

```go
type Client interface {
    ListMarketplaces() ([]Marketplace, error)
    ListPlugins(includeAvailable bool) (*PluginList, error)
    InstallPlugin(pluginID string, scope Scope) error
    UninstallPlugin(pluginID string, scope Scope) error
}

type Scope string
const (
    ScopeNone    Scope = ""
    ScopeProject Scope = "project"
    ScopeLocal   Scope = "local"
)

type PluginList struct {
    Installed []InstalledPlugin
    Available []Plugin
}
```

**TUI Model State:**

```go
type Model struct {
    // Data
    marketplaces []Marketplace
    plugins      []PluginState

    // UI state
    selectedIdx  int
    listOffset   int
    width, height int
    filterText   string
    filterActive bool

    // Pending changes (plugin ID -> desired scope, ScopeNone = uninstall)
    pending      map[string]Scope

    // View mode
    mode         Mode  // ModeMain, ModeProgress, ModeError
    progressMsgs []string
    errorMsgs    []string

    // Loading
    loading      bool
    loadErr      error
}

type Mode int
const (
    ModeMain Mode = iota
    ModeProgress
    ModeError
)
```

## Existing Patterns

This is a greenfield project with no existing codebase patterns to follow.

Design patterns are informed by:
- **jira-ticket-cli** (open-cli-collective reference): Project structure, Makefile targets, GitHub Actions workflows, CLAUDE.md format, conventional commits
- **Bubble Tea ecosystem**: Flat model architecture for medium-complexity apps, Lip Gloss for styling, Bubbles components for list and viewport

New patterns established by this design:
- Mise for tool management (enhancement over reference repo)
- Lefthook for pre-commit hooks (reference repo uses CI-only enforcement)
- release-please + GoReleaser (reference repo uses auto-release without release PRs)

## Implementation Phases

### Phase 1: Project Scaffolding

**Goal:** Initialize project with all tooling and configuration files

**Components:**
- `go.mod` with module path `github.com/open-cli-collective/cpm`
- `mise.toml` with Go, golangci-lint, gofumpt, lefthook
- `lefthook.yaml` with pre-commit hooks
- `.golangci.yml` linter configuration
- `.goreleaser.yml` release configuration
- `renovate.json` dependency update configuration
- `Makefile` as thin wrapper for mise tasks
- `CLAUDE.md` project guidance
- `.gitignore` for Go projects
- Directory structure: `cmd/cpm/`, `internal/tui/`, `internal/claude/`, `internal/version/`

**Dependencies:** None (first phase)

**Done when:** `mise install` succeeds, `mise run build` succeeds (with placeholder main.go), `lefthook install` succeeds

### Phase 2: Claude CLI Client

**Goal:** Interface for interacting with Claude Code CLI

**Components:**
- `internal/claude/types.go` - Marketplace, Plugin, InstalledPlugin, Scope, PluginList types matching Claude CLI JSON output
- `internal/claude/client.go` - Client implementation that shells out to `claude plugin` commands and parses JSON responses

**Dependencies:** Phase 1 (project structure)

**Done when:** Client can list marketplaces, list plugins (installed + available), tests pass for JSON parsing

### Phase 3: Core TUI Model & Data Loading

**Goal:** Basic Bubble Tea application that loads and displays plugin data

**Components:**
- `internal/version/version.go` - Version, Commit, BuildDate variables with injection
- `cmd/cpm/main.go` - Entry point with --version/--help flags, claude CLI check, TUI launch
- `internal/tui/model.go` - Model struct, Init (data loading), basic Update (quit handling)
- `internal/tui/view.go` - Placeholder view showing loading state and plugin count

**Dependencies:** Phase 2 (claude client)

**Done when:** Running `cpm` shows loading state, then plugin count, Ctrl+C quits cleanly

### Phase 4: Two-Pane Layout

**Goal:** Split-pane UI with plugin list and details

**Components:**
- `internal/tui/styles.go` - Lip Gloss styles for panes, headers, selection, scope colors
- `internal/tui/view.go` - Two-pane layout (1/3 left, 2/3 right), plugin list rendering with marketplace group headers, detail pane with plugin info
- `internal/tui/keys.go` - Navigation keybindings (j/k, arrows, Home/End, PgUp/PgDn)
- `internal/tui/update.go` - Navigation message handling, terminal resize handling

**Dependencies:** Phase 3 (basic TUI)

**Done when:** Two-pane layout renders correctly, navigation works, group headers are non-selectable, detail pane updates with selection

### Phase 5: Selection & Pending Changes

**Goal:** Plugin selection state management and visual feedback

**Components:**
- `internal/tui/model.go` - pending map, state transition logic
- `internal/tui/update.go` - Keybinding handlers for l/p/Tab/u/Esc
- `internal/tui/view.go` - Visual indicators for install state ([LOCAL], [PROJECT]) and pending changes ([→ LOCAL], [→ UNINSTALL])
- `internal/tui/keys.go` - Selection keybindings

**Dependencies:** Phase 4 (layout)

**Done when:** Can mark plugins for local/project install, can mark installed plugins for uninstall, Tab cycles states correctly, visual indicators show pending changes, Esc clears pending for selected plugin

### Phase 6: Execution Flow

**Goal:** Apply pending changes with progress feedback and error handling

**Components:**
- `internal/tui/model.go` - Mode switching, progress/error message storage
- `internal/tui/update.go` - Enter handler, sequential command execution via tea.Cmd, result collection
- `internal/tui/view.go` - Progress modal (with per-operation status), error summary modal, confirmation before apply

**Dependencies:** Phase 5 (selection)

**Done when:** Enter shows confirmation with pending changes summary, operations execute sequentially, progress modal updates per-operation, errors collected and shown, data refreshes after completion

### Phase 7: UX Enhancements

**Goal:** Polish features for delightful user experience

**Components:**
- `internal/tui/model.go` - filterText, filterActive fields
- `internal/tui/update.go` - Filter input handling (/), refresh (r), clear all pending (Esc in main), quit confirmation, mouse event handling
- `internal/tui/view.go` - Filter input display, filtered list rendering, install count in details
- `internal/tui/keys.go` - Additional keybindings

**Dependencies:** Phase 6 (execution)

**Done when:** Search/filter works with /, refresh with r, quit confirms if pending changes, mouse click selects, scroll wheel navigates, resize reflows gracefully

### Phase 8: CI/CD & Release

**Goal:** Complete build, test, and release automation

**Components:**
- `.github/workflows/ci.yml` - Lint, test, build on push/PR
- `.github/workflows/pr-artifacts.yml` - Build all platforms, upload artifacts on PR
- `.github/workflows/release.yml` - GoReleaser on tag, publish to Homebrew/Chocolatey/WinGet
- `.github/workflows/release-please.yml` - Automated release PR creation
- `packaging/chocolatey/` - Chocolatey package configuration
- `packaging/winget/` - WinGet manifest files

**Dependencies:** Phase 7 (complete application)

**Done when:** CI runs on all PRs, PR artifacts downloadable for testing, release-please creates release PRs, merging release PR triggers full release pipeline

## Additional Considerations

**Error handling:**
- Missing `claude` CLI at startup results in immediate exit with helpful message
- Command failures during execution are collected and shown in summary modal; remaining operations continue
- Data loading failures show error in TUI with retry option (r to refresh)

**Terminal compatibility:**
- Minimum terminal size handling (graceful message if too small)
- Mouse support optional (works without mouse)
- Color fallback for terminals without true color

**Future extensibility:**
- Architecture supports adding CLI subcommands later via Cobra if needed
- Model structure allows adding plugin update functionality
- Client interface allows mocking for testing
