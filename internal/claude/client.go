package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Client defines the interface for interacting with the Claude CLI.
type Client interface {
	// ListPlugins returns installed and optionally available plugins.
	ListPlugins(includeAvailable bool) (*PluginList, error)

	// InstallPlugin installs a plugin at the specified scope.
	InstallPlugin(pluginID string, scope Scope) error

	// UninstallPlugin removes a plugin from the specified scope.
	UninstallPlugin(pluginID string, scope Scope) error

	// EnablePlugin enables a plugin at the specified scope.
	EnablePlugin(pluginID string, scope Scope) error

	// DisablePlugin disables a plugin at the specified scope.
	DisablePlugin(pluginID string, scope Scope) error
}

// realClient implements Client by shelling out to the claude CLI.
type realClient struct {
	claudePath string
}

// NewClient creates a new Client using "claude" from PATH.
func NewClient() Client {
	return &realClient{claudePath: "claude"}
}

// NewClientWithPath creates a new Client using the specified claude binary path.
func NewClientWithPath(path string) Client {
	return &realClient{claudePath: path}
}

// ListPlugins implements Client.ListPlugins.
func (c *realClient) ListPlugins(includeAvailable bool) (*PluginList, error) {
	args := []string{"plugin", "list", "--json"}
	if includeAvailable {
		args = append(args, "--available")
	}

	// #nosec G204 -- args are hardcoded, not user input
	cmd := exec.Command(c.claudePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude plugin list failed: %w: %s", err, stderr.String())
	}

	var list PluginList
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("failed to parse plugin list: %w", err)
	}

	return &list, nil
}

// InstallPlugin implements Client.InstallPlugin.
func (c *realClient) InstallPlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "install"}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}
	args = append(args, pluginID)

	// #nosec G204 -- args are constructed safely from enum scope
	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin install failed: %w: %s", err, stderr.String())
	}

	return nil
}

// UninstallPlugin implements Client.UninstallPlugin.
func (c *realClient) UninstallPlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "uninstall"}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}
	args = append(args, pluginID)

	// #nosec G204 -- args are constructed safely from enum scope
	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin uninstall failed: %w: %s", err, stderr.String())
	}

	return nil
}

// EnablePlugin implements Client.EnablePlugin.
func (c *realClient) EnablePlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "enable"}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}
	args = append(args, pluginID)

	// #nosec G204 -- args are constructed safely from enum scope
	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin enable failed: %w: %s", err, stderr.String())
	}

	return nil
}

// DisablePlugin implements Client.DisablePlugin.
func (c *realClient) DisablePlugin(pluginID string, scope Scope) error {
	args := []string{"plugin", "disable"}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}
	args = append(args, pluginID)

	// #nosec G204 -- args are constructed safely from enum scope
	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin disable failed: %w: %s", err, stderr.String())
	}

	return nil
}
