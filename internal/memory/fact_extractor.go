// fact_extractor.go implements mem0-style single-pass fact extraction via LLM.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FactExtractor uses an LLM to extract, compare, and categorise semantic facts.
type FactExtractor struct {
	MinConfidence float64
	MaxFacts      int
}

// NewFactExtractor creates an extractor with defaults.
func NewFactExtractor() *FactExtractor {
	return &FactExtractor{
		MinConfidence: 0.7,
		MaxFacts:      20,
	}
}

// ExtractPrompt is the system prompt sent to the LLM for fact extraction.
const ExtractPrompt = `You are a knowledge extraction engine.

Given a conversation between a User and an AI assistant, extract atomic semantic facts.

For each fact output a JSON object with fields:
- "category": one of ["entity", "preference", "decision", "pattern", "task", "relation"]
- "subject": the entity or topic (e.g. "User", "Project", "Go")
- "predicate": relationship verb (e.g. "prefers", "uses", "decided")
- "object": the value or target
- "confidence": 0.0-1.0 float
- "action": one of ["ADD", "UPDATE", "DELETE", "NOOP"]

Rules:
- Prefer ADD for genuinely new information.
- Use UPDATE only when you are sure a fact overrides a prior one.
- Use NOOP for obvious, generic, or already-known facts.
- Only include high-confidence, concrete, specific facts.
- Avoid hallucinating facts not grounded in the conversation.

Output ONLY a JSON array. No markdown, no explanation outside the array.`

// Extract turns a conversation into a list of FactChange objects.
// This is designed to be called once per turn in a background goroutine.
func (e *FactExtractor) Extract(ctx context.Context, conversationText string) ([]FactChange, error) {
	// Phase 4: the actual LLM call will be wired up through the agent's Provider interface.
	// For now this is a stub that does a lightweight rule-based extraction
	// so the layer works end-to-end before LLM integration.
	return e.ruleBasedExtract(conversationText)
}

// ruleBasedExtract is a lightweight fallback that doesn't require LLM.
// It extracts preference and decision patterns using regex heuristics.
func (e *FactExtractor) ruleBasedExtract(text string) ([]FactChange, error) {
	var changes []FactChange
	lower := strings.ToLower(text)

	// Simple pattern: "I prefer X" or "we decided Y"
	if strings.Contains(lower, "prefer") || strings.Contains(lower, "喜欢") {
		changes = append(changes, FactChange{
			Action: "ADD",
			Fact: Fact{
				ID:         genID(),
				Category:   FactPreference,
				Subject:    "User",
				Predicate:  "expressed preference",
				Object:     strings.TrimSpace(text),
				Confidence: 0.6,
				CreatedAt:  time.Now().UTC(),
			},
			Reason: "rule-based extraction: preference pattern detected",
		})
	}
	if strings.Contains(lower, "decided") || strings.Contains(lower, "决定") {
		changes = append(changes, FactChange{
			Action: "ADD",
			Fact: Fact{
				ID:         genID(),
				Category:   FactDecision,
				Subject:    "User",
				Predicate:  "made decision",
				Object:     strings.TrimSpace(text),
				Confidence: 0.7,
				CreatedAt:  time.Now().UTC(),
			},
			Reason: "rule-based extraction: decision pattern detected",
		})
	}
	return changes, nil
}

// ParseFactChanges parses the LLM JSON response into structured changes.
func (e *FactExtractor) ParseFactChanges(rawJSON string) ([]FactChange, error) {
	// Strip markdown fences if present
	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var rawFacts []struct {
		Category   string  `json:"category"`
		Subject   string  `json:"subject"`
		Predicate string  `json:"predicate"`
		Object    string  `json:"object"`
		Confidence float64 `json:"confidence"`
		Action    string  `json:"action"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &rawFacts); err != nil {
		return nil, fmt.Errorf("parse fact JSON: %w", err)
	}

	var changes []FactChange
	for _, rf := range rawFacts {
		if rf.Action == "" || rf.Action == "NOOP" {
			continue
		}
		if rf.Confidence < e.MinConfidence {
			continue
		}
		changes = append(changes, FactChange{
			Action: rf.Action,
			Fact: Fact{
				ID:         genID(),
				Category:   FactCategory(rf.Category),
				Subject:    rf.Subject,
				Predicate: rf.Predicate,
				Object:     rf.Object,
				Confidence: rf.Confidence,
				CreatedAt:  time.Now().UTC(),
			},
			Reason: "LLM extraction",
		})
	}
	return changes, nil
}

// ExtractAsync runs extraction in the background after a conversation ends.
func (e *FactExtractor) ExtractAsync(store FactStore, conversationText string, source ConversationRef) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		changes, err := e.Extract(ctx, conversationText)
		if err != nil || len(changes) == 0 {
			return
		}
		// FactStore.Add doesn't need conversationText here since we already extracted
		// Instead, we'll directly upsert each change via the concrete store
		// (Phase 4 will refine this with full LLM integration)
		_ = changes
	}()
}
