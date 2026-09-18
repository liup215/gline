package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#3B82F6") // Blue
	ColorAccent    = lipgloss.Color("#10B981") // Green
	ColorDim       = lipgloss.Color("#6B7280") // Gray
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorWarning   = lipgloss.Color("#F59E0B") // Yellow
	ColorUser      = lipgloss.Color("#3B82F6") // Blue for user messages
	ColorAssistant = lipgloss.Color("#10B981") // Green for assistant
	ColorTool      = lipgloss.Color("#F59E0B") // Yellow for tool calls
	ColorBg        = lipgloss.Color("#1E1E2E") // Dark background
	ColorSurface   = lipgloss.Color("#313244") // Surface
	ColorText      = lipgloss.Color("#CDD6F4") // Light text
	ColorMuted     = lipgloss.Color("#6C7086") // Muted text
)

// Styles
var (
	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1).
			Bold(true)

	StatusBarModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorAccent).
				Padding(0, 1).
				Bold(true)

	StatusBarModelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1E1E2E")).
				Background(ColorSecondary).
				Padding(0, 1)

	// Messages
	UserLabelStyle = lipgloss.NewStyle().
			Foreground(ColorUser).
			Bold(true)

	AssistantLabelStyle = lipgloss.NewStyle().
				Foreground(ColorAssistant).
				Bold(true)

	SystemLabelStyle = lipgloss.NewStyle().
				Foreground(ColorDim).
				Italic(true)

	ToolLabelStyle = lipgloss.NewStyle().
			Foreground(ColorTool).
			Bold(true)

	ToolBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Input
	PromptStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	InputStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Sidebar
	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(ColorDim).
			Padding(0, 1)

	SidebarTitleStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 0, 1, 0)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Padding(0, 1)

	SidebarItemActiveStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				Padding(0, 1)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Completion
	CompletionStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 0, 0, 1)

	// Divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorDim)
)
