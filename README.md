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
