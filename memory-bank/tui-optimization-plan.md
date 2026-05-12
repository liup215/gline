# TUI 优化计划（Phase 6-10）

## 概述

在 MVVM 重构（Phase 1-5）完成后，TUI 架构已从单体 Model 演化为清晰的四层架构（Model/ViewModel/View/Bridge）。本计划针对审查中发现的 10 个优化点，按渐进式方式分 5 个 Phase 实施。

---

## Phase 6: 渲染性能优化（脏标记 → 增量渲染）

**目标**: 让 ViewModel 的脏标记机制真正生效，避免每次状态变更都全量重建所有消息。

**风险**: ⭐⭐（中等风险，需要确保行为等价）

**当前问题**:
```go
// viewmodel/conversation_vm.go — Refresh() 每次都全量重建
func (vm *ConversationViewModel) Refresh(conv *model.Conversation, ...) {
    var content strings.Builder
    for i := range msgs {  // ← 每次都遍历所有消息
        // 重新渲染每条消息
    }
    vm.content = content.String()
    vm.dirty = false
}
```

**优化方案**:

1. **引入消息级脏标记** — 在 ViewModel 中维护一个 `dirtyMessages map[int]bool`，只重新渲染标记为脏的消息
2. **增量拼接** — 只重建变化消息的渲染结果，不变的消息复用缓存
3. **对外接口不变** — `Refresh()` 签名不变，内部实现改为增量

```go
type ConversationViewModel struct {
    content           string
    toolAreaContent   string
    dirty             bool
    dirtyMessages     map[int]bool  // 新增：标记需要重新渲染的消息索引
    renderer          *glamour.TermRenderer
    rendererWrapWidth int
}

func (vm *ConversationViewModel) MarkMessageDirty(idx int) {
    vm.dirtyMessages[idx] = true
    vm.dirty = true
}

func (vm *ConversationViewModel) Refresh(conv *model.Conversation, ...) {
    if !vm.dirty {
        return  // 无变化，跳过
    }
    // 只重建 dirty 消息，复用其他消息的渲染结果
    // ...
}
```

**步骤**:
1. 在 `ConversationViewModel` 中添加 `dirtyMessages map[int]bool`
2. 添加 `MarkMessageDirty(idx int)` 方法
3. 修改 `Refresh()` 实现增量重建
4. 在 `tui_state.go` 的 handler 中调用 `MarkMessageDirty()` 替代 `MarkDirty()`
5. 更新 ViewModel 测试覆盖增量场景

**验证**: 所有现有测试通过，性能测试显示 100 条消息时渲染时间显著降低

---

## Phase 7: 状态 Handler 解耦（View 和 State 分离）

**目标**: 消除 `tui_state.go` 中 7 个 handler 对 `updateViewport()` 的直接调用，让状态变更逻辑可独立测试。

**风险**: ⭐（低风险，纯重构）

**当前问题**:
```go
// tui_state.go — 每个 handler 末尾都调用 updateViewport()
func handleAgentContent(m *Model, msg bridge.ContentEvent) []tea.Cmd {
    // ... 状态变更 ...
    m.updateViewport()  // ← 耦合
    return nil
}
```

**优化方案**:

1. **handler 返回脏标记** — handler 不再直接调用 `updateViewport()`，而是通过返回值或 `tea.Cmd` 通知调用方需要刷新
2. **统一刷新点** — `Update()` 方法在收集所有 handler 的 cmd 后，统一调用 `updateViewport()`

```go
// 方案 A：handler 返回是否需要刷新
func handleAgentContent(m *Model, msg bridge.ContentEvent) ([]tea.Cmd, bool) {
    // ... 状态变更 ...
    return nil, true  // 需要刷新
}

// Update 中统一处理
case bridge.AgentEvent:
    cmds2, needsRefresh := handleAgentUpdate(m, msg)
    cmds = append(cmds, cmds2...)
    if needsRefresh {
        m.updateViewport()
    }
```

**步骤**:
1. 修改 `handleAgentUpdate` 和所有 handler 签名，返回 `needsRefresh bool`
2. `Update()` 中根据返回值统一调用 `updateViewport()`
3. 为每个 handler 编写独立单元测试（不依赖 viewport）

**验证**: 每个 handler 有独立单元测试，`updateViewport()` 调用次数可验证

---

## Phase 8: 拆分 `handleAgentToolStart` + 工具显示逻辑迁移

**目标**: 将 `handleAgentToolStart`（~80 行）中的工具显示格式化逻辑提取到 `view/` 包，让状态 handler 只负责状态变更。

**风险**: ⭐（低风险，纯提取）

**当前问题**:
```go
// tui_state.go — handleAgentToolStart 混合了状态变更和展示逻辑
func handleAgentToolStart(m *Model, msg bridge.ToolStartEvent) []tea.Cmd {
    m.currentTool = msg.Name
    m.conversation.AddToolStart(msg.Name)
    
    // ↓ 以下全是展示逻辑，应该移到 view/ 包
    desc := view.GetToolDescription(msg.Name)
    display := ""
    if msg.Input != "" {
        if view.NormalizeToolName(msg.Name) == "attempt_completion" {
            // ... 大量 JSON 解析和格式化 ...
        }
    }
    // ...
}
```

**优化方案**:

1. 在 `view/` 包中创建 `tool_format.go`，提取所有工具显示格式化函数
2. `handleAgentToolStart` 只保留状态变更逻辑
3. 特殊工具（`attempt_completion`, `ask_followup_question`, `plan_mode_respond`）的处理逻辑统一到 ViewModel 或 View

**步骤**:
1. 创建 `internal/ui/view/tool_format.go`
   - 提取 `FormatToolStartDisplay(name, input string) string`
   - 提取 `FormatAttemptCompletionDisplay(input string) string`
   - 提取 `FormatToolCompleteDisplay(name, result, status string) string`
2. 简化 `handleAgentToolStart` 和 `handleAgentToolComplete`
3. 为新的格式化函数编写单元测试

**验证**: 新函数测试覆盖所有工具类型和边界情况，`handleAgentToolStart` 行数减少 50%+

---

## Phase 9: Model 层净化 + 渲染缓存迁移

**目标**: 将 `model.Message` 中的渲染缓存字段移到 ViewModel，保持 Model 层的纯净。

**风险**: ⭐（低风险，纯提取）

**当前问题**:
```go
// model/message.go — Model 层包含 ViewModel 关注点
type Message struct {
    Role      types.Role
    Content   string
    ToolCalls []types.ToolCall
    Options   []string
    Timestamp time.Time
    
    // ↓ 这些是 ViewModel 的缓存，不应该在 Model 中
    Rendered          string
    RenderedWrapWidth int
    RenderedSource    string
}
```

**优化方案**:

1. 从 `model.Message` 中删除 `Rendered`, `RenderedWrapWidth`, `RenderedSource`
2. 在 ViewModel 中创建 `cachedMessage` 结构体持有渲染缓存
3. ViewModel 维护 `map[int]*cachedMessage` 映射消息索引到缓存

```go
// viewmodel/conversation_vm.go
type cachedMessage struct {
    content           string
    rendered          string
    wrapWidth         int
}

type ConversationViewModel struct {
    content           string
    toolAreaContent   string
    dirty             bool
    dirtyMessages     map[int]bool
    messageCache      map[int]*cachedMessage  // 新增
    renderer          *glamour.TermRenderer
    rendererWrapWidth int
}
```

**步骤**:
1. 从 `model.Message` 删除 3 个缓存字段
2. 从 `model/message.go` 删除 `ResetRenderCache()` 方法
3. 在 ViewModel 中添加 `messageCache` 和缓存逻辑
4. 更新 `renderAssistantContent` 使用新的缓存机制
5. 更新所有相关测试

**验证**: Model 包零外部依赖，ViewModel 测试覆盖缓存命中/未命中场景

---

## Phase 10: 架构完整性 + 测试覆盖

**目标**: 补齐缺失的 ViewModel（status_vm, input_vm），修复 `tool_area.go` 透传问题，补充测试覆盖。

**风险**: ⭐（低风险，增量添加）

### 10a. 创建 StatusViewModel

```go
// viewmodel/status_vm.go
type StatusBarData struct {
    Mode         agent.Mode
    Provider     string
    ModelName    string
    IsProcessing bool
    IsStreaming  bool
    CurrentTool  string
    SpinnerView  string
    Width        int
}

type StatusViewModel struct {
    // 从 Model 派生状态栏数据
}

func (vm *StatusViewModel) Refresh(m *Model) StatusBarData {
    // 从 Model 提取状态栏所需数据
}
```

### 10b. 修复 `view/tool_area.go`

将 ViewModel 中的 `renderToolArea()` 移到 `view/tool_area.go`，使其成为真正的纯函数。

### 10c. 补充测试

| 文件 | 需要补充的测试 |
|------|---------------|
| `tui_state.go` | 7 个 handler 的独立单元测试（mock viewport） |
| `tui_input.go` | `handleWindowSize`, `handleKeyMsg` 各分支 |
| `tui_agent.go` | `startAgent()` 错误路径 |
| `view/tool_area.go` | 工具区域渲染测试（迁移后） |

**步骤**:
1. 创建 `viewmodel/status_vm.go`
2. 迁移 `renderToolArea` 到 `view/tool_area.go`
3. 为每个缺失的测试点编写测试

**验证**: 测试总数从 69 提升到 90+，覆盖率提升

---

## 迁移风险控制

### 基本原则

- **每个 Phase 独立提交**，保证可回滚
- **每个 Phase 前后运行 `go test ./internal/ui/...`**
- **Phase 6-7 行为等价重构**，不改变用户可见行为
- **Phase 8-10 纯增量**，不动现有行为

### 回滚策略

| Phase | 回滚复杂度 | 回滚方式 |
|-------|-----------|----------|
| Phase 6 | 中 | 恢复 `Refresh()` 为全量重建，删除 `dirtyMessages` |
| Phase 7 | 低 | 恢复 handler 中直接调用 `updateViewport()` |
| Phase 8 | 低 | 恢复 `handleAgentToolStart` 内联逻辑 |
| Phase 9 | 中 | 恢复 `model.Message` 缓存字段 |
| Phase 10 | 低 | 删除新增文件，恢复旧测试 |

### 预期收益

| 指标 | 当前 | 优化后 | 提升 |
|------|------|--------|------|
| View 渲染时间（100条消息） | O(n) 全量 | O(1) 增量 | 数量级 ↓ |
| 状态 handler 可独立测试 | ❌ | ✅ | 7 个新测试 |
| Model 层纯度 | 含缓存字段 | 零外部依赖 | 架构更纯净 |
| `handleAgentToolStart` 行数 | ~80 行 | ~30 行 | 60% ↓ |
| 总测试数 | 69 | 90+ | 30% ↑ |
