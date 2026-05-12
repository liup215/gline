package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

type toolOnlyProvider struct {
	callCount int
}

func (p *toolOnlyProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	return &MessageResponse{}, nil
}

func (p *toolOnlyProvider) CreateMessageStream(ctx context.Context, req *MessageRequest) (<-chan StreamChunk, error) {
	p.callCount++
	chunkChan := make(chan StreamChunk, 3)
	
	if p.callCount == 1 {
		// First call: return assistant message with tool call
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
	} else {
		// Second call: return empty response to end conversation
		chunkChan <- StreamChunk{Content: "Tool execution complete.", Done: true}
	}
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
 	streamStarts    int
 }
 
 func (c *recordingCallback) OnContent(delta string) {
 	c.content.WriteString(delta)
 }
 
 func (c *recordingCallback) OnStreamStart() {
 	c.streamStarts++
 }
 
 func (c *recordingCallback) OnToolCallStart(toolCall ToolCall) {
 	c.toolStartCount++
 }
 
 func (c *recordingCallback) OnToolCallComplete(toolCall ToolCall, result string) {
 	c.toolFinishCount++
 }
 
 // AskFollowupQuestion is needed by the updated StreamCallback interface.
 // For tests, provide a simple implementation that returns the first option (if any)
 // or an empty string.
 func (c *recordingCallback) AskFollowupQuestion(question string, options []string) (string, error) {
 	if len(options) > 0 {
 		return options[0], nil
 	}
 	return "", nil
 }
 
 func (c *recordingCallback) OnError(err error) {
 	// Tests don't assert on errors here.
 }
 
 func (c *recordingCallback) OnComplete() {
 	c.completeCount++
 }

func TestRunWithCallbackSurfacesToolCallsAsAssistantText(t *testing.T) {
agentInstance, err := New(Options{
	Provider:     &toolOnlyProvider{},
	ToolRegistry: tools.InitDefaultRegistry(),
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
// With a real registry the tool will execute; expect one start and one finish.
if callback.toolStartCount != 1 || callback.toolFinishCount != 1 {
t.Fatalf("expected one tool start and one finish, got starts=%d finishes=%d", callback.toolStartCount, callback.toolFinishCount)
}
if callback.completeCount != 1 {
t.Fatalf("expected OnComplete to be called once, got %d", callback.completeCount)
}

	// Find the assistant message with tool calls (should be the first assistant message)
	var assistantMsg *types.Message
	for _, msg := range agentInstance.GetConversation().GetMessages() {
		if msg.Role == types.RoleAssistant && len(msg.ToolCalls) > 0 {
			assistantMsg = &msg
			break
		}
	}
	if assistantMsg == nil {
		t.Fatal("expected to find assistant message with tool calls")
	}
	if !strings.Contains(assistantMsg.Content, `[tool:read_file] {"path":"README.md"}`) {
		t.Fatalf("assistant message missing rendered tool text: %q", assistantMsg.Content)
	}
	if !agentInstance.GetConversation().IsComplete() {
		t.Fatal("expected conversation to be complete after surfacing tool text")
	}
}