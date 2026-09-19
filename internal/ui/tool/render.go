// Package tool provides tool rendering interface and implementations.
package tool

import "github.com/liup215/gline/pkg/types"

// RenderRequest represents a request to render a tool output
type RenderRequest struct {
	Phase  types.ToolPhase
	Input  string
	Status string // "running" | "completed" | "failed"
}

// RenderResult represents the result of rendering a tool output
type RenderResult struct {
	Content  string
	Role     types.Role
	Strategy types.RenderStrategy
	Skip     bool
}

// Renderer is the interface that all tool renderers must implement
type Renderer interface {
	// Render returns the rendered content for the given request
	Render(req RenderRequest) RenderResult

	// Name returns the standard name of the tool
	Name() types.ToolName

	// Description returns a human-friendly description of the tool
	Description() string

	// Icon returns the icon for the tool
	Icon() string
}
