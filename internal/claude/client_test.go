package claude

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Verify it's a *realClient
	_, ok := client.(*realClient)
	if !ok {
		t.Error("NewClient() did not return *realClient")
	}
}

func TestNewClientWithPath(t *testing.T) {
	client := NewClientWithPath("/usr/local/bin/claude")
	if client == nil {
		t.Fatal("NewClientWithPath() returned nil")
	}

	rc, ok := client.(*realClient)
	if !ok {
		t.Fatal("NewClientWithPath() did not return *realClient")
	}
	if rc.claudePath != "/usr/local/bin/claude" {
		t.Errorf("claudePath = %q, want %q", rc.claudePath, "/usr/local/bin/claude")
	}
}
