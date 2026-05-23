package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRulesFromDir(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		got := loadRulesFromDir(dir)
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		got := loadRulesFromDir("/non/existent/path")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("single markdown file", func(t *testing.T) {
		dir := t.TempDir()
		content := "# Test Rule\n\nAlways use camelCase."
		if err := os.WriteFile(filepath.Join(dir, "coding.md"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		got := loadRulesFromDir(dir)
		want := "## coding.md\n\n# Test Rule\n\nAlways use camelCase."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("multiple files sorted", func(t *testing.T) {
		dir := t.TempDir()
		files := map[string]string{
			"z-last.md":    "Last rule.",
			"a-first.md":   "First rule.",
			"m-middle.txt": "Middle rule.",
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		got := loadRulesFromDir(dir)
		want := "## a-first.md\n\nFirst rule.\n\n## m-middle.txt\n\nMiddle rule.\n\n## z-last.md\n\nLast rule."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("skips non-md-txt files and subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "valid.md"), []byte("Valid rule."), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skip.go"), []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "subdir", "nested.md"), []byte("Nested."), 0644); err != nil {
			t.Fatal(err)
		}

		got := loadRulesFromDir(dir)
		want := "## valid.md\n\nValid rule."
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("skips empty files", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "empty.md"), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "spaces.md"), []byte("   \n\t  "), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "valid.md"), []byte("Valid"), 0644); err != nil {
			t.Fatal(err)
		}

		got := loadRulesFromDir(dir)
		want := "## valid.md\n\nValid"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ignores files that fail to read", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "valid.md"), []byte("Valid"), 0644); err != nil {
			t.Fatal(err)
		}
		// Create a directory with a file name — ReadFile will fail on it
		if err := os.Mkdir(filepath.Join(dir, "not-a-file.md"), 0755); err != nil {
			t.Fatal(err)
		}

		got := loadRulesFromDir(dir)
		want := "## valid.md\n\nValid"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestLoadCustomRules(t *testing.T) {
	// Helper to temporarily patch the workspace rules dir constant for a subtest
	// without changing working directory (which locks TempDir on Windows).
	// Since workspaceRulesDir is a const, we simulate workspace rules by creating
	// them in the actual current working directory's .gline/rules path and clean up after.
	t.Run("no rules exist in cwd", func(t *testing.T) {
		_, err := LoadCustomRules()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Result depends on user's real home / cwd; just make sure no panic.
	})

	t.Run("workspace rules in cwd", func(t *testing.T) {
		// Create workspace rules in current directory
		rulesDir := filepath.Join(workspaceRulesDir)
		_ = os.MkdirAll(rulesDir, 0755)
		t.Cleanup(func() { os.RemoveAll(rulesDir) })

		f, err := os.CreateTemp(rulesDir, "test-*.md")
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("Use Go modules.")
		f.Close()

		got, err := LoadCustomRules()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !contains(got, "Use Go modules.") {
			t.Errorf("expected loaded rules to contain 'Use Go modules.', got %q", got)
		}
		if !contains(got, "# Workspace Rules") {
			t.Errorf("expected loaded rules to contain '# Workspace Rules', got %q", got)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
