package version

import (
	"testing"
	"time"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected VersionInfo
	}{
		{"v1.0.0", VersionInfo{Major: 1, Minor: 0, Patch: 0, Pre: ""}},
		{"1.0.0", VersionInfo{Major: 1, Minor: 0, Patch: 0, Pre: ""}},
		{"v1.2.3", VersionInfo{Major: 1, Minor: 2, Patch: 3, Pre: ""}},
		{"v1.0.0-beta", VersionInfo{Major: 1, Minor: 0, Patch: 0, Pre: "beta"}},
		{"v2.0.0-alpha.1", VersionInfo{Major: 2, Minor: 0, Patch: 0, Pre: "alpha.1"}},
		{"v0.1.0", VersionInfo{Major: 0, Minor: 1, Patch: 0, Pre: ""}},
		{"dev", VersionInfo{Major: 0, Minor: 0, Patch: 0, Pre: ""}},
	}

	for _, test := range tests {
		result := ParseVersion(test.input)
		if result.Major != test.expected.Major ||
			result.Minor != test.expected.Minor ||
			result.Patch != test.expected.Patch ||
			result.Pre != test.expected.Pre {
			t.Errorf("ParseVersion(%q) = %+v, expected %+v", test.input, result, test.expected)
		}
	}
}

func TestVersionInfoGreaterThan(t *testing.T) {
	tests := []struct {
		v1       VersionInfo
		v2       VersionInfo
		expected bool
	}{
		// Major version
		{VersionInfo{1, 0, 0, ""}, VersionInfo{0, 9, 9, ""}, true},
		{VersionInfo{0, 9, 9, ""}, VersionInfo{1, 0, 0, ""}, false},

		// Minor version
		{VersionInfo{1, 1, 0, ""}, VersionInfo{1, 0, 9, ""}, true},
		{VersionInfo{1, 0, 9, ""}, VersionInfo{1, 1, 0, ""}, false},

		// Patch version
		{VersionInfo{1, 0, 1, ""}, VersionInfo{1, 0, 0, ""}, true},
		{VersionInfo{1, 0, 0, ""}, VersionInfo{1, 0, 1, ""}, false},

		// Pre-release
		{VersionInfo{1, 0, 0, ""}, VersionInfo{1, 0, 0, "beta"}, true},
		{VersionInfo{1, 0, 0, "beta"}, VersionInfo{1, 0, 0, ""}, false},
		{VersionInfo{1, 0, 0, "rc.1"}, VersionInfo{1, 0, 0, "beta"}, true},

		// Equal
		{VersionInfo{1, 0, 0, ""}, VersionInfo{1, 0, 0, ""}, false},
	}

	for _, test := range tests {
		result := test.v1.GreaterThan(test.v2)
		if result != test.expected {
			t.Errorf("%+v.GreaterThan(%+v) = %v, expected %v", test.v1, test.v2, result, test.expected)
		}
	}
}

func TestCachedResultIsExpired(t *testing.T) {
	// Test expired
	expired := &CachedResult{
		CheckedAt: time.Now().Add(-25 * time.Hour),
	}
	if !expired.IsExpired(24 * time.Hour) {
		t.Error("Expected cache to be expired")
	}

	// Test not expired
	fresh := &CachedResult{
		CheckedAt: time.Now().Add(-1 * time.Hour),
	}
	if fresh.IsExpired(24 * time.Hour) {
		t.Error("Expected cache to not be expired")
	}

	// Test exactly at boundary (should be expired since time.Since returns > interval)
	boundary := &CachedResult{
		CheckedAt: time.Now().Add(-24 * time.Hour).Add(-1 * time.Second),
	}
	if !boundary.IsExpired(24 * time.Hour) {
		t.Error("Expected cache at boundary to be expired")
	}
}

func TestCheckerShouldCheck(t *testing.T) {
	checker := NewDefaultChecker()

	// Initially should check
	if !checker.ShouldCheck() {
		t.Error("Expected ShouldCheck to be true initially")
	}

	// After check, should not check again immediately
	checker.CheckNow()
	if checker.ShouldCheck() {
		t.Error("Expected ShouldCheck to be false after just checking")
	}
}

func TestCheckerState(t *testing.T) {
	tests := []struct {
		state    CheckerState
		expected string
	}{
		{StateIdle, "idle"},
		{StateChecking, "checking"},
		{StateError, "error"},
		{CheckerState(999), "unknown"},
	}

	for _, test := range tests {
		result := test.state.String()
		if result != test.expected {
			t.Errorf("CheckerState(%d).String() = %q, expected %q", test.state, result, test.expected)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"  v1.0.0  ", "1.0.0"},
		{"v2.0.0-beta", "2.0.0-beta"},
	}

	for _, test := range tests {
		result := normalizeVersion(test.input)
		if result != test.expected {
			t.Errorf("normalizeVersion(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestVersionInfoString(t *testing.T) {
	tests := []struct {
		v        VersionInfo
		expected string
	}{
		{VersionInfo{1, 0, 0, ""}, "1.0.0"},
		{VersionInfo{1, 2, 3, ""}, "1.2.3"},
		{VersionInfo{1, 0, 0, "beta"}, "1.0.0-beta"},
		{VersionInfo{2, 0, 0, "alpha.1"}, "2.0.0-alpha.1"},
	}

	for _, test := range tests {
		result := test.v.String()
		if result != test.expected {
			t.Errorf("%+v.String() = %q, expected %q", test.v, result, test.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s       string
		maxLen  int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"exactly", 7, "exactly"},
	}

	for _, test := range tests {
		result := truncateString(test.s, test.maxLen)
		if result != test.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", test.s, test.maxLen, result, test.expected)
		}
	}
}

func TestIsDevVersion(t *testing.T) {
	// Save original version
	originalVersion := Version

	// Test dev version
	Version = "dev"
	if !IsDevVersion() {
		t.Error("Expected IsDevVersion() to be true for 'dev'")
	}

	// Test unknown version
	Version = "unknown"
	if !IsDevVersion() {
		t.Error("Expected IsDevVersion() to be true for 'unknown'")
	}

	// Test release version
	Version = "v1.0.0"
	if IsDevVersion() {
		t.Error("Expected IsDevVersion() to be false for 'v1.0.0'")
	}

	// Restore original version
	Version = originalVersion
}
