// Package agent provides shared agent types for gline: the operating Mode,
// the Provider interface implemented by LLM backends, and the streaming
// types (StreamChunk, StreamCallback) consumed by the TUI/GUI bridges.
//
// The hand-written agent loop (BaseAgent) that used to live here was
// replaced by the ADK-backed agent (internal/adkagent) in Phase 7 of the
// agent-loop refactor. This package now only hosts the shared contracts.
package agent

// Mode represents the operating mode of the Agent
type Mode string

const (
	// ModePlan is for exploration and planning without making changes
	ModePlan Mode = "plan"
	// ModeAct is for executing tasks and modifying files
	ModeAct Mode = "act"
)
