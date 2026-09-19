package tool

import (
	"regexp"

	"github.com/liup215/gline/pkg/types"
)

// Registry manages tool renderers
type Registry struct {
	renderers      map[types.ToolName]Renderer
	defaultFactory func(name types.ToolName) Renderer
	descriptions   map[types.ToolName]string
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	r := &Registry{
		renderers:    make(map[types.ToolName]Renderer),
		descriptions: make(map[types.ToolName]string),
		defaultFactory: func(name types.ToolName) Renderer {
			desc := GetDefaultDescription(name)
			return NewDefaultRenderer(name, desc)
		},
	}
	r.initDescriptions()
	return r
}

// Register adds a renderer to the registry
func (r *Registry) Register(renderer Renderer) {
	r.renderers[renderer.Name()] = renderer
}

// Get returns the renderer for a tool name.
// If no specific renderer is registered, returns a default renderer.
func (r *Registry) Get(name types.ToolName) Renderer {
	if renderer, ok := r.renderers[name]; ok {
		return renderer
	}
	return r.defaultFactory(name)
}

// Has returns true if a specific renderer is registered for this tool
func (r *Registry) Has(name types.ToolName) bool {
	_, ok := r.renderers[name]
	return ok
}

// initDescriptions sets up default descriptions for known tools
func (r *Registry) initDescriptions() {
	r.descriptions[types.ToolReadFile] = "read this file"
	r.descriptions[types.ToolWriteToFile] = "created a new file"
	r.descriptions[types.ToolReplaceInFile] = "edited this file"
	r.descriptions[types.ToolExecuteCommand] = "executed this command"
	r.descriptions[types.ToolSearchFiles] = "searched files"
	r.descriptions[types.ToolAttemptCompletion] = "completed the task"
	r.descriptions[types.ToolAskFollowupQuestion] = "asked a question"
	r.descriptions[types.ToolPlanModeRespond] = "provided a plan response"
	r.descriptions[types.ToolUseMcpTool] = "used an MCP tool"
	r.descriptions[types.ToolAccessMcpResource] = "accessed an MCP resource"
}

// GetDescription returns the description for a tool name
func (r *Registry) GetDescription(name types.ToolName) string {
	if desc, ok := r.descriptions[name]; ok {
		return desc
	}
	return "used a tool"
}

// GetDefaultDescription returns the default description for a tool name
func GetDefaultDescription(name types.ToolName) string {
	switch name {
	case types.ToolReadFile:
		return "read this file"
	case types.ToolWriteToFile:
		return "created a new file"
	case types.ToolReplaceInFile:
		return "edited this file"
	case types.ToolExecuteCommand:
		return "executed this command"
	case types.ToolSearchFiles:
		return "searched files"
	case types.ToolAttemptCompletion:
		return "completed the task"
	case types.ToolAskFollowupQuestion:
		return "asked a question"
	case types.ToolPlanModeRespond:
		return "provided a plan response"
	case types.ToolUseMcpTool:
		return "used an MCP tool"
	case types.ToolAccessMcpResource:
		return "accessed an MCP resource"
	default:
		return "used a tool"
	}
}

// NormalizeToolName converts camelCase to snake_case
func NormalizeToolName(name string) types.ToolName {
	return types.ToolName(camelToSnakeRe.ReplaceAllString(name, "${1}_${2}"))
}

var camelToSnakeRe = regexp.MustCompile("([a-z])([A-Z])")

// DefaultRegistry is the global default registry with all built-in tools
var DefaultRegistry = NewDefaultRegistry()

// NewDefaultRegistry creates a registry with all built-in tools registered
func NewDefaultRegistry() *Registry {
	reg := NewRegistry()

	// Register special tools with custom renderers
	reg.Register(&AttemptCompletionRenderer{})
	reg.Register(&AskFollowupQuestionRenderer{})
	reg.Register(&PlanModeRespondRenderer{})

	// Register standard tools
	reg.Register(&ReadFileRenderer{})
	reg.Register(NewDefaultRenderer(types.ToolWriteToFile, "created a new file"))
	reg.Register(NewDefaultRenderer(types.ToolReplaceInFile, "edited this file"))
	reg.Register(NewDefaultRenderer(types.ToolExecuteCommand, "executed this command"))
	reg.Register(NewDefaultRenderer(types.ToolSearchFiles, "searched files"))
	reg.Register(NewDefaultRenderer(types.ToolUseMcpTool, "used an MCP tool"))
	reg.Register(NewDefaultRenderer(types.ToolAccessMcpResource, "accessed an MCP resource"))

	return reg
}
