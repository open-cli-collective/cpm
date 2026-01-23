# CPM Architecture

Last updated: 2026-01-23

This document describes the architecture of cpm (Claude Plugin Manager), a TUI application for managing Claude Code plugins.

## Package Overview

```mermaid
graph TB
    subgraph "cmd/cpm"
        main[main.go<br/>Entry point]
    end

    subgraph "internal/claude"
        client[Client<br/>interface]
        types[Types<br/>Scope, Plugin structs]
        manifest[Manifest<br/>Plugin metadata]
    end

    subgraph "internal/tui"
        model[Model<br/>tea.Model impl]
        update[Update<br/>Event handlers]
        view[View<br/>Rendering]
        styles[Styles<br/>Lip Gloss]
        keys[KeyBindings<br/>Input mapping]
    end

    subgraph "internal/version"
        version[Version<br/>Build metadata]
    end

    subgraph "External"
        claude_cli[Claude CLI<br/>claude plugin ...]
        bubbletea[Bubble Tea<br/>TUI framework]
    end

    main --> client
    main --> model
    main --> version
    model --> client
    model --> styles
    model --> keys
    model --> update
    model --> view
    client --> claude_cli
    model --> bubbletea
    client --> types
    client --> manifest
```

## Package Details

### cmd/cpm

Entry point that:
- Parses command-line arguments (`--version`, `--help`)
- Creates a `claude.Client` instance
- Creates a `tui.Model` with the client and working directory
- Runs the Bubble Tea program

### internal/claude

Claude CLI wrapper providing:

```mermaid
classDiagram
    class Client {
        <<interface>>
        +ListPlugins(includeAvailable bool) PluginList, error
        +InstallPlugin(pluginID string, scope Scope) error
        +UninstallPlugin(pluginID string, scope Scope) error
    }

    class realClient {
        -claudePath string
    }

    class Scope {
        <<type string>>
        ScopeNone
        ScopeUser
        ScopeProject
        ScopeLocal
    }

    class InstalledPlugin {
        +ID string
        +Version string
        +InstallPath string
        +InstalledAt string
        +LastUpdated string
        +ProjectPath string
        +Scope Scope
        +Enabled bool
    }

    class AvailablePlugin {
        +PluginID string
        +Name string
        +Description string
        +MarketplaceName string
        +Source any
        +Version string
        +InstallCount int
    }

    class PluginList {
        +Installed []InstalledPlugin
        +Available []AvailablePlugin
    }

    class PluginManifest {
        +Name string
        +Description string
        +Version string
        +AuthorName string
        +AuthorEmail string
    }

    class PluginComponents {
        +Skills []string
        +Agents []string
        +Commands []string
        +Hooks []string
        +MCPs []string
    }

    Client <|.. realClient
    PluginList o-- InstalledPlugin
    PluginList o-- AvailablePlugin
    InstalledPlugin --> Scope
```

**Key functions:**
- `ReadPluginManifest(installPath)` - Reads plugin.json for metadata
- `ScanPluginComponents(installPath)` - Scans directories for skills, agents, etc.
- `ResolveMarketplaceSourcePath(marketplace, source)` - Resolves marketplace plugin paths

### internal/tui

Bubble Tea TUI implementation using the Elm Architecture:

```mermaid
classDiagram
    class Model {
        -client claude.Client
        -workingDir string
        -plugins []PluginState
        -pending map~string~Scope
        -operations []Operation
        -operationErrors []string
        -filteredIdx []int
        -filterText string
        -styles Styles
        -keys KeyBindings
        -selectedIdx int
        -listOffset int
        -width int
        -height int
        -mode Mode
        -currentOpIdx int
        -loading bool
        -showConfirm bool
        -filterActive bool
        -showQuitConfirm bool
        -mouseEnabled bool
        -err error
        +Init() tea.Cmd
        +Update(tea.Msg) tea.Model, tea.Cmd
        +View() string
    }

    class PluginState {
        +ID string
        +Name string
        +Description string
        +AuthorName string
        +AuthorEmail string
        +Marketplace string
        +Version string
        +InstallPath string
        +ExternalURL string
        +InstalledScope Scope
        +Components *PluginComponents
        +Enabled bool
        +IsGroupHeader bool
        +IsExternal bool
    }

    class Operation {
        +PluginID string
        +Scope Scope
        +OriginalScope Scope
        +IsInstall bool
    }

    class Mode {
        <<enum>>
        ModeMain
        ModeProgress
        ModeSummary
    }

    class Styles {
        +LeftPane Style
        +RightPane Style
        +Selected Style
        +Normal Style
        +GroupHeader Style
        +ScopeLocal Style
        +ScopeProject Style
        +ScopeUser Style
        +Pending Style
        +DetailTitle Style
        +DetailLabel Style
        +DetailValue Style
        +ComponentCategory Style
        +ComponentItem Style
        +Help Style
        +WithDimensions(w, h int) Styles
    }

    class KeyBindings {
        +Up []string
        +Down []string
        +PageUp []string
        +PageDown []string
        +Home []string
        +End []string
        +Enter []string
        +Quit []string
        +Local []string
        +Project []string
        +Toggle []string
        +Uninstall []string
        +Escape []string
        +Filter []string
        +Refresh []string
        +Mouse []string
    }

    Model --> PluginState
    Model --> Operation
    Model --> Mode
    Model --> Styles
    Model --> KeyBindings
```

**Messages:**
- `pluginsLoadedMsg` - Plugins loaded from CLI
- `pluginsErrorMsg` - Error loading plugins
- `operationDoneMsg` - Install/uninstall completed

### internal/version

Build-time metadata set via ldflags:

```mermaid
classDiagram
    class version {
        +Version string
        +Commit string
        +Date string
        +Branch string
        +String() string
    }
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI as tui.Model
    participant Client as claude.Client
    participant CLI as Claude CLI

    User->>TUI: Starts cpm
    TUI->>Client: ListPlugins(true)
    Client->>CLI: claude plugin list --json --available
    CLI-->>Client: JSON response
    Client-->>TUI: PluginList
    TUI->>TUI: mergePlugins() → []PluginState

    loop User Interaction
        User->>TUI: Key press (l/p/u)
        TUI->>TUI: Update pending map
        TUI->>TUI: Re-render View
    end

    User->>TUI: Press Enter (confirm)
    TUI->>TUI: Build []Operation

    loop Each Operation
        TUI->>Client: InstallPlugin/UninstallPlugin
        Client->>CLI: claude plugin install/uninstall
        CLI-->>Client: Result
        Client-->>TUI: operationDoneMsg
    end

    TUI->>Client: ListPlugins(true)
    Client-->>TUI: Updated PluginList
    TUI->>TUI: Refresh display
```

## UI States

```mermaid
stateDiagram-v2
    [*] --> Loading: Start
    Loading --> ModeMain: Plugins loaded
    Loading --> Error: Load failed

    ModeMain --> ModeMain: Navigate/Select
    ModeMain --> FilterActive: Press /
    FilterActive --> ModeMain: Esc/Enter
    ModeMain --> ShowConfirm: Enter (with pending)
    ShowConfirm --> ModeMain: Esc
    ShowConfirm --> ModeProgress: Enter
    ModeProgress --> ModeSummary: All ops done
    ModeSummary --> ModeMain: Enter
    ModeMain --> ShowQuitConfirm: q (with pending)
    ShowQuitConfirm --> ModeMain: n
    ShowQuitConfirm --> [*]: y
    ModeMain --> [*]: q (no pending)
```

## File Structure

```
cpm/
├── cmd/cpm/
│   └── main.go              # Entry point
├── internal/
│   ├── claude/
│   │   ├── client.go        # CLI wrapper
│   │   ├── manifest.go      # Manifest reading
│   │   └── types.go         # Data structures
│   ├── tui/
│   │   ├── model.go         # Model + state
│   │   ├── update.go        # Event handlers
│   │   ├── view.go          # Rendering
│   │   ├── styles.go        # Lip Gloss styles
│   │   └── keys.go          # Key bindings
│   └── version/
│       └── version.go       # Build metadata
└── docs/
    └── architecture.md      # This file
```
