# Claude CLI Client

Last verified: 2026-01-22

## Purpose

Provides a typed Go interface to the Claude Code CLI, isolating shell-out complexity from the TUI layer.

## Contracts

- **Exposes**: `Client` interface with `ListPlugins`, `InstallPlugin`, `UninstallPlugin`
- **Guarantees**: Returns structured `PluginList` with typed `InstalledPlugin`/`AvailablePlugin`. Errors include stderr output.
- **Expects**: `claude` binary in PATH (or custom path via `NewClientWithPath`)

## Dependencies

- **Uses**: os/exec (shells out to `claude` binary)
- **Used by**: internal/tui (consumes Client interface)
- **Boundary**: No TUI dependencies; this is a pure data layer

## Key Decisions

- Interface-based design: Enables mock clients for testing TUI without real CLI
- Shell out vs. library: Claude Code has no Go SDK; CLI is the only integration point
- JSON parsing: Uses `claude plugin list --json --available` for structured data

## Invariants

- `Scope` is always one of: `""`, `"user"`, `"project"`, `"local"`
- Plugin IDs follow `name@marketplace` format
- All errors include stderr context

## Key Files

- `types.go` - Scope enum, InstalledPlugin, AvailablePlugin, PluginList
- `client.go` - Client interface and realClient implementation
