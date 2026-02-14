package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/open-cli-collective/cpm/internal/claude"
)

func TestNewModel(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")

	if m.client == nil {
		t.Error("client is nil")
	}
	if !m.progress.loading {
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
	if !state.HasScope(claude.ScopeProject) {
		t.Errorf("HasScope(ScopeProject) = false, want true")
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
	if state.IsInstalled() {
		t.Errorf("IsInstalled() = true, want false for available plugin")
	}
}

func TestPluginStateHelpers(t *testing.T) {
	// Not installed
	ps := PluginState{InstalledScopes: map[claude.Scope]bool{}}
	if ps.IsInstalled() {
		t.Error("empty scopes should not be installed")
	}
	if ps.HasScope(claude.ScopeUser) {
		t.Error("empty scopes should not have user scope")
	}

	// Single scope
	ps.InstalledScopes = map[claude.Scope]bool{claude.ScopeLocal: true}
	if !ps.IsInstalled() {
		t.Error("should be installed")
	}
	if !ps.HasScope(claude.ScopeLocal) {
		t.Error("should have local scope")
	}
	if ps.HasScope(claude.ScopeUser) {
		t.Error("should not have user scope")
	}
	if !ps.IsSingleScope() {
		t.Error("should be single scope")
	}
	if ps.SingleScope() != claude.ScopeLocal {
		t.Errorf("SingleScope = %v, want ScopeLocal", ps.SingleScope())
	}

	// Multi scope
	ps.InstalledScopes = map[claude.Scope]bool{claude.ScopeUser: true, claude.ScopeLocal: true}
	if !ps.IsInstalled() {
		t.Error("should be installed")
	}
	if ps.IsSingleScope() {
		t.Error("should not be single scope")
	}
	if !ps.HasScope(claude.ScopeUser) {
		t.Error("should have user scope")
	}
	if !ps.HasScope(claude.ScopeLocal) {
		t.Error("should have local scope")
	}
}

// mockClient implements claude.Client for testing
type mockClient struct {
	plugins     *claude.PluginList
	err         error
	installFn   func(string, claude.Scope) error
	uninstallFn func(string, claude.Scope) error
	enableFn    func(string, claude.Scope) error
	disableFn   func(string, claude.Scope) error
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

func (m *mockClient) InstallPlugin(pluginID string, scope claude.Scope) error {
	if m.installFn != nil {
		return m.installFn(pluginID, scope)
	}
	return m.err
}

func (m *mockClient) UninstallPlugin(pluginID string, scope claude.Scope) error {
	if m.uninstallFn != nil {
		return m.uninstallFn(pluginID, scope)
	}
	return m.err
}

func (m *mockClient) EnablePlugin(pluginID string, scope claude.Scope) error {
	if m.enableFn != nil {
		return m.enableFn(pluginID, scope)
	}
	return m.err
}

func (m *mockClient) DisablePlugin(pluginID string, scope claude.Scope) error {
	if m.disableFn != nil {
		return m.disableFn(pluginID, scope)
	}
	return m.err
}

func TestSelectForInstallLocal(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeLocal)

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok || op.Type != OpInstall || op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("pendingOps[test@marketplace] = %v, want OpInstall with local scope", op)
	}
}

func TestSelectForInstallProject(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeProject)

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok || op.Type != OpInstall || op.Scopes[0] != claude.ScopeProject {
		t.Errorf("pendingOps[test@marketplace] = %v, want OpInstall with project scope", op)
	}
}

func TestSelectForInstallClearsIfSameScope(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true}},
	}
	m.selectedIdx = 0
	m.main.pendingOps["test@marketplace"] = Operation{PluginID: "test@marketplace", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall}

	// Selecting local should replace pending (not clear it, since project != local)
	m.selectForInstall(claude.ScopeLocal)

	if op, ok := m.main.pendingOps["test@marketplace"]; !ok || op.Scopes[0] != claude.ScopeLocal {
		t.Error("pendingOps should be updated to local scope")
	}
}

func TestSelectForUninstall(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true}},
	}
	m.selectedIdx = 0

	m.selectForUninstall()

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok || op.Type != OpUninstall {
		t.Errorf("pendingOps[test@marketplace] = %v, want OpUninstall", op)
	}
}

func TestSelectForUninstallToggle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true}},
	}
	m.selectedIdx = 0

	// First uninstall
	m.selectForUninstall()
	if _, ok := m.main.pendingOps["test@marketplace"]; !ok {
		t.Error("first uninstall should mark pending")
	}

	// Toggle back
	m.selectForUninstall()
	if _, ok := m.main.pendingOps["test@marketplace"]; ok {
		t.Error("second uninstall should clear pending")
	}
}

func TestClearPending(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0
	m.main.pendingOps["test@marketplace"] = Operation{PluginID: "test@marketplace", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}

	m.clearPending("test@marketplace")

	if _, ok := m.main.pendingOps["test@marketplace"]; ok {
		t.Error("clearPending should remove pending change")
	}
}

func TestToggleScopeCycle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0

	// None -> Local
	m.toggleScope()
	if op, ok := m.main.pendingOps["test@marketplace"]; !ok || op.Type != OpInstall || op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("after first toggle = %v, want OpInstall with local", op)
	}

	// Local -> Project
	m.toggleScope()
	if op, ok := m.main.pendingOps["test@marketplace"]; !ok || op.Type != OpInstall || op.Scopes[0] != claude.ScopeProject {
		t.Errorf("after second toggle = %v, want OpInstall with project", op)
	}

	// Project -> None (not installed, clears pending)
	m.toggleScope()
	if _, ok := m.main.pendingOps["test@marketplace"]; ok {
		t.Error("after third toggle, pending should be cleared")
	}
}

func TestToggleScopeInstalledPlugin(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true}},
	}
	m.selectedIdx = 0

	// Local installed -> Local pending (same scope = install, not migrate)
	m.toggleScope()
	if op, ok := m.main.pendingOps["test@marketplace"]; !ok || op.Type != OpInstall || op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("after first toggle = %v, want OpInstall with local", op)
	}

	// Local pending -> Project pending (different scope = migrate)
	m.toggleScope()
	if op, ok := m.main.pendingOps["test@marketplace"]; !ok || op.Type != OpMigrate || op.Scopes[0] != claude.ScopeProject {
		t.Errorf("after second toggle = %v, want OpMigrate with project", op)
	}
	// Also verify original scope is preserved
	if op := m.main.pendingOps["test@marketplace"]; firstScope(op.OriginalScopes) != claude.ScopeLocal {
		t.Errorf("migration should preserve original scope, got %v", firstScope(op.OriginalScopes))
	}
}

func TestSkipsGroupHeaders(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{Name: "marketplace", IsGroupHeader: true},
	}
	m.selectedIdx = 0

	// Should not panic or add to pending
	m.selectForInstall(claude.ScopeLocal)
	m.selectForUninstall()
	m.toggleScope()

	if len(m.main.pendingOps) != 0 {
		t.Error("operations on group headers should not modify pending")
	}
}

// TestUpdateConfirmationEnterStartsExecution tests that Enter in confirmation starts execution.
func TestUpdateConfirmationEnterStartsExecution(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0
	m.main.pendingOps["test@marketplace"] = Operation{PluginID: "test@marketplace", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.showConfirm = true

	// Send Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.updateConfirmation(msg)
	m = result.(*Model)

	// Should exit confirmation mode and start execution
	if m.main.showConfirm {
		t.Error("showConfirm should be false after Enter")
	}
	if m.mode != ModeProgress {
		t.Errorf("mode = %d, want ModeProgress", m.mode)
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should start first operation)")
	}
}

// TestUpdateConfirmationEscapeCancel tests that Escape cancels confirmation.
func TestUpdateConfirmationEscapeCancel(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.pendingOps["test@marketplace"] = Operation{PluginID: "test@marketplace", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.showConfirm = true

	// Send Escape key
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := m.updateConfirmation(msg)
	m = result.(*Model)

	// Should exit confirmation mode without executing
	if m.main.showConfirm {
		t.Error("showConfirm should be false after Escape")
	}
	if m.mode != ModeMain {
		t.Errorf("mode = %d, want ModeMain", m.mode)
	}
	// Pending changes should remain
	if _, ok := m.main.pendingOps["test@marketplace"]; !ok {
		t.Error("pending changes should not be cleared on cancel")
	}
}

// TestStartExecutionBuildsOperations tests that startExecution builds operations correctly.
func TestStartExecutionBuildsOperations(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@market", Name: "plugin1", InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true}},
		{ID: "plugin2@market", Name: "plugin2", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0

	// Set pending: uninstall plugin1 (local), install plugin2 (project)
	m.main.pendingOps["plugin1@market"] = Operation{PluginID: "plugin1@market", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeLocal: true}, Type: OpUninstall}
	m.main.pendingOps["plugin2@market"] = Operation{PluginID: "plugin2@market", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall}

	result, _ := m.startExecution()
	m = result.(*Model)

	if len(m.progress.operations) != 2 {
		t.Errorf("len(operations) = %d, want 2", len(m.progress.operations))
	}
	if m.progress.currentIdx != 0 {
		t.Errorf("currentOpIdx = %d, want 0", m.progress.currentIdx)
	}
	if m.mode != ModeProgress {
		t.Errorf("mode = %d, want ModeProgress", m.mode)
	}

	// Check that uninstall captured original scope
	found := false
	for _, op := range m.progress.operations {
		if op.PluginID == "plugin1@market" && op.Type == OpUninstall {
			if firstScope(op.OriginalScopes) != claude.ScopeLocal {
				t.Errorf("uninstall OriginalScope = %v, want ScopeLocal", firstScope(op.OriginalScopes))
			}
			found = true
		}
	}
	if !found {
		t.Error("uninstall operation not found or OriginalScopes not set")
	}
}

// TestExecuteOperationInstall tests that executeOperation calls InstallPlugin for installs.
func TestExecuteOperationInstall(t *testing.T) {
	calls := []struct {
		pluginID string
		scope    claude.Scope
	}{}

	client := &mockClient{
		installFn: func(pluginID string, scope claude.Scope) error {
			calls = append(calls, struct {
				pluginID string
				scope    claude.Scope
			}{pluginID, scope})
			return nil
		},
	}

	m := NewModel(client, "/test/project")
	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal},
		Type:     OpInstall,
	}

	cmd := m.executeOperation(op)
	resultMsg := cmd()

	if len(calls) != 1 {
		t.Errorf("InstallPlugin called %d times, want 1", len(calls))
	}
	if calls[0].pluginID != "test@marketplace" || calls[0].scope != claude.ScopeLocal {
		t.Errorf("InstallPlugin called with %v, want test@marketplace, local", calls[0])
	}

	// Check message
	doneMsg, ok := resultMsg.(operationDoneMsg)
	if !ok {
		t.Error("returned message is not operationDoneMsg")
	}
	if doneMsg.op.PluginID != "test@marketplace" {
		t.Errorf("doneMsg.op.PluginID = %v, want test@marketplace", doneMsg.op.PluginID)
	}
}

// TestExecuteOperationUninstallUsesOriginalScope tests that executeOperation uninstalls from specified scopes.
// This test has been updated for Phase 7 multi-scope behavior: OpUninstall now uses op.Scopes.
func TestExecuteOperationUninstallUsesOriginalScope(t *testing.T) {
	tmpDir := t.TempDir()

	// Write settings file so plugin is recognized as "in settings"
	os.Mkdir(tmpDir+"/.claude", 0o755)
	os.WriteFile(tmpDir+"/.claude/settings.json", []byte(`{"enabledPlugins":{"test@marketplace":true}}`), 0o644)

	calls := []struct {
		pluginID string
		scope    claude.Scope
	}{}

	client := &mockClient{
		uninstallFn: func(pluginID string, scope claude.Scope) error {
			calls = append(calls, struct {
				pluginID string
				scope    claude.Scope
			}{pluginID, scope})
			return nil
		},
	}

	m := NewModel(client, tmpDir)
	op := Operation{
		PluginID:       "test@marketplace",
		Scopes:         []claude.Scope{claude.ScopeProject}, // list of scopes to uninstall from
		Type:           OpUninstall,
		OriginalScopes: map[claude.Scope]bool{claude.ScopeProject: true}, // was installed at project scope
	}

	cmd := m.executeOperation(op)
	resultMsg := cmd()

	if len(calls) != 1 {
		t.Errorf("UninstallPlugin called %d times, want 1", len(calls))
	}
	if calls[0].scope != claude.ScopeProject {
		t.Errorf("UninstallPlugin called with scope %v, want ScopeProject", calls[0].scope)
	}

	// Check message
	_, ok := resultMsg.(operationDoneMsg)
	if !ok {
		t.Error("returned message is not operationDoneMsg")
	}
}

// TestUpdateProgressChainedOperations tests that operations execute sequentially.
func TestUpdateProgressChainedOperations(t *testing.T) {
	callCount := 0
	client := &mockClient{
		installFn: func(string, claude.Scope) error {
			callCount++
			return nil
		},
	}

	m := NewModel(client, "/test/project")
	m.mode = ModeProgress
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
		{PluginID: "p2@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall},
	}
	m.progress.currentIdx = 0
	m.progress.errors = make([]string, 2)

	// Simulate first operation completing
	doneMsg := operationDoneMsg{op: m.progress.operations[0], err: nil}
	result, cmd := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.progress.currentIdx != 1 {
		t.Errorf("currentOpIdx = %d, want 1 after first operation", m.progress.currentIdx)
	}
	if m.mode != ModeProgress {
		t.Errorf("mode = %d, want ModeProgress (not done yet)", m.mode)
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should execute next operation)")
	}
}

// TestUpdateProgressCompletesAndShowsSummary tests that all operations complete correctly.
func TestUpdateProgressCompletesAndShowsSummary(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeProgress
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
	}
	m.progress.currentIdx = 0
	m.progress.errors = make([]string, 1)

	// Simulate operation completing
	doneMsg := operationDoneMsg{op: m.progress.operations[0], err: nil}
	result, cmd := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary", m.mode)
	}
	if len(m.main.pendingOps) != 0 {
		t.Error("pending should be cleared after completion")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should load plugins)")
	}
}

// TestUpdateProgressRecordsErrors tests that errors are recorded correctly.
func TestUpdateProgressRecordsErrors(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeProgress
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
		{PluginID: "p2@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall},
	}
	m.progress.currentIdx = 0
	m.progress.errors = make([]string, 2)

	// Simulate first operation failing
	doneMsg := operationDoneMsg{op: m.progress.operations[0], err: fmt.Errorf("install failed")}
	result, _ := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.progress.errors[0] != "install failed" {
		t.Errorf("operationErrors[0] = %q, want 'install failed'", m.progress.errors[0])
	}
}

// TestUpdateErrorReturnsToMain tests that error summary returns to main view on Enter/Esc.
func TestUpdateErrorReturnsToMain(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeSummary
	m.progress.operations = []Operation{{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}}
	m.progress.errors = []string{""}

	// Send Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.updateError(msg)
	m = result.(*Model)

	if m.mode != ModeMain {
		t.Errorf("mode = %d, want ModeMain", m.mode)
	}
	if m.progress.operations != nil {
		t.Error("operations should be cleared when returning to main")
	}
	if m.progress.errors != nil {
		t.Error("operationErrors should be cleared when returning to main")
	}
}

// TestUpdateErrorHandlesPluginsLoaded tests that summary updates when plugins reload.
func TestUpdateErrorHandlesPluginsLoaded(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeSummary
	m.selectedIdx = 0

	newPlugins := []PluginState{
		{ID: "p1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "p2@m", Name: "plugin2", IsGroupHeader: false},
	}

	msg := pluginsLoadedMsg{plugins: newPlugins}
	result, _ := m.updateError(msg)
	m = result.(*Model)

	if len(m.plugins) != 2 {
		t.Errorf("len(plugins) = %d, want 2", len(m.plugins))
	}
	if m.plugins[0].ID != "p1@m" {
		t.Errorf("first plugin = %v, want p1@m", m.plugins[0].ID)
	}
}

// TestRenderConfirmationOutput tests that renderConfirmation produces expected content.
func TestRenderConfirmationOutput(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.main.pendingOps["p1@market"] = Operation{PluginID: "p1@market", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.pendingOps["p2@market"] = Operation{PluginID: "p2@market", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeLocal: true}, Type: OpUninstall}

	output := m.renderConfirmation(m.styles)

	if !strings.Contains(output, "Apply Changes") {
		t.Error("output should contain 'Apply Changes'")
	}
	if !strings.Contains(output, "p1@market") {
		t.Error("output should contain plugin name p1@market")
	}
	if !strings.Contains(output, "Uninstall") {
		t.Error("output should show Uninstall for uninstalls")
	}
	if !strings.Contains(output, "1 install") {
		t.Error("output should show install count")
	}
	if !strings.Contains(output, "1 uninstall") {
		t.Error("output should show uninstall count")
	}
}

// TestRenderConfirmationWithEnableDisable tests that renderConfirmation displays enable and disable operations.
func TestRenderConfirmationWithEnableDisable(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.main.pendingOps["p1@m"] = Operation{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.pendingOps["p2@m"] = Operation{PluginID: "p2@m", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeLocal: true}, Type: OpUninstall}
	m.main.pendingOps["p3@m"] = Operation{PluginID: "p3@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpEnable}
	m.main.pendingOps["p4@m"] = Operation{PluginID: "p4@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpDisable}

	output := m.renderConfirmation(m.styles)

	// Check for operation type labels
	if !strings.Contains(output, "Install") {
		t.Error("output should contain 'Install'")
	}
	if !strings.Contains(output, "Uninstall") {
		t.Error("output should contain 'Uninstall'")
	}
	if !strings.Contains(output, "Enable") {
		t.Error("output should contain 'Enable'")
	}
	if !strings.Contains(output, "Disable") {
		t.Error("output should contain 'Disable'")
	}

	// Check for summary counts
	if !strings.Contains(output, "1 install(s)") {
		t.Error("output should contain '1 install(s)'")
	}
	if !strings.Contains(output, "1 uninstall(s)") {
		t.Error("output should contain '1 uninstall(s)'")
	}
	if !strings.Contains(output, "1 enable(s)") {
		t.Error("output should contain '1 enable(s)'")
	}
	if !strings.Contains(output, "1 disable(s)") {
		t.Error("output should contain '1 disable(s)'")
	}
}

// TestRenderProgressOutput tests that renderProgress shows operation status.
func TestRenderProgressOutput(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.mode = ModeProgress
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
		{PluginID: "p2@m", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeProject: true}, Type: OpUninstall},
	}
	m.progress.currentIdx = 0
	m.progress.errors = []string{"", ""}

	output := m.renderProgress(m.styles)

	if !strings.Contains(output, "Running") {
		t.Error("output should show Running for current operation")
	}
	if !strings.Contains(output, "Pending") {
		t.Error("output should show Pending for future operations")
	}
	if !strings.Contains(output, "Install") {
		t.Error("output should show Install action")
	}
	if !strings.Contains(output, "Uninstall") {
		t.Error("output should show Uninstall action")
	}
}

// TestRenderErrorSummaryAllSuccess tests summary when all operations succeed.
func TestRenderErrorSummaryAllSuccess(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.mode = ModeSummary
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
		{PluginID: "p2@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall},
	}
	m.progress.errors = []string{"", ""}

	output := m.renderErrorSummary(m.styles)

	if !strings.Contains(output, "All Changes Applied") {
		t.Error("output should show 'All Changes Applied'")
	}
	if !strings.Contains(output, "2 succeeded") {
		t.Error("output should show success count")
	}
	if strings.Contains(output, "failed") {
		t.Error("output should not show failed count when all succeed")
	}
}

// TestRenderErrorSummaryWithErrors tests summary when some operations fail.
func TestRenderErrorSummaryWithErrors(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.mode = ModeSummary
	m.progress.operations = []Operation{
		{PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
		{PluginID: "p2@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpInstall},
	}
	m.progress.errors = []string{"", "install failed"}

	output := m.renderErrorSummary(m.styles)

	if !strings.Contains(output, "Completed With Errors") {
		t.Error("output should show 'Completed With Errors'")
	}
	if !strings.Contains(output, "1 succeeded") {
		t.Error("output should show success count")
	}
	if !strings.Contains(output, "1 failed") {
		t.Error("output should show failure count")
	}
	if !strings.Contains(output, "p2@m") {
		t.Error("output should list failed plugin")
	}
}

// --- Filter Mode Tests ---

// TestUpdateFilterEscClears tests that Esc clears filter and exits filter mode.
func TestUpdateFilterEscClears(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "test"
	m.filteredIdx = []int{0}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.updateFilter(msg)
	m = result.(*Model)

	if m.filter.active {
		t.Error("filterActive should be false after Esc")
	}
	if m.filter.text != "" {
		t.Errorf("filterText = %q, want empty", m.filter.text)
	}
	if len(m.filteredIdx) != 0 {
		t.Error("filteredIdx should be cleared after Esc")
	}
}

// TestUpdateFilterEnterSelectsFirstMatch tests that Enter selects first filtered match and exits.
func TestUpdateFilterEnterSelectsFirstMatch(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@marketplace", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@marketplace", Name: "plugin2", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "plugin"
	m.filteredIdx = []int{0, 1}
	m.selectedIdx = -1

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.updateFilter(msg)
	m = result.(*Model)

	if m.filter.active {
		t.Error("filterActive should be false after Enter")
	}
	if m.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0 (first match)", m.selectedIdx)
	}
}

// TestUpdateFilterBackspaceRemovesCharacters tests that backspace removes filter text.
func TestUpdateFilterBackspaceRemovesCharacters(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "test"
	m.filteredIdx = []int{0}

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ := m.updateFilter(msg)
	m = result.(*Model)

	if m.filter.text != "tes" {
		t.Errorf("filterText = %q, want 'tes'", m.filter.text)
	}
}

// TestUpdateFilterRunesAppends tests that runes are appended to filter text.
func TestUpdateFilterRunesAppends(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "te"

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("st")}
	result, _ := m.updateFilter(msg)
	m = result.(*Model)

	if m.filter.text != "test" {
		t.Errorf("filterText = %q, want 'test'", m.filter.text)
	}
}

// TestApplyFilterCaseInsensitive tests case-insensitive matching on name/description/ID.
func TestApplyFilterCaseInsensitive(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "TestPlugin@marketplace", Name: "TestPlugin", Description: "A test plugin", IsGroupHeader: false},
		{ID: "other@marketplace", Name: "other", Description: "Another one", IsGroupHeader: false},
	}
	m.filter.text = "test"
	m.filter.active = true

	m.applyFilter()

	if len(m.filteredIdx) != 1 {
		t.Errorf("len(filteredIdx) = %d, want 1", len(m.filteredIdx))
	}
	if m.filteredIdx[0] != 0 {
		t.Errorf("filteredIdx[0] = %d, want 0", m.filteredIdx[0])
	}
}

// TestApplyFilterSkipsGroupHeaders tests that group headers are skipped.
func TestApplyFilterSkipsGroupHeaders(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{Name: "marketplace", IsGroupHeader: true},
		{ID: "test@marketplace", Name: "test", Description: "A test", IsGroupHeader: false},
	}
	m.filter.text = "xyz" // Search for something that doesn't match
	m.filter.active = true

	m.applyFilter()

	if len(m.filteredIdx) != 0 {
		t.Errorf("len(filteredIdx) = %d, want 0 (no matches)", len(m.filteredIdx))
	}

	// Now search for something that matches only the plugin, not the header
	m.filter.text = "test"
	m.applyFilter()

	if len(m.filteredIdx) != 1 {
		t.Errorf("len(filteredIdx) = %d, want 1 (plugin matches)", len(m.filteredIdx))
	}
	// Verify it's not the header (index 0)
	if m.filteredIdx[0] == 0 {
		t.Error("should not match group header")
	}
}

// TestApplyFilterMatchesDescription tests that filters match plugin description.
func TestApplyFilterMatchesDescription(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin@m", Name: "plugin", Description: "Very useful tool", IsGroupHeader: false},
	}
	m.filter.text = "useful"
	m.filter.active = true

	m.applyFilter()

	if len(m.filteredIdx) != 1 {
		t.Errorf("len(filteredIdx) = %d, want 1 (should match description)", len(m.filteredIdx))
	}
}

// TestApplyFilterMatchesID tests that filters match plugin ID.
func TestApplyFilterMatchesID(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "unique-id@marketplace", Name: "plugin", IsGroupHeader: false},
	}
	m.filter.text = "unique"
	m.filter.active = true

	m.applyFilter()

	if len(m.filteredIdx) != 1 {
		t.Errorf("len(filteredIdx) = %d, want 1 (should match ID)", len(m.filteredIdx))
	}
}

// TestGetVisiblePluginsFiltered tests that filtered plugins are returned when active.
func TestGetVisiblePluginsFiltered(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@m", Name: "plugin2", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "plugin1"
	m.filteredIdx = []int{0}

	visible := m.getVisiblePlugins()

	if len(visible) != 1 {
		t.Errorf("len(visible) = %d, want 1", len(visible))
	}
	if visible[0].ID != "plugin1@m" {
		t.Errorf("visible[0].ID = %q, want 'plugin1@m'", visible[0].ID)
	}
}

// TestGetVisiblePluginsUnfiltered tests that all plugins returned when filter inactive.
func TestGetVisiblePluginsUnfiltered(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@m", Name: "plugin2", IsGroupHeader: false},
	}
	m.filter.active = false

	visible := m.getVisiblePlugins()

	if len(visible) != 2 {
		t.Errorf("len(visible) = %d, want 2", len(visible))
	}
}

// TestGetActualIndexWithFilter tests index mapping with filter active.
func TestGetActualIndexWithFilter(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@m", Name: "plugin2", IsGroupHeader: false},
		{ID: "plugin3@m", Name: "plugin3", IsGroupHeader: false},
	}
	m.filter.active = true
	m.filter.text = "plugin"
	m.filteredIdx = []int{0, 2} // filtered shows plugins 1 and 3 (indices 0 and 2)
	m.listOffset = 0

	actualIdx := m.getActualIndex(0)
	if actualIdx != 0 {
		t.Errorf("getActualIndex(0) = %d, want 0", actualIdx)
	}

	actualIdx = m.getActualIndex(1)
	if actualIdx != 2 {
		t.Errorf("getActualIndex(1) = %d, want 2", actualIdx)
	}
}

// TestGetActualIndexWithoutFilter tests index mapping with filter inactive.
func TestGetActualIndexWithoutFilter(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "plugin1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@m", Name: "plugin2", IsGroupHeader: false},
	}
	m.filter.active = false
	m.listOffset = 1

	actualIdx := m.getActualIndex(0)
	if actualIdx != 1 {
		t.Errorf("getActualIndex(0) = %d, want 1 (with offset)", actualIdx)
	}
}

// TestRenderFilterInput tests filter input rendering when active.
func TestRenderFilterInput(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.filter.active = true
	m.filter.text = "test"

	output := m.renderFilterInput(m.styles)

	if !strings.Contains(output, "/test") {
		t.Errorf("output should contain '/test', got %q", output)
	}
	if !strings.Contains(output, "█") {
		t.Error("output should contain cursor █")
	}
}

// TestRenderFilterInputInactive tests no output when filter inactive.
func TestRenderFilterInputInactive(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.filter.active = false

	output := m.renderFilterInput(m.styles)

	if output != "" {
		t.Errorf("output should be empty when inactive, got %q", output)
	}
}

// --- Refresh Functionality Tests ---

// TestHandleRefreshKey tests refresh key sets loading and returns loadPlugins command.
func TestHandleRefreshKey(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.progress.loading = false

	result, cmd := m.handleRefreshKey()
	m = result.(*Model)

	if !m.progress.loading {
		t.Error("loading should be true after refresh")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should return loadPlugins)")
	}
}

// --- Quit Confirmation Tests ---

// TestHandleQuitKeyShowsConfirmation tests quit confirmation when pending changes.
func TestHandleQuitKeyShowsConfirmation(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.pendingOps["test@m"] = Operation{PluginID: "test@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.showQuitConfirm = false

	result, cmd := m.handleQuitKey()
	m = result.(*Model)

	if !m.main.showQuitConfirm {
		t.Error("showQuitConfirm should be true when pending changes")
	}
	if cmd != nil {
		t.Error("cmd should be nil (not quitting yet)")
	}
}

// TestHandleQuitKeyQuitsWhenNoPending tests quit quits when no pending changes.
func TestHandleQuitKeyQuitsWhenNoPending(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.pendingOps = make(map[string]Operation)

	_, cmd := m.handleQuitKey()

	if cmd == nil {
		t.Error("cmd should not be nil (should return Quit)")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("cmd should return tea.Quit message")
	}
}

// TestRenderQuitConfirmation tests quit confirmation modal content.
func TestRenderQuitConfirmation(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.main.pendingOps["plugin1@m"] = Operation{PluginID: "plugin1@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall}
	m.main.pendingOps["plugin2@m"] = Operation{PluginID: "plugin2@m", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeLocal: true}, Type: OpUninstall}

	output := m.renderQuitConfirmation(m.styles)

	if !strings.Contains(output, "Quit Without Applying") {
		t.Error("output should contain 'Quit Without Applying'")
	}
	if !strings.Contains(output, "2") {
		t.Error("output should show pending count")
	}
	if !strings.Contains(output, "Press q again") {
		t.Error("output should show q to quit instruction")
	}
}

// --- Mouse Support Tests ---

// TestHandleMouseLeftClick tests left click selects item in left pane.
func TestHandleMouseLeftClick(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.plugins = []PluginState{
		{ID: "plugin1@m", Name: "plugin1", IsGroupHeader: false},
		{ID: "plugin2@m", Name: "plugin2", IsGroupHeader: false},
		{ID: "plugin3@m", Name: "plugin3", IsGroupHeader: false},
	}
	m.selectedIdx = 0
	m.filter.active = false
	m.listOffset = 0

	// Click at X=10 (within left pane, width/3 - 2 = 100/3 - 2 = 33 - 2 = 31)
	// Y=3 with verticalOffset=1 gives row=3-1+0=2, which is the third plugin (index 2)
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10, // Left side (within left pane)
		Y:      3,  // Calculate row based on offset
	}

	result, _ := m.handleMouse(msg)
	m = result.(*Model)

	// row = Y - verticalOffset + listOffset = 3 - 1 + 0 = 2
	// So selectedIdx should be plugins[2] = plugin3@m (index 2)
	if m.selectedIdx != 2 {
		t.Errorf("selectedIdx = %d, want 2 (third item at row 2)", m.selectedIdx)
	}
}

// TestHandleMouseWheelUp tests mouse wheel up scrolls up.
func TestHandleMouseWheelUp(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.plugins = []PluginState{
		{ID: "p1@m", Name: "p1", IsGroupHeader: false},
		{ID: "p2@m", Name: "p2", IsGroupHeader: false},
		{ID: "p3@m", Name: "p3", IsGroupHeader: false},
		{ID: "p4@m", Name: "p4", IsGroupHeader: false},
	}
	m.selectedIdx = 3

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}

	result, _ := m.handleMouse(msg)
	m = result.(*Model)

	// Should have moved up by 3
	if m.selectedIdx >= 3 {
		t.Errorf("selectedIdx = %d, should have moved up", m.selectedIdx)
	}
}

// TestHandleMouseWheelDown tests mouse wheel down scrolls down.
func TestHandleMouseWheelDown(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.plugins = []PluginState{
		{ID: "p1@m", Name: "p1", IsGroupHeader: false},
		{ID: "p2@m", Name: "p2", IsGroupHeader: false},
		{ID: "p3@m", Name: "p3", IsGroupHeader: false},
		{ID: "p4@m", Name: "p4", IsGroupHeader: false},
	}
	m.selectedIdx = 0

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	}

	result, _ := m.handleMouse(msg)
	m = result.(*Model)

	// Should have moved down by 3
	if m.selectedIdx <= 0 {
		t.Errorf("selectedIdx = %d, should have moved down", m.selectedIdx)
	}
}

// --- Enable/Disable Client Method Tests ---

// TestMockClientEnablePlugin tests that mockClient.EnablePlugin can be called with callbacks.
func TestMockClientEnablePlugin(t *testing.T) {
	called := false
	var capturedPluginID string
	var capturedScope claude.Scope

	client := &mockClient{
		enableFn: func(pluginID string, scope claude.Scope) error {
			called = true
			capturedPluginID = pluginID
			capturedScope = scope
			return nil
		},
	}

	err := client.EnablePlugin("test@marketplace", claude.ScopeLocal)
	if err != nil {
		t.Errorf("EnablePlugin returned error: %v", err)
	}
	if !called {
		t.Error("enableFn was not called")
	}
	if capturedPluginID != "test@marketplace" {
		t.Errorf("capturedPluginID = %q, want 'test@marketplace'", capturedPluginID)
	}
	if capturedScope != claude.ScopeLocal {
		t.Errorf("capturedScope = %q, want ScopeLocal", capturedScope)
	}
}

// TestMockClientDisablePlugin tests that mockClient.DisablePlugin can be called with callbacks.
func TestMockClientDisablePlugin(t *testing.T) {
	called := false
	var capturedPluginID string
	var capturedScope claude.Scope

	client := &mockClient{
		disableFn: func(pluginID string, scope claude.Scope) error {
			called = true
			capturedPluginID = pluginID
			capturedScope = scope
			return nil
		},
	}

	err := client.DisablePlugin("test@marketplace", claude.ScopeProject)
	if err != nil {
		t.Errorf("DisablePlugin returned error: %v", err)
	}
	if !called {
		t.Error("disableFn was not called")
	}
	if capturedPluginID != "test@marketplace" {
		t.Errorf("capturedPluginID = %q, want 'test@marketplace'", capturedPluginID)
	}
	if capturedScope != claude.ScopeProject {
		t.Errorf("capturedScope = %q, want ScopeProject", capturedScope)
	}
}

// TestMockClientEnablePluginWithError tests that mockClient.EnablePlugin propagates errors.
func TestMockClientEnablePluginWithError(t *testing.T) {
	client := &mockClient{
		enableFn: func(_ string, _ claude.Scope) error {
			return fmt.Errorf("enable failed: permission denied")
		},
	}

	err := client.EnablePlugin("test@marketplace", claude.ScopeLocal)

	if err == nil {
		t.Error("EnablePlugin should return error")
	}
	if !strings.Contains(err.Error(), "enable failed") {
		t.Errorf("error message = %v, should contain 'enable failed'", err)
	}
}

// TestMockClientDisablePluginWithError tests that mockClient.DisablePlugin propagates errors.
func TestMockClientDisablePluginWithError(t *testing.T) {
	client := &mockClient{
		disableFn: func(_ string, _ claude.Scope) error {
			return fmt.Errorf("disable failed: plugin not found")
		},
	}

	err := client.DisablePlugin("test@marketplace", claude.ScopeProject)

	if err == nil {
		t.Error("DisablePlugin should return error")
	}
	if !strings.Contains(err.Error(), "disable failed") {
		t.Errorf("error message = %v, should contain 'disable failed'", err)
	}
}

// TestMockClientEnablePluginDefaultBehavior tests that EnablePlugin works without callbacks.
func TestMockClientEnablePluginDefaultBehavior(t *testing.T) {
	client := &mockClient{} // No enableFn callback

	err := client.EnablePlugin("test@marketplace", claude.ScopeLocal)
	if err != nil {
		t.Errorf("EnablePlugin should return nil when no error configured: %v", err)
	}
}

// TestMockClientDisablePluginDefaultBehavior tests that DisablePlugin works without callbacks.
func TestMockClientDisablePluginDefaultBehavior(t *testing.T) {
	client := &mockClient{} // No disableFn callback

	err := client.DisablePlugin("test@marketplace", claude.ScopeProject)
	if err != nil {
		t.Errorf("DisablePlugin should return nil when no error configured: %v", err)
	}
}

// --- Enable/Disable Toggle Tests ---

// TestToggleEnablement tests basic toggle behavior.
func TestToggleEnablement(t *testing.T) {
	// Setup: plugin installed and enabled at ScopeLocal
	m := &Model{
		plugins: []PluginState{
			{
				ID:              "test@marketplace",
				InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true},
				Enabled:         true,
			},
		},
		selectedIdx: 0,
		main:        MainState{pendingOps: make(map[string]Operation)},
	}

	// First press: should add OpDisable (plugin is currently enabled)
	m.toggleEnablement()

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Fatal("expected pending operation, got none")
	}
	if op.Type != OpDisable {
		t.Errorf("Type = %v, want OpDisable", op.Type)
	}
	if op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("Scope = %v, want ScopeLocal", op.Scopes[0])
	}

	// Second press: should remove pending operation (toggle off)
	m.toggleEnablement()

	if _, ok := m.main.pendingOps["test@marketplace"]; ok {
		t.Error("expected no pending operation after second toggle")
	}
}

// TestToggleEnablementBlockedByPendingInstall tests mutual exclusion.
func TestToggleEnablementBlockedByPendingInstall(t *testing.T) {
	m := &Model{
		plugins: []PluginState{
			{
				ID:              "test@marketplace",
				InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true},
				Enabled:         true,
			},
		},
		selectedIdx: 0,
		main:        MainState{pendingOps: make(map[string]Operation)},
	}

	// Add pending install operation
	m.main.pendingOps["test@marketplace"] = Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeProject},
		Type:     OpInstall,
	}

	// Try to toggle enablement - should be blocked
	m.toggleEnablement()

	// Verify pending operation is still install (not changed to enable/disable)
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Fatal("expected pending operation")
	}
	if op.Type != OpInstall {
		t.Errorf("Type = %v, want OpInstall (should not have changed)", op.Type)
	}
}

// TestToggleEnablementNotInstalled tests blocking for uninstalled plugins.
func TestToggleEnablementNotInstalled(t *testing.T) {
	m := &Model{
		plugins: []PluginState{
			{
				ID:              "test@marketplace",
				InstalledScopes: map[claude.Scope]bool{}, // Not installed
				Enabled:         false,
			},
		},
		selectedIdx: 0,
		main:        MainState{pendingOps: make(map[string]Operation)},
	}

	// Try to toggle enablement - should be blocked (not installed)
	m.toggleEnablement()

	// Verify no pending operation was added
	if len(m.main.pendingOps) != 0 {
		t.Errorf("expected no pending operations, got %d", len(m.main.pendingOps))
	}
}

// TestExecuteOperationEnable tests that executeOperation calls client.EnablePlugin for enable operations.
func TestExecuteOperationEnable(t *testing.T) {
	var calledPluginID string
	var calledScope claude.Scope

	client := &mockClient{
		enableFn: func(pluginID string, scope claude.Scope) error {
			calledPluginID = pluginID
			calledScope = scope
			return nil
		},
	}

	m := &Model{
		client: client,
	}

	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal},
		Type:     OpEnable,
	}

	cmd := m.executeOperation(op)
	msg := cmd()

	if calledPluginID != "test@marketplace" {
		t.Errorf("EnablePlugin called with pluginID = %q, want %q", calledPluginID, "test@marketplace")
	}
	if calledScope != claude.ScopeLocal {
		t.Errorf("EnablePlugin called with scope = %v, want ScopeLocal", calledScope)
	}

	doneMsg, ok := msg.(operationDoneMsg)
	if !ok {
		t.Fatalf("expected operationDoneMsg, got %T", msg)
	}
	if doneMsg.err != nil {
		t.Errorf("expected no error, got %v", doneMsg.err)
	}
}

// TestExecuteOperationDisable tests that executeOperation calls client.DisablePlugin for disable operations.
func TestExecuteOperationDisable(t *testing.T) {
	var calledPluginID string
	var calledScope claude.Scope

	client := &mockClient{
		disableFn: func(pluginID string, scope claude.Scope) error {
			calledPluginID = pluginID
			calledScope = scope
			return nil
		},
	}

	m := &Model{
		client: client,
	}

	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeProject},
		Type:     OpDisable,
	}

	cmd := m.executeOperation(op)
	msg := cmd()

	if calledPluginID != "test@marketplace" {
		t.Errorf("DisablePlugin called with pluginID = %q, want %q", calledPluginID, "test@marketplace")
	}
	if calledScope != claude.ScopeProject {
		t.Errorf("DisablePlugin called with scope = %v, want ScopeProject", calledScope)
	}

	doneMsg, ok := msg.(operationDoneMsg)
	if !ok {
		t.Fatalf("expected operationDoneMsg, got %T", msg)
	}
	if doneMsg.err != nil {
		t.Errorf("expected no error, got %v", doneMsg.err)
	}
}

// TestOperationOrderingWithEnableDisable tests that operations are sorted correctly including enable/disable.
func TestOperationOrderingWithEnableDisable(t *testing.T) {
	m := &Model{
		main: MainState{
			pendingOps: map[string]Operation{
				"p1@m": {PluginID: "p1@m", Scopes: []claude.Scope{claude.ScopeProject}, Type: OpDisable},
				"p2@m": {PluginID: "p2@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpInstall},
				"p3@m": {PluginID: "p3@m", Scopes: []claude.Scope{claude.ScopeLocal}, Type: OpEnable},
				"p4@m": {PluginID: "p4@m", Scopes: []claude.Scope{}, OriginalScopes: map[claude.Scope]bool{claude.ScopeUser: true}, Type: OpUninstall},
			},
		},
		client: &mockClient{},
	}

	result, _ := m.startExecution()
	m = result.(*Model)

	// Verify operations are sorted: Uninstall, Install, Enable, Disable
	if len(m.progress.operations) != 4 {
		t.Fatalf("expected 4 operations, got %d", len(m.progress.operations))
	}

	expectedOrder := []OperationType{OpUninstall, OpInstall, OpEnable, OpDisable}
	for i, expectedType := range expectedOrder {
		if m.progress.operations[i].Type != expectedType {
			t.Errorf("operations[%d].Type = %v, want %v", i, m.progress.operations[i].Type, expectedType)
		}
	}
}

// TestToggleEnablementUsesInstalledScope tests that enable/disable operations use the plugin's installed scope.
func TestToggleEnablementUsesInstalledScope(t *testing.T) {
	tests := []struct {
		name           string
		installedScope claude.Scope
		enabled        bool
		wantType       OperationType
	}{
		{
			name:           "enabled plugin at ScopeLocal",
			installedScope: claude.ScopeLocal,
			enabled:        true,
			wantType:       OpDisable,
		},
		{
			name:           "disabled plugin at ScopeProject",
			installedScope: claude.ScopeProject,
			enabled:        false,
			wantType:       OpEnable,
		},
		{
			name:           "enabled plugin at ScopeUser",
			installedScope: claude.ScopeUser,
			enabled:        true,
			wantType:       OpDisable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				plugins: []PluginState{
					{
						ID:              "test@marketplace",
						InstalledScopes: map[claude.Scope]bool{tt.installedScope: true},
						Enabled:         tt.enabled,
					},
				},
				selectedIdx: 0,
				main:        MainState{pendingOps: make(map[string]Operation)},
			}

			m.toggleEnablement()

			op, ok := m.main.pendingOps["test@marketplace"]
			if !ok {
				t.Fatal("expected pending operation")
			}
			if op.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", op.Type, tt.wantType)
			}
			if op.Scopes[0] != tt.installedScope {
				t.Errorf("Scope = %v, want %v", op.Scopes[0], tt.installedScope)
			}
		})
	}
}

// --- Multi-scope context-aware key behavior tests (plugin-scope-mgmt.AC5) ---

// TestSelectForInstallMultiScopeOpensDialog tests AC5.1: multi-scope on install key opens scope dialog
func TestSelectForInstallMultiScopeOpensDialog(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeUser:  true,
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeProject)

	// Should transition to ModeScopeDialog, not create a pending operation
	if m.mode != ModeScopeDialog {
		t.Errorf("mode = %v, want ModeScopeDialog", m.mode)
	}
	if len(m.main.pendingOps) > 0 {
		t.Error("should not create pending operation for multi-scope plugin")
	}
	// Verify dialog state
	if m.main.scopeDialog.pluginID != "test@marketplace" {
		t.Errorf("scopeDialog.pluginID = %q, want 'test@marketplace'", m.main.scopeDialog.pluginID)
	}
	// Verify Project scope is pre-toggled
	if !m.main.scopeDialog.scopes[1] {
		t.Error("Project scope should be pre-toggled (scopes[1])")
	}
}

// TestSelectForInstallSingleScopeCreatesOp tests that single-scope plugins still create operations
func TestSelectForInstallSingleScopeCreatesOp(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0

	m.selectForInstall(claude.ScopeProject)

	// Should NOT transition to scope dialog; should create migrate operation
	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Error("expected pending operation for single-scope plugin")
	}
	if op.Type != OpMigrate {
		t.Errorf("Type = %v, want OpMigrate", op.Type)
	}
}

// TestSelectForUninstallMultiScopeOpensDialog tests AC5.2: multi-scope on uninstall key opens dialog
func TestSelectForUninstallMultiScopeOpensDialog(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeUser:    true,
			claude.ScopeProject: true,
		}},
	}
	m.selectedIdx = 0

	m.selectForUninstall()

	// Should transition to ModeScopeDialog, not create a pending operation
	if m.mode != ModeScopeDialog {
		t.Errorf("mode = %v, want ModeScopeDialog", m.mode)
	}
	if len(m.main.pendingOps) > 0 {
		t.Error("should not create pending operation for multi-scope plugin")
	}
	// Verify dialog state contains original scopes
	if !m.main.scopeDialog.scopes[0] || !m.main.scopeDialog.scopes[1] {
		t.Error("dialog should preserve original scopes")
	}
}

// TestSelectForUninstallSingleScopeCreatesOp tests that single-scope plugins still create uninstall ops
func TestSelectForUninstallSingleScopeCreatesOp(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0

	m.selectForUninstall()

	// Should NOT transition to scope dialog; should create uninstall operation
	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Error("expected pending operation for single-scope plugin")
	}
	if op.Type != OpUninstall {
		t.Errorf("Type = %v, want OpUninstall", op.Type)
	}
}

// TestToggleEnablementMultiScopeOpensDialog tests AC5.3: multi-scope on toggle enablement opens dialog
func TestToggleEnablementMultiScopeOpensDialog(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeUser:  true,
			claude.ScopeLocal: true,
		}, Enabled: true},
	}
	m.selectedIdx = 0
	m.main.pendingOps = make(map[string]Operation)

	m.toggleEnablement()

	// Should transition to ModeScopeDialog, not create a pending operation
	if m.mode != ModeScopeDialog {
		t.Errorf("mode = %v, want ModeScopeDialog", m.mode)
	}
	if len(m.main.pendingOps) > 0 {
		t.Error("should not create pending operation for multi-scope plugin")
	}
}

// TestToggleEnablementSingleScopeCreatesOp tests that single-scope plugins still toggle enable/disable
func TestToggleEnablementSingleScopeCreatesOp(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeLocal: true,
		}, Enabled: true},
	}
	m.selectedIdx = 0
	m.main.pendingOps = make(map[string]Operation)

	m.toggleEnablement()

	// Should NOT transition to scope dialog; should create disable operation
	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Error("expected pending operation for single-scope plugin")
	}
	if op.Type != OpDisable {
		t.Errorf("Type = %v, want OpDisable", op.Type)
	}
}

// TestToggleScopeMultiScopeIsNoOp tests AC5.4: Tab key is no-op on multi-scope plugins
func TestToggleScopeMultiScopeIsNoOp(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeUser:  true,
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0
	m.main.pendingOps = make(map[string]Operation)

	m.toggleScope()

	// Should be no-op: mode should stay ModeMain, no pending operations
	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain (should not change)", m.mode)
	}
	if len(m.main.pendingOps) > 0 {
		t.Error("toggleScope should be no-op for multi-scope, no operations should be created")
	}
}

// TestToggleScopeSingleScopeWorks tests that Tab key still works for single-scope plugins
func TestToggleScopeSingleScopeWorks(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0
	m.main.pendingOps = make(map[string]Operation)

	m.toggleScope()

	// Should create operation: none -> local
	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Error("expected pending operation for single-scope plugin")
	}
	if op.Type != OpInstall {
		t.Errorf("Type = %v, want OpInstall", op.Type)
	}
	if op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("Scope = %v, want ScopeLocal", op.Scopes[0])
	}
}

// --- Scope Dialog Tests (plugin-scope-mgmt.AC6) ---

// TestOpenScopeDialogForSelectedSingleScope tests opening scope dialog for a single-scope plugin.
// Verifies plugin-scope-mgmt.AC6.1: dialog opens with checkboxes pre-checked for installed scopes.
func TestOpenScopeDialogForSelectedSingleScope(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeLocal: true,
		}},
	}
	m.selectedIdx = 0

	m.openScopeDialogForSelected()

	if m.mode != ModeScopeDialog {
		t.Errorf("mode = %v, want ModeScopeDialog", m.mode)
	}
	if m.main.scopeDialog.pluginID != "test@marketplace" {
		t.Errorf("pluginID = %q, want 'test@marketplace'", m.main.scopeDialog.pluginID)
	}
	// Verify Local scope is checked
	if !m.main.scopeDialog.scopes[2] {
		t.Error("Local scope should be checked (scopes[2] = true)")
	}
	// Verify User and Project are not checked
	if m.main.scopeDialog.scopes[0] || m.main.scopeDialog.scopes[1] {
		t.Error("User and Project scopes should not be checked")
	}
	if m.main.scopeDialog.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.main.scopeDialog.cursor)
	}
}

// TestOpenScopeDialogForSelectedMultiScope tests opening scope dialog for multi-scope plugin.
func TestOpenScopeDialogForSelectedMultiScope(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{
			claude.ScopeUser:    true,
			claude.ScopeProject: true,
		}},
	}
	m.selectedIdx = 0

	m.openScopeDialogForSelected()

	// Verify User and Project are checked, Local is not
	if !m.main.scopeDialog.scopes[0] {
		t.Error("User scope should be checked (scopes[0] = true)")
	}
	if !m.main.scopeDialog.scopes[1] {
		t.Error("Project scope should be checked (scopes[1] = true)")
	}
	if m.main.scopeDialog.scopes[2] {
		t.Error("Local scope should not be checked (scopes[2] = false)")
	}
}

// TestOpenScopeDialogForSelectedNotInstalled tests opening dialog for uninstalled plugin.
func TestOpenScopeDialogForSelectedNotInstalled(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScopes: map[claude.Scope]bool{}},
	}
	m.selectedIdx = 0

	m.openScopeDialogForSelected()

	if m.mode != ModeScopeDialog {
		t.Errorf("mode = %v, want ModeScopeDialog", m.mode)
	}
	// No scopes should be checked
	if m.main.scopeDialog.scopes[0] || m.main.scopeDialog.scopes[1] || m.main.scopeDialog.scopes[2] {
		t.Error("no scopes should be checked for uninstalled plugin")
	}
}

// TestUpdateScopeDialogUpDown tests cursor navigation with up/down keys.
// Verifies plugin-scope-mgmt.AC6.2: Up/Down moves cursor through dialog items.
func TestUpdateScopeDialogUpDown(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeScopeDialog
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		cursor:         1,
		originalScopes: map[claude.Scope]bool{claude.ScopeProject: true},
	}
	m.keys = DefaultKeyBindings()

	// Move down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.updateScopeDialog(msg)
	m = result.(*Model)
	if m.main.scopeDialog.cursor != 2 {
		t.Errorf("cursor after down = %d, want 2", m.main.scopeDialog.cursor)
	}

	// Move up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)
	if m.main.scopeDialog.cursor != 1 {
		t.Errorf("cursor after up = %d, want 1", m.main.scopeDialog.cursor)
	}

	// Move up to 0
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)
	if m.main.scopeDialog.cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", m.main.scopeDialog.cursor)
	}

	// Try to move up past 0 (should stay at 0)
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)
	if m.main.scopeDialog.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", m.main.scopeDialog.cursor)
	}
}

// TestUpdateScopeDialogSpaceToggle tests space toggling checkbox state.
// Verifies plugin-scope-mgmt.AC6.2: Space toggles checkbox at cursor.
func TestUpdateScopeDialogSpaceToggle(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeScopeDialog
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		cursor:         0,
		scopes:         [3]bool{false, false, false},
		originalScopes: map[claude.Scope]bool{},
	}

	// Toggle on User scope
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	result, _ := m.updateScopeDialog(msg)
	m = result.(*Model)
	if !m.main.scopeDialog.scopes[0] {
		t.Error("User scope should be checked after space")
	}

	// Move to Project
	msg = tea.KeyMsg{Type: tea.KeyDown}
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)

	// Toggle on Project scope
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)
	if !m.main.scopeDialog.scopes[1] {
		t.Error("Project scope should be checked after space")
	}

	// Toggle off Project scope
	result, _ = m.updateScopeDialog(msg)
	m = result.(*Model)
	if m.main.scopeDialog.scopes[1] {
		t.Error("Project scope should be unchecked after second space")
	}
}

// TestUpdateScopeDialogEnter tests Enter key applies delta and returns to main.
// Verifies plugin-scope-mgmt.AC6.2: Enter computes delta and creates pending operations.
func TestUpdateScopeDialogEnter(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeScopeDialog
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		scopes:         [3]bool{true, false, false},
		originalScopes: map[claude.Scope]bool{},
	}
	m.main.pendingOps = make(map[string]Operation)
	m.keys = DefaultKeyBindings()

	// Press Enter to apply
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.updateScopeDialog(msg)
	m = result.(*Model)

	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}

	// Verify pending operation was created (user scope install)
	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Error("expected pending operation after Enter")
	}
	if op.Type != OpInstall {
		t.Errorf("Type = %v, want OpInstall", op.Type)
	}
	if len(op.Scopes) != 1 || op.Scopes[0] != claude.ScopeUser {
		t.Errorf("Scopes = %v, want [ScopeUser]", op.Scopes)
	}
}

// TestUpdateScopeDialogEscape tests Escape key cancels without applying.
// Verifies plugin-scope-mgmt.AC6.2: Esc exits dialog without changes.
func TestUpdateScopeDialogEscape(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.mode = ModeScopeDialog
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		scopes:         [3]bool{true, false, false},
		originalScopes: map[claude.Scope]bool{},
	}
	m.main.pendingOps = make(map[string]Operation)
	m.keys = DefaultKeyBindings()

	// Press Escape
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.updateScopeDialog(msg)
	m = result.(*Model)

	if m.mode != ModeMain {
		t.Errorf("mode = %v, want ModeMain", m.mode)
	}

	// No pending operation should be created
	if len(m.main.pendingOps) > 0 {
		t.Error("no pending operations should be created on Escape")
	}
}

// TestApplyScopeDialogDeltaCheckInstall tests delta computation for checking a new scope.
// Verifies plugin-scope-mgmt.AC6.3: Checking new scope creates OpInstall.
func TestApplyScopeDialogDeltaCheckInstall(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		scopes:         [3]bool{true, false, false},
		originalScopes: map[claude.Scope]bool{},
	}
	m.main.pendingOps = make(map[string]Operation)

	m.applyScopeDialogDelta()

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Fatal("expected pending operation")
	}
	if op.Type != OpInstall {
		t.Errorf("Type = %v, want OpInstall", op.Type)
	}
	if len(op.Scopes) != 1 || op.Scopes[0] != claude.ScopeUser {
		t.Errorf("Scopes = %v, want [ScopeUser]", op.Scopes)
	}
}

// TestApplyScopeDialogDeltaCheckUninstall tests delta computation for unchecking a scope.
// Verifies plugin-scope-mgmt.AC6.3: Unchecking scope creates OpUninstall.
func TestApplyScopeDialogDeltaCheckUninstall(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		scopes:         [3]bool{false, false, false},
		originalScopes: map[claude.Scope]bool{claude.ScopeLocal: true},
	}
	m.main.pendingOps = make(map[string]Operation)

	m.applyScopeDialogDelta()

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Fatal("expected pending operation")
	}
	if op.Type != OpUninstall {
		t.Errorf("Type = %v, want OpUninstall", op.Type)
	}
	if len(op.Scopes) != 1 || op.Scopes[0] != claude.ScopeLocal {
		t.Errorf("Scopes = %v, want [ScopeLocal]", op.Scopes)
	}
}

// TestApplyScopeDialogDeltaMixed tests delta computation with both install and uninstall.
// Verifies plugin-scope-mgmt.AC6.3: Mixed changes create OpScopeChange.
func TestApplyScopeDialogDeltaMixed(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.scopeDialog = scopeDialogState{
		pluginID: "test@marketplace",
		// Originally installed at Project and Local, now checking User and unchecking Project
		scopes:         [3]bool{true, false, true},
		originalScopes: map[claude.Scope]bool{claude.ScopeProject: true, claude.ScopeLocal: true},
	}
	m.main.pendingOps = make(map[string]Operation)

	m.applyScopeDialogDelta()

	op, ok := m.main.pendingOps["test@marketplace"]
	if !ok {
		t.Fatal("expected pending operation")
	}
	if op.Type != OpScopeChange {
		t.Errorf("Type = %v, want OpScopeChange", op.Type)
	}
	// Should have User as install scope
	if len(op.Scopes) != 1 || op.Scopes[0] != claude.ScopeUser {
		t.Errorf("Scopes (install) = %v, want [ScopeUser]", op.Scopes)
	}
	// Should have Project as uninstall scope
	if len(op.UninstallScopes) != 1 || op.UninstallScopes[0] != claude.ScopeProject {
		t.Errorf("UninstallScopes = %v, want [ScopeProject]", op.UninstallScopes)
	}
}

// TestApplyScopeDialogDeltaNoChange tests delta computation when nothing changes.
// Verifies plugin-scope-mgmt.AC6.3: No changes clears pending operation.
func TestApplyScopeDialogDeltaNoChange(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.main.scopeDialog = scopeDialogState{
		pluginID:       "test@marketplace",
		scopes:         [3]bool{true, false, false},
		originalScopes: map[claude.Scope]bool{claude.ScopeUser: true},
	}
	m.main.pendingOps = make(map[string]Operation)
	m.main.pendingOps["test@marketplace"] = Operation{PluginID: "test@marketplace", Type: OpInstall}

	m.applyScopeDialogDelta()

	// Pending operation should be cleared
	if len(m.main.pendingOps) > 0 {
		t.Error("pending operation should be cleared when no changes")
	}
}

// TestRenderScopeDialog tests dialog rendering with checkbox display.
// Verifies plugin-scope-mgmt.AC6.1 and AC6.4: Dialog renders with scope names and file paths.
func TestRenderScopeDialog(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test/project")
	m.width = 100
	m.height = 30
	m.mode = ModeScopeDialog
	m.main.scopeDialog = scopeDialogState{
		pluginID: "test@marketplace",
		scopes:   [3]bool{true, false, true},
		cursor:   1,
	}

	output := m.renderScopeDialog(m.styles)

	// Check title
	if !strings.Contains(output, "test@marketplace") {
		t.Error("output should contain plugin ID")
	}

	// Check scope names
	if !strings.Contains(output, "User") {
		t.Error("output should contain 'User' scope name")
	}
	if !strings.Contains(output, "Project") {
		t.Error("output should contain 'Project' scope name")
	}
	if !strings.Contains(output, "Local") {
		t.Error("output should contain 'Local' scope name")
	}

	// Check file paths (AC6.4)
	if !strings.Contains(output, "~/.claude/settings.json") {
		t.Error("output should contain User scope path")
	}
	if !strings.Contains(output, ".claude/settings.json") {
		t.Error("output should contain Project scope path")
	}
	if !strings.Contains(output, ".claude/settings.local.json") {
		t.Error("output should contain Local scope path")
	}

	// Check checkbox indicators
	if !strings.Contains(output, "[x]") {
		t.Error("output should contain checked boxes")
	}
	if !strings.Contains(output, "[ ]") {
		t.Error("output should contain unchecked boxes")
	}

	// Check cursor indicator
	if !strings.Contains(output, "> ") {
		t.Error("output should contain cursor indicator")
	}

	// Check help text
	if !strings.Contains(output, "space") || !strings.Contains(output, "Enter") || !strings.Contains(output, "Esc") {
		t.Error("output should contain help text with space, Enter, and Esc instructions")
	}
}

// TestExecuteOperationInstallWhenNotInSettings tests plugin-scope-mgmt.AC7.1
// When a plugin is not in settings, OpInstall should call InstallPlugin.
func TestExecuteOperationInstallWhenNotInSettings(t *testing.T) {
	client := &mockClient{}
	tmpDir := t.TempDir()

	// Track which methods were called
	var installCalled, enableCalled bool
	client.installFn = func(pluginID string, scope claude.Scope) error {
		installCalled = true
		if pluginID != "test@marketplace" || scope != claude.ScopeLocal {
			t.Errorf("InstallPlugin called with pluginID=%s, scope=%s; expected test@marketplace, ScopeLocal", pluginID, scope)
		}
		return nil
	}
	client.enableFn = func(_ string, _ claude.Scope) error {
		enableCalled = true
		return fmt.Errorf("should not call EnablePlugin when not in settings")
	}

	m := NewModel(client, tmpDir)
	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal},
		Type:     OpInstall,
	}

	// Execute the operation
	cmd := m.executeOperation(op)
	msg := cmd()

	// Check result
	if opMsg, ok := msg.(operationDoneMsg); ok {
		if opMsg.err != nil {
			t.Errorf("executeOperation returned error: %v", opMsg.err)
		}
	} else {
		t.Errorf("expected operationDoneMsg, got %T", msg)
	}

	if !installCalled {
		t.Error("InstallPlugin was not called")
	}
	if enableCalled {
		t.Error("EnablePlugin was unexpectedly called")
	}
}

// TestExecuteOperationInstallWhenInSettings tests plugin-scope-mgmt.AC7.1
// When a plugin already exists in settings, OpInstall should call EnablePlugin.
func TestExecuteOperationInstallWhenInSettings(t *testing.T) {
	client := &mockClient{}
	tmpDir := t.TempDir()

	// Write settings file with plugin already installed at ScopeLocal
	settingsPath := tmpDir + "/.claude/settings.local.json"
	os.Mkdir(tmpDir+"/.claude", 0o755)
	err := os.WriteFile(settingsPath, []byte(`{"enabledPlugins":{"test@marketplace":true}}`), 0o644)
	if err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	// Track which methods were called
	var installCalled, enableCalled bool
	client.installFn = func(_ string, _ claude.Scope) error {
		installCalled = true
		return fmt.Errorf("should not call InstallPlugin when already in settings")
	}
	client.enableFn = func(pluginID string, scope claude.Scope) error {
		enableCalled = true
		if pluginID != "test@marketplace" || scope != claude.ScopeLocal {
			t.Errorf("EnablePlugin called with pluginID=%s, scope=%s; expected test@marketplace, ScopeLocal", pluginID, scope)
		}
		return nil
	}

	m := NewModel(client, tmpDir)
	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal},
		Type:     OpInstall,
	}

	// Execute the operation
	cmd := m.executeOperation(op)
	msg := cmd()

	// Check result
	if opMsg, ok := msg.(operationDoneMsg); ok {
		if opMsg.err != nil {
			t.Errorf("executeOperation returned error: %v", opMsg.err)
		}
	} else {
		t.Errorf("expected operationDoneMsg, got %T", msg)
	}

	if installCalled {
		t.Error("InstallPlugin was unexpectedly called")
	}
	if !enableCalled {
		t.Error("EnablePlugin was not called")
	}
}

// TestExecuteOperationUninstallWhenInSettings tests plugin-scope-mgmt.AC7.2
// When a plugin exists in settings, OpUninstall should call UninstallPlugin.
func TestExecuteOperationUninstallWhenInSettings(t *testing.T) {
	client := &mockClient{}
	tmpDir := t.TempDir()

	// Write settings file with plugin installed at ScopeLocal
	os.Mkdir(tmpDir+"/.claude", 0o755)
	err := os.WriteFile(tmpDir+"/.claude/settings.local.json", []byte(`{"enabledPlugins":{"test@marketplace":true}}`), 0o644)
	if err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	// Track which methods were called
	var uninstallCalled, disableCalled bool
	client.uninstallFn = func(pluginID string, scope claude.Scope) error {
		uninstallCalled = true
		if pluginID != "test@marketplace" || scope != claude.ScopeLocal {
			t.Errorf("UninstallPlugin called with pluginID=%s, scope=%s; expected test@marketplace, ScopeLocal", pluginID, scope)
		}
		return nil
	}
	client.disableFn = func(_ string, _ claude.Scope) error {
		disableCalled = true
		return fmt.Errorf("should not call DisablePlugin when in settings")
	}

	m := NewModel(client, tmpDir)
	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal},
		Type:     OpUninstall,
	}

	// Execute the operation
	cmd := m.executeOperation(op)
	msg := cmd()

	// Check result
	if opMsg, ok := msg.(operationDoneMsg); ok {
		if opMsg.err != nil {
			t.Errorf("executeOperation returned error: %v", opMsg.err)
		}
	} else {
		t.Errorf("expected operationDoneMsg, got %T", msg)
	}

	if !uninstallCalled {
		t.Error("UninstallPlugin was not called")
	}
	if disableCalled {
		t.Error("DisablePlugin was unexpectedly called")
	}
}

// TestExecuteOperationMultiScopeStopsOnFirstError tests plugin-scope-mgmt.AC7.3
// Multi-scope operations should stop on first error and report which scope failed.
func TestExecuteOperationMultiScopeStopsOnFirstError(t *testing.T) {
	client := &mockClient{}
	tmpDir := t.TempDir()

	// Track which scopes were attempted
	var attemptedScopes []claude.Scope
	client.installFn = func(_ string, scope claude.Scope) error {
		attemptedScopes = append(attemptedScopes, scope)
		// Fail on second scope
		if len(attemptedScopes) == 2 {
			return fmt.Errorf("network error")
		}
		return nil
	}

	m := NewModel(client, tmpDir)
	op := Operation{
		PluginID: "test@marketplace",
		Scopes:   []claude.Scope{claude.ScopeLocal, claude.ScopeProject, claude.ScopeUser},
		Type:     OpInstall,
	}

	// Execute the operation
	cmd := m.executeOperation(op)
	msg := cmd()

	// Check result
	opMsg, ok := msg.(operationDoneMsg)
	if !ok {
		t.Fatalf("expected operationDoneMsg, got %T", msg)
	}

	if opMsg.err == nil {
		t.Error("expected error, got nil")
	}

	// Check error message includes scope name
	if !strings.Contains(opMsg.err.Error(), "scope") {
		t.Errorf("error message should include 'scope', got: %v", opMsg.err)
	}
	if !strings.Contains(opMsg.err.Error(), "project") {
		t.Errorf("error message should include 'project' scope, got: %v", opMsg.err)
	}

	// Only first scope should have been attempted before failure
	if len(attemptedScopes) != 2 {
		t.Errorf("expected 2 scopes attempted, got %d", len(attemptedScopes))
	}
}

// TestStartExecutionOperationOrdering tests plugin-scope-mgmt.AC7.4
// Operations should be sorted in correct order: Uninstall, Migrate, ScopeChange, Update, Install, Enable, Disable
func TestStartExecutionOperationOrdering(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client, "/test")
	m.plugins = []PluginState{}
	m.mode = ModeMain

	// Create pending operations in random order
	m.main.pendingOps = map[string]Operation{
		"plugin1": {PluginID: "plugin1", Type: OpEnable, Scopes: []claude.Scope{claude.ScopeLocal}},
		"plugin2": {PluginID: "plugin2", Type: OpInstall, Scopes: []claude.Scope{claude.ScopeLocal}},
		"plugin3": {PluginID: "plugin3", Type: OpUninstall, Scopes: []claude.Scope{claude.ScopeLocal}},
		"plugin4": {PluginID: "plugin4", Type: OpUpdate, Scopes: []claude.Scope{claude.ScopeLocal}},
		"plugin5": {PluginID: "plugin5", Type: OpMigrate, Scopes: []claude.Scope{claude.ScopeLocal}, OriginalScopes: map[claude.Scope]bool{claude.ScopeProject: true}},
		"plugin6": {PluginID: "plugin6", Type: OpDisable, Scopes: []claude.Scope{claude.ScopeLocal}},
		"plugin7": {PluginID: "plugin7", Type: OpScopeChange, Scopes: []claude.Scope{claude.ScopeLocal}, UninstallScopes: []claude.Scope{claude.ScopeProject}},
	}

	// Call startExecution
	model, _ := m.startExecution()
	m = model.(*Model)

	// Check ordering
	expectedOrder := []OperationType{
		OpUninstall,   // 0
		OpMigrate,     // 1
		OpScopeChange, // 2
		OpUpdate,      // 3
		OpInstall,     // 4
		OpEnable,      // 5
		OpDisable,     // 6
	}

	if len(m.progress.operations) != len(expectedOrder) {
		t.Fatalf("expected %d operations, got %d", len(expectedOrder), len(m.progress.operations))
	}

	for i, op := range m.progress.operations {
		if op.Type != expectedOrder[i] {
			t.Errorf("operation %d: Type = %v, want %v", i, op.Type, expectedOrder[i])
		}
	}
}

// TestFormatScopeSet_SingleScope tests AC4.1
// Call formatScopeSet with single scope map, verify output is the scope label (e.g., "USER")
func TestFormatScopeSet_SingleScope(t *testing.T) {
	scopes := map[claude.Scope]bool{claude.ScopeUser: true}
	result := formatScopeSet(scopes)
	expected := "USER"
	if result != expected {
		t.Errorf("formatScopeSet(single scope) = %q, want %q", result, expected)
	}
}

// TestFormatScopeSet_MultiScope tests AC4.1
// Call with {ScopeUser: true, ScopeLocal: true}, verify output is "USER, LOCAL"
func TestFormatScopeSet_MultiScope(t *testing.T) {
	scopes := map[claude.Scope]bool{claude.ScopeUser: true, claude.ScopeLocal: true}
	result := formatScopeSet(scopes)
	expected := "USER, LOCAL"
	if result != expected {
		t.Errorf("formatScopeSet(multi scope) = %q, want %q", result, expected)
	}
}

// TestFormatScopeSet_Ordering tests AC4.1
// Call with all 3 scopes, verify output order is "USER, PROJECT, LOCAL" (canonical order)
func TestFormatScopeSet_Ordering(t *testing.T) {
	scopes := map[claude.Scope]bool{
		claude.ScopeUser:    true,
		claude.ScopeProject: true,
		claude.ScopeLocal:   true,
	}
	result := formatScopeSet(scopes)
	expected := "USER, PROJECT, LOCAL"
	if result != expected {
		t.Errorf("formatScopeSet(all scopes) = %q, want %q", result, expected)
	}
}

// TestRenderPendingIndicator_PartialUninstall tests AC4.2
// Create Operation with OpUninstall and multi-scope PluginState, verify output contains transition text
func TestRenderPendingIndicator_PartialUninstall(t *testing.T) {
	plugin := PluginState{
		ID:              "test@marketplace",
		InstalledScopes: map[claude.Scope]bool{claude.ScopeUser: true, claude.ScopeLocal: true},
		IsGroupHeader:   false,
	}
	op := Operation{
		PluginID:       "test@marketplace",
		Type:           OpUninstall,
		Scopes:         []claude.Scope{claude.ScopeLocal},
		OriginalScopes: map[claude.Scope]bool{claude.ScopeUser: true, claude.ScopeLocal: true},
	}

	styles := DefaultStyles()
	result := renderPendingIndicator(op, plugin, styles)

	// The result contains ANSI codes, so check for expected text fragments
	if !strings.Contains(result, "->") {
		t.Errorf("renderPendingIndicator(partial uninstall) should contain transition arrow '->': %q", result)
	}
	if !strings.Contains(result, "USER") {
		t.Errorf("renderPendingIndicator(partial uninstall) should contain 'USER': %q", result)
	}
}

// TestRenderPendingIndicator_ScopeChange tests AC4.2
// Create OpScopeChange operation, verify output shows scope transition
func TestRenderPendingIndicator_ScopeChange(t *testing.T) {
	plugin := PluginState{
		ID:              "test@marketplace",
		InstalledScopes: map[claude.Scope]bool{claude.ScopeLocal: true},
		IsGroupHeader:   false,
	}
	op := Operation{
		PluginID:        "test@marketplace",
		Type:            OpScopeChange,
		Scopes:          []claude.Scope{claude.ScopeProject},
		UninstallScopes: []claude.Scope{claude.ScopeLocal},
	}

	styles := DefaultStyles()
	result := renderPendingIndicator(op, plugin, styles)

	// The result contains ANSI codes, so check for expected text fragments
	if !strings.Contains(result, "->") {
		t.Errorf("renderPendingIndicator(scope change) should contain transition arrow '->': %q", result)
	}
	if !strings.Contains(result, "LOCAL") {
		t.Errorf("renderPendingIndicator(scope change) should contain 'LOCAL': %q", result)
	}
	if !strings.Contains(result, "PROJECT") {
		t.Errorf("renderPendingIndicator(scope change) should contain 'PROJECT': %q", result)
	}
}

// TestRenderPendingIndicator_MultiScopeInstall tests AC4.2
// Create OpInstall on existing multi-scope plugin, verify output shows combined scopes
func TestRenderPendingIndicator_MultiScopeInstall(t *testing.T) {
	plugin := PluginState{
		ID:              "test@marketplace",
		InstalledScopes: map[claude.Scope]bool{claude.ScopeUser: true},
		IsGroupHeader:   false,
	}
	op := Operation{
		PluginID: "test@marketplace",
		Type:     OpInstall,
		Scopes:   []claude.Scope{claude.ScopeLocal},
	}

	styles := DefaultStyles()
	result := renderPendingIndicator(op, plugin, styles)

	// The result contains ANSI codes, so check for expected text fragments
	if !strings.Contains(result, "->") {
		t.Errorf("renderPendingIndicator(multi-scope install) should contain transition arrow '->': %q", result)
	}
	// Should show combined scopes (USER and LOCAL)
	if !strings.Contains(result, "USER") {
		t.Errorf("renderPendingIndicator(multi-scope install) should contain 'USER': %q", result)
	}
	if !strings.Contains(result, "LOCAL") {
		t.Errorf("renderPendingIndicator(multi-scope install) should contain 'LOCAL': %q", result)
	}
}
