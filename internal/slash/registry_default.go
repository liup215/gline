package slash

// NewDefaultRegistry creates a registry with all built-in slash commands pre-registered.
func NewDefaultRegistry(conversation interface{ Clear() }, onResult func(result CommandResult, message string)) *Registry {
	r := NewRegistry()
	ctx := &CommandContext{
		Conversation: conversation,
		OnResult:     onResult,
	}
	for _, cmd := range DefaultCommands(ctx) {
		r.Register(cmd)
	}
	return r
}
