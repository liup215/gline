package skills

import (
	"testing"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	orig := &types.Skill{Name: "explain", Description: "d", Prompt: "p"}
	require.NoError(t, reg.Register(orig))

	s, ok := reg.Get("explain")
	require.True(t, ok)
	assert.Equal(t, "explain", s.Name)

	_, ok = reg.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_CaseInsensitive(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "Explain", Description: "d", Prompt: "p"}))

	s, ok := reg.Get("EXPLAIN")
	assert.True(t, ok)
	assert.Equal(t, "explain", s.Name)
}

func TestRegistry_Overwrite(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "x", Description: "first", Prompt: "p"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "x", Description: "second", Prompt: "p"}))

	s, _ := reg.Get("x")
	assert.Equal(t, "second", s.Description)
}

func TestRegistry_GetAll(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "c", Description: "d", Prompt: "p"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "a", Description: "d", Prompt: "p"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "b", Description: "d", Prompt: "p"}))

	all := reg.GetAll()
	require.Len(t, all, 3)
	assert.Equal(t, "a", all[0].Name)
	assert.Equal(t, "b", all[1].Name)
	assert.Equal(t, "c", all[2].Name)
}

func TestRegistry_Activate(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "explain", Description: "d", Prompt: "p"}))

	_, ok := reg.GetActive()
	assert.False(t, ok)

	s, err := reg.Activate("explain")
	require.NoError(t, err)
	assert.Equal(t, "explain", s.Name)
	assert.True(t, reg.IsActive("explain"))
	assert.False(t, reg.IsActive("debug"))

	active, ok := reg.GetActive()
	require.True(t, ok)
	assert.Equal(t, "explain", active.Name)

	reg.Deactivate()
	_, ok = reg.GetActive()
	assert.False(t, ok)

	_, err = reg.Activate("nonexistent")
	assert.Error(t, err)
}

func TestRegistry_GetAllInfo(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "explain", Description: "d", Prompt: "p"}))
	require.NoError(t, reg.Register(&types.Skill{Name: "debug", Description: "d2", Prompt: "p"}))
	reg.Activate("debug")

	infos := reg.GetAllInfo()
	require.Len(t, infos, 2)

	for _, info := range infos {
		if info.Name == "debug" {
			assert.True(t, info.Active)
		} else {
			assert.False(t, info.Active)
		}
	}
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.Register(&types.Skill{Name: "tmp", Description: "d", Prompt: "p"}))
	reg.Activate("tmp")

	reg.Unregister("tmp")
	_, ok := reg.Get("tmp")
	assert.False(t, ok)
	_, ok = reg.GetActive()
	assert.False(t, ok)
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()
	assert.Equal(t, 0, reg.Count())
	reg.Register(&types.Skill{Name: "a", Description: "d", Prompt: "p"})
	assert.Equal(t, 1, reg.Count())
}
