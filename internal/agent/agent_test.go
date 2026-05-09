package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

type toolOnlyProvider struct{}

func (p *toolOnlyProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	return &MessageResponse{}, nil
}

func (p *toolOnlyProvider) CreateMessageStream(ctx context.Context, req *MessageRequest) (<-chan StreamChunk, error) {
	chunkChan := make(chan StreamChunk, 3)
	chunkChan <- StreamChunk{Content: "I'll inspect that for you."}
	chunkChan <- StreamChunk{
		ToolCall: &ToolCall{
			ID:    "call_1",
			Name:  "read_file",
			Input: `{"path":"README.md"}`,
		},
		IsPartial: false,
	}
	chunkChan <- StreamChunk{FinishReason: "tool_calls", Done: true}
	close(chunkChan)
	return chunkChan, nil
}

func (p *toolOnlyProvider) SupportsTools() bool { return true }
func (p *toolOnlyProvider) GetModel() string    { return "test-model" }
func (p *toolOnlyProvider) GetProviderName() string {
	return "test-provider"
}

type recordingCallback struct {
	content         strings.Builder
	toolStartCount  int
	toolFinishCount int
	completeCount   int
}

func (c *recordingCallback) OnContent(delta string) {
	c.content.WriteString(delta)
}

func (c *recordingCallback) OnToolCallStart(toolCall ToolCall) {
	c.toolStartCount++
}

func (c *recordingCallback) OnToolCallComplete(toolCall ToolCall, result string) {
	c.toolFinishCount++
}

func (c *recordingCallback) OnError(err error) {
	testErr := err
	_ = testErr
}

func (c *recordingCallback) OnComplete() {
	c.completeCount++
}

func TestRunWithCallbackSurfacesToolCallsAsAssistantText(t *testing.T) {
	agentInstance, err := New(Options{
		Provider:     &toolOnlyProvider{},
		ToolRegistry: tools.NewRegistry(),
		Mode:         ModeAct,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	callback := &recordingCallback{}
	if err := agentInstance.RunWithCallback(context.Background(), "read the file", callback); err != nil {
		t.Fatalf("RunWithCallback() error = %v", err)
	}

	gotContent := callback.content.String()
	if !strings.Contains(gotContent, "I'll inspect that for you.") {
		t.Fatalf("callback content missing assistant text: %q", gotContent)
	}
	if !strings.Contains(gotContent, `[tool:read_file] {"path":"README.md"}`) {
		t.Fatalf("callback content missing rendered tool text: %q", gotContent)
	}
	if callback.toolStartCount != 0 || callback.toolFinishCount != 0 {
		t.Fatalf("expected no tool callbacks when tool execution is bypassed, got starts=%d finishes=%d", callback.toolStartCount, callback.toolFinishCount)
	}
	if callback.completeCount != 1 {
		t.Fatalf("expected OnComplete to be called once, got %d", callback.completeCount)
	}

	lastMessage := agentInstance.GetConversation().GetLastMessage()
	if lastMessage == nil {
		t.Fatal("expected conversation to contain messages")
	}
	if lastMessage.Role != types.RoleAssistant {
		t.Fatalf("expected last message role %q, got %q", types.RoleAssistant, lastMessage.Role)
	}
	if !strings.Contains(lastMessage.Content, `[tool:read_file] {"path":"README.md"}`) {
		t.Fatalf("assistant message missing rendered tool text: %q", lastMessage.Content)
	}
	if !agentInstance.GetConversation().IsComplete() {
		t.Fatal("expected conversation to be complete after surfacing tool text")
	}
}