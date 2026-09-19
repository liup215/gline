package tool

import (
	"encoding/json"

	"github.com/liup215/gline/pkg/types"
)

// AskFollowupQuestionRenderer renders ask_followup_question tool output
type AskFollowupQuestionRenderer struct{}

func (r *AskFollowupQuestionRenderer) Render(req RenderRequest) RenderResult {
	// This tool is handled specially via AskQuestionEvent
	// Skip creating messages here
	return RenderResult{Skip: true}
}

func (r *AskFollowupQuestionRenderer) Name() types.ToolName {
	return types.ToolAskFollowupQuestion
}

func (r *AskFollowupQuestionRenderer) Description() string {
	return "asked a question"
}

func (r *AskFollowupQuestionRenderer) Icon() string {
	return "❓"
}

// ExtractQuestion extracts the question from tool input
func (r *AskFollowupQuestionRenderer) ExtractQuestion(input string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		return input
	}
	if q, ok := m["question"].(string); ok {
		return q
	}
	return input
}

// ExtractOptions extracts the options from tool input
func (r *AskFollowupQuestionRenderer) ExtractOptions(input string) []string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		return nil
	}
	if opts, ok := m["options"].([]interface{}); ok {
		result := make([]string, len(opts))
		for i, opt := range opts {
			if s, ok := opt.(string); ok {
				result[i] = s
			}
		}
		return result
	}
	return nil
}
