# Tech Context

## 技术栈

### 核心依赖

| 组件 | 库 | 版本 | 用途 |
|------|-----|------|------|
| GUI 框架 | `github.com/wailsapp/wails/v3` | v3.0-alpha | 跨平台桌面应用 |
| CLI 框架 | `github.com/spf13/cobra` | v1.10+ | 命令行解析 |
| 配置管理 | `github.com/spf13/viper` | v1.21+ | 配置加载/管理 |
| 数据库 | `modernc.org/sqlite` | v1.50+ | 纯 Go SQLite（零 CGO） |
| HTTP 客户端 | `github.com/go-resty/resty/v2` | v2.12+ | API 调用 |
| 日志 | `github.com/rs/zerolog` | v1.35+ | 结构化日志 |
| 文件监控 | `github.com/fsnotify/fsnotify` | v1.7+ | 配置热更新 |

### 开发工具

- **构建**: `go task` (Taskfile) 替代 Makefile
- **测试**: `go test` + `github.com/stretchr/testify`
- **Lint**: `golangci-lint`
- **格式化**: `gofumpt`, `golines`

## 项目结构

```
gline/
├── cmd/
│   └── gline/                   # CLI 命令入口
│       ├── main.go
│       ├── root.go
│       ├── chat.go
│       └── history.go
├── internal/
│   ├── agent/                   # Agent 核心逻辑
│   │   ├── agent.go
│   │   ├── provider.go
│   │   └── executor.go
│   ├── api/                     # LLM 提供商
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   └── registry.go
│   ├── tools/                   # 工具系统
│   │   ├── tool.go
│   │   ├── registry.go
│   │   ├── file.go
│   │   ├── command.go
│   │   └── search.go
│   ├── prompts/                 # 提示词管理
│   │   ├── system.go
│   │   ├── tools.go
│   │   └── rules.go             # 自定义规则加载
│   ├── storage/                 # 数据持久化
│   │   ├── store.go
│   │   ├── database.go
│   │   ├── sqlite.go
│   │   └── history.go
│   ├── config/                  # 配置管理
│   │   └── config.go
│   ├── ui/                      # 已废弃 (Bubbletea TUI)
│   ├── log/                     # 日志系统
│   └── utils/                   # 工具函数
├── pkg/
│   └── types/                   # 共享类型
│       ├── message.go
│       ├── tool.go
│       └── conversation.go
├── gui/                         # Wails v3 GUI 应用
│   ├── main.go                  # GUI 入口
│   ├── backend.go               # 后端服务
│   ├── chat_service.go          # 聊天服务
│   ├── frontend/                # 前端资源 (Vite + React)
│   ├── build/                   # 构建脚本
│   └── Taskfile.yml             # GUI 构建任务
├── Makefile
├── go.mod
└── README.md
```

## 配置管理

### 配置文件位置

- **macOS**: `~/.config/gline/config.yaml`
- **Linux**: `~/.config/gline/config.yaml`
- **Windows**: `%APPDATA%\gline\config.yaml`

### 配置结构

```yaml
# API 配置
api:
  provider: anthropic  # anthropic, openai, openrouter
  model: claude-3-5-sonnet-20241022
  api_key: ""  # 或通过环境变量 GLINE_API_KEY
  base_url: ""  # 可选，用于自定义端点

# 模式配置
mode:
  default: act  # plan 或 act
  separate_models: false  # Plan/Act 使用不同模型

# 自动批准
auto_approve:
  enabled: false  # yolo 模式
  read_only: true  # 自动批准只读操作

# 行为配置
behavior:
  max_consecutive_mistakes: 3
  double_check_completion: false
  auto_condense: false

# 存储
storage:
  data_dir: "~/.gline"
```

### 环境变量

- `GLINE_API_KEY` - API 密钥
- `GLINE_PROVIDER` - 默认提供商
- `GLINE_MODEL` - 默认模型
- `GLINE_CONFIG` - 配置目录覆盖

## 数据库 Schema

```sql
-- 任务历史
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    prompt TEXT NOT NULL,
    mode TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    status TEXT DEFAULT 'running' -- running, completed, failed, aborted
);

-- 消息记录
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    role TEXT NOT NULL, -- system, user, assistant
    content TEXT,
    tool_calls TEXT, -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- 工具调用记录
CREATE TABLE tool_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    input TEXT, -- JSON
    output TEXT,
    error TEXT,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- 全局状态
CREATE TABLE global_state (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 工作区状态
CREATE TABLE workspace_state (
    workspace_path TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_path, key)
);
```

## LLM 提供商接口

### Anthropic (Claude)

```go
const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

type AnthropicProvider struct {
    apiKey  string
    model   string
    baseURL string
}

func (p *AnthropicProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
    // 实现 Anthropic API 调用
}
```

### OpenAI

```go
const openaiAPIURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
    apiKey  string
    model   string
    baseURL string
}
```

### 支持的模型

| 提供商 | 模型 | 工具支持 |
|--------|------|----------|
| Anthropic | claude-3-5-sonnet-20241022 | ✅ |
| Anthropic | claude-3-opus-20240229 | ✅ |
| OpenAI | gpt-4o | ✅ |
| OpenAI | gpt-4-turbo | ✅ |
| OpenRouter | 多种 | 取决于模型 |

## 工具系统

### 工具定义格式

```json
{
  "name": "read_file",
  "description": "Read the contents of a file",
  "input_schema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "The path of the file to read"
      }
    },
    "required": ["path"]
  }
}
```

### 核心工具列表

1. **read_file** - 读取文件内容
2. **write_to_file** - 写入文件
3. **apply_patch** - 应用代码补丁
4. **search_files** - 搜索文件内容
5. **list_code_definition_names** - 列出代码定义
6. **execute_command** - 执行命令
7. **ask_followup_question** - 询问用户
8. **attempt_completion** - 完成任务

## 构建和发布

### Makefile 目标

```makefile
.PHONY: build test lint clean install

build:
	go build -o bin/gline cmd/gline/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

install:
	go install ./cmd/gline

# 交叉编译
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/gline-darwin-amd64 cmd/gline/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/gline-darwin-arm64 cmd/gline/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/gline-linux-amd64 cmd/gline/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/gline-windows-amd64.exe cmd/gline/main.go
```

## 测试策略

### 单元测试

```go
// internal/agent/agent_test.go
func TestAgent_SetMode(t *testing.T) {
    agent := NewAgent()
    agent.SetMode(ModePlan)
    assert.Equal(t, ModePlan, agent.GetMode())
}
```

### 集成测试

```go
// test/integration/task_test.go
func TestTask_Execution(t *testing.T) {
    // 测试完整任务流程
}
```

### Mock 策略

```go
// 使用接口便于 mock
type MockProvider struct {
    mock.Mock
}

func (m *MockProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*MessageResponse), args.Error(1)
}
```
