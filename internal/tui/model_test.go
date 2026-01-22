package tui

import (
	"testing"

	"github.com/open-cli-collective/cpm/internal/claude"
)

func TestNewModel(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)

	if m.client == nil {
		t.Error("client is nil")
	}
	if !m.loading {
		t.Error("loading should be true initially")
	}
	if m.err != nil {
		t.Error("err should be nil initially")
	}
}

func TestPluginStateFromInstalled(t *testing.T) {
	installed := claude.InstalledPlugin{
		ID:      "test@marketplace",
		Version: "1.0.0",
		Scope:   claude.ScopeProject,
		Enabled: true,
	}

	state := PluginStateFromInstalled(installed)

	if state.ID != "test@marketplace" {
		t.Errorf("ID = %q, want %q", state.ID, "test@marketplace")
	}
	if state.InstalledScope != claude.ScopeProject {
		t.Errorf("InstalledScope = %q, want %q", state.InstalledScope, claude.ScopeProject)
	}
	if !state.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestPluginStateFromAvailable(t *testing.T) {
	available := claude.AvailablePlugin{
		PluginID:        "test@marketplace",
		Name:            "test",
		Description:     "A test plugin",
		MarketplaceName: "marketplace",
	}

	state := PluginStateFromAvailable(available)

	if state.ID != "test@marketplace" {
		t.Errorf("ID = %q, want %q", state.ID, "test@marketplace")
	}
	if state.Name != "test" {
		t.Errorf("Name = %q, want %q", state.Name, "test")
	}
	if state.InstalledScope != claude.ScopeNone {
		t.Errorf("InstalledScope = %q, want empty", state.InstalledScope)
	}
}

// mockClient implements claude.Client for testing
type mockClient struct {
	plugins *claude.PluginList
	err     error
}

func (m *mockClient) ListPlugins(includeAvailable bool) (*claude.PluginList, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.plugins != nil {
		return m.plugins, nil
	}
	return &claude.PluginList{}, nil
}

func (m *mockClient) InstallPlugin(pluginID string, scope claude.Scope) error {
	return m.err
}

func (m *mockClient) UninstallPlugin(pluginID string, scope claude.Scope) error {
	return m.err
}
