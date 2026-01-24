# TUI Package

Last verified: 2026-01-24

## Purpose

Implements the two-pane plugin manager interface using Bubble Tea's Elm Architecture (Model-Update-View).

## Contracts

- **Exposes**: `NewModel(client, workingDir) -> *Model`, implements `tea.Model` interface
- **Guarantees**: Pending operations (install/uninstall/enable/disable) tracked until explicit Apply. Filter preserves selection when possible. Only project/local plugins for current workingDir are shown.
- **Expects**: Valid `claude.Client` implementation. Terminal with reasonable size (handles resize).

## Dependencies

- **Uses**: internal/claude (Client interface), Bubble Tea, Lip Gloss
- **Used by**: cmd/cpm (creates Model, runs tea.Program)
- **Boundary**: No direct CLI calls; all plugin operations go through Client

## Key Decisions

- Flat model: Single `Model` struct contains all state (no nested sub-models)
- Pending operations map: `map[string]Operation` tracks pending operations per plugin
- Operation type enum: `OpInstall`, `OpUninstall`, `OpEnable`, `OpDisable` define operation types
- Mode enum: `ModeMain`, `ModeProgress`, `ModeSummary` control view rendering
- Group headers: Non-selectable `PluginState` entries with `IsGroupHeader=true`

## Invariants

- Selection cursor never lands on group headers (auto-skips)
- Pending operations show visual indicator in plugin list (e.g., `[-> LOCAL]`, `[-> ENABLED]`)
- Apply shows confirmation modal before executing
- Quit with pending changes shows confirmation modal

## Key Files

- `model.go` - Model struct, Init, message types, plugin state
- `update.go` - Update logic for all modes and inputs
- `view.go` - View rendering for all modes and modals
- `styles.go` - Lip Gloss style definitions
- `keys.go` - Key binding definitions

## Gotchas

- Filter indices: `filteredIdx` maps visible index to `plugins` slice index
- Operation order: Uninstalls execute first, then installs, then enables, then disables
- Enable/disable blocked when install/uninstall pending for same plugin (prevents conflicts)
