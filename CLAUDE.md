# cpm - Claude Plugin Manager

Last verified: 2026-01-23

A TUI application for managing Claude Code plugins with clear visibility into installation scopes.

## Quick Start

```bash
mise install        # Install tools (Go, golangci-lint, gofumpt, lefthook)
lefthook install    # Set up git hooks
mise run build      # Build the application
./cpm               # Run the application
```

## Project Structure

```
cmd/cpm/           # Application entry point
internal/
  claude/          # Claude CLI client (see internal/claude/CLAUDE.md)
  tui/             # Bubble Tea TUI (see internal/tui/CLAUDE.md)
  version/         # Version info (injected at build time)
```

## Development Commands

```bash
mise run fmt       # Format code with gofumpt
mise run lint      # Run golangci-lint
mise run vet       # Run go vet
mise run test      # Run tests
mise run build     # Build binary
mise run ci        # Run all checks
```

Or use Make: `make fmt`, `make lint`, `make test`, `make build`, `make ci`

## Architecture

- **Flat model architecture**: Single Model struct contains all TUI state
- **Bubble Tea**: TUI framework using Elm Architecture (Model-Update-View)
- **Lip Gloss**: Terminal styling and layout
- **Claude CLI**: All plugin operations shell out to `claude plugin` commands

See [docs/architecture.md](docs/architecture.md) for detailed diagrams and type definitions.

### Maintaining Architecture Documentation

When making structural changes (new types, packages, or significant refactors):

1. Update `docs/architecture.md` with any new/changed types or relationships
2. Update the "Last updated" date at the top
3. Use LSP `documentSymbol` operation to extract current type definitions:
   ```
   LSP documentSymbol on internal/claude/types.go, client.go, manifest.go
   LSP documentSymbol on internal/tui/model.go, styles.go, keys.go
   ```
4. Ensure Mermaid diagrams reflect current structure

## Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Description |
|------|-------------|
| `feat` | New feature for users |
| `fix` | Bug fix for users |
| `docs` | Documentation changes |
| `style` | Code style (formatting, missing semicolons) |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks |
| `build` | Build system or external dependencies |
| `ci` | CI configuration |

Example: `feat(tui): add plugin filtering with / key`

## CI & Release Workflow

1. **On every push/PR**: CI runs lint, test, build
2. **On PR**: Artifacts built for all platforms (downloadable for testing)
3. **On merge to main with `feat:` or `fix:` commit**: Auto-release creates tag and triggers release
4. **On tag push**: GoReleaser builds and publishes to GitHub Releases and Homebrew

### Version Scheme

- Base version stored in `version.txt` (e.g., `0.1`)
- Released version: `v{base}.{run_number}` (e.g., `v0.1.42`)
- Only `feat:` and `fix:` commits trigger releases (must also change Go files)

### When to Bump version.txt

After completing work, analyze changes and suggest a version bump if appropriate:

| Change Type | Action | Example |
|-------------|--------|---------|
| Bug fixes only | No bump needed | Fix mouse click offset |
| New minor features | No bump needed | Add keyboard shortcut |
| Significant new feature | Bump minor (0.1 → 0.2) | Add plugin search/filter |
| Major new capability | Bump minor (0.2 → 0.3) | Add bulk operations |
| Breaking changes | Bump major (0.x → 1.0) | Change CLI flags, config format |
| Stability milestone | Bump major (0.x → 1.0) | Ready for production use |

**Bump triggers to watch for:**
- New commands or major UI modes
- Changes to data structures that affect saved state
- New external dependencies or integrations
- Significant UX changes users will notice

**How to bump:** Edit `version.txt` and commit with `chore: bump version to X.Y`
