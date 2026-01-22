package claude

import (
	"encoding/json"
	"testing"
)

func TestScopeConstants(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeNone, ""},
		{ScopeUser, "user"},
		{ScopeProject, "project"},
		{ScopeLocal, "local"},
	}

	for _, tt := range tests {
		if string(tt.scope) != tt.want {
			t.Errorf("Scope %v = %q, want %q", tt.scope, tt.scope, tt.want)
		}
	}
}

func TestInstalledPluginJSON(t *testing.T) {
	jsonData := `{
		"id": "context7@claude-plugins-official",
		"version": "e30768372b41",
		"scope": "user",
		"enabled": true,
		"installPath": "/Users/test/.claude/plugins/cache/claude-plugins-official/context7/e30768372b41",
		"installedAt": "2026-01-16T02:46:08.054Z",
		"lastUpdated": "2026-01-22T01:25:12.553Z"
	}`

	var plugin InstalledPlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.ID != "context7@claude-plugins-official" {
		t.Errorf("ID = %q, want %q", plugin.ID, "context7@claude-plugins-official")
	}
	if plugin.Scope != ScopeUser {
		t.Errorf("Scope = %q, want %q", plugin.Scope, ScopeUser)
	}
	if !plugin.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestInstalledPluginWithProjectPath(t *testing.T) {
	jsonData := `{
		"id": "ed3d-basic-agents@ed3d-plugins",
		"version": "1.0.0",
		"scope": "project",
		"enabled": false,
		"installPath": "/Users/test/.claude/plugins/cache/ed3d-plugins/ed3d-basic-agents/1.0.0",
		"installedAt": "2026-01-16T02:46:08.684Z",
		"lastUpdated": "2026-01-16T02:46:08.684Z",
		"projectPath": "/Users/test/Code/myproject"
	}`

	var plugin InstalledPlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.Scope != ScopeProject {
		t.Errorf("Scope = %q, want %q", plugin.Scope, ScopeProject)
	}
	if plugin.ProjectPath != "/Users/test/Code/myproject" {
		t.Errorf("ProjectPath = %q, want %q", plugin.ProjectPath, "/Users/test/Code/myproject")
	}
}

func TestAvailablePluginJSON(t *testing.T) {
	jsonData := `{
		"pluginId": "github@claude-plugins-official",
		"name": "github",
		"description": "Official GitHub MCP server for repository management.",
		"marketplaceName": "claude-plugins-official",
		"source": "./external_plugins/github",
		"installCount": 47711
	}`

	var plugin AvailablePlugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if plugin.PluginID != "github@claude-plugins-official" {
		t.Errorf("PluginID = %q, want %q", plugin.PluginID, "github@claude-plugins-official")
	}
	if plugin.Name != "github" {
		t.Errorf("Name = %q, want %q", plugin.Name, "github")
	}
	if plugin.MarketplaceName != "claude-plugins-official" {
		t.Errorf("MarketplaceName = %q, want %q", plugin.MarketplaceName, "claude-plugins-official")
	}
	// Source field should be captured (can be string or object in CLI output)
	if plugin.Source == nil {
		t.Error("Source should not be nil")
	}
}

func TestPluginListJSON(t *testing.T) {
	jsonData := `{
		"installed": [
			{
				"id": "context7@claude-plugins-official",
				"version": "e30768372b41",
				"scope": "user",
				"enabled": true,
				"installPath": "/test/path",
				"installedAt": "2026-01-16T02:46:08.054Z",
				"lastUpdated": "2026-01-22T01:25:12.553Z"
			}
		],
		"available": [
			{
				"pluginId": "github@claude-plugins-official",
				"name": "github",
				"description": "GitHub integration",
				"marketplaceName": "claude-plugins-official",
				"source": "./external_plugins/github",
				"installCount": 47711
			}
		]
	}`

	var list PluginList
	if err := json.Unmarshal([]byte(jsonData), &list); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(list.Installed) != 1 {
		t.Errorf("len(Installed) = %d, want 1", len(list.Installed))
	}
	if len(list.Available) != 1 {
		t.Errorf("len(Available) = %d, want 1", len(list.Available))
	}
}
