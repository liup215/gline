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

## 构建与 CI 注意事项

### 本地构建

必须使用 `build-all.ps1` 或等效脚本完成完整构建，禁止单独使用 `go build`（会因为缺少 embedded frontend/dist 而失败）：

1. 从 `cmd/gline` 运行 `wails3 generate bindings --ts -d "../../frontend/bindings"`
2. `npm run build`（生成 `cmd/gline/frontend/dist/`）
3. `go build`（最终生成 `bin/gline.exe`）

### CI（GitHub Actions）失败教训

- `.github/workflows/build.yml` 使用 `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95`
- 本地二进制显示 `v3.0.0-alpha2.106`，需关注版本差异是否引入 binding 差异
- 关键规则：**Wails 只会把注册到 `application.NewService()` 的 struct 方法生成 bindings**。在 `Backend` 上定义的方法即使被前端 import 也不会生成绑定。
- 因此 `internal/gui/chat_service.go` 的 `ChatService` 必须暴露前端需要的所有方法（如 `GetMCPStatus`）。

### CI 前端构建权限问题

Ubuntu 24.04 runner 通过 apt 安装 npm 后，`node_modules/.bin/vite` 可能没有可执行权限，导致 `npm run build` 报 `sh: 1: vite: Permission denied`（exit code 127）。

修复：在 `.github/workflows/build.yml` 中把前端构建命令从 `cd frontend && npm run build` 改为：
```bash
cd frontend && node ./node_modules/typescript/bin/tsc && node ./node_modules/vite/bin/vite.js build --mode production
```
直接通过 `node` 执行 TypeScript 和 Vite 的 JS 入口，绕过 apt 安装的 npm 在 Ubuntu 24.04 runner 上创建的 `.bin` 脚本无执行权限的问题。

### MCP 运行时陷阱

1. **Transport context 不能复用初始化 context**，否则 60 秒后 `context canceled`。详见 `memory-bank/mcp-design.md`。
2. **状态统计不要重复请求工具列表**：`Manager` 应在 `registerServerTools()` 阶段缓存工具列表。

## 项目结构

```
gline/
├── cmd/
│   └── gline/                   # 唯一入口：CLI + GUI 共用
│       ├── main.go              # 路由入口（无参数→GUI，有参数→CLI）
│       ├── root.go              # cobra root 命令
│       ├── gui.go               # Wails v3 GUI 初始化 + 启动
│       ├── chat.go              # `gline chat` CLI 命令
│       ├── history.go           # `gline history` CLI 命令
│       ├── kb.go                # `gline kb` CLI 命令
│       ├── wiki.go              # `gline wiki` CLI 命令
│       └── mem.go               # `gline mem` CLI 命令
│       └── frontend/            # Embed 引用的前端构建产物（//go:embed all:frontend/dist）
│           └── dist/            # npm run build 输出，非源码，不提交
├── internal/
│   ├── agent/                   # Agent 核心逻辑
│   ├── api/                     # LLM 提供商
│   ├── tools/                   # 工具系统
│   ├── prompts/                 # 提示词管理
│   ├── storage/                 # 数据持久化
│   ├── config/                  # 配置管理
│   ├── memory/                  # 四层记忆引擎
│   ├── log/                     # 日志系统
│   ├── slash/                   # Slash 命令系统
│   └── gui/                     # Wails Services（供 cmd/gline/gui.go 注册）
│       ├── chat_service.go
│       ├── file_service.go
│       └── slash_service.go
├── pkg/
│   └── types/                   # 共享类型
├── frontend/                    # 前端源码（移动自 desktop/frontend/）
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/                     # React 19 + TypeScript 源码
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── components/
│   │   ├── hooks/
│   │   └── utils/
│   ├── public/styles/           # highlight.js 主题 CSS
│   └── bindings/                # wails3 generate bindings --ts 输出
│       └── github.com/
├── build-all.sh                 # 统一构建脚本（macOS/Linux）
├── build-all.ps1                # 统一构建脚本（Windows）
│   ├── windows/
│   ├── macos/
│   ├── linux/
│   └── android/
├── build-all.ps1                # 一键构建脚本（→ bin/gline）
├── Makefile                     # bindings / dev 快捷目标
├── go.mod
└── README.md
```

### 关键目录变更历史

| 时间 | 变更 | 说明 |
|------|------|------|
| 2026-06-07 | `desktop/` 删除 | 旧独立 Wails 项目残留，前端移至 `frontend/` |
| 2026-06-07 | `build-desktop/` 删除 | Wails 构建模板未实际使用，由 `build-all.sh`/`build-all.ps1` 负责构建流程 |
| 2026-06-07 | 入口统一 | 主入口从 `gui/main.go` 改为 `cmd/gline/main.go`（CLI/GUI 共用） |
| 2026-06-07 | `gui/` 删除 | `gui/*.go` 移至 `internal/gui/`，由 `cmd/gline/gui.go` 注册 |

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

# Memory 配置
memory:
  embedding:
    provider: openai
    model: text-embedding-3-small
```

---

## KB (RAG) 与 Wiki 解耦技术说明

### 调用方式对比

| 维度 | KB (RAG) | Wiki |
|------|----------|------|
| **依赖** | 纯本地（SQLite + Go 内存） | 强依赖 LLM |
| **入口** | `IngestFile(kbID, filePath)` | `WikiIngestFile(filePath, kbID)` |
| **流程** | Parse → Chunk → Embed → Store(RAG DB) | Parse → LLM(IngestPrompt) → JSON → Write markdown |
| **触发** | 用户显式调用（工具/CLI/GUI） | 用户显式调用（独立 API） |
| **失败行为** | 同步返回 error | Caller nil 时立即返回 error |
| **搜索** | 向量相似度 + FTS5 + RRF | 关键词扫描 + 可选 LLM rerank |

### PDF 解析器

- **库**: `github.com/tsawler/tabula` (MIT, 纯 Go, 零 CGO)
- **旧库**: `github.com/ledongthuc/pdf` — 对嵌入字体、CJK 中文、复杂编码支持不足，提取可能返回二进制乱码。
- **处理格式**: `.pdf` / `.odt` / `.epub`（通过 tabula 提取）；`.docx`/`.xlsx`/`.pptx`/`.html` 保留原有解析器。
- **API**: `tabula.Open(path).ExcludeHeadersAndFooters().Text()` — 自动识别格式并提取正文，对扫描件缺少文本层时返回 warnings（不上传二进制乱码）。

### 前端 paths 说明

前端源码在 `frontend/`，bindings 生成到 `frontend/bindings/`（TypeScript 类型）。

前端引用 bindings 的路径示例：
```typescript
import {
  ChatService,
  Message,
  // ...
} from "../../bindings/github.com/liup215/gline/internal/gui";
```

### 前端调用示例

```typescript
// RAG 知识库加入
await KBIngestFile(kbNameOrID, filePath);

// Wiki 生成（独立操作，需 LLM 配置正确）
await WikiIngestFile(filePath, kbID);
```

### 后端调用示例

```go
// 只做 RAG
go engine.IngestFile(ctx, kb.ID, filePath)

// 只做 Wiki（必须 e.Caller != nil，否则错误）
go engine.WikiIngestFile(ctx, filePath, kb.ID)
```

### KB 类型变更

- `KBTypeRAG` ✅（唯一支持类型）
- `KBTypeWiki` ❌（已删除）
- `KBTypeHybrid` ❌（已删除）

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

### replace_in_file 工具

`replace_in_file` 支持两种调用风格，多 block 模式（`replacements` 数组）优于单 block 模式：

```json
// 单 block（向后兼容）
{
  "path": "src/main.go",
  "search": "oldFunc()",
  "replace": "newFunc()"
}

// 多 block（推荐：单次调用完成多个编辑）
{
  "path": "src/main.go",
  "replacements": [
    {"search": "oldFunc()", "replace": "newFunc()"},
    {"search": "const X = 1", "replace": "const X = 2"}
  ]
}
```

**5 层容错回退**:
1. 精确字符串匹配 (`strings.Contains`)
2. 空格归一化匹配（压缩空白符后比较）
3. 行锚定回退（最长行作为锚点，上下文窗口验证）
4. 失败时返回 `Jaccard bigram` 最近匹配 + 相似度分数 + 排查指南
5. 成功时返回 ```` ```diff ```` 格式修改摘要

---

### 正确的一键构建流程

本项目**不是标准 Wails 项目**，`wails3 build` **不可直接使用**。正确流程：

1. **生成 TypeScript bindings**（必须在 `cmd/gline` 目录执行才能扫描到 Service）
   ```bash
   cd cmd/gline && wails3 generate bindings --ts -d "../../frontend/bindings"
   ```
   - 在根目录执行会报 "0 Services"（找不到 `internal/gui` 的 `ChatService`）
   - 在 `cmd/gline` 执行才能正确识别 1 Service / 34 Methods / 21 Models

2. **构建前端**（npm 常规构建）
   ```bash
   cd frontend && npm install && npm run build
   ```

3. **同步产物到 embed 目录**
   ```bash
   # build-all.ps1 自动执行
   New-Item -ItemType Directory -Force -Path "cmd/gline/frontend/dist"
   Copy-Item -Recurse -Force "frontend/dist/*" "cmd/gline/frontend/dist/"
   ```

4. **编译 Go 二进制**
   ```bash
   go build -o bin/gline ./cmd/gline
   ```

### 构建脚本

- **Windows**: `build-all.ps1`（项目根目录执行）
  - 检测 Node.js / wails3 CLI
  - bindings 生成（自动切换目录）
  - 前端构建 + dist 同步
  - Go 编译 → `bin\gline.exe`（`-H=windowsgui`）

- **macOS / Linux**: `build-all.sh`
  - 相同流程，但 Go build 不添加 `-H=windowsgui`
  - macOS 保留默认 `CGO_ENABLED=1`（Wails v3 WebKit 需要）
  - 用法：`./build-all.sh [-d|--dev] [-s|--skip-bindings] [-o <output>]`

### Makefile 目标

```makefile
.PHONY: build test lint clean install

BINDINGS_DIR := frontend/bindings
FRONTEND_DIR := frontend
EMBED_DIR := cmd/gline/frontend/dist

.PHONY: bindings build test lint clean install

bindings:
	cd cmd/gline && wails3 generate bindings --ts -d "../../$(BINDINGS_DIR)"

build:
	go build -o bin/gline ./cmd/gline

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ $(EMBED_DIR)

install:
	go install ./cmd/gline

# 一键构建（前端 → 产物复制 → Go 编译）
# 推荐直接使用 build-all.ps1 (PowerShell)
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
2. **build** matrix (macos-latest, windows-latest) — `npm install` → `npm run build` → 复制产物到 `cmd/gline/frontend/dist` → `go build ./cmd/gline` → 上传裸二进制 artifact
3. **build-summary** (仅 PR) — 汇总 artifact 列表到 GitHub Step Summary
4. **release** (仅 tag `v*` 触发) — 下载 artifacts → 创建 GitHub Release + changelog + 安装说明
5. **snapshot** (仅 main/master push) — 创建/更新 `snapshot` tag 预发布版本

**关键注意事项**:
- GUI 应用通过 `//go:embed all:frontend/dist` 嵌入前端静态资源。
- **不可直接使用 `wails3 build`**（非标准 Wails 项目结构），CI 拆分为 `npm build` + `go build`。
- Linux 构建已从 CI 矩阵中移除（runner 稀缺），依赖安装复杂。Linux 用户可从源码构建。
- `CGO_ENABLED=0` 确保纯 Go 交叉编译（零 CGO 依赖）。
- 唯一入口是 `cmd/gline/main.go`（无参数→GUI，有参数→CLI）。
