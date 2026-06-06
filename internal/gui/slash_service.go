package gui

// SlashCommandInfo holds serializable metadata for a slash command.
type SlashCommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Section     string `json:"section"`
}

// SlashActionResult is returned after executing a slash command.
type SlashActionResult struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}
