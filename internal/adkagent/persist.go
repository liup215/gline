package adkagent

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/v2/session"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/pkg/types"
)

// transcriptAccumulator maps the run's final (non-partial) events onto gline
// storage messages so `gline history` and the TUI history view keep working
// exactly as they did with the legacy loop. Streaming deltas are ignored —
// the aggregate event ADK emits at turn completion is the durable record.
//
// Mapping:
//   - model text + FunctionCall parts -> one assistant message (Content +
//     ToolCalls), matching how the legacy loop stores assistant turns
//   - thinking text -> ReasoningContent on the current assistant message
//   - FunctionResponse -> a role=tool message with ToolCallID
//   - user text -> a role=user message (the prompt echo the runner yields)
type transcriptAccumulator struct {
	mu  sync.Mutex
	msg []types.Message
	as  *types.Message // assistant message being accumulated
}

func (t *transcriptAccumulator) flushAssistant() {
	if t.as != nil && (t.as.Content != "" || len(t.as.ToolCalls) > 0 || t.as.ReasoningContent != "") {
		t.msg = append(t.msg, *t.as)
	}
	t.as = nil
}

// addUserPrompt records the turn's user message. The ADK runner persists the
// user content into the session but does not yield it as an event, so the
// transcript source for storage is the prompt itself.
func (t *transcriptAccumulator) addUserPrompt(prompt string) {
	if strings.TrimSpace(prompt) == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushAssistant()
	t.msg = append(t.msg, types.Message{Role: types.RoleUser, Content: prompt, Timestamp: time.Now()})
}

func (t *transcriptAccumulator) addEvent(ev *session.Event) {
	if ev == nil || ev.Content == nil || ev.Partial {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	switch {
	case ev.Content.Role == "thinking":
		for _, p := range ev.Content.Parts {
			if p.Text == "" {
				continue
			}
			if t.as == nil {
				t.as = &types.Message{Role: types.RoleAssistant, Timestamp: ev.Timestamp}
			}
			t.as.ReasoningContent += p.Text
		}
	case ev.Content.Role == "model" || ev.Content.Role == "assistant":
		for _, p := range ev.Content.Parts {
			if p.Text != "" {
				if t.as == nil {
					t.as = &types.Message{Role: types.RoleAssistant, Timestamp: ev.Timestamp}
				}
				t.as.Content += p.Text
			}
			if p.FunctionCall != nil {
				if t.as == nil {
					t.as = &types.Message{Role: types.RoleAssistant, Timestamp: ev.Timestamp}
				}
				t.as.ToolCalls = append(t.as.ToolCalls, types.ToolCall{
					ID:    p.FunctionCall.ID,
					Name:  p.FunctionCall.Name,
					Input: json.RawMessage(toolCallJSON(p.FunctionCall.Args)),
				})
			}
		}
	default: // user-role events: prompt echoes and tool results
		for _, p := range ev.Content.Parts {
			switch {
			case p.FunctionResponse != nil:
				t.flushAssistant()
				t.msg = append(t.msg, types.Message{
					Role:       types.RoleTool,
					Content:    functionResponseText(p.FunctionResponse.Response),
					ToolCallID: p.FunctionResponse.ID,
					Timestamp:  ev.Timestamp,
				})
			case p.Text != "":
				t.flushAssistant()
				t.msg = append(t.msg, types.Message{Role: types.RoleUser, Content: p.Text, Timestamp: ev.Timestamp})
			}
		}
	}
}

// messages returns the collected transcript in storage shape.
func (t *transcriptAccumulator) messages() []types.Message {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushAssistant()
	return t.msg
}

// finishTask records the terminal status of the current task (best effort).
func (a *Agent) finishTask(status string) {
	a.mu.Lock()
	id := a.taskID
	a.mu.Unlock()
	if a.opts.Store == nil || id == "" {
		return
	}
	if err := a.opts.Store.UpdateTaskStatus(id, status); err != nil {
		log.Warnf("failed to update task status: %v", err)
	}
}

// persistTranscript writes the run's messages into the task index. Storage
// failures are logged, never fatal: the ADK session log remains the source
// of truth even if the index write fails.
func (a *Agent) persistTranscript(acc *transcriptAccumulator) {
	if a.opts.Store == nil || a.taskID == "" {
		return
	}
	for _, m := range acc.messages() {
		if strings.TrimSpace(m.Content) == "" && len(m.ToolCalls) == 0 && strings.TrimSpace(m.ReasoningContent) == "" {
			continue
		}
		if err := a.opts.Store.SaveMessage(a.taskID, m); err != nil {
			log.Warnf("failed to save message to task index: %v", err)
		}
	}
}
