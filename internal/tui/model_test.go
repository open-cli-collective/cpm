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

func (m *mockClient) ListPlugins(_ bool) (*claude.PluginList, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.plugins != nil {
		return m.plugins, nil
	}
	return &claude.PluginList{}, nil
}

func (m *mockClient) InstallPlugin(_ string, _ claude.Scope) error {
	return m.err
}

func (m *mockClient) UninstallPlugin(_ string, _ claude.Scope) error {
	return m.err
}

func TestSelectForInstallLocal(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeLocal)

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeLocal {
		t.Errorf("pending = %v, want local scope", m.pending)
	}
}

func TestSelectForInstallProject(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeProject)

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeProject {
		t.Errorf("pending = %v, want project scope", m.pending)
	}
}

func TestSelectForInstallClearsIfSameScope(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0
	m.pending["test@marketplace"] = claude.ScopeProject

	// Selecting local should clear pending since it's the same as installed
	m.selectForInstall(claude.ScopeLocal)

	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("pending should be cleared when selecting same scope as installed")
	}
}

func TestSelectForUninstall(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	m.selectForUninstall()

	if scope, ok := m.pending["test@marketplace"]; !ok || scope != claude.ScopeNone {
		t.Errorf("pending = %v, want ScopeNone for uninstall", m.pending)
	}
}

func TestSelectForUninstallToggle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	// First uninstall
	m.selectForUninstall()
	if _, ok := m.pending["test@marketplace"]; !ok {
		t.Error("first uninstall should mark pending")
	}

	// Toggle back
	m.selectForUninstall()
	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("second uninstall should clear pending")
	}
}

func TestClearPending(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0
	m.pending["test@marketplace"] = claude.ScopeLocal

	m.clearPending()

	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("clearPending should remove pending change")
	}
}

func TestToggleScopeCycle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	// None -> Local
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeLocal {
		t.Errorf("after first toggle = %v, want local", scope)
	}

	// Local -> Project
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeProject {
		t.Errorf("after second toggle = %v, want project", scope)
	}

	// Project -> None (not installed, clears pending)
	m.toggleScope()
	if _, ok := m.pending["test@marketplace"]; ok {
		t.Error("after third toggle, pending should be cleared")
	}
}

func TestToggleScopeInstalledPlugin(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeLocal},
	}
	m.selectedIdx = 0

	// Local installed -> Project pending
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeProject {
		t.Errorf("after first toggle = %v, want project", scope)
	}

	// Project pending -> Uninstall pending
	m.toggleScope()
	if scope := m.pending["test@marketplace"]; scope != claude.ScopeNone {
		t.Errorf("after second toggle = %v, want ScopeNone (uninstall)", scope)
	}
}

func TestSkipsGroupHeaders(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{Name: "marketplace", IsGroupHeader: true},
	}
	m.selectedIdx = 0

	// Should not panic or add to pending
	m.selectForInstall(claude.ScopeLocal)
	m.selectForUninstall()
	m.toggleScope()

	if len(m.pending) != 0 {
		t.Error("operations on group headers should not modify pending")
	}
}
