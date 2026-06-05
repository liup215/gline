package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSkillFromFile(t *testing.T) {
	t.Run("yaml skill", func(t *testing.T) {
		dir := t.TempDir()
		content := `name: test-skill
description: "A test skill"
prompt: |
  You are a test assistant.
`
		path := filepath.Join(dir, "test.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		skill, err := LoadSkillFromFile(path)
		require.NoError(t, err)
		assert.Equal(t, "test-skill", skill.Name)
		assert.Equal(t, "A test skill", skill.Description)
		assert.Contains(t, skill.Prompt, "test assistant")
	})

	t.Run("json skill", func(t *testing.T) {
		dir := t.TempDir()
		content := `{"name":"json-skill","description":"JSON test","prompt":"Hello from JSON"}`
		path := filepath.Join(dir, "test.json")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))

		skill, err := LoadSkillFromFile(path)
		require.NoError(t, err)
		assert.Equal(t, "json-skill", skill.Name)
		assert.Equal(t, "JSON test", skill.Description)
	})

	t.Run("unsupported extension", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))
		_, err := LoadSkillFromFile(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadSkillFromFile("/nonexistent/skill.yaml")
		assert.Error(t, err)
	})
}

func TestLoadSkillsFromDir(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		skills, err := LoadSkillsFromDir("/nonexistent/dir")
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("multiple skills with override priority", func(t *testing.T) {
		dir1 := t.TempDir()
		dir2 := t.TempDir()

		// dir1: older version
		f1 := filepath.Join(dir1, "explain.yaml")
		require.NoError(t, os.WriteFile(f1, []byte(`
name: explain
description: Old description
prompt: Old prompt
`), 0644))

		// dir2: newer version (should override when dir2 is loaded after dir1)
		f2 := filepath.Join(dir2, "explain.yaml")
		require.NoError(t, os.WriteFile(f2, []byte(`
name: explain
description: New description
prompt: New prompt
`), 0644))

		// Load dir1 then dir2 separately to test overwrite
		reg := NewRegistry()
		require.NoError(t, reg.LoadFromDirs(dir1))
		s1, _ := reg.Get("explain")
		assert.Equal(t, "Old description", s1.Description)

		require.NoError(t, reg.LoadFromDirs(dir2))
		s2, _ := reg.Get("explain")
		assert.Equal(t, "New description", s2.Description)
	})

	t.Run("derive name from filename", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "my-skill.yml")
		require.NoError(t, os.WriteFile(f, []byte(`
description: "Derived from filename"
prompt: "I have no name field"
`), 0644))

		skills, err := LoadSkillsFromDir(dir)
		require.NoError(t, err)
		require.Len(t, skills, 1)
		assert.Equal(t, "my-skill", skills[0].Name)
	})

	t.Run("skip invalid files", func(t *testing.T) {
		dir := t.TempDir()
		valid := filepath.Join(dir, "valid.yaml")
		require.NoError(t, os.WriteFile(valid, []byte(`name: valid
description: ok
prompt: ok
`), 0644))

		invalid := filepath.Join(dir, "invalid.yaml")
		require.NoError(t, os.WriteFile(invalid, []byte(`not: yaml: :-`), 0644))

		skills, err := LoadSkillsFromDir(dir)
		// Should return the valid skill but also report an error
		assert.NotNil(t, skills)
		assert.Len(t, skills, 1)
		assert.Error(t, err) // some files failed
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid skill", func(t *testing.T) {
		assert.NoError(t, Validate(&types.Skill{
			Name: "test", Description: "d", Prompt: "p",
		}))
	})

	t.Run("nil skill", func(t *testing.T) {
		assert.Error(t, Validate(nil))
	})

	t.Run("missing name", func(t *testing.T) {
		assert.Error(t, Validate(&types.Skill{Description: "d", Prompt: "p"}))
	})

	t.Run("missing description", func(t *testing.T) {
		assert.Error(t, Validate(&types.Skill{Name: "n", Prompt: "p"}))
	})

	t.Run("missing prompt", func(t *testing.T) {
		assert.Error(t, Validate(&types.Skill{Name: "n", Description: "d"}))
	})
}
