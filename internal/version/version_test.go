package version

import "testing"

func TestString(t *testing.T) {
	// Test default values
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if Date == "" {
		Date = "unknown"
	}
	if Branch == "" {
		Branch = "unknown"
	}

	s := String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// Should contain version
	if !contains(s, Version) {
		t.Errorf("String() = %q, should contain version %q", s, Version)
	}

	// Should contain branch
	if !contains(s, Branch) {
		t.Errorf("String() = %q, should contain branch %q", s, Branch)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
