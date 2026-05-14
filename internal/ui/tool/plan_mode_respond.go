package tool

import (
	"github.com/liup215/gline/pkg/types"
)

// PlanModeRespondRenderer renders plan_mode_respond tool output
type PlanModeRespondRenderer struct{}

func (r *PlanModeRespondRenderer) Render(req RenderRequest) RenderResult {
	if req.Phase == types.ToolPhaseStart {
		// Skip in start phase, will handle in complete
		return RenderResult{Skip: true}
	}

	// ToolPhaseComplete: render the result as assistant message with markdown
	if req.Input != "" {
		return RenderResult{
			Content:  req.Input,
			Role:     types.RoleAssistant,
			Strategy: types.StrategyMarkdown,
			Skip:     false,
		}
	}
	return RenderResult{Skip: true}
}

func (r *PlanModeRespondRenderer) Name() types.ToolName {
	return types.ToolPlanModeRespond
}

func (r *PlanModeRespondRenderer) Description() string {
	return "provided a plan response"
}

func (r *PlanModeRespondRenderer) Icon() string {
	return "📋"
}
