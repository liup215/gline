# TUI MVVM 架构重构方案

## 1. 当前架构分析

### 1.1 文件结构

```
internal/ui/
├── tui.go              # 核心: Model 结构体、Bubbletea 生命周期 (321行)
├── tui_agent.go        # Agent 集成: tuiCallback、startAgent() (109行)
├── tui_state.go        # 状态变更: 6个 handler 函数 (236行)
├── tui_update.go       # 分发器: handleAgentUpdate switch (31行)
├── tui_view.go         # 视图渲染: updateViewport/renderToolArea/renderStatusBar (111行)
├── tui_view_render.go  # 消息渲染: renderMessageHeader/renderAssistantContent/renderToolCalls (122行)
├── tui_helpers.go      # 工具函数: renderMarkdown/formatToolCallsInline (80行)
├── tui_styles.go       # Lipgloss 样式 + 工具描述映射 (190行)
├── tui_input.go        # 键盘输入处理 (127行)
├── tui_test.go         # 测试
├── model/              # 空目录 (预留)
├── viewmodel/          # 空目录 (预留)
├── agent/              # 空目录 (预留，废弃)
└── core/               # 空目录 (预留，废弃)
```

### 1.2 Model 结构体分析（上帝对象）

```go
type Model struct {
    // --- UI 组件状态 ---
    viewport     viewport.Model      // Bubbletea viewport
    textarea     textarea.Model      // Bubbletea textarea
    spinner      spinner.Model       // Bubbletea spinner
    inputHeight  int                 // UI 布局
    toolAreaHeight int               // UI 布局
    width        int                 // 终端尺寸
    height       int                 // 终端尺寸

    // --- 业务状态 ---
    messages            []Message    // 对话消息
    mode                agent.Mode   // Plan/Act
    provider            string       // Provider 名称
    model               string       // 模型名称
    toolHistory         []ToolStatus // 工具执行历史
    currentTool         string       // 当前执行的工具
    activeAssistantIndex int        // 当前流式消息的索引
    isProcessing        bool         // 是否正在处理
    isStreaming         bool         // 是否正在流式输出
    err                 error        // 当前错误

    // --- Agent 胶水代码 ---
    agentInstance *agent.BaseAgent   // Agent 实例
    ctx           context.Context    // 上下文
    cancelFn      context.CancelFunc // 取消函数
    program       *tea.Program       // Bubbletea 程序引用
    pendingReply  chan string        // 问答同步通道

    // --- 渲染缓存 ---
    renderer          *glamour.TermRenderer
    rendererWrapWidth int
}
```

**问题**: 一个结构体混合了 3 层关注点，违反了单一职责原则。

### 1.3 魔法字符串问题

```go
// tui_update.go
switch msg.updateType {
case "content":       // 魔法字符串
case "toolStart":     // 魔法字符串
case "toolComplete":  // 魔法字符串
case "error":         // 魔法字符串
case "complete":      // 魔法字符串
case "streamStart":   // 魔法字符串
case "streamEnd":     // 魔法字符串
}
```

**问题**: 无编译时检查，拼写错误只能在运行时发现。

### 1.4 View 和 State 紧耦合

```go
// tui_state.go: handleAgentContent
func handleAgentContent(m *Model, msg agentUpdateMsg) []tea.Cmd {
    m.messages[m.activeAssistantIndex].Content += msg.content
    m.updateViewport()  // ← 状态变更后立即渲染
    return nil
}
```

**问题**: 每个状态变更函数都直接调用 `updateViewport()`，无法独立测试状态变更逻辑。

### 1.5 Agent 回调直接耦合 TUI

```go
// tui_agent.go
type tuiCallback struct {
    program *tea.Program  // ← 直接依赖 Bubbletea
}

func (c *tuiCallback) OnContent(delta string) {
    c.program.Send(agentUpdateMsg{updateType: "content", content: delta})
}
```

**问题**: `tuiCallback` 无法脱离 `tea.Program` 进行测试。

### 1.6 全量 Viewport 重建

```go
// tui_view.go: updateViewport
func (m *Model) updateViewport() {
    var content strings.Builder
    for i := range m.messages {  // ← 遍历所有消息
        // 重新渲染每条消息
    }
    m.viewport.SetContent(content.String())
}
```

**问题**: 每次收到一个字符都要遍历所有消息重新渲染，时间复杂度 O(n)。

---

## 2. 优化目标

### 2.1 目标架构

```
internal/ui/
├── model/              # Domain Model（纯数据，零外部依赖）
│   ├── conversation.go
│   ├── message.go
│   └── model_test.go
│
├── viewmodel/          # ViewModel（派生展示状态 + 命令）
│   ├── conversation_vm.go
│   ├── status_vm.go
│   ├── input_vm.go
│   └── viewmodel_test.go
│
├── view/               # View（纯渲染函数）
│   ├── messages.go
│   ├── header.go
│   ├── input.go
│   ├── status_bar.go
│   ├── tool_area.go
│   ├── styles.go
│   └── view_test.go
│
├── bridge/             # Agent-TUI 桥接层
│   ├── callback.go
│   ├── messages.go
│   └── bridge_test.go
│
├── tui.go              # Bubbletea 薄壳
└── tui_test.go
```

### 2.2 各层职责

| 层级 | 职责 | 外部依赖 | 测试策略 |
|------|------|----------|----------|
| **Model** | 纯数据结构、业务规则、状态变更方法 | 仅标准库 + `pkg/types` | 纯 Go 单元测试 |
| **ViewModel** | 从 Model 派生展示状态、渲染缓存管理、脏标记 | Model + lipgloss/glamour | 纯 Go 单元测试 |
| **View** | 纯函数：输入 ViewModel → 输出渲染字符串 | ViewModel + lipgloss | 纯 Go 单元测试 |
| **Bridge** | Agent 回调 → 类型安全事件转换 | Agent 接口 | 纯 Go 单元测试（mock Agent） |
| **TUI Shell** | Bubbletea 生命周期管理、消息分发 | 以上全部 | 集成测试 |

### 2.3 核心优化点

| 问题 | 当前状态 | 优化后 |
|------|----------|--------|
| 可测试代码比例 | ~10% | ~80% |
| 新增消息类型改动文件数 | 4-5 个 | 2-3 个 |
| View 渲染性能（100条消息） | O(n) 全量重建 | O(1) 增量更新 |
| Agent 回调脱离 TUI 测试 | ❌ | ✅ |
| 添加新 UI 模式（Web UI） | 大量改动 | 只需实现 Bridge |

---

## 3. 渐进优化方案（5 个 Phase）

### Phase 1: 类型安全的消息系统

**目标**: 消除魔法字符串，不改变现有行为

**风险**: ⭐（低风险，纯增量）

**步骤**:
1. 创建 `bridge/messages.go`
2. 定义 `AgentEvent` 接口和具体事件类型
3. 替换 `agentUpdateMsg` 为 `AgentEvent`
4. `handleAgentUpdate` 改为 type switch

```go
// bridge/messages.go
package bridge

type AgentEvent interface { agentEvent() }

type ContentEvent struct { Delta string }
type ToolStartEvent struct { Name string; Input string }
type ToolCompleteEvent struct { Name string; Result string }
type ErrorEvent struct { Err error }
type StreamStartEvent struct{}
type StreamEndEvent struct{}
type CompleteEvent struct{}

func (ContentEvent) agentEvent()      {}
func (ToolStartEvent) agentEvent()    {}
func (ToolCompleteEvent) agentEvent() {}
func (ErrorEvent) agentEvent()        {}
func (StreamStartEvent) agentEvent()  {}
func (StreamEndEvent) agentEvent()    {}
func (CompleteEvent) agentEvent()     {}
```

**验证**: 所有现有测试无变化通过

---

### Phase 2: 抽离纯数据 Model

**目标**: 将 `Model` 中的纯数据字段提取到 `model/` 包

**风险**: ⭐（低风险，纯提取）

**步骤**:
1. 创建 `model/message.go`:
   ```go
   type Message struct {
       Role      types.Role
       Content   string
       ToolCalls []types.ToolCall
       Options   []string
       Timestamp time.Time
       // 渲染缓存移到 ViewModel
   }

   type ToolStatus struct {
       Name      string
       Status    string
       StartTime time.Time
   }
   ```

2. 创建 `model/conversation.go`:
   ```go
   type Conversation struct {
       Messages    []Message
       ToolHistory []ToolStatus
       Mode        agent.Mode
       Provider    string
       ModelName   string
       // 业务方法
   }

   func (c *Conversation) AppendUserMessage(content string)
   func (c *Conversation) AppendAssistantMessage(content string)
   func (c *Conversation) AppendSystemMessage(content string)
   func (c *Conversation) UpdateLastAssistantContent(delta string)
   func (c *Conversation) AddToolStart(name, input string)
   func (c *Conversation) AddToolComplete(name, result string)
   func (c *Conversation) MarkToolFailed(name string)
   func (c *Conversation) Clear()
   ```

3. `Model` 结构体改为:
   ```go
   type Model struct {
       conversation *model.Conversation  // ← 替换 messages, toolHistory, mode 等
       // ... 保留 UI 组件和尺寸
   }
   ```

4. 所有状态变更函数改为调用 `conversation` 方法

**验证**: 添加 `model/` 包单元测试，覆盖率 > 80%

---

### Phase 3: 引入 ViewModel 层

**目标**: 将展示逻辑从 View 中分离到 ViewModel

**风险**: ⭐⭐（中等风险，行为等价）

**步骤**:
1. 创建 `viewmodel/conversation_vm.go`:
   ```go
   type FormattedMessage struct {
       Role           types.Role
       Header         string      // 预渲染的头部 (Author + Timestamp)
       Body           string      // 预渲染的内容 (markdown + 样式)
       IsQuestion     bool        // 是否为问答消息
       Options        []string    // 选项列表
       IsStreaming    bool        // 是否正在流式输出
   }

   type ConversationViewModel struct {
       Messages      []FormattedMessage
       ScrollToBottom bool
       dirty         bool  // 脏标记
   }

   func (vm *ConversationViewModel) Refresh(conv *model.Conversation, width int) {
       // 增量更新：只重新渲染变化的消息
       // 使用脏标记避免不必要的重建
   }
   ```

2. 将 `renderMarkdown`、`renderAssistantContent` 等移到 ViewModel
3. `updateViewport()` 简化为:
   ```go
   func (m *Model) updateViewport() {
       m.conversationVM.Refresh(m.conversation, m.viewport.Width)
       m.viewport.SetContent(m.conversationVM.RenderedContent())
       m.viewport.GotoBottom()
   }
   ```

4. 实现脏标记机制:
   ```go
   // Model.Update 中
   case bridge.ContentEvent:
       m.conversation.UpdateLastAssistantContent(msg.Delta)
       m.conversationVM.MarkDirty()  // 标记脏，下次刷新时重建
   ```

**验证**: ViewModel 单元测试覆盖所有渲染场景

---

### Phase 4: Bridge 层重构

**目标**: Agent 回调不依赖 `tea.Program`，可独立测试

**风险**: ⭐⭐（中等风险，解耦 Agent 和 TUI）

**步骤**:
1. 创建 `bridge/callback.go`:
   ```go
   type TUIBridge struct {
       eventCh chan<- bridge.AgentEvent  // 发送事件到 TUI
       replyCh chan string              // 用于同步问答
   }

   func NewBridge(eventCh chan<- bridge.AgentEvent) *TUIBridge

   func (b *TUIBridge) OnContent(delta string) {
       b.eventCh <- bridge.ContentEvent{Delta: delta}
   }

   func (b *TUIBridge) OnToolCallStart(tc agent.ToolCall) {
       b.eventCh <- bridge.ToolStartEvent{Name: tc.Name, Input: tc.Input}
   }

   // ... 其他回调方法

   func (b *TUIBridge) AskFollowupQuestion(question string, options []string) (string, error) {
       reply := make(chan string, 1)
       b.eventCh <- bridge.AskQuestionEvent{Question: question, Options: options, Reply: reply}
       return <-reply, nil
   }
   ```

2. `tuiCallback` 改为使用 `TUIBridge`
3. `AskFollowupQuestion` 的同步等待从 TUI Model 移到 Bridge
4. TUI Model 中的 `pendingReply` 逻辑简化

**验证**: Bridge 层独立单元测试（不需要 Bubbletea）

---

### Phase 5: View 纯函数化 + Bubbletea 薄壳

**目标**: TUI Model 成为薄壳，View 全部纯函数

**风险**: ⭐（低风险，最终整理）

**步骤**:
1. 精简 Bubbletea `Model`:
   ```go
   type Model struct {
       // 业务层
       conversation *model.Conversation
       convVM       *viewmodel.ConversationViewModel
       statusVM     *viewmodel.StatusViewModel
       inputVM      *viewmodel.InputViewModel
       bridge       *bridge.TUIBridge

       // UI 组件（仅 Bubbletea 相关）
       viewport viewport.Model
       textarea textarea.Model
       spinner  spinner.Model

       // 尺寸
       width  int
       height int
   }
   ```

2. `Update()` 保持为薄分发层:
   ```go
   func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.WindowSizeMsg:
           return m.handleResize(msg)
       case tea.KeyMsg:
           return m.handleKey(msg)
       case bridge.AgentEvent:
           return m.handleAgentEvent(msg)
       // ...
       }
       // 更新子组件
       // ...
   }
   ```

3. `View()` 组装纯函数调用的结果:
   ```go
   func (m *Model) View() string {
       sections := []string{
           view.RenderHeader(m.statusVM),
           view.RenderMessages(m.convVM),
           view.RenderToolArea(m.convVM),
           view.RenderInput(m.inputVM),
           view.RenderStatusBar(m.statusVM),
           view.RenderHelp(),
       }
       return lipgloss.JoinVertical(lipgloss.Left, sections...)
   }
   ```

4. 所有 View 函数移到 `view/` 包

**验证**: 完整测试覆盖，行为与重构前完全一致

---

## 4. 迁移风险控制

### 4.1 基本原则

- **每个 Phase 独立提交**，保证可回滚
- **每个 Phase 前后运行现有测试套件**
- **Phase 1-2 纯增量**，不动现有行为
- **Phase 3-4 行为等价重构**，依赖前两个 Phase 的测试
- **Phase 5 为清理收尾**

### 4.2 回滚策略

| Phase | 回滚复杂度 | 回滚方式 |
|-------|-----------|----------|
| Phase 1 | 低 | 删除 `bridge/messages.go`，恢复 `agentUpdateMsg` |
| Phase 2 | 中 | 恢复 `Model` 内联字段，删除 `model/` 包 |
| Phase 3 | 中 | 恢复 `updateViewport()` 内联逻辑 |
| Phase 4 | 中 | 恢复 `tuiCallback` 内联实现 |
| Phase 5 | 低 | 恢复 `View()` 内联实现 |

### 4.3 测试策略

```
Phase 1: 现有测试 + 新增 bridge 测试
Phase 2: 现有测试 + 新增 model 测试
Phase 3: 现有测试 + 新增 viewmodel 测试
Phase 4: 现有测试 + 新增 bridge 测试（完整）
Phase 5: 现有测试 + 新增 view 测试 + 集成测试
```

---

## 5. 预期收益

### 5.1 量化指标

| 指标 | 当前 | 优化后 | 提升 |
|------|------|--------|------|
| 可独立测试的代码比例 | ~10% | ~80% | 8x |
| 单文件平均行数 | ~140 | ~80 | 43% ↓ |
| 新增功能平均改动文件数 | 4-5 | 2-3 | 40% ↓ |
| View 渲染时间（100条消息） | O(n) | O(1) | 数量级 ↓ |
| 引入回归 bug 的概率 | 高 | 低 | 显著 ↓ |

### 5.2 质量提升

- **可测试性**: Model/ViewModel/View/Bridge 均可脱离 Bubbletea 独立测试
- **可维护性**: 清晰的职责边界，新增功能只需改动 2-3 个文件
- **可扩展性**: 添加 Web UI 只需实现新的 Bridge，复用 Model 和 ViewModel
- **性能**: 增量渲染 + 脏标记，避免全量重建
- **类型安全**: 编译时检查所有 Agent 事件类型

---

## 6. 与现有系统的关系

### 6.1 不受影响的模块

- `internal/agent/` — Agent 核心逻辑不变
- `internal/api/` — Provider 实现不变
- `internal/tools/` — 工具系统不变
- `internal/prompts/` — 提示词管理不变
- `pkg/types/` — 共享类型不变

### 6.2 受影响的接口

| 接口 | 变化 |
|------|------|
| `agent.StreamCallback` | 不变，但实现从 `tuiCallback` 改为 `bridge.TUIBridge` |
| `tea.Model` | `Update()` 和 `View()` 逻辑重构，但接口不变 |

### 6.3 废弃的空目录

- `internal/ui/model/` → 正式启用
- `internal/ui/viewmodel/` → 正式启用
- `internal/ui/agent/` → 废弃（改为 `bridge/`）
- `internal/ui/core/` → 废弃（功能并入 `model/` 和 `viewmodel/`）

---

## 7. 参考

- [Bubbletea 架构文档](https://github.com/charmbracelet/bubbletea)
- [MVVM 模式](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93viewmodel)
- [Go 接口设计最佳实践](https://go.dev/doc/effective_go#interfaces)