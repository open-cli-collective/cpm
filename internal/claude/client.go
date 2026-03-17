package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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

	// Write stdout to a temp file instead of a pipe. The claude CLI (Node.js)
	// can truncate pipe output at 64KB when the process exits before the OS
	// pipe buffer is fully drained. File-based capture avoids this.
	tmpFile, err := os.CreateTemp("", "cpm-plugins-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName) //nolint:errcheck // best-effort cleanup

	// #nosec G204 -- args are hardcoded, not user input
	cmd := exec.Command(c.claudePath, args...)
	cmd.Stdout = tmpFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	_ = tmpFile.Close()
	if runErr != nil {
		return nil, fmt.Errorf("claude plugin list failed: %w: %s", runErr, stderr.String())
	}

	stdout, err := os.ReadFile(tmpName) // #nosec G304 -- path from CreateTemp, not user input
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin list output: %w", err)
	}

	var list PluginList
	if err := json.Unmarshal(stdout, &list); err != nil {
		return nil, fmt.Errorf("failed to parse plugin list: %w", err)
	}

	return &list, nil
}

// runPluginCommand executes a claude plugin subcommand (install, uninstall, enable, disable).
func (c *realClient) runPluginCommand(command, pluginID string, scope Scope) error {
	args := []string{"plugin", command}
	if scope != ScopeNone {
		args = append(args, "--scope", string(scope))
	}
	args = append(args, pluginID)

	// #nosec G204 -- args are constructed safely from enum scope
	cmd := exec.Command(c.claudePath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude plugin %s failed: %w: %s", command, err, stderr.String())
	}

	return nil
}

// InstallPlugin implements Client.InstallPlugin.
func (c *realClient) InstallPlugin(pluginID string, scope Scope) error {
	return c.runPluginCommand("install", pluginID, scope)
}

// UninstallPlugin implements Client.UninstallPlugin.
func (c *realClient) UninstallPlugin(pluginID string, scope Scope) error {
	return c.runPluginCommand("uninstall", pluginID, scope)
}

// EnablePlugin implements Client.EnablePlugin.
func (c *realClient) EnablePlugin(pluginID string, scope Scope) error {
	return c.runPluginCommand("enable", pluginID, scope)
}

// DisablePlugin implements Client.DisablePlugin.
func (c *realClient) DisablePlugin(pluginID string, scope Scope) error {
	return c.runPluginCommand("disable", pluginID, scope)
}
