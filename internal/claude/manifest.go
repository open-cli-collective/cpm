package claude

import (
	"encoding/json"
	"io/fs"
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
	root, err := os.OpenRoot(installPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	return ReadPluginManifestFS(root.FS())
}

// ReadPluginManifestFS reads the plugin.json manifest from the given filesystem.
func ReadPluginManifestFS(fsys fs.FS) (*PluginManifest, error) {
	data, err := fs.ReadFile(fsys, filepath.Join(".claude-plugin", "plugin.json"))
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
	root, err := os.OpenRoot(installPath)
	if err != nil {
		return &PluginComponents{}
	}
	defer func() { _ = root.Close() }()

	return ScanPluginComponentsFS(root.FS())
}

// ScanPluginComponentsFS scans the filesystem for skills, agents, commands, hooks, and MCPs.
func ScanPluginComponentsFS(fsys fs.FS) *PluginComponents {
	components := &PluginComponents{}

	// Scan skills/ directory (subdirectories are skill names)
	components.Skills = listSubdirectoriesFS(fsys, "skills")

	// Scan agents/ directory (.md files are agent definitions)
	components.Agents = listMarkdownFilesFS(fsys, "agents")

	// Scan commands/ directory (.md files are command definitions)
	components.Commands = listMarkdownFilesFS(fsys, "commands")

	// Scan hooks/ directory (can be subdirectories or .md files)
	hooks := listSubdirectoriesFS(fsys, "hooks")
	hooks = append(hooks, listMarkdownFilesFS(fsys, "hooks")...)
	components.Hooks = hooks

	// Scan mcp-servers/ directory (subdirectories are MCP server names)
	components.MCPs = listSubdirectoriesFS(fsys, "mcp-servers")

	// Also check for .mcp.json at root (indicates MCP-only plugin)
	if _, err := fs.Stat(fsys, ".mcp.json"); err == nil && len(components.MCPs) == 0 {
		// Plugin has an MCP server defined at root level
		components.MCPs = append(components.MCPs, "mcp-server")
	}

	return components
}

// listSubdirectoriesFS returns names of immediate subdirectories within an fs.FS.
func listSubdirectoriesFS(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
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

// listMarkdownFilesFS returns names of .md files (without extension) within an fs.FS.
func listMarkdownFilesFS(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
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
	dir := filepath.Dir(settingsPath)
	file := filepath.Base(settingsPath)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	return readSettingsFromFS(root.FS(), file)
}

// readSettingsFromFS reads a settings file from the given filesystem.
func readSettingsFromFS(fsys fs.FS, name string) (*ProjectSettings, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}

	var settings ProjectSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// ScopeState tracks the enabled state of a plugin at a specific scope.
// The outer map key is the plugin ID, the inner map key is the scope,
// and the bool value is true=enabled, false=disabled-but-present.
type ScopeState map[string]map[Scope]bool

// GetAllEnabledPlugins reads all three settings files (user, project, local)
// and returns a map of plugin ID to scope set with enabled state.
// Missing settings files are silently ignored.
func GetAllEnabledPlugins(workingDir string) ScopeState {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "" // Will fail to read user settings, which is handled gracefully
	}
	return getAllEnabledPlugins(workingDir, homeDir)
}

// getAllEnabledPlugins is the internal implementation with injectable homeDir for testing.
func getAllEnabledPlugins(workingDir, homeDir string) ScopeState {
	result := make(ScopeState)

	// Helper to accumulate plugins from a .claude directory root
	addFromRoot := func(claudeDir string, entries []struct {
		file  string
		scope Scope
	},
	) {
		root, err := os.OpenRoot(claudeDir)
		if err != nil {
			return // Directory doesn't exist — skip silently
		}
		defer func() { _ = root.Close() }()

		rootFS := root.FS()
		for _, e := range entries {
			settings, err := readSettingsFromFS(rootFS, e.file)
			if err != nil {
				continue // Missing or unreadable file — skip silently
			}
			for pluginID, enabled := range settings.EnabledPlugins {
				if result[pluginID] == nil {
					result[pluginID] = make(map[Scope]bool)
				}
				result[pluginID][e.scope] = enabled
			}
		}
	}

	// User scope: {homeDir}/.claude/settings.json
	if homeDir != "" {
		addFromRoot(filepath.Join(homeDir, ".claude"), []struct {
			file  string
			scope Scope
		}{
			{"settings.json", ScopeUser},
		})
	}

	// Project + local scope: {workingDir}/.claude/settings.json and settings.local.json
	addFromRoot(filepath.Join(workingDir, ".claude"), []struct {
		file  string
		scope Scope
	}{
		{"settings.json", ScopeProject},
		{"settings.local.json", ScopeLocal},
	})

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
	root, err := os.OpenRoot(installPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	return ReadPluginConfigsFS(root.FS())
}

// ReadPluginConfigsFS reads all JSON configuration files from the given filesystem.
func ReadPluginConfigsFS(fsys fs.FS) ([]ConfigFile, error) {
	var configs []ConfigFile

	// Always include the manifest first if it exists
	if content, err := readAndFormatJSONFS(fsys, filepath.Join(".claude-plugin", "plugin.json")); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: ".claude-plugin/plugin.json",
			Content:      content,
		})
	}

	// Check for hooks.json
	if content, err := readAndFormatJSONFS(fsys, filepath.Join("hooks", "hooks.json")); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: "hooks/hooks.json",
			Content:      content,
		})
	}

	// Check for .mcp.json (MCP server config)
	if content, err := readAndFormatJSONFS(fsys, ".mcp.json"); err == nil {
		configs = append(configs, ConfigFile{
			RelativePath: ".mcp.json",
			Content:      content,
		})
	}

	// Scan mcp-servers/ for config files
	if entries, err := fs.ReadDir(fsys, "mcp-servers"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				relPath := filepath.Join("mcp-servers", entry.Name(), "config.json")
				if content, err := readAndFormatJSONFS(fsys, relPath); err == nil {
					configs = append(configs, ConfigFile{
						RelativePath: relPath,
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

// readAndFormatJSONFS reads a JSON file from the filesystem and returns it pretty-printed.
func readAndFormatJSONFS(fsys fs.FS, name string) (string, error) {
	data, err := fs.ReadFile(fsys, name)
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
