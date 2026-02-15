# Human Test Plan: Plugin Scope Management

## Prerequisites
- Build the application: `mise run build`
- All automated tests passing: `mise run test`
- A working directory that is a Git repository with Claude Code configured
- At least one plugin installed at different scopes (user, project, local)
- Terminal with at least 80 columns width

## Phase 1: Scope Display Verification

| Step | Action | Expected |
|------|--------|----------|
| 1.1 | Launch `./cpm` in a project directory | Application starts, plugin list loads without errors |
| 1.2 | Locate a plugin installed at a single scope | Scope indicator shows as `[USER]`, `[PROJECT]`, or `[LOCAL]` in uppercase |
| 1.3 | Locate a plugin installed at multiple scopes (if available) | Scope indicator shows combined format like `[USER, LOCAL]` with scopes in order: user, project, local |
| 1.4 | Resize terminal to narrow width (~60 columns) | Left pane adjusts width but remains readable; scope indicators do not overflow |

## Phase 2: Single-Scope Key Behavior

| Step | Action | Expected |
|------|--------|----------|
| 2.1 | Select a single-scope installed plugin, press `l` | Pending indicator shows migration arrow (e.g., `[PROJECT -> LOCAL]`) |
| 2.2 | Press `l` again on the same plugin | Pending operation clears (toggle behavior) |
| 2.3 | Select a single-scope installed plugin, press `p` | Pending indicator shows migration to project scope |
| 2.4 | Select a single-scope installed plugin, press `u` | Pending indicator shows uninstall |
| 2.5 | Select a single-scope installed plugin, press `e` | Pending indicator shows enable/disable toggle |
| 2.6 | Select a single-scope installed plugin, press `Tab` | Scope cycles through available options |

## Phase 3: Multi-Scope Key Behavior

| Step | Action | Expected |
|------|--------|----------|
| 3.1 | Select a multi-scope installed plugin, press `l` | Scope dialog opens (not a direct operation) |
| 3.2 | Press `Esc` in the scope dialog | Dialog closes, no pending operation created |
| 3.3 | Select the same multi-scope plugin, press `u` | Scope dialog opens |
| 3.4 | Press `Esc` in the scope dialog | Dialog closes, no changes |
| 3.5 | Select the same multi-scope plugin, press `e` | Scope dialog opens |
| 3.6 | Select the same multi-scope plugin, press `Tab` | Nothing happens (no-op for multi-scope) |

## Phase 4: Scope Dialog Interaction

| Step | Action | Expected |
|------|--------|----------|
| 4.1 | Select an installed plugin, press `S` | Scope dialog opens with checkboxes. Installed scopes pre-checked. File paths shown: `~/.claude/settings.json` (User), `.claude/settings.json` (Project), `.claude/settings.local.json` (Local) |
| 4.2 | Press `Up`/`Down` arrows | Cursor moves between User, Project, Local rows |
| 4.3 | Press `Space` on an unchecked scope | Checkbox toggles to checked `[x]` |
| 4.4 | Press `Space` on a checked scope | Checkbox toggles to unchecked `[ ]` |
| 4.5 | Check a new scope, press `Enter` | Dialog closes, pending indicator shows scope transition |
| 4.6 | Uncheck an installed scope and check a different one, press `Enter` | Pending indicator shows scope change transition |
| 4.7 | Open dialog again, restore to original state, press `Enter` | Pending operation clears (no net change) |

## Phase 5: Apply Operations

| Step | Action | Expected |
|------|--------|----------|
| 5.1 | Create one or more pending operations | Pending indicators visible in plugin list |
| 5.2 | Press `a` (Apply) | Confirmation modal shows all pending operations with scope details |
| 5.3 | Press `Enter` to confirm | Progress view shows operations executing in order: uninstalls, migrates, installs, enables, disables |
| 5.4 | Wait for completion | Summary screen shows success/failure for each operation |
| 5.5 | Press any key to return | Plugin list refreshes with updated scope information |

## End-to-End: Install Plugin to Multiple Scopes

1. Launch `./cpm`, select an available (not installed) plugin
2. Press `S` to open the scope dialog
3. Check both "User" and "Local" checkboxes
4. Press `Enter` -- pending indicator should show `[-> USER, LOCAL]`
5. Press `a` to apply, confirm with `Enter`
6. Verify progress shows two install operations
7. After completion, plugin should show `[USER, LOCAL]` scope indicator
8. Verify `~/.claude/settings.json` and `.claude/settings.local.json` both contain the plugin

## End-to-End: Migrate Plugin Between Scopes

1. Start with a plugin installed at `[LOCAL]` scope
2. Press `S` to open the scope dialog
3. Uncheck "Local", check "Project"
4. Press `Enter` -- pending indicator should show `[LOCAL -> PROJECT]`
5. Press `a` to apply, confirm
6. Verify progress shows uninstall from local, then install to project
7. After completion, plugin should show `[PROJECT]` scope indicator

## Traceability

| AC | Automated Test | Manual Step |
|----|----------------|-------------|
| AC1.1 | TestExecuteOperationInstall | 5.3 |
| AC1.2 | TestExecuteOperationUninstallUsesOriginalScope | 5.3 |
| AC1.3 | TestExecuteOperationEnable, TestExecuteOperationDisable | 2.5, 5.3 |
| AC2.1 | TestGetAllEnabledPluginsUserScope | E2E Install step 8 |
| AC2.2 | TestGetAllEnabledPluginsProjectAndLocalScope | E2E Install step 8 |
| AC2.3 | TestGetAllEnabledPluginsMissingFiles | N/A (automated) |
| AC2.4 | TestGetAllEnabledPluginsMultipleScopes | 1.3 |
| AC3.1 | TestPluginStateFromInstalled, TestPluginStateFromAvailable | 1.2, 1.3 |
| AC3.2 | TestPluginStateHelpers | N/A (automated) |
| AC3.3 | TestStartExecutionBuildsOperations | 5.3 |
| AC4.1 | TestFormatScopeSet_* | 1.2, 1.3 |
| AC4.2 | TestRenderPendingIndicator_* | 4.5, 4.6 |
| AC4.3 | TestWithDimensions_MinimumLeftPaneWidth | 1.4 |
| AC5.1 | TestSelectForInstall*ScopeCreatesOp/OpensDialog | 2.1, 3.1 |
| AC5.2 | TestSelectForUninstall*ScopeCreatesOp/OpensDialog | 2.4, 3.3 |
| AC5.3 | TestToggleEnablement*ScopeCreatesOp/OpensDialog | 2.5, 3.5 |
| AC5.4 | TestToggleScopeMultiScopeIsNoOp | 3.6 |
| AC6.1 | TestOpenScopeDialogForSelected* | 4.1 |
| AC6.2 | TestUpdateScopeDialog* | 4.2-4.5 |
| AC6.3 | TestApplyScopeDialogDelta* | 4.5-4.7 |
| AC6.4 | TestRenderScopeDialog | 4.1 |
| AC7.1 | TestExecuteOperationInstallWhen*InSettings | 5.3 |
| AC7.2 | TestExecuteOperationUninstallWhenInSettings | 5.3 |
| AC7.3 | TestExecuteOperationMultiScopeStopsOnFirstError | N/A (automated) |
| AC7.4 | TestStartExecutionOperationOrdering | 5.3 |
