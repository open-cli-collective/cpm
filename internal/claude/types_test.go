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

func TestMarketplaceSourceRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		sourceType string
		input      string
	}{
		{"github", "github", `{"source":"github","repo":"owner/repo","ref":"main","path":"sub"}`},
		{"github minimal", "github", `{"source":"github","repo":"owner/repo"}`},
		{"git", "git", `{"source":"git","url":"https://example.com/repo.git","ref":"v1"}`},
		{"url", "url", `{"source":"url","url":"https://example.com/marketplace.json"}`},
		{"url with headers", "url", `{"source":"url","url":"https://example.com","headers":{"Authorization":"Bearer tok"}}`},
		{"npm", "npm", `{"source":"npm","package":"@scope/pkg"}`},
		{"file", "file", `{"source":"file","path":"/tmp/marketplace.json"}`},
		{"directory", "directory", `{"source":"directory","path":"/tmp/marketplace"}`},
		{"hostPattern", "hostPattern", `{"source":"hostPattern","hostPattern":"*.example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := unmarshalSource([]byte(tt.input))
			if err != nil {
				t.Fatalf("unmarshalSource: %v", err)
			}
			if src.SourceType() != tt.sourceType {
				t.Errorf("SourceType() = %q, want %q", src.SourceType(), tt.sourceType)
			}

			// Round-trip through marshalSource
			data, err := marshalSource(src)
			if err != nil {
				t.Fatalf("marshalSource: %v", err)
			}
			src2, err := unmarshalSource(data)
			if err != nil {
				t.Fatalf("unmarshalSource round-trip: %v", err)
			}
			if src2.SourceType() != tt.sourceType {
				t.Errorf("round-trip SourceType() = %q, want %q", src2.SourceType(), tt.sourceType)
			}
		})
	}
}

func TestUnmarshalSourceUnknownType(t *testing.T) {
	_, err := unmarshalSource([]byte(`{"source":"unknown"}`))
	if err == nil {
		t.Error("expected error for unknown source type, got nil")
	}
}

func TestMarketplaceEntryJSON(t *testing.T) {
	entry := MarketplaceEntry{
		Source: GitHubSource{Repo: "owner/repo"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded MarketplaceEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	gh, ok := decoded.Source.(*GitHubSource)
	if !ok {
		t.Fatalf("Source is %T, want *GitHubSource", decoded.Source)
	}
	if gh.Repo != "owner/repo" {
		t.Errorf("Repo = %q, want %q", gh.Repo, "owner/repo")
	}
}

func TestMarketplaceEntryUnmarshalReal(t *testing.T) {
	// Real format from extraKnownMarketplaces
	input := `{"source":{"source":"github","repo":"ed3dai/ed3d-plugins"}}`

	var entry MarketplaceEntry
	if err := json.Unmarshal([]byte(input), &entry); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	gh, ok := entry.Source.(*GitHubSource)
	if !ok {
		t.Fatalf("Source is %T, want *GitHubSource", entry.Source)
	}
	if gh.Repo != "ed3dai/ed3d-plugins" {
		t.Errorf("Repo = %q, want %q", gh.Repo, "ed3dai/ed3d-plugins")
	}
}

func TestKnownMarketplaceUnmarshal(t *testing.T) {
	input := `{
		"source": {"source": "github", "repo": "anthropics/claude-plugins-official"},
		"installLocation": "/home/test/.claude/plugins/marketplaces/claude-plugins-official",
		"lastUpdated": "2026-02-28T12:55:10.957Z",
		"autoUpdate": true
	}`

	var km KnownMarketplace
	if err := json.Unmarshal([]byte(input), &km); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	gh, ok := km.Source.(*GitHubSource)
	if !ok {
		t.Fatalf("Source is %T, want *GitHubSource", km.Source)
	}
	if gh.Repo != "anthropics/claude-plugins-official" {
		t.Errorf("Repo = %q, want %q", gh.Repo, "anthropics/claude-plugins-official")
	}
	if km.InstallLocation != "/home/test/.claude/plugins/marketplaces/claude-plugins-official" {
		t.Errorf("InstallLocation = %q", km.InstallLocation)
	}
	if !km.AutoUpdate {
		t.Error("AutoUpdate = false, want true")
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
