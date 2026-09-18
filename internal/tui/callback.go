package tui

import (
	"github.com/liup215/gline/internal/agent"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Tea Messages ---

// streamStartMsg signals the start of a new streaming response.
type streamStartMsg struct{}

// contentDeltaMsg carries incremental text content from the assistant.
type contentDeltaMsg struct {
	Delta string
}

// reasoningDeltaMsg carries reasoning/thinking content.
type reasoningDeltaMsg struct {
	Delta string
}

// toolStartMsg signals that a tool call has started.
type toolStartMsg struct {
	ID    string
	Name  string
	Input string
}

// toolCompleteMsg signals that a tool call has completed.
type toolCompleteMsg struct {
	ID     string
	Name   string
	Result string
}

// streamErrorMsg carries an error from the streaming process.
type streamErrorMsg struct {
	Err error
}

// streamCompleteMsg signals the streaming response is done.
type streamCompleteMsg struct{}

// followupQuestionMsg carries a followup question from the agent.
type followupQuestionMsg struct {
	Question string
	Options  []string
	AnswerCh chan<- string
}

// taskCreatedMsg carries the ID of a newly created task.
type taskCreatedMsg struct {
	TaskID string
}

// providerInfoMsg carries provider/model info for the status bar.
type providerInfoMsg struct {
	Provider string
	Model    string
}

// tokenUsageMsg carries token usage info.
type tokenUsageMsg struct {
	InputTokens  int
	OutputTokens int
}

// --- TuiCallback ---

// TuiCallback adapts the agent.StreamCallback interface to send tea.Msg
// values through a channel, which are then processed by the Bubbletea Update loop.
type TuiCallback struct {
	program *tea.Program
}

// NewTuiCallback creates a new callback that sends messages to the given program.
func NewTuiCallback(program *tea.Program) *TuiCallback {
	return &TuiCallback{program: program}
}

func (c *TuiCallback) OnContent(delta string) {
	c.program.Send(contentDeltaMsg{Delta: delta})
}

func (c *TuiCallback) OnReasoning(delta string) {
	c.program.Send(reasoningDeltaMsg{Delta: delta})
}

func (c *TuiCallback) OnStreamStart() {
	c.program.Send(streamStartMsg{})
}

func (c *TuiCallback) OnToolCallStart(toolCall agent.ToolCall) {
	c.program.Send(toolStartMsg{
		ID:    toolCall.ID,
		Name:  toolCall.Name,
		Input: toolCall.Input,
	})
}

func (c *TuiCallback) OnToolCallComplete(toolCall agent.ToolCall, result string) {
	c.program.Send(toolCompleteMsg{
		ID:     toolCall.ID,
		Name:   toolCall.Name,
		Result: result,
	})
}

func (c *TuiCallback) AskFollowupQuestion(question string, options []string) (string, error) {
	answerCh := make(chan string, 1)
	c.program.Send(followupQuestionMsg{
		Question: question,
		Options:  options,
		AnswerCh: answerCh,
	})
	answer := <-answerCh
	return answer, nil
}

func (c *TuiCallback) OnError(err error) {
	c.program.Send(streamErrorMsg{Err: err})
}

func (c *TuiCallback) OnComplete() {
	c.program.Send(streamCompleteMsg{})
}

func (c *TuiCallback) OnTaskCreated(taskID string) {
	c.program.Send(taskCreatedMsg{TaskID: taskID})
}

// Ensure TuiCallback implements agent.StreamCallback
var _ agent.StreamCallback = (*TuiCallback)(nil)
