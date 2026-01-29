package claude

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExportVersion is the current export file format version.
const ExportVersion = 1

// ExportedPlugin represents a plugin in an export file.
type ExportedPlugin struct {
	ID      string `json:"id"`
	Scope   Scope  `json:"scope"`
	Enabled bool   `json:"enabled"`
}

// ExportFile represents the structure of a plugin export file.
type ExportFile struct {
	Version int              `json:"version"`
	Plugins []ExportedPlugin `json:"plugins"`
}

// ExportPlugins exports the list of installed plugins to a JSON file.
func ExportPlugins(client Client, filePath string) error {
	list, err := client.ListPlugins(false)
	if err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	exported := ExportFile{
		Version: ExportVersion,
		Plugins: make([]ExportedPlugin, 0, len(list.Installed)),
	}

	for _, p := range list.Installed {
		exported.Plugins = append(exported.Plugins, ExportedPlugin{
			ID:      p.ID,
			Scope:   p.Scope,
			Enabled: p.Enabled,
		})
	}

	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// ImportResult contains the results of an import operation.
type ImportResult struct {
	Installed []string // Plugin IDs that were installed
	Skipped   []string // Plugin IDs that were already installed
	Failed    []string // Plugin IDs that failed to install
	Errors    []error  // Errors for failed plugins
}

// ReadExportFile reads and validates a plugin export file.
func ReadExportFile(filePath string) (*ExportFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	var exported ExportFile
	if err := json.Unmarshal(data, &exported); err != nil {
		return nil, fmt.Errorf("failed to parse export file: %w", err)
	}

	if exported.Version != ExportVersion {
		return nil, fmt.Errorf("unsupported export file version: %d (expected %d)", exported.Version, ExportVersion)
	}

	return &exported, nil
}

// ImportPlugins imports plugins from an export file.
// It skips plugins that are already installed.
func ImportPlugins(client Client, exported *ExportFile) *ImportResult {
	result := &ImportResult{}

	// Get currently installed plugins
	list, err := client.ListPlugins(false)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to list plugins: %w", err))
		return result
	}

	// Build set of installed plugin IDs
	installed := make(map[string]bool)
	for _, p := range list.Installed {
		installed[p.ID] = true
	}

	// Install missing plugins
	for _, p := range exported.Plugins {
		if installed[p.ID] {
			result.Skipped = append(result.Skipped, p.ID)
			continue
		}

		if err := client.InstallPlugin(p.ID, p.Scope); err != nil {
			result.Failed = append(result.Failed, p.ID)
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", p.ID, err))
			continue
		}

		result.Installed = append(result.Installed, p.ID)

		// Handle enabled state - disable if needed
		if !p.Enabled {
			if err := client.DisablePlugin(p.ID, p.Scope); err != nil {
				// Non-fatal: plugin is installed but enable state may differ
				result.Errors = append(result.Errors, fmt.Errorf("%s: failed to set enabled state: %w", p.ID, err))
			}
		}
	}

	return result
}
