// Package ui provides the TUI (Terminal User Interface) for gline
package ui

import (
"context"
"fmt"
"sync"
"time"
  
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/textarea"
"github.com/charmbracelet/bubbles/viewport"
tea "github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
	"strings"
 
"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/slash"
"github.com/liup215/gline/internal/ui/bridge"
"github.com/liup215/gline/internal/ui/model"
"github.com/liup215/gline/internal/ui/tool"
	"github.com/liup215/gline/internal/ui/view"
"github.com/liup215/gline/internal/ui/viewmodel"
"github.com/liup215/gline/pkg/types"
)

// Model represents the TUI state.
// Business data (messages, tool history, mode, provider, model name) lives in
// *model.Conversation; UI-specific state (activeAssistantIndex, isProcessing,
// etc.) stays here.
type Model struct {
 // Domain model (extracted from the former god-object)
 conversation *model.Conversation
 
 // UI components
 viewport viewport.Model
 textarea textarea.Model
 spinner  spinner.Model
 
 // UI-only state
 inputHeight          int
 toolAreaHeight       int
 isProcessing         bool
 isStreaming          bool
 err                  error
 currentTool          string
 activeAssistantIndex int
 
 // Agent components
 agentInstance *agent.BaseAgent
 ctx           context.Context
    // Backwards-compatible cancel channel (kept for tests). Prefer agentCtx/agentCancel.
    // Buffered size 1 to mimic previous single-value container behavior.
    cancelCh chan context.CancelFunc

    // Use a context + cancel func pair protected by a RWMutex to broadcast cancel
    // and avoid data races when multiple goroutines may read/call cancel.
    agentCtx    context.Context
    agentCancel context.CancelFunc
    cancelLock  sync.RWMutex
 
 // Bridge channel: TUIBridge sends events here; a forwarding goroutine
 // relays them to tea.Program.Send so that Bridge stays decoupled from Bubbletea.
 eventCh chan bridge.AgentEvent
 done    chan struct{} // signals the forwarding goroutine to stop
 
 // Pending reply channel when the UI is answering an AskFollowupQuestion
 pendingReply chan string
 
 // ViewModel derives rendered display state from the conversation.
 convVM *viewmodel.ConversationViewModel

 // Tool registry for rendering tool outputs
 toolRegistry *tool.Registry

 // Dimensions
 width  int
 height int

 // Performance optimization: only refresh viewport when content actually changed
 contentChanged bool

	// Slash command pending quit flag
	quitting         bool

	// Slash command menu state
	slashMenu        *SlashMenuState
}

// New creates a new TUI model
func New(agentInstance *agent.BaseAgent) *Model {
	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = ""

	// Create viewport for messages
	vp := viewport.New(80, 20)

	// Create spinner for loading animation
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))

	// Get provider and model info from agent
	var providerName, modelName string
	if agentInstance != nil {
		if provider := agentInstance.GetProvider(); provider != nil {
			providerName = provider.GetProviderName()
			modelName = provider.GetModel()
		}
	}

	conv := model.NewConversation()
	conv.Provider = providerName
	conv.ModelName = modelName

	m := &Model{
		textarea:             ta,
		viewport:             vp,
		spinner:              s,
		conversation:         conv,
		convVM:               viewmodel.NewConversationViewModel(),
		toolRegistry:         tool.NewDefaultRegistry(),
		inputHeight:          3,
		toolAreaHeight:       3,
		activeAssistantIndex: -1,
		agentInstance:        agentInstance,
		ctx:                  context.Background(),
		cancelCh:             make(chan context.CancelFunc, 1),
		pendingReply:         nil,
	}
	m.slashMenu = NewSlashMenuState(slash.NewDefaultRegistry(conv, func(result slash.CommandResult, message string) {
		handleSlashCommandResult(m, result, message)
	}))
	return m
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		textarea.Blink,
		m.spinner.Tick,
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	needsRefresh := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmds = append(cmds, handleWindowSize(m, msg)...)

	case tea.KeyMsg:
		cmds = append(cmds, handleKeyMsg(m, msg)...)

	case spinner.TickMsg:
		// Update spinner animation
		if m.isProcessing {
			newSpinner, cmd := m.spinner.Update(msg)
			m.spinner = newSpinner
			cmds = append(cmds, cmd)
		}

	case bridge.AskQuestionEvent:
		// Display the follow-up question and options, set pending reply channel
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   "❓ " + msg.Question,
			Options:   msg.Options,
			MsgType:   types.TypeQuestion,
			Strategy:  types.StrategyPlain,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
		// Set the reply channel so Enter will send the answer back to the agent
		m.pendingReply = msg.Reply
		m.textarea.Reset()
		m.textarea.Placeholder = "Type option number or your answer..."
		m.textarea.Focus()
		cmds = append(cmds, textarea.Blink)
		needsRefresh = true

	case bridge.AgentEvent:
		// Delegate to extracted handler; collect needsRefresh for unified viewport update
		needsAgentRefresh, agentCmds := handleAgentUpdate(m, msg)
		cmds = append(cmds, agentCmds...)
		if needsAgentRefresh {
			needsRefresh = true
		}
		// Force scroll to bottom on stream start to follow streaming output
		if _, ok := msg.(bridge.StreamStartEvent); ok {
			m.updateViewportForceScroll()
			needsRefresh = false // already handled
		}

	// Handle internal messages for real-time viewport updates
	case tickMsg:
		if m.isProcessing && m.contentChanged {
			m.updateViewport()
			m.contentChanged = false
		}
		if m.isProcessing {
			cmds = append(cmds, m.tick())
		}
	}

	// Unified viewport refresh: if any handler signaled a state change,
	// refresh the viewport content once instead of each handler doing it individually.
	if needsRefresh {
		m.contentChanged = true
		m.updateViewport()
	}

	// Update textarea
	newTextarea, textareaCmd := m.textarea.Update(msg)
	m.textarea = newTextarea
	cmds = append(cmds, textareaCmd)

	// Update slash menu query based on current textarea content.
	// We detect slash mode by checking if the value starts with / and has no space.
	if m.slashMenu != nil {
		v := m.textarea.Value()
		if !m.slashMenu.Active {
			if strings.HasPrefix(v, "/") && !strings.Contains(strings.TrimPrefix(v, "/"), " ") {
				m.slashMenu.EnterSlashMode()
				m.slashMenu.UpdateQuery(v, len(v))
			}
		} else {
			m.slashMenu.UpdateQuery(v, len(v))
		}
	}

	// Update viewport
	newViewport, viewportCmd := m.viewport.Update(msg)
	m.viewport = newViewport
	cmds = append(cmds, viewportCmd)

	// If a slash command requested quit, append tea.Quit so Bubbletea exits.
	if m.quitting {
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

// tickMsg is used for periodic viewport refresh during processing
type tickMsg time.Time

// tick returns a command that sends a tick message for periodic updates
func (m *Model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// isNearBottom checks if viewport is near bottom (within threshold lines).
// This is more forgiving than AtBottom() which requires exact bottom position.
func isNearBottom(v viewport.Model, content string, threshold int) bool {
	totalLines := len(strings.Split(content, "\n"))
	visibleEnd := v.YOffset + v.Height
	return totalLines - visibleEnd <= threshold
}

// updateViewport refreshes the viewport content via the ViewModel.
// forceScroll parameter controls whether to force scroll to bottom.
func (m *Model) updateViewport() {
	m.updateViewportWithOptions(false)
}

// updateViewportForceScroll refreshes the viewport and forces scroll to bottom.
// Use this when user sends a new message or when streaming starts.
func (m *Model) updateViewportForceScroll() {
	m.updateViewportWithOptions(true)
}

// updateViewportWithOptions refreshes the viewport content with options.
func (m *Model) updateViewportWithOptions(forceScroll bool) {
	m.convVM.Refresh(m.conversation, m.viewport.Width, m.toolAreaHeight, m.isStreaming, m.activeAssistantIndex)
	content := m.convVM.Content()
	m.viewport.SetContent(content)
	// Scroll to bottom when:
	// 1. forceScroll is true (user sent new message or streaming started)
	// 2. User is at bottom or near bottom (within 10 lines) to follow streaming output
	if forceScroll || isNearBottom(m.viewport, content, 10) {
		m.viewport.GotoBottom()
	}
}

// View renders the TUI by composing pure view functions.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Render slash command menu if active
	var menu string
	if m.slashMenu != nil && m.slashMenu.Active && len(m.slashMenu.Filtered) > 0 {
		menu = view.RenderSlashMenu(view.SlashMenuData{
			Commands:      m.slashMenu.Filtered,
			SelectedIndex: m.slashMenu.Selected,
			Width:         m.width,
			Query:         m.slashMenu.Query,
		}, 5)
	}

	// Render input status bar (below input, cline style)
	inputStatusBar := view.RenderInputStatusBar(view.InputStatusBarData{
		Mode:      m.conversation.Mode,
		Provider:  m.conversation.Provider,
		ModelName: m.conversation.ModelName,
		Width:     m.width,
	})

	layout := view.RenderLayout(view.LayoutData{
		CompactBar: view.RenderCompactBar(view.CompactBarData{
			Mode:         m.conversation.Mode,
			Provider:     m.conversation.Provider,
			ModelName:    m.conversation.ModelName,
			IsProcessing: m.isProcessing,
			IsStreaming:  m.isStreaming,
			CurrentTool:  m.currentTool,
			SpinnerView:  m.spinner.View(),
			Width:        m.width,
		}),
		Content:        m.viewport.View(),
		InputView:      m.textarea.View(),
		InputStatusBar: inputStatusBar,
		Help:           view.RenderHelp(),
		Menu:           menu,
		Height:         m.height,
		InputHeight:    m.inputHeight,
	})

	return layout
}

// handleSlashCommandResult processes the result of a slash command execution.
func handleSlashCommandResult(m *Model, result slash.CommandResult, message string) {
	switch result {
	case slash.ResultClearScreen:
		// If agent is processing, abort it first so the clear is clean
		if m.isProcessing && m.agentInstance != nil {
			m.agentInstance.Abort()
		}
		m.conversation.Clear()
		m.convVM.InvalidateCache()
		if m.agentInstance != nil {
			if conv := m.agentInstance.GetConversation(); conv != nil {
				conv.Clear()
			}
		}
		m.isProcessing = false
		m.isStreaming = false
		m.currentTool = ""
		m.textarea.Focus()
		m.updateViewport()
	case slash.ResultQuit:
		m.quitting = true
	case slash.ResultShowHelp:
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   message,
			MsgType:   types.TypeNormal,
			Strategy: types.StrategyPlain,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
		m.updateViewport()
	case slash.ResultNewTask:
		// Abort any running agent work before starting a new task
		if m.isProcessing && m.agentInstance != nil {
			m.agentInstance.Abort()
		}
		m.conversation.Clear()
		m.convVM.InvalidateCache()
		if m.agentInstance != nil {
			if conv := m.agentInstance.GetConversation(); conv != nil {
				conv.Clear()
			}
		}
		m.isProcessing = false
		m.isStreaming = false
		m.currentTool = ""
		m.activeAssistantIndex = -1
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   message,
			MsgType:   types.TypeNormal,
			Strategy: types.StrategyPlain,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
		m.updateViewport()
	case slash.ResultCompact:
		// Trim the agent conversation to fit within token budget.
		if m.agentInstance != nil {
			if conv := m.agentInstance.GetConversation(); conv != nil {
				before := conv.MessageCount()
				conv.TrimToMaxTokens()
				after := conv.MessageCount()
				message = fmt.Sprintf("Context compacted: %d messages removed, %d remaining.", before-after, after)
			}
		}
		idx := m.conversation.AppendMessage(model.Message{
			Role:      types.RoleSystem,
			Content:   message,
			MsgType:   types.TypeNormal,
			Strategy:  types.StrategyPlain,
			Timestamp: time.Now(),
		})
		m.convVM.MarkMessageDirty(idx)
		m.updateViewport()
	}
}

// sendMessage adds a user message and prepares for response
func (m *Model) sendMessage(content string) {
	// Clear tool history for new conversation turn
	m.conversation.ClearToolHistory()

	// Add user message
	idx := m.conversation.AppendMessage(model.Message{
		Role:      types.RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.convVM.MarkMessageDirty(idx)

	// Do not pre-create assistant message slot here.
	// The UI will create the assistant slot when the agent signals streamStart.
	m.activeAssistantIndex = -1

	m.isProcessing = true
	m.isStreaming = false
	m.contentChanged = true
	// Force scroll to bottom to show the new user message
	m.updateViewportForceScroll()
}

// addErrorMessage adds an error message with proper typing
func (m *Model) addErrorMessage(content string) {
	msg := model.Message{
		Role:     types.RoleSystem,
		Content:  content,
		MsgType:  types.TypeError,
		Strategy: types.StrategyPlain,
		Timestamp: time.Now(),
	}
	// Optionally set metadata for complex errors
	// msg.SetMeta(model.ErrorMeta{Code: 500, Retryable: false})
	idx := m.conversation.AppendMessage(msg)
	m.convVM.MarkMessageDirty(idx)
	m.contentChanged = true
	m.updateViewport()
}

// Run starts the TUI with a Bridge-based event forwarding architecture.
// A buffered channel carries events from TUIBridge (Agent side) to the
// Bubbletea Program; a goroutine relays them via program.Send.
func Run(agentInstance *agent.BaseAgent) error {
	m := New(agentInstance)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Create the event channel and wire it into the Model
	eventCh := make(chan bridge.AgentEvent, 64)
	m.eventCh = eventCh
	done := make(chan struct{})
	m.done = done

	// Forward goroutine: reads events from the bridge channel and sends
	// them into the Bubbletea event loop. This keeps TUIBridge decoupled
	// from tea.Program while preserving ordered delivery.
	go func() {
		for {
			select {
			case evt := <-eventCh:
				p.Send(evt)
			case <-done:
				return
			}
		}
	}()

	_, err := p.Run()
	close(done) // signal the forwarding goroutine to stop
	return err
}

// calculateLayout calculates flexible height distribution for TUI components.
// Fixed elements and their heights:
//   - CompactBar: 1 line
//   - Input box border (top + bottom): 2 lines
//   - InputStatusBar: 1 line
//   - Help: 1 line
//   Total fixed overhead: 5 lines
// Variable elements:
//   - inputH: textarea content height (10% of total, min 3, max 5)
//   - toolH: tool area height (10% of total, min 2, max 6, currently unused in layout)
//   - viewportH: remaining space for message content, min 3
//   - menuHeight: dynamically calculated when slash menu is active
func calculateLayout(totalHeight int) (viewportH, toolH, inputH int) {
	if totalHeight < 10 {
		// Minimum viable layout for very small terminals
		return 3, 2, 3
	}

	// Fixed overhead: CompactBar(1) + InputBoxBorders(2) + InputStatusBar(1) + Help(1) = 5
	fixedOverhead := 5

	// Calculate proportional heights for input
	inputH = totalHeight / 10
	if inputH < 3 {
		inputH = 3
	} else if inputH > 5 {
		inputH = 5
	}

	// Tool area height (for future use, not currently rendered in layout)
	toolH = totalHeight / 10
	if toolH < 2 {
		toolH = 2
	} else if toolH > 6 {
		toolH = 6
	}

	// Viewport gets remaining space after subtracting fixed overhead and input content height
	viewportH = totalHeight - fixedOverhead - inputH
	if viewportH < 3 {
		viewportH = 3
	}

	return viewportH, toolH, inputH
}