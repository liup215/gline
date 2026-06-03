// Package storage provides persistent storage for conversation state,
// task history, and tool call records using SQLite.
package storage

import (
	"encoding/json"
	"time"

	"github.com/liup215/gline/pkg/types"
)

// Store defines the contract for persisting conversation state.
type Store interface {
	// === Task lifecycle ===

	// CreateTask creates a new task record and returns its UUID.
	CreateTask(title, prompt, mode, provider, model, workingDir string) (taskID string, err error)

	// UpdateTaskStatus updates the task status.
	UpdateTaskStatus(taskID, status string) error

	// UpdateTaskWorkingDir updates the working directory for a task.
	UpdateTaskWorkingDir(taskID, workingDir string) error

	// CompleteTask marks the task as completed.
	CompleteTask(taskID string) error

	// FailTask marks the task as failed with an optional error message.
	FailTask(taskID string, errMsg string) error

	// === Message persistence ===

	// SaveMessage persists a single message to a task.
	SaveMessage(taskID string, msg types.Message) error

	// GetMessages retrieves all messages for a task, in order.
	GetMessages(taskID string) ([]MessageRecord, error)

	// === Tool call tracking ===

	// StartToolCall records the beginning of a tool call.
	StartToolCall(taskID, toolName string, input json.RawMessage) (callID int64, err error)

	// CompleteToolCall records successful completion of a tool call.
	CompleteToolCall(callID int64, output string) error

	// FailToolCall records failed tool call.
	FailToolCall(callID int64, err error) error

	// === History queries (for CLI) ===

	// ListTasks returns the most recent tasks with optional pagination.
	ListTasks(limit, offset int) ([]TaskRecord, error)

	// GetTaskByID returns a single task with its metadata.
	GetTaskByID(id string) (*TaskRecord, error)

	// GetTaskSummary returns a task with its messages.
	GetTaskSummary(id string) (*TaskRecord, []MessageRecord, error)

	// DeleteTask removes a task and all associated records.
	DeleteTask(id string) error

	// Close closes the underlying database connection.
	Close() error
}

// TaskRecord is the database model for a task.
type TaskRecord struct {
	ID          string
	Title       string
	Prompt      string
	Mode        string
	Provider    string
	Model       string
	Status      string
	WorkingDir  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// MessageRecord is the database model for a message.
type MessageRecord struct {
	ID               int64
	TaskID           string
	Role             string
	Content          string
	ReasoningContent string
	ToolCalls        string // JSON
	ToolCallID       string
	AvailableTools   string // JSON - the list of tools available when this message was sent
	ToolChoice       string // the tool_choice setting sent with the request (e.g. "required", "auto")
	CreatedAt        time.Time
}

// ToolCallRecord is the database model for a tool call.
type ToolCallRecord struct {
	ID          int64
	TaskID      string
	ToolName    string
	Input       string // JSON
	Output      string
	Error       string
	StartedAt   time.Time
	CompletedAt *time.Time
}

// String returns a human-readable description of the task.
func (t TaskRecord) String() string {
	return t.Title
}
