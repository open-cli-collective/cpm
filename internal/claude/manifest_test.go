package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
