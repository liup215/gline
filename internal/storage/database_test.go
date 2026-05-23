// Package storage provides persistent storage for gline.
package storage

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/liup215/gline/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *SQLiteStore {
	s, err := NewSQLiteStoreInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSQLiteStore_CreateTask(t *testing.T) {
	s := newTestStore(t)

	t.Run("create basic task", func(t *testing.T) {
		id, err := s.CreateTask("", "hello world", "act", "openai", "gpt-4")
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		task, err := s.GetTaskByID(id)
		require.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "hello world", task.Title) // prompt becomes title
		assert.Equal(t, "act", task.Mode)
		assert.Equal(t, "openai", task.Provider)
		assert.Equal(t, "gpt-4", task.Model)
		assert.Equal(t, "running", task.Status)
	})

	t.Run("create task with title", func(t *testing.T) {
		id, err := s.CreateTask("My Bug Fix", "fix the bug", "plan", "anthropic", "claude-3")
		require.NoError(t, err)

		task, err := s.GetTaskByID(id)
		require.NoError(t, err)
		assert.Equal(t, "My Bug Fix", task.Title)
	})

	t.Run("long prompt generates truncated title", func(t *testing.T) {
		longPrompt := "this is a very long prompt that should be truncated when used as a title because it exceeds the fifty character limit"
		id, err := s.CreateTask("", longPrompt, "act", "openai", "gpt-4")
		require.NoError(t, err)

		task, err := s.GetTaskByID(id)
		require.NoError(t, err)
		assert.Len(t, task.Title, 44) // "this is a very long prompt that should be..."
	})
}

func TestSQLiteStore_TaskLifecycle(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateTask("", "test", "act", "openai", "gpt-4")
	require.NoError(t, err)

	t.Run("complete task", func(t *testing.T) {
		err := s.CompleteTask(id)
		require.NoError(t, err)

		task, err := s.GetTaskByID(id)
		require.NoError(t, err)
		assert.Equal(t, "completed", task.Status)
		assert.NotNil(t, task.CompletedAt)
	})

	t.Run("fail task", func(t *testing.T) {
		id2, err := s.CreateTask("", "test2", "act", "openai", "gpt-4")
		require.NoError(t, err)

		err = s.FailTask(id2, "something went wrong")
		require.NoError(t, err)

		task, err := s.GetTaskByID(id2)
		require.NoError(t, err)
		assert.Equal(t, "failed", task.Status)
	})
}

func TestSQLiteStore_Messages(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateTask("", "test", "act", "openai", "gpt-4")
	require.NoError(t, err)

	t.Run("save and retrieve messages", func(t *testing.T) {
		msgs := []types.Message{
			{Role: types.RoleSystem, Content: "You are helpful", Timestamp: time.Now()},
			{Role: types.RoleUser, Content: "Hello", Timestamp: time.Now()},
			{Role: types.RoleAssistant, Content: "Hi there!", Timestamp: time.Now()},
		}

		for _, m := range msgs {
			err := s.SaveMessage(id, m)
			require.NoError(t, err)
		}

		records, err := s.GetMessages(id)
		require.NoError(t, err)
		assert.Len(t, records, 3)
		assert.Equal(t, "system", records[0].Role)
		assert.Equal(t, "You are helpful", records[0].Content)
		assert.Equal(t, "user", records[1].Role)
		assert.Equal(t, "assistant", records[2].Role)
	})

	t.Run("save message with tool calls", func(t *testing.T) {
		msg := types.Message{
			Role: types.RoleAssistant,
			ToolCalls: []types.ToolCall{
				{ID: "call_1", Name: "read_file", Input: []byte(`{"path": "/tmp/test"}`)},
			},
			Timestamp: time.Now(),
		}
		err := s.SaveMessage(id, msg)
		require.NoError(t, err)

		records, err := s.GetMessages(id)
		require.NoError(t, err)
		found := false
		for _, r := range records {
			if r.Role == "assistant" && r.ToolCalls != "" {
				found = true
				var tcs []types.ToolCall
				err := json.Unmarshal([]byte(r.ToolCalls), &tcs)
				require.NoError(t, err)
				assert.Len(t, tcs, 1)
				assert.Equal(t, "read_file", tcs[0].Name)
			}
		}
		assert.True(t, found, "should find message with tool calls")
	})
}

func TestSQLiteStore_ToolCalls(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateTask("", "test", "act", "openai", "gpt-4")
	require.NoError(t, err)

	t.Run("start and complete tool call", func(t *testing.T) {
		callID, err := s.StartToolCall(id, "read_file", []byte(`{"path": "/tmp/test"}`))
		require.NoError(t, err)
		assert.Greater(t, callID, int64(0))

		err = s.CompleteToolCall(callID, "file contents")
		require.NoError(t, err)
	})

	t.Run("start and fail tool call", func(t *testing.T) {
		callID, err := s.StartToolCall(id, "write_to_file", []byte(`{"path": "/tmp/test"}`))
		require.NoError(t, err)

		err = s.FailToolCall(callID, assert.AnError)
		require.NoError(t, err)
	})
}

func TestSQLiteStore_ListTasks(t *testing.T) {
	s := newTestStore(t)

	// Create multiple tasks
	for i := 0; i < 5; i++ {
		_, err := s.CreateTask("", "task", "act", "openai", "gpt-4")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // ensure different timestamps
	}

	t.Run("list with default limit", func(t *testing.T) {
		tasks, err := s.ListTasks(10, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
		assert.NotEmpty(t, tasks[0].ID)
	})

	t.Run("list with pagination", func(t *testing.T) {
		tasks, err := s.ListTasks(3, 0)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		tasks2, err := s.ListTasks(3, 3)
		require.NoError(t, err)
		assert.Len(t, tasks2, 2)
	})

	t.Run("order is descending by creation time", func(t *testing.T) {
		tasks, err := s.ListTasks(5, 0)
		require.NoError(t, err)
		for i := 0; i < len(tasks)-1; i++ {
			assert.True(t, tasks[i].CreatedAt.After(tasks[i+1].CreatedAt) || tasks[i].CreatedAt.Equal(tasks[i+1].CreatedAt),
				"tasks should be ordered newest first")
		}
	})
}

func TestSQLiteStore_DeleteTask(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateTask("", "test", "act", "openai", "gpt-4")
	require.NoError(t, err)

	// Save a message
	err = s.SaveMessage(id, types.Message{Role: types.RoleUser, Content: "hello", Timestamp: time.Now()})
	require.NoError(t, err)

	// Delete task
	err = s.DeleteTask(id)
	require.NoError(t, err)

	// Verify deleted
	task, err := s.GetTaskByID(id)
	require.NoError(t, err)
	assert.Nil(t, task)

	// Verify cascade delete for messages
	msgs, err := s.GetMessages(id)
	require.NoError(t, err)
	assert.Len(t, msgs, 0)
}

func TestGenerateDefaultTitle(t *testing.T) {
	assert.Equal(t, "Untitled task", generateDefaultTitle(""))
	assert.Equal(t, "hello", generateDefaultTitle("hello"))
	assert.Equal(t, "hello world", generateDefaultTitle("hello\nworld"))
	assert.Equal(t, "hello world", generateDefaultTitle("hello   world"))

	long := "this is a very long prompt that definitely exceeds fifty characters in length"
	result := generateDefaultTitle(long)
	assert.True(t, len(result) <= 53)
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestFormatTaskList(t *testing.T) {
	now := time.Now().UTC()
	tasks := []TaskRecord{
		{ID: "1", Title: "Fix bug", Mode: "act", Provider: "openai", Model: "gpt-4", Status: "completed", CreatedAt: now},
		{ID: "2", Title: "Plan refactor", Mode: "plan", Provider: "anthropic", Model: "claude-3", Status: "running", CreatedAt: now},
	}
	out := FormatTaskList(tasks, true)
	assert.Contains(t, out, "Fix bug")
	assert.Contains(t, out, "Plan refactor")
	assert.Contains(t, out, "[0]")
	assert.Contains(t, out, "[1]")

	// Empty list
	assert.Equal(t, "No tasks found.", FormatTaskList(nil, false))
}

func TestFormatTaskDetail(t *testing.T) {
	now := time.Now().UTC()
	task := &TaskRecord{
		ID:        "abc",
		Title:     "Test task",
		Prompt:    "do something",
		Mode:      "act",
		Provider:  "openai",
		Model:     "gpt-4",
		Status:    "running",
		CreatedAt: now,
	}
	msgs := []MessageRecord{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "", ToolCalls: `[{"id":"1","name":"read_file","input":"eyJwYXRoIjoiL3RtcC90ZXN0In0="}]`},
	}
	out := FormatTaskDetail(task, msgs)
	assert.Contains(t, out, "Test task")
	assert.Contains(t, out, "do something")
	assert.Contains(t, out, "Messages (2)")
	assert.Contains(t, out, "[tool call]")

	// Not found
	assert.Equal(t, "Task not found.", FormatTaskDetail(nil, nil))
}

func TestToTypesMessage(t *testing.T) {
	rec := MessageRecord{
		Role:             "assistant",
		Content:          "test",
		ReasoningContent: "thinking",
		ToolCallID:       "call_1",
		ToolCalls:        `[{"id":"call_1","name":"read_file","input":"eyJwYXRoIjoiL3RtcC90ZXN0In0="}]`,
		CreatedAt:        time.Now(),
	}
	msg, err := rec.ToTypesMessage()
	require.NoError(t, err)
	assert.Equal(t, types.RoleAssistant, msg.Role)
	assert.Equal(t, "test", msg.Content)
	assert.Equal(t, "thinking", msg.ReasoningContent)
	assert.Equal(t, "call_1", msg.ToolCallID)
	assert.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "read_file", msg.ToolCalls[0].Name)
}
