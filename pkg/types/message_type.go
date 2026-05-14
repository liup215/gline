package types

// MessageType defines the semantic type of a system message
// This is separate from RenderStrategy (how to render) to allow flexibility
type MessageType int

const (
	TypeNormal       MessageType = 0 // Default system message
	TypeError        MessageType = 1 // Error message (red styling)
	TypeQuestion     MessageType = 2 // Question with options
	TypeToolStart    MessageType = 3 // Tool start notification
	TypeToolComplete MessageType = 4 // Tool completion notification
)
