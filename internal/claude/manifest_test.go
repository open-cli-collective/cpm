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
