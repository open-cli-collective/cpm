# cpm - Claude Plugin Manager

Last verified: 2026-01-22

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
3. **On merge to main**: release-please creates/updates release PR
4. **On release PR merge**: GoReleaser builds and publishes to GitHub Releases, Homebrew, Chocolatey, WinGet
