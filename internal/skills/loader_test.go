package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillMarkdown(t *testing.T) {
	t.Run("valid SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "my-skill")
		require.NoError(t, os.MkdirAll(skillDir, 0755))

		content := `---
name: my-skill
description: "A test skill"
---
# Instructions
You are a test assistant.
Do testing things.
`
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

		skills, err := LoadSkillsFromDir(dir)
		require.NoError(t, err)
		require.Len(t, skills, 1)
		assert.Equal(t, "my-skill", skills[0].Name)
		assert.Equal(t, "A test skill", skills[0].Description)
		assert.Contains(t, skills[0].Contents, "You are a test assistant.")
	})

	t.Run("name mismatch", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "wrong-name")
		require.NoError(t, os.MkdirAll(skillDir, 0755))

		content := `---
name: expected-name
description: "Name mismatch"
---
# Instructions
Test.
`
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

		_, err := LoadSkillsFromDir(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not match directory name")
	})

	t.Run("missing frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "bad-skill")
		require.NoError(t, os.MkdirAll(skillDir, 0755))

		content := `# Instructions
No frontmatter here.
`
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

		_, err := LoadSkillsFromDir(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "skill description is required")
	})

	t.Run("missing SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "empty-skill")
		require.NoError(t, os.MkdirAll(skillDir, 0755))
		// no SKILL.md file

		skills, err := LoadSkillsFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("missing description", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "no-desc")
		require.NoError(t, os.MkdirAll(skillDir, 0755))

		content := `---
name: no-desc
---
# Instructions
No description.
`
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

		_, err := LoadSkillsFromDir(dir)
		assert.Error(t, err)
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		skills, err := LoadSkillsFromDir(dir)
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		skills, err := LoadSkillsFromDir("/nonexistent/dir")
		require.NoError(t, err)
		assert.Empty(t, skills)
	})
}

func TestLoadSkillsFromDir(t *testing.T) {
	t.Run("multiple skills", func(t *testing.T) {
		dir := t.TempDir()

		// skill1: explain
		s1Dir := filepath.Join(dir, "explain")
		require.NoError(t, os.MkdirAll(s1Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte(`---
name: explain
description: Explain code
---
Explain code clearly.
`), 0644))

		// skill2: debug
		s2Dir := filepath.Join(dir, "debug")
		require.NoError(t, os.MkdirAll(s2Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s2Dir, "SKILL.md"), []byte(`---
name: debug
description: Debug code
---
Debug code step by step.
`), 0644))

		skills, err := LoadSkillsFromDir(dir)
		require.NoError(t, err)
		require.Len(t, skills, 2)

		names := make([]string, 2)
		for i, s := range skills {
			names[i] = s.Name
		}
		assert.Contains(t, names, "explain")
		assert.Contains(t, names, "debug")
	})

	t.Run("override priority via registry", func(t *testing.T) {
		dir1 := t.TempDir()
		dir2 := t.TempDir()

		// dir1: older version
		s1Dir := filepath.Join(dir1, "explain")
		require.NoError(t, os.MkdirAll(s1Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte(`---
name: explain
description: Old description
---
Old instructions.
`), 0644))

		// dir2: newer version
		s2Dir := filepath.Join(dir2, "explain")
		require.NoError(t, os.MkdirAll(s2Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s2Dir, "SKILL.md"), []byte(`---
name: explain
description: New description
---
New instructions.
`), 0644))

		reg := NewRegistry()
		require.NoError(t, reg.LoadFromDirs(dir1))
		s1, _ := reg.Get("explain")
		assert.Equal(t, "Old description", s1.Description)

		require.NoError(t, reg.LoadFromDirs(dir2))
		s2, _ := reg.Get("explain")
		assert.Equal(t, "New description", s2.Description)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		skills, err := LoadSkillsFromDir("/nonexistent/dir")
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("skip invalid skill", func(t *testing.T) {
		dir := t.TempDir()

		// valid skill
		s1Dir := filepath.Join(dir, "valid")
		require.NoError(t, os.MkdirAll(s1Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte(`---
name: valid
description: ok
---
Ok.
`), 0644))

		// invalid name mismatch
		s2Dir := filepath.Join(dir, "bad")
		require.NoError(t, os.MkdirAll(s2Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s2Dir, "SKILL.md"), []byte(`---
name: wrong-name
description: bad
---
Bad.
`), 0644))

		skills, err := LoadSkillsFromDir(dir)
		// Should return the valid skill but also report an error
		assert.NotNil(t, skills)
		assert.Len(t, skills, 1)
		assert.Error(t, err)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid skill", func(t *testing.T) {
		assert.NoError(t, Validate(&types.Skill{
			Name: "test", Description: "d", Contents: "content",
		}))
	})

	t.Run("nil skill", func(t *testing.T) {
		assert.Error(t, Validate(nil))
	})

	t.Run("missing name", func(t *testing.T) {
		assert.Error(t, Validate(&types.Skill{Description: "d", Contents: "c"}))
	})

	t.Run("missing description", func(t *testing.T) {
		assert.Error(t, Validate(&types.Skill{Name: "n", Contents: "c"}))
	})
}
