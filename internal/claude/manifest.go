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
