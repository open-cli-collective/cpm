# cpm - Phase 1: Project Scaffolding

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Initialize project with all tooling and configuration files

**Architecture:** Greenfield Go project using mise for tool management, lefthook for git hooks, golangci-lint for linting, and GoReleaser for releases. No application code in this phase - only scaffolding.

**Tech Stack:** Go 1.23, mise, lefthook, golangci-lint, gofumpt, GoReleaser, release-please

**Scope:** Phase 1 of 8 from original design

**Codebase verified:** 2026-01-21 - Confirmed greenfield project with only .gitignore and docs/design-plans/ present

---

## Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`

**Step 1: Create go.mod**

```bash
cd /Users/brajkovic/Code/claude-plugin-manager/.worktrees/cpm-plugin-manager
go mod init github.com/open-cli-collective/cpm
```

**Step 2: Verify operationally**

Run: `cat go.mod`
Expected: Shows module path `github.com/open-cli-collective/cpm` with Go version

**Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: initialize go module"
```

---

## Task 2: Create Directory Structure

**Files:**
- Create: `cmd/cpm/.gitkeep`
- Create: `internal/tui/.gitkeep`
- Create: `internal/claude/.gitkeep`
- Create: `internal/version/.gitkeep`

**Step 1: Create directories with .gitkeep files**

```bash
mkdir -p cmd/cpm internal/tui internal/claude internal/version
touch cmd/cpm/.gitkeep internal/tui/.gitkeep internal/claude/.gitkeep internal/version/.gitkeep
```

**Step 2: Verify operationally**

Run: `find cmd internal -type f`
Expected: Lists all .gitkeep files

**Step 3: Commit**

```bash
git add cmd internal
git commit -m "chore: create project directory structure"
```

---

## Task 3: Create mise.toml

**Files:**
- Create: `mise.toml`

**Step 1: Create mise.toml**

Create file `mise.toml`:

```toml
[tools]
go = "1.23"
"aqua:golangci/golangci-lint" = "1.64.5"
"aqua:mvdan/gofumpt" = "0.7.0"
lefthook = "1.11.5"

[env]
GOFLAGS = "-mod=mod"

[tasks.fmt]
description = "Format Go code with gofumpt"
run = "gofumpt -l -w -extra ."

[tasks.lint]
description = "Run golangci-lint"
run = "golangci-lint run ./..."

[tasks.vet]
description = "Run go vet"
run = "go vet ./..."

[tasks.test]
description = "Run Go tests"
run = "go test -v -race -count=1 ./..."

[tasks.build]
description = "Build the application"
run = """
go build -ldflags="-s -w \
  -X github.com/open-cli-collective/cpm/internal/version.Version=dev \
  -X github.com/open-cli-collective/cpm/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/open-cli-collective/cpm/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o cpm ./cmd/cpm
"""

[tasks.ci]
description = "Run all CI checks"
run = """
set -e
mise run fmt
mise run vet
mise run lint
mise run test
mise run build
"""

[settings]
task_output = "prefix"
```

**Step 2: Verify operationally**

Run: `mise install`
Expected: Installs Go 1.23, golangci-lint, gofumpt, lefthook without errors

**Step 3: Commit**

```bash
git add mise.toml
git commit -m "chore: add mise configuration for tool management"
```

---

## Task 4: Create lefthook.yaml

**Files:**
- Create: `lefthook.yaml`

**Step 1: Create lefthook.yaml**

Create file `lefthook.yaml`:

```yaml
pre-commit:
  parallel: true
  commands:
    gofumpt:
      glob: "*.go"
      run: gofumpt -l -w -extra {staged_files}
      stage_fixed: true

    golangci-lint:
      glob: "*.go"
      run: golangci-lint run --fix {staged_files}
      stage_fixed: true

    govet:
      glob: "*.go"
      run: go vet ./...

commit-msg:
  commands:
    conventional-commits:
      run: |
        message=$(cat {1})
        if ! echo "$message" | grep -qE '^(feat|fix|docs|style|refactor|perf|test|chore|build|ci)(\(.+\))?!?: .'; then
          echo "Commit message must follow conventional commits format:"
          echo "  type(scope): description"
          echo ""
          echo "Types: feat, fix, docs, style, refactor, perf, test, chore, build, ci"
          exit 1
        fi
```

**Step 2: Verify operationally**

Run: `lefthook install`
Expected: Hooks installed successfully

**Step 3: Commit**

```bash
git add lefthook.yaml
git commit -m "chore: add lefthook configuration for git hooks"
```

---

## Task 5: Create .golangci.yml

**Files:**
- Create: `.golangci.yml`

**Step 1: Create .golangci.yml**

Create file `.golangci.yml`:

```yaml
version: "2"

run:
  timeout: 5m
  modules-download-mode: readonly

linters-settings:
  gocyclo:
    min-complexity: 15

  errcheck:
    check-type-assertions: true

  gofumpt:
    extra-rules: true

  govet:
    enable-all: true

linters:
  enable:
    # Error handling
    - errcheck
    - errorlint

    # Code quality
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused

    # Style and formatting
    - gofmt
    - gofumpt
    - unconvert

    # Complexity
    - gocyclo

    # Security
    - gosec

    # Best practices
    - revive
    - misspell
    - gocritic

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
        - gosec
```

**Step 2: Verify operationally**

Run: `golangci-lint config verify`
Expected: No errors (config is valid)

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "chore: add golangci-lint configuration"
```

---

## Task 6: Create .goreleaser.yml

**Files:**
- Create: `.goreleaser.yml`

**Step 1: Create .goreleaser.yml**

Create file `.goreleaser.yml`:

```yaml
version: 2

project_name: cpm

before:
  hooks:
    - go mod tidy
    - go mod download

builds:
  - id: default
    main: ./cmd/cpm
    binary: cpm
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/open-cli-collective/cpm/internal/version.Version={{ .Version }}
      - -X github.com/open-cli-collective/cpm/internal/version.Commit={{ .FullCommit }}
      - -X github.com/open-cli-collective/cpm/internal/version.Date={{ .Date }}
    flags:
      - -trimpath
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: default
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - README.md
      - LICENSE

checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
  algorithm: sha256

changelog:
  use: github-native

release:
  github:
    owner: open-cli-collective
    name: cpm
  draft: false
  prerelease: auto
```

**Step 2: Verify operationally**

Run: `goreleaser check`
Expected: Config is valid (or minor warnings about missing files like LICENSE)

Note: goreleaser is not installed via mise - this check will be run in CI. Skip if not available locally.

**Step 3: Commit**

```bash
git add .goreleaser.yml
git commit -m "chore: add goreleaser configuration for releases"
```

---

## Task 7: Create renovate.json

**Files:**
- Create: `renovate.json`

**Step 1: Create renovate.json**

Create file `renovate.json`:

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    "config:recommended",
    ":semanticCommits",
    ":semanticCommitTypeAll(chore)",
    "group:allNonMajor"
  ],
  "labels": ["dependencies"],
  "golang": {
    "enabled": true
  },
  "postUpdateOptions": ["gomodTidy"]
}
```

**Step 2: Verify operationally**

Run: `cat renovate.json | python3 -m json.tool`
Expected: Valid JSON output

**Step 3: Commit**

```bash
git add renovate.json
git commit -m "chore: add renovate configuration for dependency updates"
```

---

## Task 8: Create Makefile

**Files:**
- Create: `Makefile`

**Step 1: Create Makefile**

Create file `Makefile`:

```makefile
.PHONY: all fmt lint vet test build ci clean

all: ci

fmt:
	mise run fmt

lint:
	mise run lint

vet:
	mise run vet

test:
	mise run test

build:
	mise run build

ci:
	mise run ci

clean:
	rm -f cpm
	rm -rf dist/
```

**Step 2: Verify operationally**

Run: `make --dry-run ci`
Expected: Shows the commands that would run

**Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile as thin wrapper for mise tasks"
```

---

## Task 9: Create Placeholder main.go

**Files:**
- Create: `cmd/cpm/main.go`
- Delete: `cmd/cpm/.gitkeep`

**Step 1: Create placeholder main.go**

Create file `cmd/cpm/main.go`:

```go
package main

import "fmt"

func main() {
	fmt.Println("cpm - Claude Plugin Manager")
}
```

**Step 2: Remove .gitkeep**

```bash
rm cmd/cpm/.gitkeep
```

**Step 3: Verify operationally**

Run: `go build -o cpm ./cmd/cpm && ./cpm`
Expected: Prints "cpm - Claude Plugin Manager"

**Step 4: Verify .gitkeep was removed**

Run: `ls cmd/cpm/`
Expected: Only `main.go` present, no `.gitkeep`

**Step 5: Commit**

```bash
git add cmd/cpm/main.go
git rm cmd/cpm/.gitkeep
git commit -m "chore: add placeholder main.go"
```

---

## Task 10: Create CLAUDE.md

**Files:**
- Create: `CLAUDE.md`

**Step 1: Create CLAUDE.md**

Create file `CLAUDE.md`:

```markdown
# cpm - Claude Plugin Manager

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
  claude/          # Claude CLI client (shells out to `claude` command)
  tui/             # Bubble Tea TUI implementation
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
```

**Step 2: Verify operationally**

Run: `cat CLAUDE.md | head -20`
Expected: Shows first 20 lines of CLAUDE.md

**Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md project guidance"
```

---

## Task 11: Run Full CI Check

**Step 1: Run mise ci**

```bash
mise run ci
```

Expected output:
- fmt: No changes (code already formatted)
- vet: No issues
- lint: No issues
- test: No test files (ok)
- build: Builds successfully, creates `cpm` binary

**Step 2: Verify binary works**

```bash
./cpm
```

Expected: Prints "cpm - Claude Plugin Manager"

**Step 3: Clean up**

```bash
rm -f cpm
```

---

## Phase 1 Complete

**Verification:**
- `mise install` succeeds
- `lefthook install` succeeds
- `mise run build` succeeds
- `./cpm` prints placeholder message

**Files created:**
- `go.mod`
- `mise.toml`
- `lefthook.yaml`
- `.golangci.yml`
- `.goreleaser.yml`
- `renovate.json`
- `Makefile`
- `CLAUDE.md`
- `cmd/cpm/main.go`
- `internal/tui/.gitkeep`
- `internal/claude/.gitkeep`
- `internal/version/.gitkeep`
