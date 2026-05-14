# System Patterns

## 架构概述

```
┌─────────────────────────────────────────────────────────────┐
│                         gline CLI                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Command   │  │     UI      │  │      Config       │  │
│  │   Layer     │  │   Layer     │  │      Layer        │  │
│  │  (cobra)    │  │ (bubbletea) │  │     (viper)       │  │
│  └──────┬──────┘  └──────┬──────┘  └─────────┬───────────┘  │
│         │                │                   │              │
│         └────────────────┴───────────────────┘              │
│                          │                                  │
│                   ┌──────┴──────┐                          │
│                   │   Agent     │                           │
│                   │   Core      │                           │
│                   └──────┬──────┘                           │
│                          │                                  │
│         ┌────────────────┼────────────────┐              │
│         │                │                │              │
│    ┌────┴────┐    ┌─────┴─────┐    ┌────┴────┐        │
│    │  Tools  │    │   LLM     │    │ Storage │        │
│    │Registry │    │ Providers │    │  Layer  │        │
│    └─────────┘    └───────────┘    └─────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## 核心设计模式

### 1. 模块化架构

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

### 4. 工具注册表模式

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
- 支持多种 UI 模式（TUI、纯文本）
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

## TUI 架构设计

### 整体架构（MVVM + Bridge）

```
┌─────────────────────────────────────────────────────────────┐
│                         TUI Layer                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │    Model    │  │  ViewModel  │  │   tool.Registry   │ │
│  │  (纯数据)   │  │ (派生状态)  │  │  (工具渲染器注册表) │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                │                    │            │
│         └────────────────┴────────────────────┘            │
│                          │                                  │
│                   ┌──────┴──────┐                          │
│                   │   Bridge    │                          │
│                   │  (事件转换)  │                          │
│                   └──────┬──────┘                          │
│                          │                                  │
│         ┌────────────────┼────────────────┐                │
│         │                │                │                │
│    ┌────┴────┐    ┌─────┴─────┐    ┌────┴────┐           │
│    │  Agent  │    │   LLM     │    │  Tools  │           │
│    │  Core   │    │ Providers │    │ Registry│           │
│    └─────────┘    └───────────┘    └─────────┘           │
└─────────────────────────────────────────────────────────────┘
```

### 目录结构

```
internal/ui/
├── model/              # Domain Model（纯数据，零外部依赖）
│   ├── message.go      # Message: Role, Content, MsgType, Strategy, Meta
│   ├── meta.go         # ErrorMeta, ToolMeta + 辅助方法
│   └── conversation.go # Conversation: Messages, ToolHistory
│
├── viewmodel/          # ViewModel（派生展示状态 + 渲染缓存）
│   └── conversation_vm.go  # 格式化消息列表，增量渲染
│
├── view/               # View（纯渲染函数）
│   ├── tool_area.go    # 工具状态区域渲染
│   └── styles.go       # Lipgloss 样式定义
│
├── tool/               # 工具自描述渲染
│   ├── render.go       # Renderer 接口
│   ├── registry.go     # 工具注册表
│   ├── attempt_completion.go
│   ├── ask_followup_question.go
│   ├── plan_mode_respond.go
│   ├── read_file.go
│   └── default.go      # 通用工具默认渲染器
│
├── bridge/             # Agent-TUI 桥接层（类型安全）
│   ├── callback.go     # TUIBridge 实现 StreamCallback
│   └── messages.go     # AgentEvent 接口定义
│
└── tui.go              # Bubbletea 薄壳（Init/Update/View）
```

### 核心分层原则

| 层级 | 职责 | 依赖 | 测试方式 |
|------|------|------|----------|
| Model | 纯数据、业务规则 | 无外部依赖 | 纯 Go 单元测试 |
| ViewModel | 派生状态、格式化 | Model | 纯 Go 单元测试 |
| View | 纯函数渲染 | ViewModel, lipgloss | 纯 Go 单元测试 |
| Bridge | Agent 回调 → TUI 事件 | Agent 接口 | 纯 Go 单元测试（mock Agent）|
| TUI Shell | Bubbletea 生命周期 | 以上全部 | 集成测试 |

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

---

## 原始架构模式

### 模块化架构
