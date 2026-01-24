# Sidecar Sync and Restore Design

## Summary

This design adds persistent plugin state tracking to `cpm` through automatically-maintained sidecar files that mirror Claude's scope structure (user, project, and local). When you install or remove plugins using `cpm`, the corresponding sidecar file is updated to reflect the current state. This enables two key capabilities: detecting when plugins have been removed or added outside of `cpm` (showing what changed), and restoring plugin configurations across machines or after a fresh install by reading from a sidecar file.

The implementation migrates from the current single-binary TUI to a Cobra-based CLI with subcommands (`cpm sync`, `cpm restore`, `cpm init`). The sync operation compares Claude's actual plugin state to the sidecar and updates it, while the restore operation provides an interactive TUI for selectively adding marketplaces and installing plugins from a sidecar file. Optional integration with Claude's SessionStart hook allows automatic background synchronization whenever a Claude session starts, keeping sidecars up-to-date without manual intervention.

## Definition of Done

- [ ] Sidecars auto-maintain plugin state per scope (`~/.claude/cpm.json`, `.claude/cpm.json`, `.claude/cpm.local.json`)
- [ ] `cpm sync` command syncs sidecars with Claude's actual state (detailed output)
- [ ] `cpm sync --quiet` provides minimal output for hook usage
- [ ] `cpm restore <file>` launches interactive restore TUI
- [ ] Restore TUI allows selective marketplace addition and plugin installation
- [ ] Version mismatches are detected and reported after restore
- [ ] `cpm init` installs Claude SessionStart hook for automatic sync
- [ ] First-run and post-restore prompts offer hook installation
- [ ] CLI uses Cobra for subcommand structure

## Glossary

- **Sidecar file**: A JSON file stored alongside Claude's configuration that mirrors plugin state, enabling state tracking and portability. Named `cpm.json` or `cpm.local.json` and stored in `.claude/` directories.
- **Scope**: Claude's three-tier configuration hierarchy—User (global `~/.claude/`), Project (committed `.claude/`), and Local (gitignored `.claude/`). Each scope can have independent plugin configurations.
- **Marketplace**: A third-party registry of Claude plugins, typically a GitHub repository. Users add marketplaces to access plugins beyond Claude's built-in catalog.
- **Cobra**: A Go library for building CLI applications with subcommands (like `git commit`, `docker run`). Provides flag parsing, help generation, and command structure.
- **Bubble Tea**: A Go framework for building terminal user interfaces using the Elm Architecture (Model-Update-View pattern). Already used by the existing `cpm` TUI.
- **SessionStart hook**: A Claude feature that runs commands automatically when a Claude session begins. Used here to trigger `cpm sync --quiet` for background synchronization.
- **TUI**: Terminal User Interface—an interactive text-based interface in the terminal (as opposed to a GUI or plain CLI).
- **Version mismatch**: When a plugin's current installed version differs from the version recorded in the sidecar. Advisory-only since Claude CLI doesn't support version-specific installs.

## Architecture

### Core Concept

Auto-maintained sidecar files that track plugin state alongside Claude's settings, with optional Claude hook integration for background synchronization.

### CLI Structure

Migrate from direct TUI launch to Cobra-based CLI with subcommands:

```
cpm                           # Default: main TUI (current behavior)
cpm restore <file>            # Restore TUI from sidecar/export file
cpm sync [--quiet]            # Force sync sidecars
cpm init [--remove|--status]  # Manage Claude SessionStart hook
cpm version                   # Version info
```

### Sidecar Files

Sidecars parallel Claude's scope structure:

| Scope | Path | Git Committed |
|-------|------|---------------|
| User | `~/.claude/cpm.json` | N/A |
| Project | `.claude/cpm.json` | Yes |
| Local | `.claude/cpm.local.json` | No |

### Sidecar Format

```json
{
  "version": 1,
  "lastSynced": "2026-01-23T10:30:00Z",
  "hookInstalled": true,
  "hookPromptDismissed": false,
  "marketplaces": [
    {
      "name": "ed3d-plugins",
      "source": "github",
      "repo": "ed3dai/ed3d-plugins"
    }
  ],
  "plugins": [
    {
      "id": "some-plugin@ed3d-plugins",
      "version": "abc123f",
      "enabled": true,
      "discoveredAt": "2026-01-20T08:00:00Z"
    }
  ]
}
```

Key fields:
- `version`: Format version for future compatibility
- `lastSynced`: Timestamp of last sync operation
- `hookInstalled`/`hookPromptDismissed`: Track hook installation state to avoid repeated prompts
- `marketplaces`: Full marketplace definitions (name, source type, repo/URL) for portability
- `plugins`: Plugin state including version at discovery time

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        cpm startup                              │
├─────────────────────────────────────────────────────────────────┤
│ 1. Read Claude state via CLI                                    │
│    - claude plugin list --json --available                      │
│    - claude plugin marketplace list --json                      │
│                                                                 │
│ 2. Read existing sidecars (if any)                              │
│    - ~/.claude/cpm.json                                         │
│    - .claude/cpm.json                                           │
│    - .claude/cpm.local.json                                     │
│                                                                 │
│ 3. Merge: compare Claude state to sidecar                       │
│    - Add plugins in Claude but not in sidecar (discovered)      │
│    - Flag plugins in sidecar but not in Claude (removed)        │
│                                                                 │
│ 4. Write updated sidecars                                       │
│                                                                 │
│ 5. Check for uninstalled plugins in sidecar                     │
│    - If found: prompt to enter restore mode                     │
│    - If not: continue to main TUI                               │
└─────────────────────────────────────────────────────────────────┘
```

### Restore Flow

Two entry points:
1. Explicit: `cpm restore <file>` launches restore TUI directly
2. Auto-detect: On startup, if sidecar contains plugins not currently installed, prompt user

Restore TUI stages:
1. Display marketplaces to add (checkboxes, all selected by default)
2. Display plugins to install (checkboxes, all selected by default)
3. Show already-installed plugins (skipped)
4. Execute: add marketplaces, install plugins
5. Compare installed versions to sidecar versions, warn on mismatch

### Hook Integration

Claude SessionStart hook enables automatic sync without running cpm directly.

Hook configuration (added to `~/.claude/settings.json`):
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cpm sync --quiet"
          }
        ]
      }
    ]
  }
}
```

`cpm init` manages this hook:
- `cpm init`: Install hook (merge with existing hooks, don't clobber)
- `cpm init --remove`: Remove hook
- `cpm init --status`: Check if hook is installed

### First-Run and Post-Restore Prompts

When no sidecar exists (first run) or after a successful restore, prompt user to install the hook if not already installed:

```
Would you like to install automatic sync?

This adds a Claude hook that keeps your plugin
configuration synced whenever you start a session.

[y] Yes, install hook
[n] No, I'll sync manually
[?] Learn more
```

Track state in user-scope sidecar:
- `hookInstalled: true` - don't prompt
- `hookPromptDismissed: true` - user declined, don't prompt again

## Existing Patterns

Investigation found the following relevant patterns in the codebase:

**CLI entry point:** Current `cmd/cpm/main.go` calls TUI directly. This will be refactored to use Cobra with the default command launching the TUI.

**Client interface:** `internal/claude/client.go` defines the Client interface for Claude CLI operations. New operations needed:
- `ListMarketplaces() ([]Marketplace, error)` - wraps `claude plugin marketplace list --json`
- Marketplace type needs to be added to `internal/claude/types.go`

**TUI patterns:** `internal/tui/` uses Bubble Tea with flat model architecture. The restore TUI will follow the same patterns but as a separate model/view.

**No existing sidecar handling:** This is new functionality. Sidecar read/write will be added to `internal/claude/` package.

## Implementation Phases

### Phase 1: Cobra CLI Migration

**Goal:** Migrate from direct TUI launch to Cobra-based CLI structure

**Components:**
- Cobra dependency in `go.mod`
- `cmd/cpm/main.go` - Cobra root command setup
- `cmd/cpm/root.go` - Default command (launches existing TUI)
- `cmd/cpm/version.go` - Version command

**Dependencies:** None (first phase)

**Done when:** `cpm` launches TUI as before, `cpm version` shows version info, `cpm --help` shows available commands

### Phase 2: Marketplace Support

**Goal:** Add marketplace listing to Claude client

**Components:**
- `Marketplace` type in `internal/claude/types.go` - name, source, repo fields
- `ListMarketplaces()` method on Client interface in `internal/claude/client.go`
- `AddMarketplace(source string)` method for restore operations

**Dependencies:** Phase 1

**Done when:** Can list marketplaces with full metadata (name, source type, repo/URL), can add a marketplace programmatically

### Phase 3: Sidecar Management

**Goal:** Implement sidecar read/write/sync logic

**Components:**
- `Sidecar` type in `internal/claude/sidecar.go` - matches JSON schema
- `ReadSidecar(scope Scope, projectPath string)` - reads sidecar for given scope
- `WriteSidecar(sidecar Sidecar, scope Scope, projectPath string)` - writes sidecar
- `SyncSidecar(current PluginList, existing *Sidecar)` - merges state, returns updated sidecar and diff

**Dependencies:** Phase 2 (needs Marketplace type)

**Done when:** Can read/write sidecars for all scopes, sync logic correctly identifies discovered/removed plugins

### Phase 4: Sync Command

**Goal:** Implement `cpm sync` command with detailed and quiet modes

**Components:**
- `cmd/cpm/sync.go` - sync command implementation
- `--quiet` flag for minimal output
- Integration with sidecar management from Phase 3

**Dependencies:** Phase 3

**Done when:** `cpm sync` outputs detailed changes, `cpm sync --quiet` outputs only errors/significant changes

### Phase 5: Restore TUI

**Goal:** Interactive restore flow with marketplace and plugin selection

**Components:**
- `internal/tui/restore/` - restore TUI model, view, update
- Checkbox list for marketplaces and plugins
- Progress display during restore
- Version mismatch warning display
- `cmd/cpm/restore.go` - restore command launching TUI

**Dependencies:** Phase 2, Phase 3

**Done when:** `cpm restore <file>` shows interactive selection, executes restore, reports version mismatches

### Phase 6: Startup Integration

**Goal:** Integrate sync and restore-prompt into main TUI startup

**Components:**
- Modify `cmd/cpm/root.go` to run sync on startup
- Add restore prompt when uninstalled plugins detected in sidecar
- Route to restore TUI or main TUI based on user choice

**Dependencies:** Phase 4, Phase 5

**Done when:** Starting `cpm` syncs sidecars, prompts for restore if needed, then shows main TUI

### Phase 7: Hook Management

**Goal:** Implement `cpm init` for Claude hook installation

**Components:**
- `cmd/cpm/init.go` - init command with `--remove` and `--status` flags
- `internal/claude/hooks.go` - read/write/merge Claude settings.json hooks
- First-run prompt in startup flow
- Post-restore prompt in restore TUI

**Dependencies:** Phase 4, Phase 6

**Done when:** `cpm init` installs hook, `cpm init --remove` removes it, first-run and post-restore show hook prompt

### Phase 8: Documentation, Polish, and Edge Cases

**Goal:** Document the sync/restore features, handle edge cases, and improve UX

**Components:**
- `docs/sync-and-restore.md` - User-facing documentation explaining:
  - What sidecars are and where they're stored
  - How sync works (automatic on startup, manual via `cpm sync`)
  - How to restore plugins on a new machine or share with teammates
  - How to set up automatic sync via Claude hook (`cpm init`)
  - Sidecar file format for advanced users
- Update `README.md` with sync/restore feature overview and links to detailed docs
- Handle missing `cpm` in PATH for hook (use absolute path or warn)
- Handle corrupt/invalid sidecar files gracefully
- Handle Claude CLI errors during sync/restore
- Add `[?] Learn more` option to hook prompt

**Dependencies:** All previous phases

**Done when:** Documentation is complete and accurate, error cases handled gracefully, UX is polished

## Additional Considerations

**Version tracking limitations:** Claude CLI does not support installing specific plugin versions. Sidecars record version at discovery/install time for informational purposes only. Version mismatch warnings are advisory - users cannot act on them without Claude CLI changes.

**Sidecar portability:** The sidecar format is designed to be portable across machines. Marketplace definitions include full source information so a new machine can add the marketplace before installing plugins.

**Hook PATH considerations:** The SessionStart hook runs `cpm sync --quiet`. If `cpm` is not in PATH during Claude sessions, the hook will fail silently. `cpm init` should detect this and either use an absolute path or warn the user.

**Backward compatibility:** Sidecars use a `version` field. Future format changes can be handled by migration logic that checks the version and upgrades if needed.
