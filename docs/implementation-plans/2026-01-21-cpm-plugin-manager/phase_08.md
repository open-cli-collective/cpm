# cpm - Phase 8: CI/CD & Release

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Complete build, test, and release automation

**Architecture:** GitHub Actions for CI (lint, test, build), PR artifacts (all platforms), release-please for automated release PRs, GoReleaser for binary distribution. Package managers: Homebrew, Chocolatey, WinGet.

**Tech Stack:** GitHub Actions, GoReleaser, release-please, Chocolatey, WinGet

**Scope:** Phase 8 of 8 from original design

**Codebase verified:** 2026-01-21 - Phase 7 complete with full TUI functionality

---

## Task 1: Create CI Workflow

**Files:**
- Create: `.github/workflows/ci.yml`

**Step 1: Create CI workflow**

Create file `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.64.5

      - name: Run golangci-lint
        run: golangci-lint run ./...

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          fail_ci_if_error: false

  build:
    name: Build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: Build
        run: go build -o cpm ./cmd/cpm

      - name: Verify binary
        run: ./cpm --version
```

**Step 2: Verify operationally**

Run: `cat .github/workflows/ci.yml | head -20`
Expected: Shows beginning of workflow file

**Step 3: Commit**

```bash
mkdir -p .github/workflows
git add .github/workflows/ci.yml
git commit -m "ci: add CI workflow for lint, test, and build"
```

---

## Task 2: Create PR Artifacts Workflow

**Files:**
- Create: `.github/workflows/pr-artifacts.yml`

**Step 1: Create PR artifacts workflow**

Create file `.github/workflows/pr-artifacts.yml`:

```yaml
name: PR Artifacts

on:
  pull_request:
    branches: [main]

permissions:
  contents: read
  pull-requests: write

jobs:
  build-artifacts:
    name: Build ${{ matrix.goos }}-${{ matrix.goarch }}
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - goos: linux
            goarch: amd64
          - goos: linux
            goarch: arm64
          - goos: darwin
            goarch: amd64
          - goos: darwin
            goarch: arm64
          - goos: windows
            goarch: amd64

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: "0"
        run: |
          BINARY_NAME="cpm"
          if [ "$GOOS" = "windows" ]; then
            BINARY_NAME="cpm.exe"
          fi
          go build -ldflags="-s -w \
            -X github.com/open-cli-collective/cpm/internal/version.Version=pr-${{ github.event.pull_request.number }} \
            -X github.com/open-cli-collective/cpm/internal/version.Commit=${{ github.sha }} \
            -X github.com/open-cli-collective/cpm/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            -o ${BINARY_NAME} ./cmd/cpm

      - name: Create archive
        run: |
          ARCHIVE_NAME="cpm_pr-${{ github.event.pull_request.number }}_${{ matrix.goos }}_${{ matrix.goarch }}"
          if [ "${{ matrix.goos }}" = "windows" ]; then
            zip "${ARCHIVE_NAME}.zip" cpm.exe README.md
          else
            tar -czvf "${ARCHIVE_NAME}.tar.gz" cpm README.md
          fi

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: cpm_${{ matrix.goos }}_${{ matrix.goarch }}
          path: |
            *.tar.gz
            *.zip
          retention-days: 7
```

**Step 2: Commit**

```bash
git add .github/workflows/pr-artifacts.yml
git commit -m "ci: add PR artifacts workflow for multi-platform builds"
```

---

## Task 3: Create Release Please Workflow

**Files:**
- Create: `.github/workflows/release-please.yml`

**Step 1: Create release-please workflow**

Create file `.github/workflows/release-please.yml`:

```yaml
name: Release Please

on:
  push:
    branches: [main]

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    name: Release Please
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.release.outputs.release_created }}
      tag_name: ${{ steps.release.outputs.tag_name }}
    steps:
      - name: Run release-please
        id: release
        uses: googleapis/release-please-action@v4
        with:
          release-type: go

  goreleaser:
    name: GoReleaser
    needs: release-please
    if: needs.release-please.outputs.release_created
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  homebrew:
    name: Update Homebrew
    needs: [release-please, goreleaser]
    if: needs.release-please.outputs.release_created
    runs-on: ubuntu-latest
    steps:
      - name: Update Homebrew tap
        uses: mislav/bump-homebrew-formula-action@v3
        with:
          formula-name: cpm
          homebrew-tap: open-cli-collective/homebrew-tap
          tag-name: ${{ needs.release-please.outputs.tag_name }}
        env:
          COMMITTER_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

**Step 2: Commit**

```bash
git add .github/workflows/release-please.yml
git commit -m "ci: add release-please workflow with GoReleaser and Homebrew"
```

---

## Task 4: Create Chocolatey Package Configuration

**Files:**
- Create: `packaging/chocolatey/cpm.nuspec`
- Create: `packaging/chocolatey/tools/chocolateyinstall.ps1`
- Create: `packaging/chocolatey/tools/chocolateyuninstall.ps1`

**Step 1: Create nuspec file**

Create file `packaging/chocolatey/cpm.nuspec`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>cpm</id>
    <version>$version$</version>
    <title>cpm - Claude Plugin Manager</title>
    <authors>open-cli-collective</authors>
    <projectUrl>https://github.com/open-cli-collective/cpm</projectUrl>
    <licenseUrl>https://github.com/open-cli-collective/cpm/blob/main/LICENSE</licenseUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <projectSourceUrl>https://github.com/open-cli-collective/cpm</projectSourceUrl>
    <bugTrackerUrl>https://github.com/open-cli-collective/cpm/issues</bugTrackerUrl>
    <tags>claude ai plugin manager tui cli</tags>
    <summary>TUI for managing Claude Code plugins with clear scope visibility</summary>
    <description>
cpm (Claude Plugin Manager) is a terminal user interface application that manages Claude Code plugins with clear visibility into installation scopes (local vs project).

Features:
- Two-pane TUI for plugin management
- Clear scope indicators (local, project, user)
- Batch install/uninstall operations
- Search and filter plugins
- Keyboard and mouse navigation
    </description>
  </metadata>
  <files>
    <file src="tools\**" target="tools" />
  </files>
</package>
```

**Step 2: Create install script**

Create file `packaging/chocolatey/tools/chocolateyinstall.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$packageName = 'cpm'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$url64 = "https://github.com/open-cli-collective/cpm/releases/download/v$($env:chocolateyPackageVersion)/cpm_$($env:chocolateyPackageVersion)_windows_amd64.zip"

$packageArgs = @{
  packageName    = $packageName
  unzipLocation  = $toolsDir
  url64bit       = $url64
  checksum64     = '$checksum64$'
  checksumType64 = 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
```

**Step 3: Create uninstall script**

Create file `packaging/chocolatey/tools/chocolateyuninstall.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$packageName = 'cpm'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

Remove-Item -Path "$toolsDir\cpm.exe" -Force -ErrorAction SilentlyContinue
```

**Step 4: Commit**

```bash
mkdir -p packaging/chocolatey/tools
git add packaging/chocolatey/
git commit -m "ci: add Chocolatey package configuration"
```

---

## Task 5: Create WinGet Manifest

**Files:**
- Create: `packaging/winget/cpm.yaml`

**Step 1: Create WinGet manifest**

Create file `packaging/winget/cpm.yaml`:

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.singleton.1.6.0.schema.json
PackageIdentifier: OpenCLICollective.cpm
PackageVersion: "$version$"
PackageName: cpm
Publisher: open-cli-collective
License: MIT
ShortDescription: TUI for managing Claude Code plugins with clear scope visibility
Description: |
  cpm (Claude Plugin Manager) is a terminal user interface application that manages
  Claude Code plugins with clear visibility into installation scopes (local vs project).
Moniker: cpm
Tags:
  - claude
  - ai
  - plugin
  - manager
  - tui
  - cli
PackageUrl: https://github.com/open-cli-collective/cpm
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/open-cli-collective/cpm/releases/download/v$version$/cpm_$version$_windows_amd64.zip
    InstallerType: zip
    InstallerSha256: "$checksum$"
    NestedInstallerFiles:
      - RelativeFilePath: cpm.exe
        PortableCommandAlias: cpm
ManifestType: singleton
ManifestVersion: 1.6.0
```

**Step 2: Commit**

```bash
mkdir -p packaging/winget
git add packaging/winget/
git commit -m "ci: add WinGet manifest configuration"
```

---

## Task 6: Create README.md

**Files:**
- Create: `README.md`

**Step 1: Create README**

Create file `README.md`:

```markdown
# cpm - Claude Plugin Manager

[![CI](https://github.com/open-cli-collective/cpm/actions/workflows/ci.yml/badge.svg)](https://github.com/open-cli-collective/cpm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/open-cli-collective/cpm)](https://github.com/open-cli-collective/cpm/releases/latest)

A terminal user interface for managing Claude Code plugins with clear visibility into installation scopes.

![cpm screenshot](docs/screenshot.png)

## Features

- **Two-pane TUI** - Plugin list on the left, details on the right
- **Clear scope indicators** - See if plugins are installed globally (user), for the project, or locally
- **Batch operations** - Mark multiple plugins for install/uninstall, apply all at once
- **Search and filter** - Quickly find plugins with `/`
- **Keyboard and mouse** - Full keyboard navigation plus mouse support

## Installation

### Homebrew (macOS/Linux)

```bash
brew install open-cli-collective/tap/cpm
```

### Chocolatey (Windows)

```powershell
choco install cpm
```

### WinGet (Windows)

```powershell
winget install OpenCLICollective.cpm
```

### Go Install

```bash
go install github.com/open-cli-collective/cpm/cmd/cpm@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/open-cli-collective/cpm/releases/latest).

## Usage

```bash
cpm
```

### Key Bindings

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `PgUp/Ctrl+u` | Page up |
| `PgDn/Ctrl+d` | Page down |
| `Home/g` | Go to top |
| `End/G` | Go to bottom |
| `l` | Mark for local install |
| `p` | Mark for project install |
| `Tab` | Toggle between scopes |
| `u` | Mark for uninstall |
| `Enter` | Apply pending changes |
| `Esc` | Clear pending / Cancel |
| `/` | Filter plugins |
| `r` | Refresh plugin list |
| `q` | Quit |

## Requirements

- Claude Code CLI (`claude`) must be installed and in PATH
- Terminal with color support

## Building from Source

```bash
# Clone the repository
git clone https://github.com/open-cli-collective/cpm.git
cd cpm

# Install tools (requires mise)
mise install

# Build
mise run build

# Run
./cpm
```

## Contributing

Contributions are welcome! Please read our [contributing guidelines](CONTRIBUTING.md) first.

## License

MIT License - see [LICENSE](LICENSE) for details.
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with installation and usage instructions"
```

---

## Task 7: Create LICENSE

**Files:**
- Create: `LICENSE`

**Step 1: Create MIT license**

Create file `LICENSE`:

```
MIT License

Copyright (c) 2026 open-cli-collective

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

**Step 2: Commit**

```bash
git add LICENSE
git commit -m "docs: add MIT license"
```

---

## Task 8: Final Integration Test

**Step 1: Run full CI locally**

```bash
mise run ci
```

Expected: All lint, test, build checks pass

**Step 2: Verify workflow files**

```bash
ls -la .github/workflows/
```

Expected: ci.yml, pr-artifacts.yml, release-please.yml

**Step 3: Verify packaging files**

```bash
ls -la packaging/
```

Expected: chocolatey/, winget/ directories

**Step 4: Build and test final binary**

```bash
mise run build
./cpm --version
./cpm --help
./cpm
```

Expected: All commands work correctly

---

## Phase 8 Complete

**Verification:**
- `.github/workflows/ci.yml` runs lint, test, build on push/PR
- `.github/workflows/pr-artifacts.yml` builds all platforms on PR
- `.github/workflows/release-please.yml` creates release PRs and triggers GoReleaser
- `packaging/chocolatey/` contains nuspec and PowerShell scripts
- `packaging/winget/` contains manifest template
- `README.md` documents installation and usage
- `LICENSE` contains MIT license

**Files created:**
- `.github/workflows/ci.yml`
- `.github/workflows/pr-artifacts.yml`
- `.github/workflows/release-please.yml`
- `packaging/chocolatey/cpm.nuspec`
- `packaging/chocolatey/tools/chocolateyinstall.ps1`
- `packaging/chocolatey/tools/chocolateyuninstall.ps1`
- `packaging/winget/cpm.yaml`
- `README.md`
- `LICENSE`

**CI/CD flow:**
1. Push/PR → CI runs (lint, test, build)
2. PR → Artifacts built for all platforms
3. Merge to main → release-please creates/updates release PR
4. Merge release PR → GoReleaser builds and publishes
5. Post-release → Homebrew tap updated, manual Chocolatey/WinGet submissions
