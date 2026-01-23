package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	plugins     *claude.PluginList
	err         error
	installFn   func(string, claude.Scope) error
	uninstallFn func(string, claude.Scope) error
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

// TestUpdateConfirmationEnterStartsExecution tests that Enter in confirmation starts execution.
func TestUpdateConfirmationEnterStartsExecution(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "test@marketplace", Name: "test", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0
	m.pending["test@marketplace"] = claude.ScopeLocal
	m.showConfirm = true

	// Send Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.updateConfirmation(msg)
	m = result.(*Model)

	// Should exit confirmation mode and start execution
	if m.showConfirm {
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
	m := NewModel(client)
	m.pending["test@marketplace"] = claude.ScopeLocal
	m.showConfirm = true

	// Send Escape key
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := m.updateConfirmation(msg)
	m = result.(*Model)

	// Should exit confirmation mode without executing
	if m.showConfirm {
		t.Error("showConfirm should be false after Escape")
	}
	if m.mode != ModeMain {
		t.Errorf("mode = %d, want ModeMain", m.mode)
	}
	// Pending changes should remain
	if _, ok := m.pending["test@marketplace"]; !ok {
		t.Error("pending changes should not be cleared on cancel")
	}
}

// TestStartExecutionBuildsOperations tests that startExecution builds operations correctly.
func TestStartExecutionBuildsOperations(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.plugins = []PluginState{
		{ID: "plugin1@market", Name: "plugin1", InstalledScope: claude.ScopeLocal},
		{ID: "plugin2@market", Name: "plugin2", InstalledScope: claude.ScopeNone},
	}
	m.selectedIdx = 0

	// Set pending: uninstall plugin1 (local), install plugin2 (project)
	m.pending["plugin1@market"] = claude.ScopeNone    // uninstall
	m.pending["plugin2@market"] = claude.ScopeProject // install

	result, _ := m.startExecution()
	m = result.(*Model)

	if len(m.operations) != 2 {
		t.Errorf("len(operations) = %d, want 2", len(m.operations))
	}
	if m.currentOpIdx != 0 {
		t.Errorf("currentOpIdx = %d, want 0", m.currentOpIdx)
	}
	if m.mode != ModeProgress {
		t.Errorf("mode = %d, want ModeProgress", m.mode)
	}

	// Check that uninstall captured original scope
	found := false
	for _, op := range m.operations {
		if op.PluginID == "plugin1@market" && !op.IsInstall {
			if op.OriginalScope != claude.ScopeLocal {
				t.Errorf("uninstall OriginalScope = %v, want ScopeLocal", op.OriginalScope)
			}
			found = true
		}
	}
	if !found {
		t.Error("uninstall operation not found or OriginalScope not set")
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

	m := NewModel(client)
	op := Operation{
		PluginID:  "test@marketplace",
		Scope:     claude.ScopeLocal,
		IsInstall: true,
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

// TestExecuteOperationUninstallUsesOriginalScope tests that executeOperation uses OriginalScope for uninstalls.
func TestExecuteOperationUninstallUsesOriginalScope(t *testing.T) {
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

	m := NewModel(client)
	op := Operation{
		PluginID:      "test@marketplace",
		Scope:         claude.ScopeNone, // marked for uninstall
		IsInstall:     false,
		OriginalScope: claude.ScopeProject, // was installed at project scope
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

	m := NewModel(client)
	m.mode = ModeProgress
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
		{PluginID: "p2@m", Scope: claude.ScopeProject, IsInstall: true},
	}
	m.currentOpIdx = 0
	m.operationErrors = make([]string, 2)

	// Simulate first operation completing
	doneMsg := operationDoneMsg{op: m.operations[0], err: nil}
	result, cmd := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.currentOpIdx != 1 {
		t.Errorf("currentOpIdx = %d, want 1 after first operation", m.currentOpIdx)
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
	m := NewModel(client)
	m.mode = ModeProgress
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
	}
	m.currentOpIdx = 0
	m.operationErrors = make([]string, 1)

	// Simulate operation completing
	doneMsg := operationDoneMsg{op: m.operations[0], err: nil}
	result, cmd := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary", m.mode)
	}
	if len(m.pending) != 0 {
		t.Error("pending should be cleared after completion")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (should load plugins)")
	}
}

// TestUpdateProgressRecordsErrors tests that errors are recorded correctly.
func TestUpdateProgressRecordsErrors(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.mode = ModeProgress
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
		{PluginID: "p2@m", Scope: claude.ScopeProject, IsInstall: true},
	}
	m.currentOpIdx = 0
	m.operationErrors = make([]string, 2)

	// Simulate first operation failing
	doneMsg := operationDoneMsg{op: m.operations[0], err: fmt.Errorf("install failed")}
	result, _ := m.updateProgress(doneMsg)
	m = result.(*Model)

	if m.operationErrors[0] != "install failed" {
		t.Errorf("operationErrors[0] = %q, want 'install failed'", m.operationErrors[0])
	}
}

// TestUpdateErrorReturnsToMain tests that error summary returns to main view on Enter/Esc.
func TestUpdateErrorReturnsToMain(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.mode = ModeSummary
	m.operations = []Operation{{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true}}
	m.operationErrors = []string{""}

	// Send Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.updateError(msg)
	m = result.(*Model)

	if m.mode != ModeMain {
		t.Errorf("mode = %d, want ModeMain", m.mode)
	}
	if m.operations != nil {
		t.Error("operations should be cleared when returning to main")
	}
	if m.operationErrors != nil {
		t.Error("operationErrors should be cleared when returning to main")
	}
}

// TestUpdateErrorHandlesPluginsLoaded tests that summary updates when plugins reload.
func TestUpdateErrorHandlesPluginsLoaded(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
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
	m := NewModel(client)
	m.width = 100
	m.height = 30
	m.pending["p1@market"] = claude.ScopeLocal
	m.pending["p2@market"] = claude.ScopeNone

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

// TestRenderProgressOutput tests that renderProgress shows operation status.
func TestRenderProgressOutput(t *testing.T) {
	client := &mockClient{}
	m := NewModel(client)
	m.width = 100
	m.height = 30
	m.mode = ModeProgress
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
		{PluginID: "p2@m", Scope: claude.ScopeNone, IsInstall: false, OriginalScope: claude.ScopeProject},
	}
	m.currentOpIdx = 0
	m.operationErrors = []string{"", ""}

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
	m := NewModel(client)
	m.width = 100
	m.height = 30
	m.mode = ModeSummary
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
		{PluginID: "p2@m", Scope: claude.ScopeProject, IsInstall: true},
	}
	m.operationErrors = []string{"", ""}

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
	m := NewModel(client)
	m.width = 100
	m.height = 30
	m.mode = ModeSummary
	m.operations = []Operation{
		{PluginID: "p1@m", Scope: claude.ScopeLocal, IsInstall: true},
		{PluginID: "p2@m", Scope: claude.ScopeProject, IsInstall: true},
	}
	m.operationErrors = []string{"", "install failed"}

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
