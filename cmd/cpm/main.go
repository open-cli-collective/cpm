package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
	"github.com/open-cli-collective/cpm/internal/tui"
	"github.com/open-cli-collective/cpm/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	theme := tui.ThemeAuto

	// Handle flags
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--version" || arg == "-v":
			fmt.Println(version.String())
			return nil
		case arg == "--help" || arg == "-h":
			printUsage()
			return nil
		case arg == "--theme" || arg == "-t":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "Error: --theme requires an argument (auto, light, dark)")
				os.Exit(1)
			}
			i++
			var ok bool
			theme, ok = parseTheme(os.Args[i])
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: invalid theme '%s'. Use: auto, light, dark\n", os.Args[i])
				os.Exit(1)
			}
		case strings.HasPrefix(arg, "--theme="):
			val := strings.TrimPrefix(arg, "--theme=")
			var ok bool
			theme, ok = parseTheme(val)
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: invalid theme '%s'. Use: auto, light, dark\n", val)
				os.Exit(1)
			}
		case strings.HasPrefix(arg, "-t="):
			val := strings.TrimPrefix(arg, "-t=")
			var ok bool
			theme, ok = parseTheme(val)
			if !ok {
				fmt.Fprintf(os.Stderr, "Error: invalid theme '%s'. Use: auto, light, dark\n", val)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n\n", arg)
			printUsage()
			os.Exit(1)
		}
	}

	// Check for claude CLI
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Please install Claude Code first")
	}

	// Get current working directory for filtering project-scoped plugins
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create client and model
	client := claude.NewClient()
	model := tui.NewModelWithTheme(client, workingDir, theme)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// parseTheme converts a theme string to a Theme constant.
func parseTheme(s string) (tui.Theme, bool) {
	switch strings.ToLower(s) {
	case "auto":
		return tui.ThemeAuto, true
	case "light":
		return tui.ThemeLight, true
	case "dark":
		return tui.ThemeDark, true
	default:
		return tui.ThemeAuto, false
	}
}

func printUsage() {
	fmt.Println("cpm - Claude Plugin Manager")
	fmt.Println()
	fmt.Println("A TUI for managing Claude Code plugins with clear scope visibility.")
	fmt.Println()
	fmt.Println("Usage: cpm [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help           Show this help message")
	fmt.Println("  -v, --version        Show version information")
	fmt.Println("  -t, --theme <theme>  Set color theme: auto, light, dark (default: auto)")
}
