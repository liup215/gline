package model

import (
	"encoding/json"
	"fmt"
)

// ErrorMeta contains structured data for error messages
type ErrorMeta struct {
	Code      int    `json:"code,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
	Stack     string `json:"stack,omitempty"`
}

// AsErrorMeta extracts ErrorMeta from message metadata
func (m Message) AsErrorMeta() (*ErrorMeta, error) {
	if len(m.Meta) == 0 {
		return nil, nil
	}
	var meta ErrorMeta
	if err := json.Unmarshal(m.Meta, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal error meta: %w", err)
	}
	return &meta, nil
}

// ToolMeta contains structured data for tool status messages
type ToolMeta struct {
	ToolName string `json:"tool_name"`
	Status   string `json:"status"` // "running", "completed", "failed"
	Duration int64  `json:"duration_ms,omitempty"`
}

// AsToolMeta extracts ToolMeta from message metadata
func (m Message) AsToolMeta() (*ToolMeta, error) {
	if len(m.Meta) == 0 {
		return nil, nil
	}
	var meta ToolMeta
	if err := json.Unmarshal(m.Meta, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool meta: %w", err)
	}
	return &meta, nil
}

// SetMeta sets the metadata for a message (helper for fluent API)
func (m *Message) SetMeta(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal meta: %w", err)
	}
	m.Meta = data
	return nil
}
