package version

import "fmt"

// 这些变量将在构建时通过 -ldflags 注入
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// String 返回格式化的版本信息
func String() string {
	return fmt.Sprintf("gline version %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}

// Short 返回简短版本号
func Short() string {
	return Version
}
