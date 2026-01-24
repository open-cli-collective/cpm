# Plugin Enable/Disable Toggle Design

## Summary

This design extends the Claude Plugin Manager TUI to support toggling plugins between enabled and disabled states. Users will press the 'e' key on any installed plugin to queue an enable or disable operation, which executes when they apply changes (similar to the existing install/uninstall workflow). The feature follows the established pending changes pattern: operations are added to a queue, displayed with visual indicators (`[→ ENABLED]`, `[→ DISABLED]`), and executed sequentially when the user confirms.

The implementation refactors the existing pending operations architecture from separate tracking structures to a unified map (`pendingOps map[string]Operation`) keyed by plugin ID. This creates natural mutual exclusion where each plugin can have exactly one pending operation at any time. A new `OperationType` enum (OpInstall, OpUninstall, OpEnable, OpDisable) replaces the previous boolean-based approach. Enable/disable operations use the plugin's current installation scope to determine which scope to pass to the underlying `claude plugin enable/disable` CLI commands, maintaining scope-aware behavior consistent with how the application handles install/uninstall operations.

## Definition of Done

1. **Core functionality**: Users can press the 'e' key on any installed plugin to toggle its enabled/disabled state, which adds the operation to the pending changes queue
2. **UI feedback**: The plugin list and detail pane clearly show pending enable/disable operations using visual indicators consistent with the existing install/uninstall pattern
3. **Apply workflow**: Pending enable/disable operations execute sequentially alongside install/uninstall operations when the user applies changes, with full progress tracking and error handling
4. **Constraints**: Enable/disable operations are blocked on plugins that have pending install/uninstall operations to prevent conflicting states

## Glossary

- **TUI (Text User Interface)**: Terminal-based graphical interface built with the Bubble Tea framework, providing interactive visual components in the terminal
- **Bubble Tea**: Go framework implementing the Elm Architecture (Model-Update-View) pattern for building terminal user interfaces
- **Pending changes pattern**: Design pattern where user actions queue operations that execute in batch when confirmed, rather than executing immediately
- **Operation**: A queued action (install, uninstall, enable, disable) that will execute when the user applies changes
- **Scope**: Installation context for a plugin - either USER (installed for current user) or LOCAL (installed for current project directory)
- **InstalledScope**: The scope where a plugin is currently installed, used to determine which scope to target for enable/disable operations
- **Client interface**: Abstraction layer that encapsulates all Claude CLI interactions, enabling testing through mock implementations
- **mockClient**: Test implementation of the Client interface that uses callback functions instead of shelling out to the real Claude CLI
- **Mutual exclusion**: Constraint ensuring each plugin has at most one pending operation, preventing conflicting states like "install AND uninstall"
- **Sequential execution**: Pattern where operations run one at a time in defined order (uninstalls → installs → enable/disable), with each operation triggering the next via message passing
- **Command pattern**: Bubble Tea's execution model where long-running operations return command functions that run asynchronously and send messages back to the Update function

## Architecture

The enable/disable feature extends the existing pending changes architecture using a unified operation model. All four operation types (install, uninstall, enable, disable) flow through a single pending operations queue that executes sequentially when the user applies changes.

**Key architectural decisions:**

1. **Unified pending operations map**: `pendingOps map[string]Operation` (keyed by pluginID) replaces the previous separate maps. Each plugin can have exactly one pending operation at a time, creating natural mutual exclusion.

2. **Operation type enum**: A new `OperationType` enum (OpInstall, OpUninstall, OpEnable, OpDisable) replaces the previous boolean-based approach, making operation types explicit and extensible.

3. **Scope-aware execution**: Enable/disable operations use the plugin's current `InstalledScope` to determine which scope to pass to the Claude CLI commands (`claude plugin enable/disable --scope <scope> <pluginID>`).

4. **Sequential execution order**: Operations execute in phases - uninstalls first, then installs, then enable/disable - ensuring plugins exist before their state is toggled.

**Component interactions:**

- User presses 'e' → TUI validates (installed? no pending conflict?) → adds Operation to `pendingOps` map → updates display indicators
- User presses Enter → confirmation modal shows all pending operations → user confirms → operations execute sequentially via Client interface → progress modal tracks status → summary shows results
- Client shells out to Claude CLI for each operation → captures errors → returns results to TUI

## Existing Patterns

Investigation of the existing codebase (`internal/tui/model.go`, `internal/tui/update.go`, `internal/claude/client.go`) revealed a well-established pending changes pattern for install/uninstall operations. This design follows that pattern exactly:

**Patterns followed:**

- **Pending changes map**: Operations tracked in Model map, displayed with visual indicators (`[→ LOCAL]`, `[→ UNINSTALL]`), applied on user confirmation
- **Operation struct**: Encapsulates operation type, plugin ID, scope information - used to build execution queue
- **Sequential execution**: Operations run one at a time via `executeOperation()` command pattern, with `operationDoneMsg` triggering next operation
- **Three-mode workflow**: ModeMain (selection) → ModeProgress (execution) → ModeSummary (results)
- **Error isolation**: Failed operations don't abort the batch; errors captured in array and displayed in summary
- **Client interface abstraction**: All CLI interaction through interface methods, enabling mockClient for testing

**Pattern evolution:**

The previous implementation used `pending map[string]claude.Scope` which mapped plugin ID to desired scope. This design replaces it with `pendingOps map[string]Operation` for several reasons:
- Natural mutual exclusion (one operation per plugin)
- Cleaner handling of multiple operation types
- No need to distinguish between pending install/uninstall and pending enable/disable in separate maps
- Operations are already built when applying (no conversion step)

This evolution maintains the same user experience while simplifying the internal model.

## Implementation Phases

### Phase 1: Core Types and Client Interface

**Goal:** Add enable/disable operation types and Client interface methods

**Components:**
- `OperationType` enum in `internal/tui/model.go` - defines OpInstall, OpUninstall, OpEnable, OpDisable
- Updated `Operation` struct in `internal/tui/model.go` - replaces `IsInstall bool` with `Type OperationType`
- `EnablePlugin(pluginID, scope)` method in `internal/claude/client.go` - shells out to `claude plugin enable`
- `DisablePlugin(pluginID, scope)` method in `internal/claude/client.go` - shells out to `claude plugin disable`
- Extended `mockClient` in `internal/tui/model_test.go` - adds `enableFn` and `disableFn` callbacks

**Dependencies:** None (extends existing types)

**Done when:** Code compiles, Client interface includes new methods, mockClient implements them

### Phase 2: Pending Operations Map Migration

**Goal:** Replace `pending map[string]Scope` with `pendingOps map[string]Operation`

**Components:**
- `pendingOps map[string]Operation` field in `Model` struct (`internal/tui/model.go`)
- Updated `selectForInstall()` in `internal/tui/model.go` - creates OpInstall/OpUninstall operations
- Updated `selectForUninstall()` in `internal/tui/model.go` - creates OpUninstall operations
- Updated `toggleScope()` in `internal/tui/model.go` - creates operations instead of scopes
- Updated `clearPending()` in `internal/tui/model.go` - works with pendingOps map

**Dependencies:** Phase 1 (OperationType exists)

**Done when:** All existing install/uninstall functionality works with new pendingOps map, existing tests pass with updated assertions

### Phase 3: Toggle Enablement Logic

**Goal:** Add 'e' key handling to toggle enabled/disabled state

**Components:**
- `toggleEnablement()` method in `internal/tui/model.go` - handles 'e' key press logic (validate installed, check conflicts, create OpEnable/OpDisable)
- Updated `KeyBindings` struct in `internal/tui/keys.go` - adds `Enable []string` field with `["e"]`
- Updated `DefaultKeyBindings()` in `internal/tui/keys.go` - initializes Enable binding
- Updated `updateMain()` in `internal/tui/update.go` - calls `toggleEnablement()` when 'e' pressed

**Dependencies:** Phase 2 (pendingOps map exists)

**Done when:** Pressing 'e' adds pending enable/disable operation, pressing 'e' again cancels it, tests verify toggle behavior and mutual exclusion

### Phase 4: Display Indicators

**Goal:** Show pending enable/disable operations in UI

**Components:**
- Updated `getScopeIndicator()` in `internal/tui/view.go` - switches on `Operation.Type` to show `[→ ENABLED]`/`[→ DISABLED]`
- Updated `appendPendingChange()` in `internal/tui/view.go` - displays pending enable/disable text in detail pane
- Updated `renderConfirmation()` in `internal/tui/view.go` - groups enable/disable operations in confirmation modal

**Dependencies:** Phase 3 (toggle logic exists)

**Done when:** Pending enable/disable operations visible in list indicator, detail pane, and confirmation modal using existing Pending style

### Phase 5: Operation Execution

**Goal:** Execute enable/disable operations through Client interface

**Components:**
- Updated `executeOperation()` in `internal/tui/update.go` - adds OpEnable and OpDisable cases to switch statement
- Updated `startExecution()` in `internal/tui/update.go` - builds operations from pendingOps map, sorts by type (uninstall, install, enable/disable)
- Updated `renderProgress()` in `internal/tui/view.go` - displays enable/disable operations in progress modal
- Updated `renderErrorSummary()` in `internal/tui/view.go` - shows enable/disable operations in summary

**Dependencies:** Phase 4 (UI displays operations), Phase 1 (Client methods exist)

**Done when:** Enable/disable operations execute sequentially after install/uninstall, progress tracks correctly, errors captured and displayed in summary

### Phase 6: Testing

**Goal:** Comprehensive test coverage for enable/disable feature

**Components:**
- `TestToggleEnablement` in `internal/tui/model_test.go` - verifies 'e' adds pending, pressing again cancels
- `TestToggleEnablementBlockedByPendingInstall` in `internal/tui/model_test.go` - mutual exclusion test
- `TestToggleEnablementNotInstalled` in `internal/tui/model_test.go` - blocks on non-installed plugins
- `TestExecuteOperationEnable` in `internal/tui/model_test.go` - verifies EnablePlugin called with correct params
- `TestExecuteOperationDisable` in `internal/tui/model_test.go` - verifies DisablePlugin called with correct params
- `TestOperationOrderingWithEnableDisable` in `internal/tui/model_test.go` - verifies uninstall → install → enable/disable order
- Updated existing tests to use OperationType enum instead of IsInstall bool

**Dependencies:** Phase 5 (full implementation complete)

**Done when:** All new tests pass, existing test suite passes with updated assertions, code coverage maintained

## Additional Considerations

**Error handling:** Enable/disable CLI errors are captured identically to install/uninstall errors. The Claude CLI may succeed silently when enabling an already-enabled plugin or disabling an already-disabled plugin - this is acceptable, as the post-apply plugin list reload will reflect the true state.

**Operation ordering:** The three-phase execution order (uninstalls → installs → enable/disable) ensures operations execute in dependency order. Enable/disable operations only apply to installed plugins, so they execute after any pending installs complete.

**State consistency:** The pendingOps map enforces one-operation-per-plugin invariant at the model level. The user cannot create conflicting operations (e.g., "install to local AND disable") without first applying changes. This simplifies the model and prevents impossible states.
