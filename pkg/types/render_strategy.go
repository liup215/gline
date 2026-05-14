package types

// RenderStrategy defines how a message should be rendered
type RenderStrategy int

const (
	StrategyPlain    RenderStrategy = 0
	StrategyMarkdown RenderStrategy = 1
	StrategyJSON     RenderStrategy = 2
	StrategySpecial  RenderStrategy = 3
	StrategySkip     RenderStrategy = 4
)

