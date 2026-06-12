// fact_extractor.go implements mem0-style single-pass fact extraction via LLM.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FactExtractor uses an LLM (or fallback rules) to extract semantic facts.
type FactExtractor struct {
	MinConfidence float64
	MaxFacts      int
	// Caller is invoked when set; otherwise ruleBasedExtract is used as fallback.
	// Signature: func(ctx, systemPrompt, userContent string) (response string, err error)
	Caller func(ctx context.Context, systemPrompt, userContent string) (string, error)
}

// NewFactExtractor creates an extractor with defaults.
func NewFactExtractor() *FactExtractor {
	return &FactExtractor{
		MinConfidence: 0.7,
		MaxFacts:      20,
	}
}

// ExtractPrompt is the system prompt sent to the LLM for fact extraction.
const ExtractPrompt = `You are a knowledge extraction engine for an AI coding assistant.

Given a conversation between a User and an AI assistant, extract atomic semantic facts about the user, their project, and their technical preferences.

For each meaningful fact output a JSON object with exactly these fields:
- "category": one of ["entity", "preference", "decision", "pattern", "task", "relation"]
- "subject": the entity or topic (e.g. "User", "Project", "Go")
- "predicate": relationship verb (e.g. "prefers", "uses", "decided", "avoids")
- "object": the value or target (keep concise, under 100 characters)
- "confidence": 0.0-1.0 float (be conservative; use <0.7 for uncertain facts)
- "action": one of ["ADD", "DELETE", "NOOP"]

RULES:
1. ADD for genuinely NEW information not already implied by the conversation.
2. DELETE only when the user explicitly contradicts or revokes a prior statement.
3. NOOP for obvious, generic, or already-known facts (e.g. "Go is a programming language").
4. Be SPECIFIC and CONCRETE. Prefer: "User prefers double quotes for strings" over "User has coding style preferences".
5. Extract categories:
   - preference: coding style, tools, languages, UI patterns
   - decision: architecture choices, library selections, deployment strategy
   - pattern: recurring bugs, habits, anti-patterns the user falls into
   - entity: important people, projects, APIs, services mentioned
6. Do NOT hallucinate facts not grounded in the conversation.
7. If the conversation contains no extractable facts, return an empty array [].

OUTPUT: ONLY a JSON array. No markdown fences, no explanation.`

// Extract turns a conversation into a list of FactChange objects.
// This is designed to be called once per turn in a background goroutine.
// If Caller is set, it uses the LLM for extraction; otherwise falls back to
// a lightweight rule-based heuristic.
func (e *FactExtractor) Extract(ctx context.Context, conversationText string) ([]FactChange, error) {
	if e.Caller == nil {
		return e.ruleBasedExtract(conversationText)
	}
	resp, err := e.Caller(ctx, ExtractPrompt, conversationText)
	if err != nil {
		return nil, fmt.Errorf("llm extraction call failed: %w", err)
	}
	changes, err := e.ParseFactChanges(resp)
	if err != nil {
		return nil, fmt.Errorf("parse fact changes: %w", err)
	}
	if len(changes) == 0 {
		return nil, nil
	}
	if e.MaxFacts > 0 && len(changes) > e.MaxFacts {
		changes = changes[:e.MaxFacts]
	}
	return changes, nil
}

// ruleBasedExtract is a lightweight fallback that doesn't require LLM.
// It extracts preference and decision patterns using regex heuristics.
func (e *FactExtractor) ruleBasedExtract(text string) ([]FactChange, error) {
	var changes []FactChange
	lower := strings.ToLower(text)

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
				UpdatedAt:  time.Now().UTC(),
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
				UpdatedAt:  time.Now().UTC(),
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
		Subject    string  `json:"subject"`
		Predicate  string  `json:"predicate"`
		Object     string  `json:"object"`
		Confidence float64 `json:"confidence"`
		Action     string  `json:"action"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &rawFacts); err != nil {
		return nil, fmt.Errorf("parse fact JSON: %w", err)
	}

	var changes []FactChange
	now := time.Now().UTC()
	for _, rf := range rawFacts {
		if rf.Action == "" || rf.Action == "NOOP" {
			continue
		}
		if rf.Confidence < e.MinConfidence {
			continue
		}
		// Normalize action: only ADD and DELETE are meaningful here;
		// UPDATE is handled by the store's Apply() smart-merge logic.
		action := rf.Action
		if action == "UPDATE" {
			action = "ADD"
		}
		changes = append(changes, FactChange{
			Action: action,
			Fact: Fact{
				ID:         genID(),
				Category:   FactCategory(rf.Category),
				Subject:    rf.Subject,
				Predicate:  rf.Predicate,
				Object:     rf.Object,
				Confidence: rf.Confidence,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			Reason: "LLM extraction",
		})
	}
	return changes, nil
}

// EnrichFacts sets default fields for facts that the LLM cannot know.
func EnrichFacts(changes []FactChange, source string, kbID string) []FactChange {
	now := time.Now().UTC()
	for i := range changes {
		if changes[i].Fact.Source == "" {
			changes[i].Fact.Source = source
		}
		if changes[i].Fact.KBID == "" && kbID != "" {
			changes[i].Fact.KBID = kbID
		}
		if changes[i].Fact.CreatedAt.IsZero() {
			changes[i].Fact.CreatedAt = now
		}
		if changes[i].Fact.UpdatedAt.IsZero() {
			changes[i].Fact.UpdatedAt = now
		}
		if changes[i].Fact.LastAccess.IsZero() {
			changes[i].Fact.LastAccess = now
		}
	}
	return changes
}
