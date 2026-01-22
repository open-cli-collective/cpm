// Package claude provides a client for interacting with the Claude Code CLI.
package claude

// Scope represents the installation scope of a plugin.
type Scope string

const (
	// ScopeNone indicates no scope (used for uninstall operations).
	ScopeNone Scope = ""
	// ScopeUser is the global user scope (~/.claude/settings.json).
	ScopeUser Scope = "user"
	// ScopeProject is the shared project scope (.claude/settings.json).
	ScopeProject Scope = "project"
	// ScopeLocal is the local project scope (.claude/settings.local.json).
	ScopeLocal Scope = "local"
)

// InstalledPlugin represents a plugin that is currently installed.
type InstalledPlugin struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	InstallPath string `json:"installPath"`
	InstalledAt string `json:"installedAt"`
	LastUpdated string `json:"lastUpdated"`
	ProjectPath string `json:"projectPath,omitempty"`
	Scope       Scope  `json:"scope"`
	Enabled     bool   `json:"enabled"`
}

// AvailablePlugin represents a plugin available from a marketplace.
type AvailablePlugin struct {
	PluginID        string `json:"pluginId"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	MarketplaceName string `json:"marketplaceName"`
	Source          any    `json:"source,omitempty"` // Can be string or object depending on plugin type
	Version         string `json:"version,omitempty"`
	InstallCount    int    `json:"installCount,omitempty"`
}

// PluginList is the response from `claude plugin list --json --available`.
type PluginList struct {
	Installed []InstalledPlugin `json:"installed"`
	Available []AvailablePlugin `json:"available"`
}
