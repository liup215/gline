package types

// ToolPhase represents the phase of a tool execution
type ToolPhase int

const (
	ToolPhaseStart    ToolPhase = 0
	ToolPhaseComplete ToolPhase = 1
)

func (p ToolPhase) String() string {
	switch p {
	case ToolPhaseStart:
		return "start"
	case ToolPhaseComplete:
		return "complete"
	default:
		return "unknown"
	}
}
