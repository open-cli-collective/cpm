package main

import (
	"fmt"
	"os"
	"os/exec"

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
	// Handle --version and --help
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version.String())
			return nil
		case "--help", "-h":
			printUsage()
			return nil
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n\n", os.Args[1])
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
	model := tui.NewModel(client, workingDir)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

func printUsage() {
	fmt.Println("cpm - Claude Plugin Manager")
	fmt.Println()
	fmt.Println("A TUI for managing Claude Code plugins with clear scope visibility.")
	fmt.Println()
	fmt.Println("Usage: cpm [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help     Show this help message")
	fmt.Println("  -v, --version  Show version information")
}
