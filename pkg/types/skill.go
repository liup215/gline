// Package types defines skill types shared across the codebase.
package types

// Skill describes an externally-defined capability that modifies the
// system prompt when activated.  Skills are discovered from well-known
// directories (e.g. ~/.gline/skills, ~/.agents/skills) and exposed as
// slash commands (/<name>).
type Skill struct {
	// Name is the command identifier without the leading slash.
	// Must match the YAML/JSON file stem in most cases.
	Name string `yaml:"name" json:"name"`

	// Description is shown in the slash-command menu and help text.
	Description string `yaml:"description" json:"description"`

	// Prompt is injected into the system prompt when this skill is active.
	// It may contain multi-line instructions, examples, or role definitions.
	Prompt string `yaml:"prompt" json:"prompt"`

	// Tools optionally restricts the tool set available while this skill
	// is active.  When empty all registered tools are available.
	Tools []string `yaml:"tools,omitempty" json:"tools,omitempty"`

	// Tags is an optional list of category labels for grouping skills.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Author is optional attribution.
	Author string `yaml:"author,omitempty" json:"author,omitempty"`

	// Version is an optional version string (semver recommended).
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Source records the absolute path of the file this skill was loaded from.
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

// SkillInfo is a lightweight summary of a skill for UI lists.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}
