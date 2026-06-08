package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	orig := &types.Skill{Name: "explain", Description: "d", Contents: "instructions"}
	require.NoError(t, reg.Register(orig))

	s, ok := reg.Get("explain")
	require.True(t, ok)
	assert.Equal(t, "explain", s.Name)

	_, ok = reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_CaseInsensitive(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "Explain", Description: "d", Contents: "instructions"}))

	s, ok := reg.Get("EXPLAIN")
	assert.True(t, ok)
	assert.Equal(t, "explain", s.Name)
}

func TestRegistry_Overwrite(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "x", Description: "first", Contents: "old"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "x", Description: "second", Contents: "new"}))

	s, _ := reg.Get("x")
	assert.Equal(t, "second", s.Description)
}

func TestRegistry_GetAll(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "c", Description: "d", Contents: "i"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "a", Description: "d", Contents: "i"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "b", Description: "d", Contents: "i"}))

	all := reg.GetAll()
	require.Len(t, all, 3)
	assert.Equal(t, "a", all[0].Name)
	assert.Equal(t, "b", all[1].Name)
	assert.Equal(t, "c", all[2].Name)
}

func TestRegistry_GetMeta(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "explain", Description: "Explain skill", Contents: "explain things"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "debug", Description: "Debug skill", Contents: "debug things"}))

	metas := reg.GetMeta()
	require.Len(t, metas, 2)

	names := make([]string, 2)
	for i, m := range metas {
		names[i] = m.Name
	}
	assert.Contains(t, names, "explain")
	assert.Contains(t, names, "debug")

	for _, m := range metas {
		assert.NotEmpty(t, m.Description)
	}
}

func TestRegistry_GetInstructions(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "explain", Description: "d", Contents: "full markdown here"}))

	instructions, err := reg.GetInstructions("explain")
	require.NoError(t, err)
	assert.Equal(t, "full markdown here", instructions)

	_, err = reg.GetInstructions("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "tmp", Description: "d", Contents: "i"}))

	reg.Unregister("tmp")
	_, ok := reg.Get("tmp")
	assert.False(t, ok)
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()
	assert.Equal(t, 0, reg.Count())
	reg.Register(&types.Skill{Name: "a", Description: "d", Contents: "i"})
	assert.Equal(t, 1, reg.Count())
}

func TestRegistry_LoadFromDirs(t *testing.T) {
	t.Run("load from temp directories", func(t *testing.T) {
		dir := t.TempDir()
		s1Dir := filepath.Join(dir, "maven")
		require.NoError(t, os.MkdirAll(s1Dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(s1Dir, "SKILL.md"), []byte(`---
name: maven
description: Maven helper
---
# Maven
Maven instructions.
`), 0644))

		reg := NewRegistry()
		require.NoError(t, reg.LoadFromDirs(dir))
		assert.Equal(t, 1, reg.Count())

		s, ok := reg.Get("maven")
		require.True(t, ok)
		assert.Equal(t, "Maven helper", s.Description)
		assert.Contains(t, s.Contents, "Maven instructions")
	})

	t.Run("missing directory is skipped", func(t *testing.T) {
		reg := NewRegistry()
		require.NoError(t, reg.LoadFromDirs("/nonexistent/path"))
		assert.Equal(t, 0, reg.Count())
	})
}
