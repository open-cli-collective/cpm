// Package claude provides a client for interacting with the Claude Code CLI.
package claude

import (
	"encoding/json"
	"fmt"
)

// Scope represents the installation scope of a plugin.
type Scope string

const (
	// ScopeNone indicates no scope (used for uninstall operations).
	ScopeNone Scope = ""
	// ScopeUser is the global user scope (~/.claude/settings.json).
	ScopeUser Scope = "user"
	// ScopeProject is the shared project scope (.claude/settings.json).
	ScopeProject Scope = "project"
	// ScopeLocal is the local project scope (.claude/settings.local.json).
	ScopeLocal Scope = "local"
)

// InstalledPlugin represents a plugin that is currently installed.
type InstalledPlugin struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	InstallPath string `json:"installPath"`
	InstalledAt string `json:"installedAt"`
	LastUpdated string `json:"lastUpdated"`
	ProjectPath string `json:"projectPath,omitempty"`
	Scope       Scope  `json:"scope"`
	Enabled     bool   `json:"enabled"`
}

// AvailablePlugin represents a plugin available from a marketplace.
type AvailablePlugin struct {
	PluginID        string `json:"pluginId"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	MarketplaceName string `json:"marketplaceName"`
	Source          any    `json:"source,omitempty"` // Can be string or object depending on plugin type
	Version         string `json:"version,omitempty"`
	InstallCount    int    `json:"installCount,omitempty"`
}

// PluginList is the response from `claude plugin list --json --available`.
type PluginList struct {
	Installed []InstalledPlugin `json:"installed"`
	Available []AvailablePlugin `json:"available"`
}

// MarketplaceSource is the interface for marketplace source type discriminated union.
type MarketplaceSource interface {
	SourceType() string
}

// GitHubSource represents a GitHub-hosted marketplace source.
type GitHubSource struct {
	Repo string `json:"repo"`
	Ref  string `json:"ref,omitempty"`
	Path string `json:"path,omitempty"`
}

func (s GitHubSource) SourceType() string { return "github" }

// GitSource represents a generic Git repository source.
type GitSource struct {
	URL  string `json:"url"`
	Ref  string `json:"ref,omitempty"`
	Path string `json:"path,omitempty"`
}

func (s GitSource) SourceType() string { return "git" }

// URLSource represents a URL-based marketplace source.
type URLSource struct {
	Headers map[string]string `json:"headers,omitempty"`
	URL     string            `json:"url"`
}

func (s URLSource) SourceType() string { return "url" }

// NPMSource represents an NPM package source.
type NPMSource struct {
	Package string `json:"package"`
}

func (s NPMSource) SourceType() string { return "npm" }

// FileSource represents a local file source.
type FileSource struct {
	Path string `json:"path"`
}

func (s FileSource) SourceType() string { return "file" }

// DirectorySource represents a local directory source.
type DirectorySource struct {
	Path string `json:"path"`
}

func (s DirectorySource) SourceType() string { return "directory" }

// HostPatternSource represents a host-pattern based source.
type HostPatternSource struct {
	HostPattern string `json:"hostPattern"`
}

func (s HostPatternSource) SourceType() string { return "hostPattern" }

// unmarshalSource parses JSON with a "source" discriminator field into the correct concrete type.
func unmarshalSource(data []byte) (MarketplaceSource, error) {
	var disc struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return nil, err
	}

	target := newSourceByType(disc.Source)
	if target == nil {
		return nil, fmt.Errorf("unknown marketplace source type: %q", disc.Source)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return nil, err
	}
	return target, nil
}

// newSourceByType returns a pointer to a zero-value concrete source type.
func newSourceByType(sourceType string) MarketplaceSource {
	switch sourceType {
	case "github":
		return &GitHubSource{}
	case "git":
		return &GitSource{}
	case "url":
		return &URLSource{}
	case "npm":
		return &NPMSource{}
	case "file":
		return &FileSource{}
	case "directory":
		return &DirectorySource{}
	case "hostPattern":
		return &HostPatternSource{}
	default:
		return nil
	}
}

// marshalSource serializes a MarketplaceSource, injecting the "source" discriminator.
func marshalSource(s MarketplaceSource) (json.RawMessage, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	// Inject "source" discriminator into the JSON object
	var fields map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &fields); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	disc, marshalErr := json.Marshal(s.SourceType())
	if marshalErr != nil {
		return nil, marshalErr
	}
	fields["source"] = disc
	return json.Marshal(fields)
}

// MarketplaceEntry represents a value in the extraKnownMarketplaces settings map.
type MarketplaceEntry struct {
	Source MarketplaceSource
}

func (e MarketplaceEntry) MarshalJSON() ([]byte, error) {
	src, err := marshalSource(e.Source)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Source json.RawMessage `json:"source"`
	}{Source: src})
}

func (e *MarketplaceEntry) UnmarshalJSON(data []byte) error {
	var raw struct {
		Source json.RawMessage `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	src, err := unmarshalSource(raw.Source)
	if err != nil {
		return err
	}
	e.Source = src
	return nil
}

// KnownMarketplace represents a value in known_marketplaces.json.
type KnownMarketplace struct {
	Source          MarketplaceSource
	InstallLocation string `json:"installLocation"`
	LastUpdated     string `json:"lastUpdated"`
	AutoUpdate      bool   `json:"autoUpdate,omitempty"`
}

func (k *KnownMarketplace) UnmarshalJSON(data []byte) error {
	// Parse non-source fields normally
	type Alias KnownMarketplace
	var raw struct {
		Alias
		Source json.RawMessage `json:"source"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*k = KnownMarketplace(raw.Alias)
	src, err := unmarshalSource(raw.Source)
	if err != nil {
		return err
	}
	k.Source = src
	return nil
}
