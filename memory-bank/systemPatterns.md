# System Patterns

## 架构概述

```
┌─────────────────────────────────────────────────────────────┐
│              gline (CLI + GUI 共用入口)                      │
│                    cmd/gline/main.go                         │
│                    (无参数 → GUI, 有参数 → CLI)              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │  GUI 模式 (Wails v3)                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │  Frontend   │  │  internal/  │  │    Config   │ │   │
│  │  │ (Webview)   │  │  gui/*      │  │   (viper)   │ │   │
│  │  │ frontend/   │  │  Services   │  │             │ │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │   │
│  │         │                │                │        │   │
│  │         └────────────────┴────────────────┘        │   │
│  │                          │                         │   │
│  │                   ┌──────┴──────┐                   │   │
│  │                   │  Agent Core │                   │   │
│  │                   │  (internal/)│                   │   │
│  │                   └──────┬──────┘                   │   │
│  └────────────────────────┼──────────────────────────┘   │
│                           │                                │
│    ┌──────────────────────┼──────────────────────┐        │
│    │                      │                      │        │
│    ├────Tools────┬────LLM Providers────┬───Storage───┤   │
│    │  Registry   │                     │  (SQLite)   │   │
│    └─────────────┴─────────────────────┴─────────────┘    │
└─────────────────────────────────────────────────────────────┘

CLI 子命令: gline chat / gline history / gline kb / gline wiki / gline mem
```

## 核心设计模式

### 1. 模块化架构

#### 1.1 GUI 前端模块化 (2026-06-02)

前端代码按 **共享层 → Hooks 层 → 基础组件 → 复合组件 → 入口层** 分层拆分：

```
frontend/src/                    # 前端源码目录（原 desktop/frontend/）
├── theme.ts                    # THEME 常量 + CSS 变量映射 + applyThemeColors()
├── ThemeContext.tsx            # React Context 主题管理 (localStorage 持久化)
├── types.ts                    # Message、AppStatus 等类型
├── utils/
│   └── format.ts               # formatContent、useHighlightCode、代码复制
├── hooks/
│   ├── useChat.ts              # 聊天状态 + 事件监听 + slash/追问
│   ├── useTaskHistory.ts    # 历史任务加载/选择/删除
│   ├── useAppStatus.ts      # mode/status 管理
│   ├── useSettings.ts       # 设置弹窗状态
│   └── useKeyboardShortcuts.ts  # 全局快捷键
├── components/
│   ├── UserMessage.tsx      # 用户消息气泡
│   ├── AssistantMessage.tsx # AI 消息 + streaming 光标
│   ├── ToolMessage.tsx      # 工具调用消息
│   ├── SystemMessage.tsx    # 系统/错误消息
│   ├── Sidebar.tsx          # 侧边栏
│   ├── Header.tsx           # 顶部栏
│   ├── MessageList.tsx      # 消息列表容器（自动滚动、代码高亮）
│   ├── InputArea.tsx        # 输入区（含 SlashMenu）
│   ├── ChatArea.tsx         # 主聊天区域组合
│   ├── SettingsPanel.tsx    # 设置弹窗
│   └── FollowupModal.tsx    # 追问弹窗
└── App.tsx                 # ~93 行入口组合层
```

**设计原则**: 每个文件职责单一；Props 自上而下传递；Hooks 封装业务逻辑；保持 inline style（不引入新样式系统）。

#### 1.2 后端模块化

每个功能模块独立，通过接口通信：

```go
// Agent 模块
package agent

type Agent interface {
    Run(ctx context.Context, prompt string) error
    SetMode(mode Mode)
    Abort()
}

// LLM Provider 模块
package api

type Provider interface {
    CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
    SupportsTools() bool
}

// Tool 模块
package tools

type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, input json.RawMessage) (string, error)
}
```

### 2. Agent 循环模式

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│   Start  │────▶│  System  │────▶│  User    │
└──────────┘     │  Prompt  │     │  Input   │
                 └────┬─────┘     └────┬─────┘
                      │                │
                      └────────┬───────┘
                               │
                          ┌────┴────┐
                          │   LLM   │
                          └────┬────┘
                               │
                    ┌──────────┼──────────┐
                    │          │          │
               ┌────┴───┐ ┌────┴───┐ ┌───┴────┐
               │  Text  │ │  Tool  │ │ Finish │
               │Response│ │  Call  │ │        │
               └────────┘ └───┬────┘ └────────┘
                              │
                         ┌────┴────┐
                         │ Execute │
                         │  Tool   │
                         └────┬────┘
                              │
                              ▼
                         ┌────────┐
                         │ Result │
                         │ (loop) │
                         └────────┘
```

### 3. Plan/Act 模式切换

```go
type Mode string

const (
    ModePlan Mode = "plan"
    ModeAct  Mode = "act"
)

type ModeManager struct {
    currentMode Mode
    toolFilter  map[Mode][]string // 每种模式允许的工具
}

func (m *ModeManager) CanUseTool(mode Mode, toolName string) bool {
    allowedTools := m.toolFilter[mode]
    for _, t := range allowedTools {
        if t == toolName {
            return true
        }
    }
    return false
}
```

### 4. KB (RAG) 与 Wiki 解耦模式（2026-06-05）

**核心原则**: KB 只做本地精确检索（RAG），Wiki 走 LLM 生成。两者完全独立调用。

**架构变更**:
- KB 类型只保留 `rag`，删除 `hybrid`/`wiki`。
- `IngestFile()` → 纯 RAG（chunk → embed → store），无 wiki 副作用。
- `WikiIngestFile()` → 独立显式入口，直接调用 `WikiEngine.IngestAsync()`，强依赖 `e.Caller`（LLM）。

**调用路径对比**:

```
KB/RAG 路径（本地，无 LLM）:
  User: "把这文件加入知识库"
  → KBIngestFile(kbID, filePath)
  → ParseDocument → Chunk → Embed → StoreDocument(RAG DB)
  ✓ Chunk + FTS5 本地搜索即可

Wiki 路径（需 LLM）:
  User: "生成 wiki 笔记"
  → WikiIngestFile(filePath, kbID)
  → ParseDocument → LLM(IngestPrompt) → JSON → Write markdown pages
  要求 e.Caller != nil，否则直接返回 error
```

**前端预留**: GUI `ChatService` 同时暴露 `KBIngestFile()` 和 `WikiIngestFile()`，后续前端可添加独立 Wiki 操作面板。

---

### 5. 工具注册表模式

```go
type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(tool Tool) {
    r.tools[tool.Name()] = tool
}

func (r *Registry) Get(name string) (Tool, error) {
    tool, ok := r.tools[name]
    if !ok {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    return tool, nil
}

func (r *Registry) GetAll() []Tool {
    // 返回所有工具
}
```

### 5. 状态管理模式

```go
type StateManager struct {
    globalState    map[string]interface{}
    workspaceState map[string]interface{}
    sessionState   map[string]interface{} // 非持久化
    db             *sql.DB
}

func (s *StateManager) GetGlobalState(key string) interface{}
func (s *StateManager) SetGlobalState(key string, value interface{})
func (s *StateManager) GetWorkspaceState(key string) interface{}
func (s *StateManager) SetSessionOverride(key string, value interface{}) // 仅当前会话
```

### 6. 消息流模式

```go
type Message struct {
    Role    string          // "system", "user", "assistant"
    Content string
    ToolCalls []ToolCall    // 工具调用
}

type Conversation struct {
    Messages []Message
    MaxTokens int
}

func (c *Conversation) AddMessage(msg Message) {
    c.Messages = append(c.Messages, msg)
    c.trimIfNeeded()
}
```

## 目录结构模式

```
gline/
├── cmd/
│   └── gline/
│       └── main.go              # 入口点，最小化
├── internal/
│   ├── agent/                   # Agent 核心（私有）
│   ├── api/                     # LLM 提供商
│   ├── tools/                   # 工具实现
│   ├── prompts/                 # 提示词管理
│   ├── storage/                 # 状态管理
│   ├── ui/                      # 用户界面
│   │   ├── tui/                 # TUI 实现
│   │   └── plain/               # 纯文本模式
│   └── config/                  # 配置管理
├── pkg/
│   └── types/                   # 共享类型（可导出）
└── memory-bank/                 # 项目文档
```

## 关键决策

### 1. 为什么使用 internal/
- 明确区分公共 API 和内部实现
- 防止外部依赖内部模块
- 便于未来重构

### 2. 为什么分离 UI 层
- 已全面迁移到 Wails GUI 桌面应用（取代 TUI）
- 支持 GUI + CLI 双模式（GUI 日常使用，CLI 脚本自动化）
- 便于测试（可以 mock UI）
- 清晰的关注点分离

### 3. 为什么使用接口定义
- 便于测试（mock 实现）
- 支持多种实现（不同 LLM 提供商）
- 降低模块间耦合

### 4. 状态管理策略
- **Global State**: 用户配置、认证信息（持久化）
- **Workspace State**: 项目特定设置（持久化）
- **Session State**: 临时覆盖（非持久化）

## GUI 架构设计 (Wails v3)

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    gline GUI (Wails v3)                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐      ┌─────────────────────────────┐ │
│  │   Frontend      │◄────►│      Backend (Go)           │ │
│  │  (Webview)      │      │  ┌───────────────────────┐  │ │
│  │  - Chat UI      │      │  │   ChatService         │  │ │
│  │  - Settings     │      │  │   (Wails Service)     │  │ │
│  │  - History      │      │  └───────────┬───────────┘  │ │
│  │  - Markdown     │      │              │              │ │
│  │    renderer     │      │  ┌─────────┴─────────┐    │ │
│  └─────────────────┘      │  │     Agent Core      │    │ │
│         HTML/CSS/JS       │  │  - Plan/Act 模式    │    │ │
│                           │  │  - Tool 调用          │    │ │
│                           │  │  - Conversation     │    │ │
│                           │  └─────────┬───────────┘    │ │
│                           │            │                │ │
│                           │  ┌─────────┼─────────────┐  │ │
│                           │  │         │             │  │ │
│                           │ ┌┴┐      ┌─┴─┐       ┌──┴─┐│ │
│                           │ │LLM│     │Tool│       │Store││
│                           │ │Providers   │Registry   │(SQLite)│
│                           └─┴──┴──────┴───┴───────┴────┘│
└─────────────────────────────────────────────────────────────┘
```

### 目录结构

```
cmd/gline/
├── main.go              # 路由入口：无参数→GUI，有参数→CLI
├── gui.go               # Wails v3 应用初始化 + 窗口配置
├── root.go              # cobra root 命令
├── chat.go              # CLI chat 命令
├── history.go           # CLI history 命令
├── kb.go                # CLI kb 命令
├── wiki.go              # CLI wiki 命令
├── mem.go               # CLI mem 命令
└── frontend/dist/       # 前端构建产物（//go:embed all:frontend/dist）

frontend/                # 前端源码（React 19 + TypeScript + Vite）
├── src/                 # 组件、Hooks、Utils
├── public/styles/       # highlight.js 主题 CSS
├── bindings/            # wails3 generate bindings --ts 输出
└── dist/                # npm run build 产物

build-desktop/           # Wails 构建资产（图标、manifest、各平台配置）
├── windows/
├── macos/
├── linux/
└── android/

internal/
├── agent/               # Agent 核心（GUI/CLI 复用）
├── api/                 # LLM Provider
├── tools/               # 工具系统
├── prompts/             # 提示词 + 自定义规则
├── storage/             # SQLite 持久化
├── config/              # 配置管理
├── memory/              # 四层记忆引擎
├── slash/               # Slash 命令系统
└── gui/                 # Wails Services（chat_service.go, file_service.go, slash_service.go）
```

### 核心分层原则

| 层级 | 职责 | 依赖 | 测试方式 |
|------|------|------|----------|
| Frontend | 用户界面（聊天、设置、历史） | Wails JS Runtime | 浏览器/集成测试 |
| ChatService | Wails Service 绑定，事件转发 | Agent, Storage | Go 单元测试 |
| Backend | 初始化、生命周期管理 | Config, Storage, Agent | Go 单元测试 |
| Agent Core | 业务逻辑、模式、工具调用 | Provider, Tools, Storage | Go 单元测试 |
| Provider | LLM API 通信 | HTTP 客户端 | Go 单元测试 |
| Storage | SQLite 持久化 | modernc.org/sqlite | Go 单元测试 |

### Message 数据结构

```go
type Message struct {
    Role      types.Role           // System | Assistant | User
    Content   string               // 显示内容
    ToolCalls []types.ToolCall     // 工具调用
    Options   []string             // 问题选项
    MsgType   types.MessageType    // 语义类型 (Error/Question/Tool/Normal)
    Strategy  types.RenderStrategy // 渲染策略 (Plain/Markdown/JSON)
    Meta      json.RawMessage     // 结构化元数据 (ErrorMeta, ToolMeta)
    Timestamp time.Time
}
```

### 渲染优先级

ViewModel.renderSystemMessage 的三层渲染优先级：

1. **MsgType (语义类型)**
   - TypeError → ErrorStyle (红色)
   - TypeQuestion → QuestionStyle (带选项)
   - TypeToolStart → ToolRunningStyle (橙色)
   - TypeToolComplete → ToolCompletedStyle (绿色)

2. **Strategy (渲染策略)**
   - StrategyMarkdown → Glamour 渲染
   - StrategyJSON → 代码块
   - StrategyPlain → 纯文本

3. **Fallback (向后兼容)**
   - 字符串前缀检测 (保留用于旧消息兼容)

### 工具自描述渲染

```go
// Renderer 接口让工具自己决定如何渲染
type Renderer interface {
    Render(req RenderRequest) RenderResult
    Name() types.ToolName
    Description() string
    Icon() string
}

// RenderResult 包含渲染决策
type RenderResult struct {
    Content  string               // 显示内容
    Role     types.Role           // System | Assistant
    Strategy types.RenderStrategy // Plain | Markdown
    Skip     bool                 // 是否跳过创建消息
}
```

### 常量定义

**pkg/types/**:
- `message_type.go` - MessageType (TypeError, TypeQuestion, etc.)
- `render_strategy.go` - RenderStrategy (StrategyPlain, StrategyMarkdown, etc.)
- `tool_phases.go` - ToolPhase (ToolPhaseStart, ToolPhaseComplete)
- `tool_names.go` - ToolName (ToolAttemptCompletion, ToolReadFile, etc.)

### 与现有架构的关系

TUI MVVM 是 UI 层内部细化，不影响系统整体架构：
- Agent、Provider、Tools 层保持不变
- 仅 UI 层内部重新组织
- `Agent.StreamCallback` 接口不变，Bridge 层提供新的实现

## 扩展点

### 添加新的消息类型
1. 在 `pkg/types/message_type.go` 添加常量
2. 在 `internal/ui/model/meta.go` 添加对应的 Meta 结构体
3. 在 `viewmodel/conversation_vm.go` 添加渲染逻辑

### 添加新的工具
1. 实现 `tool.Renderer` 接口
2. 在 `tool/registry.go` 注册
3. 工具自动使用自己的渲染策略

### 添加新的 LLM 提供商
1. 实现 `api.Provider` 接口
2. 在 `api/registry.go` 中注册

### 添加新的 Slash 命令
1. 在 `internal/slash/commands.go` 的 `DefaultCommands()` 中定义命令
2. 通过 `CommandContext.OnResult` 与 TUI 交互（使用 `slash.CommandResult`）
3. 在 `NewDefaultRegistry()` 中注册到 `slash.Registry`

---

## Slash 命令架构

### 分层设计

```
┌─────────────────────────────────────────────┐
│              TUI (Bubbletea)                │
│  ┌──────────────────────────────────────┐  │
│  │  Model.Update() → handleKeyMsg()     │  │
│  │    ↓                                 │  │
│  │  executeSlashCommand()               │  │
│  │    ↓                                 │  │
│  │  slashMenu.Registry.Get(name)        │  │
│  │    ↓                                 │  │
│  │  cmd.Handler(args) → OnResult()      │  │
│  │    ↓                                 │  │
│  │  handleSlashCommandResult()          │  │
│  └──────────────────────────────────────┘  │
├─────────────────────────────────────────────┤
│           UI Layer (internal/ui)              │
│  ┌──────────────┐      ┌─────────────────┐  │
│  │ SlashMenuState│     │ SlashMenuState  │  │
│  │  (菜单/导航)   │     │  (过滤/选择)    │  │
│  └──────────────┘      └─────────────────┘  │
├─────────────────────────────────────────────┤
│         Slash Layer (internal/slash)          │
│  ┌──────────────┐      ┌─────────────────┐  │
│  │   Registry   │      │  DefaultCommands │  │
│  │  (命令注册表) │      │  (内置命令定义)  │  │
│  └──────────────┘      └─────────────────┘  │
├─────────────────────────────────────────────┤
│         Domain Layer (model/agent)           │
│  ┌──────────────┐      ┌─────────────────┐  │
│  │ model.Conversation  │  types.Conversation│ │
│  │   (UI 状态)         │   (Agent 状态)     │ │
│  └──────────────┘      └─────────────────┘  │
└─────────────────────────────────────────────┘
```

### 核心交互流程

```
用户输入: "/clear"
    ↓
handleKeyMsg() → IsStandaloneCommand("/clear") == true
    ↓
executeSlashCommand(m, "/clear")
    ↓
slash.ParseCommand("/clear") → ("clear", "")
    ↓
m.slashMenu.Registry.Get("clear") → *SlashCommand
    ↓
cmd.Handler("") —— 调用内置 handler
    ↓
handler 内部: ctx.OnResult(ResultClearScreen, "Conversation cleared")
    ↓
OnResult 闭包: handleSlashCommandResult(m, ResultClearScreen, "...")
    ↓
结果处理:
  - abort 运行中 agent
  - m.conversation.Clear()      ← UI 层
  - m.agentInstance.GetConversation().Clear()  ← Agent 层
  - m.updateViewport()
```

### 关键数据流

**1. 命令定义 → 结果通知 → UI 响应**
```go
// internal/slash/commands.go
func DefaultCommands(ctx *CommandContext) []*types.SlashCommand {
    return []*types.SlashCommand{
        {
            Name: "clear",
            Handler: func(args string) (bool, error) {
                ctx.Conversation.Clear()
                ctx.OnResult(ResultClearScreen, "Conversation cleared")
                return true, nil
            },
        },
    }
}

// internal/ui/tui.go — New() 中建立连接
m.slashMenu = NewSlashMenuState(slash.NewDefaultRegistry(conv, 
    func(result slash.CommandResult, message string) {
        handleSlashCommandResult(m, result, message)
    }))
```

**2. 双 Conversation 同步**

Slash 命令影响两个独立但关联的 conversation：

| 层级 | 类型 | 职责 | 影响命令 |
|------|------|------|----------|
| UI 层 | `model.Conversation` | 用户可见消息、工具历史、渲染缓存 | `/clear`, `/newtask` |
| Agent 层 | `types.Conversation` | LLM 请求消息、token 预算管理 | `/compact` |

命令执行时必须同步更新两层，避免 UI 显示和 Agent 实际状态不一致。

### Cline 参考设计

对比 Cline CLI (TypeScript) 的 slash 命令处理：

**Cline 三层分类**:
1. **local execution** (`execution: "local"`) — TUI 本地处理
   - `/help`, `/exit`, `/settings`, `/clear`, `/q`
   - 直接修改 TUI 状态，不发送到后端
   
2. **runtime execution** (`execution: "runtime"`) — 发送到后端处理
   - `/newtask` → 在后端转换为 `<new_task>` 工具调用
   - `/smol`, `/compact` → 在后端运行 condense 逻辑
   
3. **user-command execution** (`execution: "user-command"`) — 展开为提示词注入
   - `/workflow-name` → 展开为 `<explicit_instructions>` 块
   - `/skill-name` → 展开为对应的系统提示词

**Cline 的 `parseSlashCommands()`**:
```typescript
// src/core/slash-commands/index.ts
export async function parseSlashCommands(text, ...): Promise<...> {
    // 1. 在 XML 标签内查找 /command
    // 2. 默认命令 → 替换为对应的工具响应提示词
    // 3. MCP 命令 → 获取 MCP prompt 内容
    // 4. Workflow 命令 → 读取工作流文件内容
    // 5. 返回处理后的文本 + 是否需要检查 clinerules
}
```

### gline 当前实现

**已实现** (与 Cline `execution: "local"` 等价):
- `/clear` — 清空对话 (UI + Agent 双清空)
- `/exit`, `/q` — 退出程序
- `/help` — 显示帮助信息
- `/newtask` — 开始新任务 (UI + Agent 双清空)
- `/smol`, `/compact` — 压缩上下文 (调用 `TrimToMaxTokens()`)

**待实现** (Cline 的 `execution: "runtime"/"user-command"`):
- `/deep-planning` — 转换为深度规划提示词
- `/explain-changes` — 需要 git diff 支持
- `/reportbug` — 需要 GitHub issue 集成
- Workflow/skill 命令 — 需要工作流系统

---

## 原始架构模式

### 模块化架构
