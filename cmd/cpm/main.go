package main

import (
	"bufio"
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
	// Parse flags
	exportPath, importPath, done := parseFlags()
	if done {
		return nil
	}

	// Check for claude CLI
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Please install Claude Code first")
	}

	client := claude.NewClient()

	// Handle export
	if exportPath != "" {
		return handleExport(client, exportPath)
	}

	// Handle import
	if importPath != "" {
		return handleImport(client, importPath)
	}

	return runTUI(client)
}

// parseFlags parses command-line flags and returns export/import paths.
// Returns done=true if the program should exit (help/version shown).
func parseFlags() (exportPath, importPath string, done bool) {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--version" || arg == "-v":
			fmt.Println(version.String())
			return "", "", true
		case arg == "--help" || arg == "-h":
			printUsage()
			return "", "", true
		case arg == "--export" || arg == "-e":
			if i+1 >= len(os.Args) {
				exitWithError("--export requires a file path argument")
			}
			i++
			exportPath = os.Args[i]
		case arg == "--import" || arg == "-i":
			if i+1 >= len(os.Args) {
				exitWithError("--import requires a file path argument")
			}
			i++
			importPath = os.Args[i]
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n\n", arg)
			printUsage()
			os.Exit(1)
		}
	}
	return exportPath, importPath, false
}

// exitWithError prints an error message and exits.
func exitWithError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}

// handleExport exports installed plugins to a file.
func handleExport(client claude.Client, filePath string) error {
	if err := claude.ExportPlugins(client, filePath); err != nil {
		return err
	}
	fmt.Printf("Exported plugins to %s\n", filePath)
	return nil
}

// handleImport imports plugins from a file.
func handleImport(client claude.Client, filePath string) error {
	exported, err := claude.ReadExportFile(filePath)
	if err != nil {
		return err
	}

	if len(exported.Plugins) == 0 {
		fmt.Println("No plugins to import.")
		return nil
	}

	// Show what will be imported
	fmt.Printf("Plugins to import from %s:\n", filePath)
	for _, p := range exported.Plugins {
		scope := string(p.Scope)
		if scope == "" {
			scope = "default"
		}
		fmt.Printf("  - %s (scope: %s)\n", p.ID, scope)
	}
	fmt.Println()

	// Confirm
	fmt.Print("Proceed with import? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Import cancelled.")
		return nil
	}

	// Perform import
	result := claude.ImportPlugins(client, exported)

	// Print results
	if len(result.Installed) > 0 {
		fmt.Printf("\nInstalled %d plugin(s):\n", len(result.Installed))
		for _, id := range result.Installed {
			fmt.Printf("  ✓ %s\n", id)
		}
	}

	if len(result.Skipped) > 0 {
		fmt.Printf("\nSkipped %d already-installed plugin(s):\n", len(result.Skipped))
		for _, id := range result.Skipped {
			fmt.Printf("  - %s\n", id)
		}
	}

	if len(result.Failed) > 0 {
		fmt.Printf("\nFailed to install %d plugin(s):\n", len(result.Failed))
		for i, id := range result.Failed {
			fmt.Printf("  ✗ %s: %v\n", id, result.Errors[i])
		}
	}

	return nil
}

// runTUI runs the interactive TUI.
func runTUI(client claude.Client) error {
	// Get current working directory for filtering project-scoped plugins
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create model
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
	fmt.Println("  -h, --help              Show this help message")
	fmt.Println("  -v, --version           Show version information")
	fmt.Println("  -e, --export <file>     Export installed plugins to a JSON file")
	fmt.Println("  -i, --import <file>     Import plugins from a JSON file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cpm                          Launch the interactive TUI")
	fmt.Println("  cpm --export plugins.json    Export plugins to plugins.json")
	fmt.Println("  cpm --import plugins.json    Import plugins from plugins.json")
}
