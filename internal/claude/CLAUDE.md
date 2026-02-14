# Claude CLI Client

Last verified: 2026-02-14

## Purpose

Provides a typed Go interface to the Claude Code CLI, plugin manifest reading, and project settings, isolating shell-out complexity from the TUI layer.

## Contracts

- **Exposes**: `Client` interface with `ListPlugins`, `InstallPlugin`, `UninstallPlugin`, `EnablePlugin`, `DisablePlugin`
- **Exposes**: `ReadPluginManifest(installPath)` - reads plugin.json for metadata
- **Exposes**: `ScanPluginComponents(installPath)` - discovers skills, agents, commands, hooks, MCPs
- **Exposes**: `ResolveMarketplaceSourcePath(marketplace, source)` - resolves local marketplace paths
- **Exposes**: `GetAllEnabledPlugins(workingDir)` - reads all three settings files (user, project, local) to get multi-scope plugin data
- **Exposes**: `ReadProjectSettings(settingsPath)` - reads a single settings file
- **Guarantees**: Returns structured `PluginList` with typed `InstalledPlugin`/`AvailablePlugin`. Errors include stderr output.
- **Expects**: `claude` binary in PATH (or custom path via `NewClientWithPath`)

## Dependencies

- **Uses**: os/exec (shells out to `claude` binary), os (file reading)
- **Used by**: internal/tui (consumes Client interface and manifest functions)
- **Boundary**: No TUI dependencies; this is a pure data layer

## Key Decisions

- Interface-based design: Enables mock clients for testing TUI without real CLI
- Shell out vs. library: Claude Code has no Go SDK; CLI is the only integration point
- JSON parsing: Uses `claude plugin list --json --available` for structured data
- Manifest parsing: Handles flexible author field (string or object format)
- Install/Uninstall use install/uninstall: `InstallPlugin` calls `claude plugin install`, `UninstallPlugin` calls `claude plugin uninstall`. `EnablePlugin`/`DisablePlugin` use `enable`/`disable` for toggling state of already-installed plugins.
- Multi-scope detection: All three settings files (user, project, local) are read to build a complete `ScopeState` map. This enables later phases to render and manage plugins across multiple scopes from a single data structure.

## Invariants

- `Scope` is always one of: `""`, `"user"`, `"project"`, `"local"`
- Plugin IDs follow `name@marketplace` format
- All CLI errors include stderr context
- Manifest functions return nil/empty on missing files (not errors for optional data)

## Key Files

- `types.go` - Scope enum, InstalledPlugin, AvailablePlugin, PluginList
- `client.go` - Client interface and realClient implementation
- `manifest.go` - PluginManifest, PluginComponents, ProjectSettings, manifest/settings reading and component scanning
