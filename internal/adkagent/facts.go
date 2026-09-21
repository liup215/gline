// facts.go — background fact extraction after a completed run.
// Ported from the legacy loop's BaseAgent.extractFactsAsync
// (internal/agent/agent.go) onto the ADK agent: same memory.FactExtractor
// contract, but the LLM caller goes through the ADK model.LLM interface and
// the transcript is built from the persisted accumulator messages.
package adkagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

	"github.com/liup215/gline/internal/log"
	"github.com/liup215/gline/internal/memory"
	"github.com/liup215/gline/pkg/types"
)

// transcriptFromMessages renders user/assistant message text the way the
// legacy fact extractor expects ("User: …" / "Assistant: …" lines).
func transcriptFromMessages(msgs []types.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case types.RoleUser:
			b.WriteString("User: ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		case types.RoleAssistant:
			b.WriteString("Assistant: ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// extractFactsAsync runs fact extraction in the background after a
// completed conversation. It never blocks the caller and silently no-ops
// when the memory engine is absent or the transcript is too short.
func (a *Agent) extractFactsAsync(transcript string) {
	engine := a.opts.MemoryEngine
	if engine == nil || engine.FactStore == nil || a.llm == nil {
		return
	}
	if len(strings.Fields(transcript)) < 4 {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		extractor := memory.NewFactExtractor()
		extractor.Caller = func(ctx context.Context, systemPrompt, userContent string) (string, error) {
			return generateText(ctx, a.llm, systemPrompt, userContent)
		}

		changes, err := extractor.Extract(ctx, transcript)
		if err != nil || len(changes) == 0 {
			return
		}
		source := memory.ConversationRef{TaskID: a.taskID}.String()
		changes = memory.EnrichFacts(changes, source, "")
		if err := engine.FactStore.Apply(ctx, changes); err != nil {
			log.Warnf("fact extraction persist failed: %v", err)
		} else {
			log.Infof("fact extraction: %d facts applied (task=%s)", len(changes), a.taskID)
		}
	}()
}

// generateText runs a non-streaming, tool-free one-shot completion through
// the ADK model.LLM and concatenates the text parts of the response.
func generateText(ctx context.Context, llm model.LLM, systemPrompt, userContent string) (string, error) {
	temp := float32(0.0)
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText(userContent, "user")},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(systemPrompt, "user"),
			Temperature:       &temp,
			MaxOutputTokens:   2048,
		},
	}
	var out strings.Builder
	for resp, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		if resp == nil || resp.Content == nil {
			continue
		}
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				out.WriteString(part.Text)
			}
		}
	}
	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", fmt.Errorf("empty fact extraction response")
	}
	return text, nil
}
