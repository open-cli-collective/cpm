package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// PluginManifest represents the plugin.json file in a plugin's .claude-plugin directory.
type PluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	AuthorName  string // Author name (from string or object.name)
	AuthorEmail string // Author email (from object.email, if available)
}

// pluginManifestRaw is used for initial parsing to handle flexible author field.
type pluginManifestRaw struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version,omitempty"`
	Author      json.RawMessage `json:"author,omitempty"`
}

// authorObject represents the object form of author field.
type authorObject struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// PluginComponents represents the skills, agents, commands, hooks, and MCPs a plugin provides.
type PluginComponents struct {
	Skills   []string
	Agents   []string
	Commands []string
	Hooks    []string
	MCPs     []string
}

// ReadPluginManifest reads the plugin.json manifest from the given install path.
func ReadPluginManifest(installPath string) (*PluginManifest, error) {
	manifestPath := filepath.Join(installPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	// First parse into raw struct to handle flexible author field
	var raw pluginManifestRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	manifest := &PluginManifest{
		Name:        raw.Name,
		Description: raw.Description,
		Version:     raw.Version,
	}

	// Parse author field (can be string or object)
	if len(raw.Author) > 0 {
		// Try string first
		var authorStr string
		if err := json.Unmarshal(raw.Author, &authorStr); err == nil {
			manifest.AuthorName = authorStr
		} else {
			// Try object form
			var authorObj authorObject
			if err := json.Unmarshal(raw.Author, &authorObj); err == nil {
				manifest.AuthorName = authorObj.Name
				manifest.AuthorEmail = authorObj.Email
			}
		}
	}

	return manifest, nil
}

// ScanPluginComponents scans the plugin directory for skills, agents, commands, hooks, and MCPs.
func ScanPluginComponents(installPath string) *PluginComponents {
	components := &PluginComponents{}

	// Scan skills/ directory (subdirectories are skill names)
	components.Skills = listSubdirectories(filepath.Join(installPath, "skills"))

	// Scan agents/ directory (.md files are agent definitions)
	components.Agents = listMarkdownFiles(filepath.Join(installPath, "agents"))

	// Scan commands/ directory (.md files are command definitions)
	components.Commands = listMarkdownFiles(filepath.Join(installPath, "commands"))

	// Scan hooks/ directory (can be subdirectories or .md files)
	hooks := listSubdirectories(filepath.Join(installPath, "hooks"))
	hooks = append(hooks, listMarkdownFiles(filepath.Join(installPath, "hooks"))...)
	components.Hooks = hooks

	// Scan mcp-servers/ directory (subdirectories are MCP server names)
	components.MCPs = listSubdirectories(filepath.Join(installPath, "mcp-servers"))

	// Also check for .mcp.json at root (indicates MCP-only plugin)
	mcpJSONPath := filepath.Join(installPath, ".mcp.json")
	if _, err := os.Stat(mcpJSONPath); err == nil && len(components.MCPs) == 0 {
		// Plugin has an MCP server defined at root level
		components.MCPs = append(components.MCPs, "mcp-server")
	}

	return components
}

// listSubdirectories returns names of immediate subdirectories.
func listSubdirectories(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			result = append(result, entry.Name())
		}
	}
	return result
}

// listMarkdownFiles returns names of .md files (without extension).
func listMarkdownFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var result []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			result = append(result, name)
		}
	}
	return result
}

// ProjectSettings represents the .claude/settings.json or .claude/settings.local.json file.
type ProjectSettings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// ReadProjectSettings reads the settings file at the given path.
func ReadProjectSettings(settingsPath string) (*ProjectSettings, error) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings ProjectSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// GetProjectEnabledPlugins returns the set of plugin IDs enabled for the given working directory.
// It reads both .claude/settings.json (project scope) and .claude/settings.local.json (local scope)
// and returns a map of plugin ID to scope.
func GetProjectEnabledPlugins(workingDir string) map[string]Scope {
	result := make(map[string]Scope)

	// Read project-scoped settings
	projectSettingsPath := filepath.Join(workingDir, ".claude", "settings.json")
	if settings, err := ReadProjectSettings(projectSettingsPath); err == nil {
		for pluginID, enabled := range settings.EnabledPlugins {
			if enabled {
				result[pluginID] = ScopeProject
			}
		}
	}

	// Read local-scoped settings (overrides project if present)
	localSettingsPath := filepath.Join(workingDir, ".claude", "settings.local.json")
	if settings, err := ReadProjectSettings(localSettingsPath); err == nil {
		for pluginID, enabled := range settings.EnabledPlugins {
			if enabled {
				result[pluginID] = ScopeLocal
			}
		}
	}

	return result
}

// ConfigFile represents a JSON configuration file found in a plugin.
type ConfigFile struct {
	RelativePath string // Path relative to plugin root (e.g., ".claude-plugin/plugin.json")
	Content      string // Pretty-printed JSON content
}

// ReadPluginConfigs reads all JSON configuration files from a plugin directory.
// Returns the manifest and any other config files found.
func ReadPluginConfigs(installPath string) ([]ConfigFile, error) {
	var configs []ConfigFile

	// Always include the manifest first if it exists
	manifestPath := filepath.Join(installPath, ".claude-plugin", "plugin.json")
	if content, err := readAndFormatJSON(manifestPath); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: ".claude-plugin/plugin.json",
			Content:      content,
		})
	}

	// Check for hooks.json
	hooksPath := filepath.Join(installPath, "hooks", "hooks.json")
	if content, err := readAndFormatJSON(hooksPath); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: "hooks/hooks.json",
			Content:      content,
		})
	}

	// Check for .mcp.json (MCP server config)
	mcpPath := filepath.Join(installPath, ".mcp.json")
	if content, err := readAndFormatJSON(mcpPath); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: ".mcp.json",
			Content:      content,
		})
	}

	// Scan mcp-servers/ for config files
	mcpServersDir := filepath.Join(installPath, "mcp-servers")
	if entries, err := os.ReadDir(mcpServersDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				configPath := filepath.Join(mcpServersDir, entry.Name(), "config.json")
				if content, err := readAndFormatJSON(configPath); err == nil {
					configs = append(configs, ConfigFile{
						RelativePath: filepath.Join("mcp-servers", entry.Name(), "config.json"),
						Content:      content,
					})
				}
			}
		}
	}

	if len(configs) == 0 {
		return nil, os.ErrNotExist
	}

	return configs, nil
}

// readAndFormatJSON reads a JSON file and returns it pretty-printed.
func readAndFormatJSON(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Parse and re-format for consistent pretty printing
	var parsed any
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// ResolveMarketplaceSourcePath resolves the full path to a plugin in a marketplace.
// It combines ~/.claude/plugins/marketplaces/<marketplace>/ with the source field.
func ResolveMarketplaceSourcePath(marketplace string, source any) string {
	// Get source as string - source can be string or object
	var sourcePath string
	switch s := source.(type) {
	case string:
		sourcePath = s
	default:
		// If source is not a string (e.g., an object), we can't resolve it
		return ""
	}

	if sourcePath == "" {
		return ""
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Clean up source path (remove leading ./)
	sourcePath = strings.TrimPrefix(sourcePath, "./")

	// Construct full path: ~/.claude/plugins/marketplaces/<marketplace>/<source>
	return filepath.Join(homeDir, ".claude", "plugins", "marketplaces", marketplace, sourcePath)
}
