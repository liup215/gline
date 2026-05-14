# TUI 优化计划（Phase 6-10）

## 概述

在 MVVM 重构（Phase 1-5）完成后，TUI 架构已从单体 Model 演化为清晰的四层架构（Model/ViewModel/View/Bridge）。Phase 6-9 已完成增量渲染、handler 解耦、工具格式化提取和 Model 层净化。Phase 10 基于 2026-05-12 全面审查（发现 18 个优化点）修订为 4 个子阶段，优先修复并发安全 bug，逐步推进用户体验、性能布局和架构完整性。

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

## Phase 10: 并发安全 + 用户体验 + 性能布局 + 架构完整性

> **修订记录** (2026-05-12): 基于 TUI 全面审查（发现 18 个优化点），将原 Phase 10 从 3 项扩展为 4 个子阶段。
> 原 Phase 10 仅覆盖 3/18 项，遗漏了 2 个 P0 级并发安全 bug。handler 测试已在 Phase 7 补齐。

**目标**: 修复并发安全 bug、改善用户体验、优化渲染性能与布局、补齐架构完整性。

**当前测试基线**: 90 个（ui:27 + bridge:16 + model:17 + view:18 + viewmodel:26[含子测试]）

---

### Phase 10a: 并发安全修复 [P0 — 必须立即做]

**风险**: ⭐⭐⭐（高 — 潜在 crash 和 goroutine 泄漏）

#### 问题 1: `cancelFn` 并发 data race

三个位置跨 goroutine 访问 `cancelFn`，无同步保护：
```go
m.cancelFn = cancel   // startAgent (后台 goroutine) 写
m.cancelFn()          // handleKeyMsg (主 goroutine) 读+调用
m.cancelFn = nil      // handleAgentComplete/Error (主 goroutine) 清除
```

**修复方案**: 添加 `sync.Mutex` 保护 `cancelFn` 的读写：
```go
type Model struct {
    // ...
    cancelMu sync.Mutex
    cancelFn context.CancelFunc
}

func (m *Model) setCancelFn(fn context.CancelFunc) {
    m.cancelMu.Lock()
    defer m.cancelMu.Unlock()
    m.cancelFn = fn
}

func (m *Model) getAndClearCancelFn() context.CancelFunc {
    m.cancelMu.Lock()
    defer m.cancelMu.Unlock()
    fn := m.cancelFn
    m.cancelFn = nil
    return fn
}
```

**涉及文件**: `tui.go`, `tui_agent.go`, `tui_input.go`, `tui_state.go`

#### 问题 2: `pendingReply` 通道泄漏

用户按 Esc/Ctrl+C 中断时，`pendingReply` channel 未关闭，Agent goroutine 永久阻塞：
```go
// tui_input.go — Esc handler 只调用 cancelFn，不处理 pendingReply
case tea.KeyEsc:
    if m.isProcessing {
        if m.cancelFn != nil {
            m.cancelFn()   // ← 取消了 context，但 pendingReply channel 仍打开
        }
    }
```

**修复方案**: Esc/Ctrl+C 中断时关闭 pendingReply 通道，让 Agent 收到空值并退出：
```go
case tea.KeyEsc:
    if m.isProcessing {
        if m.pendingReply != nil {
            close(m.pendingReply)  // 通知 Agent 中断
            m.pendingReply = nil
        }
        if m.cancelFn != nil {
            m.cancelFn()
        }
        // ... 其余清理 ...
    }
```
同时在 `AskFollowupQuestion` 的 Agent 侧处理 channel 关闭：
```go
func (b *TUIBridge) AskFollowupQuestion(question string, options []string) (string, error) {
    reply := make(chan string, 1)
    b.eventCh <- AskQuestionEvent{...}
    answer, ok := <-reply  // ok=false 表示 channel 被关闭
    if !ok {
        return "", context.Canceled  // 优雅退出
    }
    return answer, nil
}
```

**涉及文件**: `tui_input.go`, `bridge/callback.go`

#### 步骤

1. 在 `Model` 中添加 `cancelMu sync.Mutex` 字段
2. 创建 `setCancelFn()` / `getAndClearCancelFn()` 方法
3. 替换所有直接访问 `m.cancelFn` 的代码
4. 修改 Esc/Ctrl+C handler 关闭 `pendingReply`
5. 修改 `AskFollowupQuestion` 处理 channel 关闭
6. 新增并发安全测试（`-race` 检测）

#### 验证

- `go test -race ./internal/ui/...` 无 data race 报告
- Esc 中断 AskFollowupQuestion 时 Agent goroutine 正常退出
- 新增 2-3 个测试

---

### Phase 10b: 用户体验修复 [P1] — ✅ 已完成 2026-05-14

**风险**: ⭐（低风险）

#### 问题 3: 错误双重显示

`handleAgentError` 同时添加两条系统消息：
```go
// 1. "🔧 Failed: read_file"
// 2. "Error: something went wrong"
```
同一错误在视图中出现两次。

**修复方案**: 只保留 `addErrorMessage`（更详细的错误描述），删除 "🔧 Failed" 系统消息。
在 `handleAgentError` 中保留工具状态标记为 failed（用于 ToolArea 显示），但不再追加 "🔧 Failed" 系统消息。

#### 问题 4: GotoBottom 阻止用户滚动

`updateViewport()` 无条件调用 `m.viewport.GotoBottom()`，用户无法在流式输出时向上滚动。

**修复方案**: 只在用户已处于底部时自动滚动：
```go
func (m *Model) updateViewport() {
    m.convVM.Refresh(...)
    m.viewport.SetContent(m.convVM.Content())
    if m.viewport.AtBottom() {
        m.viewport.GotoBottom()
    }
}
```

#### 步骤

1. 修改 `handleAgentError` 删除重复的 "🔧 Failed" 消息
2. 修改 `updateViewport()` 检测 `viewport.AtBottom()`
3. 新增 2 个测试验证行为

#### 验证

- 错误事件只产生一条可见消息
- 用户向上滚动时不再被强制跳回底部
- 流式输出时自动滚到底部行为不变（用户未手动滚动时）

---

### Phase 10c: 性能与布局优化 [P2] — ✅ 已完成 2026-05-14

**风险**: ⭐⭐（中等风险 — 布局变化需多终端尺寸测试）

#### 问题 5: tickMsg 无差别刷新

处理期间每 100ms 发 tick，即使无新内容也走 Update 循环。

**修复方案**: 在 Model 中添加 `contentChanged bool` 标志，tick handler 只在标志为 true 时触发 viewport 刷新并重置标志。

#### 问题 6: Header/StatusBar 信息重复

Header 和 StatusBar 都显示 Mode+Provider+Model，浪费两行屏幕空间。

**修复方案**: 合并 Header 和 StatusBar 为一行：
- 左侧：🚀 gline · Provider/Model · [MODE]
- 右侧：动态状态（Processing/Streaming/Tool）

#### 问题 7: 固定高度配比不灵活

`inputHeight=3`, `toolAreaHeight=3`, 魔数 `4`，小窗口体验差。

**修复方案**: 按比例分配 + 最小/最大值约束：
```go
func calculateLayout(totalHeight int) (viewportH, toolH, inputH int) {
    inputH = clamp(3, totalHeight/10, 5)
    toolH = clamp(2, totalHeight/10, 6)
    reserved := inputH + toolH + 2  // header + help
    viewportH = totalHeight - reserved
    if viewportH < 3 { viewportH = 3 }
    return
}
```

#### 步骤

1. 添加 `contentChanged` 标志到 Model，修改 tickMsg 处理
2. 合并 Header+StatusBar 为单行 `RenderCompactBar()`
3. 创建 `calculateLayout()` 函数替换硬编码
4. 更新 `handleWindowSize` 使用新的布局计算
5. 新增 3-4 个测试

#### 验证

- 无新内容时 tick 不触发 viewport 刷新
- 屏幕空间节省一行
- 小窗口（<20 行）和大窗口（>60 行）布局合理

---

### Phase 10d: 架构完整性 [P3 — 原 Phase 10 更新版]

**风险**: ⭐（低风险，增量添加）

#### 问题 8: Tool Area 渲染逻辑位置错误

`view/tool_area.go` 是空壳透传，真正逻辑在 `viewmodel/conversation_vm.go` 的 `renderToolArea()`。

**修复方案**: 将 `renderToolArea()` 移到 `view/tool_area.go`，改为纯函数：
```go
func RenderToolAreaContent(history []model.ToolStatus, width, maxEntries int) string {
    // 纯函数，零副作用
}
```
ViewModel 的 `Refresh()` 调用 `view.RenderToolAreaContent()` 替代内联渲染。

#### 问题 9: 缺少 StatusViewModel

状态栏数据在 `View()` 中内联组装，无法独立测试。

**修复方案**: 创建 `viewmodel/status_vm.go`：
```go
type StatusViewModel struct {
    data view.StatusBarData
}

func (vm *StatusViewModel) Refresh(mode, provider, model string, isProcessing, isStreaming bool, currentTool, spinnerView string, width int) view.StatusBarData {
    vm.data = view.StatusBarData{...}
    return vm.data
}
```
注意：若 Phase 10c 已合并 Header/StatusBar，此处应适配新的合并布局。

#### 问题 10: messageCache 无驱逐机制

`messageCache` 只增不减，长会话内存增长。`Clear()` 后靠隐式 `len(msgs) != len(cache)` 兜底。

**修复方案**:
- `Conversation.Clear()` 时调用 `vm.InvalidateCache()` 清空缓存
- 或在 `Refresh()` 全量重建时自动清理已删除消息的缓存条目

#### 问题 11: 系统消息静默丢弃

`renderSystemMessage` 对不匹配前缀的消息返回空字符串，用户看不到。

**修复方案**: 不匹配任何已知前缀的系统消息，以默认灰色样式显示，而非丢弃：
```go
default:
    b.WriteString(view.SystemStyle.Render(content))
    b.WriteString("\n\n")
```

#### 补充测试

| 文件 | 需要补充的测试 | 当前状态 |
|------|---------------|---------|
| `tui_input.go` | `handleKeyMsg` 各分支（Ctrl+C/Esc/Tab/Enter/Ctrl+L） | ❌ 完全缺失 |
| `tui_input.go` | `handleWindowSize` 小窗口/宽度0/重复resize | ⚠️ 仅有1个 |
| `tui_agent.go` | `startAgent()` 错误路径（nil agent/空消息） | ❌ 完全缺失 |
| `view/tool_area.go` | 工具区域渲染测试（迁移后） | ⚠️ 只有透传测试 |

#### 步骤

1. 迁移 `renderToolArea` 到 `view/tool_area.go`
2. 创建 `viewmodel/status_vm.go`（适配 Phase 10c 的布局变更）
3. 添加 `InvalidateCache()` 方法，在 `Clear()` 时调用
4. 修改 `renderSystemMessage` 默认显示而非丢弃
5. 补充 handleKeyMsg / startAgent / handleWindowSize 测试
6. 目标：测试总数 90 → 110+

#### 验证

- `view/tool_area.go` 包含真正的渲染逻辑和测试
- 系统消息不再被静默丢弃
- messageCache 在 Clear 后正确清空
- 测试总数达到 110+

---

## Phase 10 总体迁移风险控制

### 基本原则

- **每个子阶段独立提交**，保证可回滚
- **每个子阶段前后运行 `go test ./internal/ui/...`**
- **Phase 10a 必须最先做** — 并发 bug 影响程序正确性
- **Phase 10b-10d 按优先级顺序** — 每个可独立完成

### 回滚策略

| 子阶段 | 回滚复杂度 | 回滚方式 |
|--------|-----------|----------|
| Phase 10a | 低 | 移除 mutex，恢复直接访问 cancelFn；恢复旧 Esc handler |
| Phase 10b | 低 | 恢复双重错误消息；移除 AtBottom 检测 |
| Phase 10c | 中 | 恢复 Header+StatusBar 分离；恢复硬编码高度 |
| Phase 10d | 低 | 恢复 tool_area.go 空壳；删除 StatusViewModel |

### 预期收益

| 指标 | 当前 | 优化后 | 提升 |
|------|------|--------|------|
| 并发安全 | ❌ data race | ✅ mutex 保护 | 修复潜在 crash |
| goroutine 泄漏 | ❌ pendingReply 未关闭 | ✅ Esc 正确清理 | 修复资源泄漏 |
| 错误显示 | 双重 | 单一 | UX 改善 |
| 滚动体验 | 强制跳底 | 智能跟随 | UX 改善 |
| 屏幕利用率 | Header+StatusBar 重复 | 合并为一行 | 节省 1 行 |
| 系统消息 | 静默丢弃 | 默认显示 | 信息完整 |
| 总测试数 | 90 | 110+ | 22% ↑ |

---

## 延后项（Phase 11+）

以下优化点按需实施，不在 Phase 10 范围内：

| # | 问题 | 优先级 | 工作量 |
|---|------|--------|--------|
| 1 | 用户消息无 Markdown 渲染 | P4 | 小 |
| 2 | 无输入历史（上下箭头翻阅） | P4 | 中 |
| 3 | 无代码语法高亮（需 chroma） | P4 | 大 |
| 4 | 工具区条目数计算有误（border 占一行未计入） | P4 | 小 |
| 5 | 硬编码颜色无主题支持 | P4 | 中 |
| 6 | 无视觉层次（缩进/边框/分隔线） | P4 | 中 |
| 7 | reasoning_content 不显示 | P4 | 小 |
