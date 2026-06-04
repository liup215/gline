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

## 前端技术栈

| 技术 | 用途 | 说明 |
|------|------|------|
| React 19 | UI 框架 | 函数组件 + Hooks |
| TypeScript | 类型安全 | React.RefObject 与 MutableRefObject 兼容性需注意 |
| Vite | 构建工具 | wails3 serve / npm run build |
| Marked | Markdown 渲染 | 自定义 token 渲染器支持代码块复制、脚注 |
| KaTeX | 数学公式 | 行内 `$$...$$` 和块级 `$$...$$` |
| Highlight.js | 代码高亮 | 自动语言检测 + 常见语言支持；动态切换 github-dark/light 主题 |
| Inline Style | CSS-in-JS | 所有组件使用 `style={{...}}`，无 CSS 模块 |
| CSS Variables | 主题系统 | 28+ 个 CSS 自定义属性，ThemeContext + localStorage 持久化 |

## 项目结构

```
gline/
├── cmd/
│   └── gline/                   # CLI 命令入口（保留子命令）
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
│   │   └── rules.go             # 自定义规则加载
│   ├── storage/                 # 数据持久化
│   │   ├── store.go
│   │   ├── database.go
│   │   ├── sqlite.go
│   │   └── history.go
│   ├── config/                  # 配置管理
│   │   └── config.go
│   ├── log/                     # 日志系统
│   ├── slash/                   # Slash 命令系统
│   │   ├── commands.go
│   │   ├── parser.go
│   │   └── registry.go
│   └── version/                 # 版本信息
├── pkg/
│   └── types/                   # 共享类型
│       ├── message.go
│       ├── message_type.go
│       ├── render_strategy.go
│       ├── slash_command.go
│       ├── tool_names.go
│       └── tool_phases.go
├── gui/                         # Wails v3 GUI 应用（主入口）
│   ├── main.go                  # Wails 应用入口
│   ├── backend.go               # Backend 初始化
│   ├── chat_service.go          # ChatService（Wails Service，暴露给前端）
│   ├── file_service.go          # FileService（@ 引用 + 目录浏览）
│   ├── slash_service.go         # SlashCommand service（Wails bridge）
│   ├── frontend/                # 前端资源 (Vite + React + TypeScript)
│   │   ├── src/
│   │   │   ├── theme.ts         # 主题色板 + CSS 变量
│   │   │   ├── ThemeContext.tsx # 主题 Context + localStorage
│   │   │   ├── main.tsx         # 入口（ThemeProvider 包裹 App）
│   │   │   ├── App.tsx
│   │   │   ├── components/      # 18+ UI 组件
│   │   │   ├── hooks/           # 业务逻辑 Hooks
│   │   │   └── utils/           # 格式化工具 + 测试
│   │   ├── public/styles/       # highlight.js 主题 CSS
│   │   └── index.html           # FOUC prevention script
│   ├── build/                   # 构建脚本（Taskfile 按 OS 分发）
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

---

## CI/CD (GitHub Actions)

**触发条件**: `push` 到 `main/master`、PR、tag `v*`

**工作流文件**: `.github/workflows/build.yml` (统一工作流)

**构建流程**:
1. **test** job (ubuntu-24.04) — `go test -v ./...` + `go vet ./...`
2. **build** matrix (macos-latest, windows-latest) — 安装 wails3 CLI → `cd gui && wails3 build` → 上传裸二进制 artifact
3. **build-summary** (仅 PR) — 汇总 artifact 列表到 GitHub Step Summary
4. **release** (仅 tag `v*` 触发) — 下载 artifacts → 创建 GitHub Release + changelog + 安装说明
5. **snapshot** (仅 main/master push) — 创建/更新 `snapshot` tag 预发布版本

**关键注意事项**:
- GUI 应用通过 `//go:embed all:frontend/dist` 嵌入前端静态资源；wails3 build 自动集成前端编译。
- Linux 构建已从 CI 矩阵中移除（runner 稀缺），依赖安装复杂。Linux 用户可从源码构建。
- `CGO_ENABLED=0` 确保纯 Go 交叉编译（零 CGO 依赖）。
- CLI 入口（`cmd/gline`）已废弃，当前主入口为 `gui/` 目录下的 Wails 应用。
