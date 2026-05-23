// Package storage provides persistent storage for gline.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liup215/gline/pkg/types"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed store.
// Uses the default database path if dbPath is empty.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		dbPath = DefaultDBPath()
	}

	db, err := Open(dbPath)
	if err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// NewSQLiteStoreInMemory creates an in-memory SQLite store (for testing).
func NewSQLiteStoreInMemory() (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", ":memory:?_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// CreateTask creates a new task record.
func (s *SQLiteStore) CreateTask(title, prompt, mode, provider, model string) (taskID string, err error) {
	uuid, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate task ID: %w", err)
	}

	// Generate a default title from prompt if not provided
	if title == "" {
		title = generateDefaultTitle(prompt)
	}

	_, err = s.db.Exec(`
		INSERT INTO tasks (id, title, prompt, mode, provider, model, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'running', ?, ?)
	`, uuid, title, prompt, mode, provider, model, now(), now())
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	return uuid, nil
}

// UpdateTaskStatus updates the task status.
func (s *SQLiteStore) UpdateTaskStatus(taskID, status string) error {
	_, err := s.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?",
		status, now(), taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	return nil
}

// CompleteTask marks the task as completed.
func (s *SQLiteStore) CompleteTask(taskID string) error {
	n := now()
	_, err := s.db.Exec(
		"UPDATE tasks SET status = 'completed', completed_at = ?, updated_at = ? WHERE id = ?",
		&n, n, taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}
	return nil
}

// FailTask marks the task as failed.
func (s *SQLiteStore) FailTask(taskID string, errMsg string) error {
	n := now()
	// For now store error in title prefix, or we can add an error column later.
	// We'll append error info to title for now, or better, just update status.
	_, err := s.db.Exec(
		"UPDATE tasks SET status = 'failed', completed_at = ?, updated_at = ? WHERE id = ?",
		&n, n, taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to fail task: %w", err)
	}
	return nil
}

// SaveMessage persists a message.
func (s *SQLiteStore) SaveMessage(taskID string, msg types.Message) error {
	// Serialize tool calls to JSON if present
	var toolCallsJSON []byte
	if len(msg.ToolCalls) > 0 {
		var err error
		toolCallsJSON, err = json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO messages (task_id, role, content, reasoning_content, tool_calls, tool_call_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, taskID, string(msg.Role), msg.Content, msg.ReasoningContent, string(toolCallsJSON), msg.ToolCallID, msg.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

// GetMessages retrieves all messages for a task.
func (s *SQLiteStore) GetMessages(taskID string) ([]MessageRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, task_id, role, content, reasoning_content, tool_calls, tool_call_id, created_at
		FROM messages
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageRecord
	for rows.Next() {
		var m MessageRecord
		var toolCalls sql.NullString
		var reasoningContent sql.NullString
		var toolCallID sql.NullString
		if err := rows.Scan(&m.ID, &m.TaskID, &m.Role, &m.Content, &reasoningContent, &toolCalls, &toolCallID, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		m.ReasoningContent = reasoningContent.String
		m.ToolCalls = toolCalls.String
		m.ToolCallID = toolCallID.String
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating messages: %w", err)
	}
	return messages, nil
}

// StartToolCall records the start of a tool call.
func (s *SQLiteStore) StartToolCall(taskID, toolName string, input json.RawMessage) (callID int64, err error) {
	result, err := s.db.Exec(`
		INSERT INTO tool_calls (task_id, tool_name, input, started_at)
		VALUES (?, ?, ?, ?)
	`, taskID, toolName, string(input), now())
	if err != nil {
		return 0, fmt.Errorf("failed to start tool call: %w", err)
	}
	return result.LastInsertId()
}

// CompleteToolCall records successful completion.
func (s *SQLiteStore) CompleteToolCall(callID int64, output string) error {
	_, err := s.db.Exec(
		"UPDATE tool_calls SET output = ?, completed_at = ? WHERE id = ?",
		output, now(), callID,
	)
	if err != nil {
		return fmt.Errorf("failed to complete tool call: %w", err)
	}
	return nil
}

// FailToolCall records failed completion.
func (s *SQLiteStore) FailToolCall(callID int64, err error) error {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	_, dbErr := s.db.Exec(
		"UPDATE tool_calls SET error = ?, completed_at = ? WHERE id = ?",
		errStr, now(), callID,
	)
	if dbErr != nil {
		return fmt.Errorf("failed to fail tool call: %w", dbErr)
	}
	return nil
}

// === History queries ===

// ListTasks returns tasks ordered by most recent first.
func (s *SQLiteStore) ListTasks(limit, offset int) ([]TaskRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.Query(`
		SELECT id, title, prompt, mode, provider, model, status, created_at, updated_at, completed_at
		FROM tasks
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var t TaskRecord
		var completedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.Title, &t.Prompt, &t.Mode, &t.Provider, &t.Model, &t.Status, &t.CreatedAt, &t.UpdatedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating tasks: %w", err)
	}
	return tasks, nil
}

// GetTaskByID fetches a single task by ID.
func (s *SQLiteStore) GetTaskByID(id string) (*TaskRecord, error) {
	var t TaskRecord
	var completedAt sql.NullTime
	row := s.db.QueryRow(`
		SELECT id, title, prompt, mode, provider, model, status, created_at, updated_at, completed_at
		FROM tasks WHERE id = ?
	`, id)
	err := row.Scan(&t.ID, &t.Title, &t.Prompt, &t.Mode, &t.Provider, &t.Model, &t.Status, &t.CreatedAt, &t.UpdatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return &t, nil
}

// GetTaskSummary returns a task with its messages.
func (s *SQLiteStore) GetTaskSummary(id string) (*TaskRecord, []MessageRecord, error) {
	task, err := s.GetTaskByID(id)
	if err != nil {
		return nil, nil, err
	}
	if task == nil {
		return nil, nil, nil
	}

	msgs, err := s.GetMessages(id)
	if err != nil {
		return nil, nil, err
	}
	return task, msgs, nil
}

// DeleteTask removes a task and all associated records.
func (s *SQLiteStore) DeleteTask(id string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// generateDefaultTitle creates a default title from the prompt.
func generateDefaultTitle(prompt string) string {
	// Take first few words, up to 50 chars
	if len(prompt) == 0 {
		return "Untitled task"
	}

	// Remove newlines and extra spaces
	prompt = strings.Join(strings.Fields(prompt), " ")

	if len(prompt) <= 50 {
		return prompt
	}
	// Try to cut at a word boundary
	cut := strings.LastIndexAny(prompt[:50], " ")
	if cut > 20 {
		return prompt[:cut] + "..."
	}
	return prompt[:50] + "..."
}

// ToTypesMessage converts a MessageRecord to a types.Message.
func (m MessageRecord) ToTypesMessage() (types.Message, error) {
	msg := types.Message{
		Role:             types.Role(m.Role),
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCallID:       m.ToolCallID,
		Timestamp:        m.CreatedAt,
	}
	if m.ToolCalls != "" {
		var tcs []types.ToolCall
		if err := json.Unmarshal([]byte(m.ToolCalls), &tcs); err != nil {
			return msg, fmt.Errorf("failed to unmarshal tool calls: %w", err)
		}
		msg.ToolCalls = tcs
	}
	return msg, nil
}

// AutoClose wraps a closer to be called on a delay (useful for deferred cleanup).
func AutoClose(s Store, delay time.Duration) {
	if delay > 0 {
		time.Sleep(delay)
	}
	_ = s.Close()
}
