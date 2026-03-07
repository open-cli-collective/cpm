package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestReadPluginManifestFS(t *testing.T) {
	t.Run("string author", func(t *testing.T) {
		fsys := fstest.MapFS{
			".claude-plugin/plugin.json": &fstest.MapFile{
				Data: []byte(`{"name":"test-plugin","description":"A test","version":"1.0.0","author":"Alice"}`),
			},
		}

		manifest, err := ReadPluginManifestFS(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifest.Name != "test-plugin" {
			t.Errorf("Name = %q, want %q", manifest.Name, "test-plugin")
		}
		if manifest.Description != "A test" {
			t.Errorf("Description = %q, want %q", manifest.Description, "A test")
		}
		if manifest.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", manifest.Version, "1.0.0")
		}
		if manifest.AuthorName != "Alice" {
			t.Errorf("AuthorName = %q, want %q", manifest.AuthorName, "Alice")
		}
	})

	t.Run("object author", func(t *testing.T) {
		fsys := fstest.MapFS{
			".claude-plugin/plugin.json": &fstest.MapFile{
				Data: []byte(`{"name":"test","description":"desc","author":{"name":"Bob","email":"bob@example.com"}}`),
			},
		}

		manifest, err := ReadPluginManifestFS(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if manifest.AuthorName != "Bob" {
			t.Errorf("AuthorName = %q, want %q", manifest.AuthorName, "Bob")
		}
		if manifest.AuthorEmail != "bob@example.com" {
			t.Errorf("AuthorEmail = %q, want %q", manifest.AuthorEmail, "bob@example.com")
		}
	})

	t.Run("missing manifest", func(t *testing.T) {
		fsys := fstest.MapFS{}

		_, err := ReadPluginManifestFS(fsys)
		if err == nil {
			t.Error("expected error for missing manifest, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		fsys := fstest.MapFS{
			".claude-plugin/plugin.json": &fstest.MapFile{
				Data: []byte(`{invalid`),
			},
		}

		_, err := ReadPluginManifestFS(fsys)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}

func TestScanPluginComponentsFS(t *testing.T) {
	t.Run("all component types", func(t *testing.T) {
		fsys := fstest.MapFS{
			"skills/my-skill/SKILL.md":      &fstest.MapFile{},
			"agents/helper.md":              &fstest.MapFile{},
			"commands/run.md":               &fstest.MapFile{},
			"hooks/pre-commit/hook.sh":      &fstest.MapFile{},
			"hooks/post-build.md":           &fstest.MapFile{},
			"mcp-servers/my-server/main.go": &fstest.MapFile{},
		}

		components := ScanPluginComponentsFS(fsys)
		if len(components.Skills) != 1 || components.Skills[0] != "my-skill" {
			t.Errorf("Skills = %v, want [my-skill]", components.Skills)
		}
		if len(components.Agents) != 1 || components.Agents[0] != "helper" {
			t.Errorf("Agents = %v, want [helper]", components.Agents)
		}
		if len(components.Commands) != 1 || components.Commands[0] != "run" {
			t.Errorf("Commands = %v, want [run]", components.Commands)
		}
		if len(components.Hooks) != 2 {
			t.Errorf("Hooks length = %d, want 2", len(components.Hooks))
		}
		if len(components.MCPs) != 1 || components.MCPs[0] != "my-server" {
			t.Errorf("MCPs = %v, want [my-server]", components.MCPs)
		}
	})

	t.Run("mcp.json fallback", func(t *testing.T) {
		fsys := fstest.MapFS{
			".mcp.json": &fstest.MapFile{Data: []byte(`{}`)},
		}

		components := ScanPluginComponentsFS(fsys)
		if len(components.MCPs) != 1 || components.MCPs[0] != "mcp-server" {
			t.Errorf("MCPs = %v, want [mcp-server]", components.MCPs)
		}
	})

	t.Run("empty plugin", func(t *testing.T) {
		fsys := fstest.MapFS{}

		components := ScanPluginComponentsFS(fsys)
		if len(components.Skills) != 0 || len(components.Agents) != 0 ||
			len(components.Commands) != 0 || len(components.Hooks) != 0 ||
			len(components.MCPs) != 0 {
			t.Errorf("expected empty components, got %+v", components)
		}
	})
}

func TestReadPluginConfigsFS(t *testing.T) {
	t.Run("manifest only", func(t *testing.T) {
		fsys := fstest.MapFS{
			".claude-plugin/plugin.json": &fstest.MapFile{
				Data: []byte(`{"name":"test"}`),
			},
		}

		configs, err := ReadPluginConfigsFS(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(configs) != 1 {
			t.Fatalf("len(configs) = %d, want 1", len(configs))
		}
		if configs[0].RelativePath != ".claude-plugin/plugin.json" {
			t.Errorf("RelativePath = %q, want %q", configs[0].RelativePath, ".claude-plugin/plugin.json")
		}
	})

	t.Run("multiple configs", func(t *testing.T) {
		fsys := fstest.MapFS{
			".claude-plugin/plugin.json": &fstest.MapFile{
				Data: []byte(`{"name":"test"}`),
			},
			"hooks/hooks.json": &fstest.MapFile{
				Data: []byte(`{"hooks":[]}`),
			},
			".mcp.json": &fstest.MapFile{
				Data: []byte(`{"server":"test"}`),
			},
		}

		configs, err := ReadPluginConfigsFS(fsys)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(configs) != 3 {
			t.Errorf("len(configs) = %d, want 3", len(configs))
		}
	})

	t.Run("no configs", func(t *testing.T) {
		fsys := fstest.MapFS{}

		_, err := ReadPluginConfigsFS(fsys)
		if err == nil {
			t.Error("expected error for no configs, got nil")
		}
	})
}

func TestReadSettingsFromFS(t *testing.T) {
	t.Run("valid settings", func(t *testing.T) {
		fsys := fstest.MapFS{
			"settings.json": &fstest.MapFile{
				Data: []byte(`{"enabledPlugins":{"plugin-a@mp":true,"plugin-b@mp":false}}`),
			},
		}

		settings, err := readSettingsFromFS(fsys, "settings.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !settings.EnabledPlugins["plugin-a@mp"] {
			t.Error("plugin-a@mp should be enabled")
		}
		if settings.EnabledPlugins["plugin-b@mp"] {
			t.Error("plugin-b@mp should be disabled")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		fsys := fstest.MapFS{}

		_, err := readSettingsFromFS(fsys, "settings.json")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

// TestGetAllEnabledPluginsUserScope verifies AC2.1: user-scoped plugins
func TestGetAllEnabledPluginsUserScope(t *testing.T) {
	// Create temp directories
	tmpHome, err := os.MkdirTemp("", "claude-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpWork, err := os.MkdirTemp("", "claude-work-*")
	if err != nil {
		t.Fatalf("Failed to create temp work: %v", err)
	}
	defer os.RemoveAll(tmpWork)

	// Create user settings file
	userSettingsDir := filepath.Join(tmpHome, ".claude")
	if err = os.MkdirAll(userSettingsDir, 0o755); err != nil {
		t.Fatalf("Failed to create user settings dir: %v", err)
	}

	userSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-a@mp": true,
		},
	}
	userSettingsData, err := json.Marshal(userSettings)
	if err != nil {
		t.Fatalf("Failed to marshal user settings: %v", err)
	}

	userSettingsPath := filepath.Join(userSettingsDir, "settings.json")
	if err = os.WriteFile(userSettingsPath, userSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write user settings: %v", err)
	}

	// Call getAllEnabledPlugins with temp dirs
	result := getAllEnabledPlugins(tmpWork, tmpHome)

	// Verify plugin-a@mp is present with ScopeUser
	if _, ok := result["plugin-a@mp"]; !ok {
		t.Errorf("plugin-a@mp not found in result")
	}
	if scopes, ok := result["plugin-a@mp"]; ok {
		if enabled, ok := scopes[ScopeUser]; !ok {
			t.Errorf("ScopeUser not found for plugin-a@mp")
		} else if !enabled {
			t.Errorf("plugin-a@mp ScopeUser = %v, want true", enabled)
		}
	}
}

// TestGetAllEnabledPluginsProjectAndLocalScope verifies AC2.2: project and local scopes
func TestGetAllEnabledPluginsProjectAndLocalScope(t *testing.T) {
	// Create temp directories
	tmpHome, err := os.MkdirTemp("", "claude-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpWork, err := os.MkdirTemp("", "claude-work-*")
	if err != nil {
		t.Fatalf("Failed to create temp work: %v", err)
	}
	defer os.RemoveAll(tmpWork)

	// Create project settings directory
	projectSettingsDir := filepath.Join(tmpWork, ".claude")
	if err = os.MkdirAll(projectSettingsDir, 0o755); err != nil {
		t.Fatalf("Failed to create project settings dir: %v", err)
	}

	// Create project settings file
	projectSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-b@mp": true,
		},
	}
	projectSettingsData, err := json.Marshal(projectSettings)
	if err != nil {
		t.Fatalf("Failed to marshal project settings: %v", err)
	}

	projectSettingsPath := filepath.Join(projectSettingsDir, "settings.json")
	if err = os.WriteFile(projectSettingsPath, projectSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write project settings: %v", err)
	}

	// Create local settings file
	localSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-c@mp": true,
		},
	}
	localSettingsData, err := json.Marshal(localSettings)
	if err != nil {
		t.Fatalf("Failed to marshal local settings: %v", err)
	}

	localSettingsPath := filepath.Join(projectSettingsDir, "settings.local.json")
	if err = os.WriteFile(localSettingsPath, localSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write local settings: %v", err)
	}

	// Call getAllEnabledPlugins
	result := getAllEnabledPlugins(tmpWork, tmpHome)

	// Verify plugin-b@mp has ScopeProject
	if _, ok := result["plugin-b@mp"]; !ok {
		t.Errorf("plugin-b@mp not found in result")
	}
	if scopes, ok := result["plugin-b@mp"]; ok {
		if enabled, ok := scopes[ScopeProject]; !ok {
			t.Errorf("ScopeProject not found for plugin-b@mp")
		} else if !enabled {
			t.Errorf("plugin-b@mp ScopeProject = %v, want true", enabled)
		}
	}

	// Verify plugin-c@mp has ScopeLocal
	if _, ok := result["plugin-c@mp"]; !ok {
		t.Errorf("plugin-c@mp not found in result")
	}
	if scopes, ok := result["plugin-c@mp"]; ok {
		if enabled, ok := scopes[ScopeLocal]; !ok {
			t.Errorf("ScopeLocal not found for plugin-c@mp")
		} else if !enabled {
			t.Errorf("plugin-c@mp ScopeLocal = %v, want true", enabled)
		}
	}
}

// TestGetAllEnabledPluginsMissingFiles verifies AC2.3: gracefully handle missing files
func TestGetAllEnabledPluginsMissingFiles(t *testing.T) {
	// Create temp directories with no settings files
	tmpHome, err := os.MkdirTemp("", "claude-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpWork, err := os.MkdirTemp("", "claude-work-*")
	if err != nil {
		t.Fatalf("Failed to create temp work: %v", err)
	}
	defer os.RemoveAll(tmpWork)

	// Call getAllEnabledPlugins with no settings files
	result := getAllEnabledPlugins(tmpWork, tmpHome)

	// Verify empty map returned, no error/panic
	if result == nil {
		t.Errorf("result is nil, want empty ScopeState")
	}
	if len(result) != 0 {
		t.Errorf("result length = %d, want 0", len(result))
	}
}

// TestGetAllEnabledPluginsMultipleScopes verifies AC2.4: plugin in multiple scopes
func TestGetAllEnabledPluginsMultipleScopes(t *testing.T) {
	// Create temp directories
	tmpHome, err := os.MkdirTemp("", "claude-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpWork, err := os.MkdirTemp("", "claude-work-*")
	if err != nil {
		t.Fatalf("Failed to create temp work: %v", err)
	}
	defer os.RemoveAll(tmpWork)

	// Create user settings with plugin-x@mp
	userSettingsDir := filepath.Join(tmpHome, ".claude")
	if err = os.MkdirAll(userSettingsDir, 0o755); err != nil {
		t.Fatalf("Failed to create user settings dir: %v", err)
	}

	userSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-x@mp": true,
		},
	}
	userSettingsData, err := json.Marshal(userSettings)
	if err != nil {
		t.Fatalf("Failed to marshal user settings: %v", err)
	}

	userSettingsPath := filepath.Join(userSettingsDir, "settings.json")
	if err = os.WriteFile(userSettingsPath, userSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write user settings: %v", err)
	}

	// Create local settings with plugin-x@mp
	projectSettingsDir := filepath.Join(tmpWork, ".claude")
	if err = os.MkdirAll(projectSettingsDir, 0o755); err != nil {
		t.Fatalf("Failed to create project settings dir: %v", err)
	}

	localSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-x@mp": false, // disabled in local scope
		},
	}
	localSettingsData, err := json.Marshal(localSettings)
	if err != nil {
		t.Fatalf("Failed to marshal local settings: %v", err)
	}

	localSettingsPath := filepath.Join(projectSettingsDir, "settings.local.json")
	if err = os.WriteFile(localSettingsPath, localSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write local settings: %v", err)
	}

	// Call getAllEnabledPlugins
	result := getAllEnabledPlugins(tmpWork, tmpHome)

	// Verify plugin-x@mp appears in both ScopeUser and ScopeLocal
	if _, ok := result["plugin-x@mp"]; !ok {
		t.Errorf("plugin-x@mp not found in result")
	}
	if scopes, ok := result["plugin-x@mp"]; ok {
		if enabled, ok := scopes[ScopeUser]; !ok {
			t.Errorf("ScopeUser not found for plugin-x@mp")
		} else if !enabled {
			t.Errorf("plugin-x@mp ScopeUser = %v, want true", enabled)
		}

		if enabled, ok := scopes[ScopeLocal]; !ok {
			t.Errorf("ScopeLocal not found for plugin-x@mp")
		} else if enabled {
			t.Errorf("plugin-x@mp ScopeLocal = %v, want false", enabled)
		}
	}
}

// TestGetAllEnabledPluginsDisabledButPresent verifies disabled plugins are included
func TestGetAllEnabledPluginsDisabledButPresent(t *testing.T) {
	// Create temp directories
	tmpHome, err := os.MkdirTemp("", "claude-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	tmpWork, err := os.MkdirTemp("", "claude-work-*")
	if err != nil {
		t.Fatalf("Failed to create temp work: %v", err)
	}
	defer os.RemoveAll(tmpWork)

	// Create project settings with disabled plugin
	projectSettingsDir := filepath.Join(tmpWork, ".claude")
	if err = os.MkdirAll(projectSettingsDir, 0o755); err != nil {
		t.Fatalf("Failed to create project settings dir: %v", err)
	}

	projectSettings := ProjectSettings{
		EnabledPlugins: map[string]bool{
			"plugin-d@mp": false, // disabled
		},
	}
	projectSettingsData, err := json.Marshal(projectSettings)
	if err != nil {
		t.Fatalf("Failed to marshal project settings: %v", err)
	}

	projectSettingsPath := filepath.Join(projectSettingsDir, "settings.json")
	if err = os.WriteFile(projectSettingsPath, projectSettingsData, 0o644); err != nil {
		t.Fatalf("Failed to write project settings: %v", err)
	}

	// Call getAllEnabledPlugins
	result := getAllEnabledPlugins(tmpWork, tmpHome)

	// Verify plugin-d@mp is present even though disabled
	if _, ok := result["plugin-d@mp"]; !ok {
		t.Errorf("plugin-d@mp not found in result")
	}
	if scopes, ok := result["plugin-d@mp"]; ok {
		if enabled, ok := scopes[ScopeProject]; !ok {
			t.Errorf("ScopeProject not found for plugin-d@mp")
		} else if enabled {
			t.Errorf("plugin-d@mp ScopeProject = %v, want false", enabled)
		}
	}
}

func TestMarketplaceNameFromPluginID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"plugin-a@ed3d-plugins", "ed3d-plugins"},
		{"context7@claude-plugins-official", "claude-plugins-official"},
		{"no-marketplace", ""},
		{"multi@at@signs", "signs"},
	}
	for _, tt := range tests {
		got := MarketplaceNameFromPluginID(tt.id)
		if got != tt.want {
			t.Errorf("MarketplaceNameFromPluginID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestSettingsPathForScope(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeProject, "/work/.claude/settings.json"},
		{ScopeLocal, "/work/.claude/settings.local.json"},
		{ScopeUser, ""},
		{ScopeNone, ""},
	}
	for _, tt := range tests {
		got := SettingsPathForScope("/work", tt.scope)
		if got != tt.want {
			t.Errorf("SettingsPathForScope(%q) = %q, want %q", tt.scope, got, tt.want)
		}
	}
}

// setupTempDir creates a temp directory and returns its path with a cleanup function.
func setupTempDir(t *testing.T, pattern string) string {
	t.Helper()
	tmp, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmp) })
	return tmp
}

// setupClaudeDir creates a .claude directory inside tmp and writes a settings file.
func setupClaudeDir(t *testing.T, tmp, settingsContent string) string {
	t.Helper()
	claudeDir := filepath.Join(tmp, ".claude")
	if mkErr := os.MkdirAll(claudeDir, 0o750); mkErr != nil {
		t.Fatal(mkErr)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if writeErr := os.WriteFile(settingsPath, []byte(settingsContent), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	return settingsPath
}

// readRawSettingsFile reads a settings file and returns it as a raw JSON map.
func readRawSettingsFile(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var raw map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	return raw
}

func TestReadKnownMarketplaces(t *testing.T) {
	tmp := setupTempDir(t, "known-mp-*")

	data := `{
		"ed3d-plugins": {
			"source": {"source": "github", "repo": "ed3dai/ed3d-plugins"},
			"installLocation": "/test/path",
			"lastUpdated": "2026-01-01T00:00:00Z",
			"autoUpdate": true
		}
	}`
	writeErr := os.WriteFile(filepath.Join(tmp, "known_marketplaces.json"), []byte(data), 0o644)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	result, readErr := readKnownMarketplaces(tmp)
	if readErr != nil {
		t.Fatalf("readKnownMarketplaces: %v", readErr)
	}

	km, ok := result["ed3d-plugins"]
	if !ok {
		t.Fatal("ed3d-plugins not found")
	}
	gh, ok := km.Source.(*GitHubSource)
	if !ok {
		t.Fatalf("Source is %T, want *GitHubSource", km.Source)
	}
	if gh.Repo != "ed3dai/ed3d-plugins" {
		t.Errorf("Repo = %q, want %q", gh.Repo, "ed3dai/ed3d-plugins")
	}
}

func TestSyncExtraMarketplacesAddsEntry(t *testing.T) {
	tmp := setupTempDir(t, "sync-mp-*")
	settingsPath := setupClaudeDir(t, tmp, `{"enabledPlugins":{"my-plugin@ed3d-plugins":true}}`)

	known := map[string]KnownMarketplace{
		"ed3d-plugins": {Source: GitHubSource{Repo: "ed3dai/ed3d-plugins"}},
	}

	syncErr := SyncExtraMarketplaces(settingsPath, known)
	if syncErr != nil {
		t.Fatalf("SyncExtraMarketplaces: %v", syncErr)
	}

	raw := readRawSettingsFile(t, settingsPath)

	extraRaw, ok := raw["extraKnownMarketplaces"]
	if !ok {
		t.Fatal("extraKnownMarketplaces not found in settings")
	}

	var extra map[string]MarketplaceEntry
	unmarshalErr := json.Unmarshal(extraRaw, &extra)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}

	entry, ok := extra["ed3d-plugins"]
	if !ok {
		t.Fatal("ed3d-plugins not found in extraKnownMarketplaces")
	}
	gh, ok := entry.Source.(*GitHubSource)
	if !ok {
		t.Fatalf("Source is %T, want *GitHubSource", entry.Source)
	}
	if gh.Repo != "ed3dai/ed3d-plugins" {
		t.Errorf("Repo = %q, want %q", gh.Repo, "ed3dai/ed3d-plugins")
	}
}

func TestSyncExtraMarketplacesRemovesEntry(t *testing.T) {
	tmp := setupTempDir(t, "sync-mp-*")
	settingsPath := setupClaudeDir(t, tmp,
		`{"enabledPlugins":{},"extraKnownMarketplaces":{"ed3d-plugins":{"source":{"source":"github","repo":"ed3dai/ed3d-plugins"}}}}`)

	known := map[string]KnownMarketplace{
		"ed3d-plugins": {Source: GitHubSource{Repo: "ed3dai/ed3d-plugins"}},
	}

	syncErr := SyncExtraMarketplaces(settingsPath, known)
	if syncErr != nil {
		t.Fatalf("SyncExtraMarketplaces: %v", syncErr)
	}

	raw := readRawSettingsFile(t, settingsPath)
	if _, ok := raw["extraKnownMarketplaces"]; ok {
		t.Error("extraKnownMarketplaces should have been removed")
	}
}

func TestSyncExtraMarketplacesPreservesUnknownFields(t *testing.T) {
	tmp := setupTempDir(t, "sync-mp-*")
	settingsPath := setupClaudeDir(t, tmp,
		`{"enabledPlugins":{"p@mp":true},"env":{"FOO":"bar"},"permissions":{"allow":["*"]}}`)

	known := map[string]KnownMarketplace{
		"mp": {Source: GitHubSource{Repo: "owner/repo"}},
	}

	syncErr := SyncExtraMarketplaces(settingsPath, known)
	if syncErr != nil {
		t.Fatal(syncErr)
	}

	raw := readRawSettingsFile(t, settingsPath)
	if _, ok := raw["env"]; !ok {
		t.Error("env field was not preserved")
	}
	if _, ok := raw["permissions"]; !ok {
		t.Error("permissions field was not preserved")
	}
	if _, ok := raw["extraKnownMarketplaces"]; !ok {
		t.Error("extraKnownMarketplaces should have been added")
	}
}

func TestSyncExtraMarketplacesIdempotent(t *testing.T) {
	tmp := setupTempDir(t, "sync-mp-*")
	settingsPath := setupClaudeDir(t, tmp,
		`{"enabledPlugins":{"p@mp":true},"extraKnownMarketplaces":{"mp":{"source":{"source":"github","repo":"owner/repo"}}}}`)

	known := map[string]KnownMarketplace{
		"mp": {Source: GitHubSource{Repo: "owner/repo"}},
	}

	infoBefore, _ := os.Stat(settingsPath)

	syncErr := SyncExtraMarketplaces(settingsPath, known)
	if syncErr != nil {
		t.Fatal(syncErr)
	}

	infoAfter, _ := os.Stat(settingsPath)
	if !infoBefore.ModTime().Equal(infoAfter.ModTime()) {
		t.Error("file was rewritten despite no changes needed")
	}
}

func TestSyncExtraMarketplacesCreatesFile(t *testing.T) {
	tmp := setupTempDir(t, "sync-mp-*")
	settingsPath := filepath.Join(tmp, ".claude", "settings.json")

	known := map[string]KnownMarketplace{
		"mp": {Source: GitHubSource{Repo: "owner/repo"}},
	}

	syncErr := SyncExtraMarketplaces(settingsPath, known)
	if syncErr != nil {
		t.Fatal(syncErr)
	}

	if _, statErr := os.Stat(settingsPath); statErr == nil {
		t.Error("file should not have been created when there are no plugins")
	}
}
