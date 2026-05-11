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

## 扩展点

### 添加新的 LLM 提供商
1. 实现 `api.Provider` 接口
2. 在 `api/registry.go` 中注册

### 添加新的工具
1. 实现 `tools.Tool` 接口
2. 在 `tools/registry.go` 中注册
3. 更新系统提示词

### 添加新的 UI 模式
1. 实现 UI 接口
2. 在 `ui/` 下创建新包

## TUI MVVM 架构模式（规划中）

### 背景

当前 TUI 层（`internal/ui/`）采用传统 Bubbletea 架构，所有状态、视图渲染和 Agent 交互集中在单一的 `Model` 结构体中。随着功能增加，出现了以下问题：
- 上帝对象：Model 混合 UI 状态、业务状态、Agent 胶水代码
- 类型不安全：`agentUpdateMsg.updateType` 使用魔法字符串
- 渲染效率低：每次状态变更后全量重建消息列表
- 测试困难：无法脱离 Bubbletea 框架进行单元测试

### 目标架构：MVVM + Bridge

```
internal/ui/
├── model/         # Domain Model（纯数据，零外部依赖）
│   ├── conversation.go   # 对话：messages、toolHistory、mode
│   └── message.go        # 消息：Role、Content、ToolCalls、渲染缓存
│
├── viewmodel/     # ViewModel（派生展示状态）
│   ├── conversation_vm.go  # 格式化消息列表、滚动状态
│   ├── status_vm.go        # 状态栏信息
│   └── input_vm.go         # 输入框状态
│
├── view/          # View（纯渲染函数）
│   ├── messages.go      # 消息列表渲染
│   ├── header.go        # 标题栏
│   ├── input.go         # 输入框
│   ├── status_bar.go    # 状态栏
│   ├── tool_area.go     # 工具状态区域
│   └── styles.go        # Lipgloss 样式定义
│
├── bridge/        # Agent-TUI 桥接层
│   ├── callback.go      # 类型安全的事件发送
│   └── messages.go      # AgentEvent 接口 + 具体事件类型
│
└── tui.go         # Bubbletea 薄壳
    # 仅组合以上各层，处理 tea.Msg 分发
```

### 核心分层原则

| 层级 | 职责 | 依赖 | 测试方式 |
|------|------|------|----------|
| Model | 纯数据、业务规则 | 无外部依赖 | 纯 Go 单元测试 |
| ViewModel | 派生状态、格式化 | Model | 纯 Go 单元测试 |
| View | 纯函数：输入 ViewModel → 输出 string | ViewModel, lipgloss | 纯 Go 单元测试 |
| Bridge | Agent 回调 → TUI 事件转换 | Agent 接口 | 纯 Go 单元测试（mock Agent） |
| TUI Shell | Bubbletea 生命周期 | 以上全部 | 集成测试 |

### 当前状态

- **空目录已预留**: `internal/ui/model/`, `internal/ui/viewmodel/`, `internal/ui/view/`, `internal/ui/agent/`, `internal/ui/core/`（后两者废弃，将改为 `bridge/`）
- **当前代码**: 10 个文件分散在 `internal/ui/` 根目录
- **重构计划**: 5 阶段渐进方案（详见 `memory-bank/tui-mvvm-refactor.md`）

### 与现有架构的关系

MVVM 架构是 UI 层内部的细化，不影响系统整体架构：
- Agent、Provider、Tools 层保持不变
- 仅 UI 层内部重新组织
- `Agent.StreamCallback` 接口不变，Bridge 层提供新的实现
