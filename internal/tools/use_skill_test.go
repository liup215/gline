package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSkillRegistry implements the SkillRegistry interface for tests
type mockSkillRegistry struct {
	skills map[string]*types.Skill
}

func (m *mockSkillRegistry) Get(name string) (*types.Skill, bool) {
	s, ok := m.skills[name]
	return s, ok
}

func (m *mockSkillRegistry) GetMeta() []types.SkillMeta {
	var metas []types.SkillMeta
	for _, s := range m.skills {
		metas = append(metas, types.SkillMeta{
			Name:        s.Name,
			Description: s.Description,
		})
	}
	return metas
}

func TestUseSkillTool_NameAndDescription(t *testing.T) {
	reg := &mockSkillRegistry{skills: make(map[string]*types.Skill)}
	tool := NewUseSkillTool(reg)
	assert.Equal(t, "use_skill", tool.Name())
	assert.Contains(t, tool.Description(), "Load and activate")
}

func TestUseSkillTool_Execute_Success(t *testing.T) {
	reg := &mockSkillRegistry{
		skills: map[string]*types.Skill{
			"explain": {
				Name:        "explain",
				Description: "Explain code",
				Contents:    "Open with 'Let me explain...'",
			},
		},
	}
	tool := NewUseSkillTool(reg)

	input := json.RawMessage(`{"skill_name": "explain"}`)
	result, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	assert.Contains(t, result, `Skill "explain" is now active`)
	assert.Contains(t, result, "Open with 'Let me explain...'")
	assert.Contains(t, result, "Do NOT call use_skill again")
}

func TestUseSkillTool_Execute_NotFound(t *testing.T) {
	reg := &mockSkillRegistry{
		skills: map[string]*types.Skill{
			"explain": {Name: "explain", Description: "Explain code"},
			"debug":   {Name: "debug", Description: "Debug code"},
		},
	}
	tool := NewUseSkillTool(reg)

	input := json.RawMessage(`{"skill_name": "nonexistent"}`)
	_, err := tool.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "\"explain\"")
	assert.Contains(t, err.Error(), "\"debug\"")
}

func TestUseSkillTool_Execute_EmptyName(t *testing.T) {
	reg := &mockSkillRegistry{skills: make(map[string]*types.Skill)}
	tool := NewUseSkillTool(reg)

	input := json.RawMessage(`{"skill_name": ""}`)
	_, err := tool.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill_name is required")
}

func TestUseSkillTool_Execute_NoSkillsAvailable(t *testing.T) {
	reg := &mockSkillRegistry{skills: make(map[string]*types.Skill)}
	tool := NewUseSkillTool(reg)

	input := json.RawMessage(`{"skill_name": "foo"}`)
	_, err := tool.Execute(context.Background(), input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No skills are available")
}