# Active Context

## 当前焦点

### TUI MVVM 架构重构 ✅ 已完成

**5 阶段渐进方案**（全部完成 2026-05-11）:
- ✅ Phase 1: 类型安全的消息系统（消除魔法字符串）
- ✅ Phase 2: 抽离纯数据 Model
- ✅ Phase 3: 引入 ViewModel 层
- ✅ Phase 4: Bridge 层重构（解耦 Agent 和 TUI）
- ✅ Phase 5: View 纯函数化 + Bubbletea 薄壳

### TUI 优化 Phase 6-10（进行中）🔧

**目标**: 在 MVVM 重构基础上，进一步优化渲染性能、解耦状态 handler、净化 Model 层、补齐架构完整性。

**详细计划**: 见 `memory-bank/tui-optimization-plan.md`

**5 阶段渐进方案**:
- ✅ Phase 6: 渲染性能优化（脏标记 → 增量渲染）— **已完成** (2026-05-11)
- ✅ Phase 7: 状态 Handler 解耦（View 和 State 分离）— **已完成** (2026-05-11)
- ✅ Phase 8: 拆分 `handleAgentToolStart` + 工具显示逻辑迁移 — **已完成** (2026-05-11)
- ✅ Phase 9: Model 层净化 + 渲染缓存迁移 — **已完成** (2026-05-12)
- ⬜ Phase 10: 并发安全 + 用户体验 + 性能布局 + 架构完整性 — **待开始**
  - ⬜ Phase 10a [P0]: 并发安全修复 — cancelFn data race + pendingReply 通道泄漏
  - ⬜ Phase 10b [P1]: 用户体验修复 — 错误双重显示 + GotoBottom 阻止滚动
  - ⬜ Phase 10c [P2]: 性能与布局优化 — tickMsg 无差别刷新 + Header/StatusBar 合并 + 灵活高度配比
  - ⬜ Phase 10d [P3]: 架构完整性 — Tool Area 渲染迁移 + StatusViewModel + cache 驱逐 + 系统消息默认显示 + 补充测试

**详细计划**: 见 `memory-bank/tui-optimization-plan.md`（Phase 10 已于 2026-05-12 修订，从原 3 项扩展为 4 个子阶段 11 项优化）

**Phase 9 完成记录**:
1. 从 `model.Message` 删除渲染缓存字段 — 删除 `Rendered`, `RenderedWrapWidth`, `RenderedSource` 三个字段
2. 删除 `ResetRenderCache()` 方法 — Model 层现在完全纯净，零外部 UI 依赖
3. 扩展 ViewModel `cachedMessage` 结构体 — 添加 `content` 和 `wrapWidth` 字段用于缓存验证
4. 更新 `renderAssistantContent()` — 从访问 `msg.Rendered` 改为使用 `vm.messageCache` 进行缓存命中/未命中判断
5. 更新 `Refresh()` 方法 — 全量重建和增量刷新时正确保存 `content` 和 `wrapWidth` 到缓存
6. 删除 `model/conversation_test.go` 中的 `TestMessageResetRenderCache` 测试
7. 添加 4 个 ViewModel 缓存测试：
   - `TestViewModelCacheHit` — 验证缓存正确存储 content/wrapWidth/rendered
   - `TestViewModelCacheMissOnContentChange` — 验证内容变更时缓存失效
   - `TestViewModelCacheMissOnWidthChange` — 验证宽度变更时缓存失效
   - `TestViewModelCacheNoDirectAccessToMessageCache` — 编译时验证 Message 无缓存字段
8. `go build ./...` 成功，`go test ./internal/ui/...` 全部通过（90+ 测试）

**Phase 8 完成记录**:
1. 创建 `internal/ui/view/tool_format.go` — 提取 3 个工具显示格式化函数：
   - `FormatToolStartDisplay(name, input string) string` — 格式化工具启动显示（处理 attempt_completion 保持完整输入、普通工具显示主要参数、JSON 回退格式化）
   - `FormatAttemptCompletionContent(input string) string` — 从 attempt_completion 的 JSON 输入中提取人类可读的结果（优先 result/content 字符串、对象转 JSON 代码块、无效 JSON 返回原始输入）
   - `FormatToolCompleteDisplay(name, result, status string) string` — 格式化工具完成显示（包含 Completed/Failed 状态和截断的结果预览）
2. 简化 `handleAgentToolStart`（~80 行 → ~30 行）— 删除内联的 JSON 解析和格式化逻辑，委托给 `view.FormatToolStartDisplay()` 和 `view.FormatAttemptCompletionContent()`
3. 简化 `handleAgentToolComplete`（~30 行 → ~20 行）— 删除内联的格式化逻辑，委托给 `view.FormatToolCompleteDisplay()`
4. 删除 `tui_state.go` 中不再需要的 `bytes`、`encoding/json`、`strings` 导入
5. 为 3 个新格式化函数编写 13 个单元测试（`view/view_test.go`）：
   - `TestFormatToolStartDisplay`: 4 个测试（空输入、主要参数、attempt_completion、未知工具）
   - `TestFormatAttemptCompletionContent`: 5 个测试（字符串 result、字符串 content、对象 result、无效 JSON、空对象）
   - `TestFormatToolCompleteDisplay`: 4 个测试（完成状态、失败状态、空结果、截断行）
6. `go build ./...` 成功，`go test ./internal/ui/...` 全部通过

**Phase 7 完成记录**:
1. 修改 `tui_state.go` 中全部 7 个 handler 签名，从 `(m *Model, msg) []tea.Cmd` 改为 `(m *Model, msg) (bool, []tea.Cmd)`，返回 `needsRefresh` 替代直接调用 `updateViewport()`
   - `handleAgentContent` — 返回 `true, nil`
   - `handleAgentToolStart` — 返回 `true, nil`
   - `handleAgentToolComplete` — 返回 `true, nil`
   - `handleAgentError` — 返回 `true, cmds`（含 textarea.Blink）
   - `handleAgentComplete` — 返回 `true, cmds`（含 textarea.Blink）
   - `handleAgentStreamStart` — 返回 `true, nil`
   - `handleAgentStreamEnd` — 返回 `true, nil`
2. 修改 `tui_update.go` 的 `handleAgentUpdate()` — 返回 `(bool, []tea.Cmd)`，收集各 handler 的 `needsRefresh`
3. 修改 `tui.go` 的 `Update()` — 添加 `needsRefresh` 变量，在 switch 后统一调用 `m.updateViewport()` 当 `needsRefresh == true`
4. `AskQuestionEvent` handler 也改为设置 `needsRefresh = true` 替代直接调用 `m.updateViewport()`
5. 创建 `internal/ui/tui_state_test.go` — 21 个独立单元测试覆盖所有 handler：
   - `handleAgentContent`: 3 个测试（返回脏标记、创建缺失 slot、追加 delta）
   - `handleAgentToolStart`: 5 个测试（返回脏标记、attempt_completion、ask_followup_question、plan_mode_respond、普通工具系统消息）
   - `handleAgentToolComplete`: 4 个测试（返回脏标记、attempt_completion、plan_mode_respond、ask_followup_question）
   - `handleAgentError`: 2 个测试（有/无运行中工具）
   - `handleAgentComplete`: 1 个测试（返回脏标记和状态重置）
   - `handleAgentStreamStart`: 1 个测试（返回脏标记和创建消息）
   - `handleAgentStreamEnd`: 1 个测试（返回脏标记和停止流）
   - 集成测试: 4 个（Update 触发 viewport 刷新、AskQuestionEvent、tickMsg 不触发、所有事件类型返回脏标记）
6. `go build ./...` 成功，`go test ./internal/ui/...` 全部通过（76 个测试）

**Phase 6 完成记录**:
1. 在 `ConversationViewModel` 中添加 `dirtyMessages map[int]bool` 和 `messageCache map[int]cachedMessage`
2. 添加 `MarkMessageDirty(idx int)` 方法 — 支持标记单个消息为脏
3. 修改 `Refresh()` 实现增量重建：
   - 当 `dirtyMessages` 非空且消息数量匹配时，只重新渲染脏消息
   - 其他消息复用 `messageCache` 中的缓存
   - 消息数量变化或首次调用时自动回退到全量重建
4. 将 `writeUserMessage`/`writeAssistantMessage`/`writeSystemMessage` 重构为 `renderUserMessage`/`renderAssistantMessage`/`renderSystemMessage` 纯函数
5. 在 `tui_state.go` 和 `tui.go` 的所有 handler 中调用 `MarkMessageDirty()` 替代 `MarkDirty()`
6. 新增 6 个单元测试覆盖增量渲染场景
7. `go build ./...` 成功，`go test ./internal/ui/...` 全部 55 个测试通过

**Phase 1 完成记录**:
1. 创建 `internal/ui/bridge/messages.go` — 定义 `AgentEvent` 接口 + 8 个具体事件类型
2. 创建 `internal/ui/bridge/messages_test.go` — 8 个单元测试全部通过
3. `tuiCallback` 和 `startAgent()` 改为发送 `bridge.XXXEvent`
4. `Update()` 改为处理 `bridge.AgentEvent` / `bridge.AskQuestionEvent`
5. `handleAgentUpdate` 改为类型 switch（编译时检查）
6. 7 个状态 handler 签名改为接收类型化事件（如 `bridge.ContentEvent`）
7. 删除 `agentUpdateMsg` 和 `askQuestionMsg` 旧类型
8. 所有现有测试更新并通过，`go build` 成功

**Phase 2 完成记录**:
1. 创建 `internal/ui/model/message.go` — 纯数据 `Message` 结构体（零外部依赖）
2. 创建 `internal/ui/model/conversation.go` — `Conversation` 结构体 + 业务方法
   - 消息操作：`AppendMessage`, `UpdateMessageContent`, `SetMessageContent`, `LastUserMessage`, `Clear`
   - 工具历史：`AddToolStart`, `MarkToolComplete`, `MarkToolFailed`, `ClearToolHistory`, `LastRunningToolName`
3. 重构 `ui.Model` 结构体 — 删除 `messages`, `toolHistory`, `mode`, `provider`, `model` 字段
   - 替换为 `conversation *model.Conversation`
   - 保留 UI 状态：`activeAssistantIndex`, `isProcessing`, `isStreaming`, `currentTool`, `err`
4. 更新 `tui_state.go` — 7 个 handler 全部委托给 `conversation` 方法
5. 更新 `tui_agent.go` — `startAgent()` 使用 `conversation.LastUserMessage()`
6. 更新 `tui_view.go`, `tui_view_render.go` — 遍历 `conversation.Messages`
7. 更新 `tui_input.go` — `Ctrl+L` 用 `conversation.Clear()`，`Tab` 切换 `conversation.Mode`
8. 更新 `tui_test.go` — 测试适配新字段（使用 `uimodel` 别名避免包名冲突）
9. 创建 `internal/ui/model/conversation_test.go` — 23 个单元测试全部通过
10. `go build ./...` 成功，现有测试 + 新测试全部通过

**Phase 4 完成记录**:
1. 创建 `internal/ui/bridge/callback.go` — `TUIBridge` 结构体
   - 实现 `agent.StreamCallback` 接口，通过 `eventCh chan<- AgentEvent` 发送事件
   - 替代原来的 `tuiCallback`（依赖 `*tea.Program`），Bridge 层可独立测试
   - 编译时断言 `var _ agent.StreamCallback = (*TUIBridge)(nil)`
2. 创建 `internal/ui/bridge/callback_test.go` — 8 个单元测试全部通过
   - 测试每个回调方法发送正确的事件类型
   - 测试 `AskFollowupQuestion` 的同步阻塞行为
   - 测试多事件顺序发送
3. 修改 `internal/ui/tui.go` — 集成 eventCh + 转发 goroutine
   - Model 新增 `eventCh chan bridge.AgentEvent` 和 `done chan struct{}`
   - 删除 `program *tea.Program` 字段和 `SetProgram()` 方法
   - `Run()` 函数：创建 buffered channel (64)，启动转发 goroutine 将事件中转到 `p.Send()`
   - `p.Run()` 返回后 `close(done)` 通知 goroutine 退出
4. 修改 `internal/ui/tui_agent.go` — 用 `bridge.NewTUIBridge(m.eventCh)` 替代 `tuiCallback`
   - 删除 `tuiCallback` 结构体及其全部 7 个方法
   - `startAgent()` 改为创建 `TUIBridge` 实例
5. `go build ./...` 成功，`go test ./internal/ui/...` 全部 49 个测试通过

**Phase 5 完成记录**:
1. 创建 `internal/ui/view/styles.go` — 所有样式变量和工具格式化函数迁移到 `view` 包
   - 20+ lipgloss 样式变量（`UserStyle`, `AssistantStyle`, `StatusBarStyle` 等）导出供其他包引用
   - `NormalizeToolName`, `GetToolDescription`, `GetToolMainArg`, `FormatToolResultLines` 导出
   - `ToolDescriptions` map 导出
2. 创建 `internal/ui/view/status_bar.go` — `RenderStatusBar(StatusBarData) string` 纯函数
3. 创建 `internal/ui/view/tool_area.go` — `RenderToolArea(content) string` 纯函数
4. 创建 `internal/ui/view/layout.go` — `RenderHeader`, `RenderHelp`, `RenderInputBox`, `RenderLayout` 纯函数
5. 重写 `tui.go` 的 `View()` — 委托给 `view.RenderLayout()` + 各纯函数，Model 成为薄壳
6. 更新 `tui_state.go` — `normalizeToolName`/`getToolDescription`/`getToolMainArg`/`formatToolResultLines` 改为 `view.NormalizeToolName`/`view.GetToolDescription`/`view.GetToolMainArg`/`view.FormatToolResultLines`
7. 更新 `viewmodel/conversation_vm.go` — 删除重复的样式定义，改用 `view.UserStyle`/`view.AssistantStyle` 等
8. 删除 `tui_view.go` 和 `tui_styles.go` — 逻辑已迁移到 `view/` 包
9. 删除空目录 `agent/` 和 `core/`
10. 创建 `internal/ui/view/view_test.go` — 12 个单元测试全部通过
11. `go build ./...` 成功，`go test ./internal/ui/...` 全部通过

**Phase 3 完成记录**:
1. 创建 `internal/ui/viewmodel/conversation_vm.go` — `ConversationViewModel` 结构体
   - 持有 Glamour renderer（从 Model 移入）
   - `Refresh()` 从 `model.Conversation` 重建完整 viewport 内容和工具区域
   - `Content()` / `ToolAreaContent()` 提供渲染后的字符串
   - 脏标记机制：`MarkDirty()` / `IsDirty()`（为后续增量渲染预留）
2. 迁移渲染逻辑到 ViewModel
   - `renderMarkdown` → ViewModel 私有方法
   - `renderAssistantContent` → ViewModel 私有方法
   - `renderMessageHeader` → ViewModel 私有方法
   - `formatToolCallsInline` → ViewModel 私有方法
   - `renderToolCalls` → ViewModel 私有方法
   - `updateViewport()` 的全量渲染循环 → `Refresh()`
3. 重构 `ui.Model` — 移除 `renderer` / `rendererWrapWidth` 字段，添加 `convVM *viewmodel.ConversationViewModel`
4. 重写 `updateViewport()` — 委托给 `convVM.Refresh()`，然后设置 viewport 内容
5. 重写 `renderToolArea()` — 委托给 `convVM.ToolAreaContent()`
6. 删除 `internal/ui/tui_view_render.go` — 所有函数已迁移到 ViewModel
7. 删除 `internal/ui/tui_helpers.go` — `renderMarkdown` / `formatToolCallsInline` 已迁移
8. 更新 `tui_test.go` — `TestToolStatusArea` 在直接修改数据后调用 `updateViewport()`
9. 创建 `internal/ui/viewmodel/conversation_vm_test.go` — 18 个单元测试全部通过
10. `go build ./...` 成功，`go test ./internal/ui/...` 全部通过

### 已完成任务 ✅
```

1. **Phase 0: 项目初始化** ✅
   - ✅ 创建最小可运行项目结构
   - ✅ 配置 GitHub Actions CI/CD
   - ✅ 本地构建测试通过
   - ✅ 支持跨平台编译 (5个目标平台)

2. **Phase 1: 基础框架** ✅
   - ✅ 配置管理系统 (Viper)
   - ✅ 日志系统 (Zerolog)
   - ✅ CLI 命令结构 (Cobra)

3. **Phase 2: 核心模块** ✅
   - ✅ Agent 接口定义和 BaseAgent 实现
   - ✅ Provider 接口定义
   - ✅ Tool 接口定义和 10个基础工具
   - ✅ Agent 核心循环 (消息处理、工具调用)
   - ✅ Plan/Act 模式切换
   - ✅ 工具注册表 (线程安全)
   - ✅ Anthropic Provider 实现
   - ✅ Provider 注册表
   - ✅ 系统提示词管理

4. **Phase 3: LLM 集成** ✅
   - ✅ **OpenAI Provider** - 通用 OpenAI 兼容 Provider
     - ✅ 支持 OpenAI 官方 API
     - ✅ 支持自定义 base_url (OpenRouter, DashScope, Ollama 等)
     - ✅ 支持工具调用
     - ✅ 完整的错误处理
     - ✅ 单元测试覆盖
   - ✅ **DashScope 兼容性修复**
     - ✅ 修复 URL 拼接问题 - 使用 `buildFullURL` 函数
     - ✅ 添加 SSE 调试日志
     - ✅ 验证 DashScope 流式响应兼容性

### 已完成任务 ✅

**Phase 3: LLM 集成 - 已完成**

根据用户要求，`gline chat` 已默认进入**交互式多轮对话（TUI）模式**。

已实现的组件：
1. ✅ **OpenAI Provider** - 已完成
2. ✅ **流式响应处理** - 为 TUI 提供实时输出能力
3. ✅ **TUI 交互式界面** - Bubbletea 多轮对话界面
4. ✅ **CLI 命令集成** - gline chat 默认 TUI 模式
5. ⏳ **错误处理增强** - 基础错误处理已集成

### 最新完成任务 ✅

**✅ TUI 交互式问答功能（ask_followup_question）已完成** (2026-05-09)

实现了 TUI 模式下的完整交互式问答流程，现在 AI 可以通过 `ask_followup_question` 工具与用户进行多轮交互。

**实现功能**:
1. **TUI 问答界面** - 美观的问题和选项显示样式
2. **双向通信** - Agent 通过回调向 TUI 提问，TUI 通过 channel 返回答案
3. **优雅降级** - CLI 模式下自动降级到 stdin 输入
4. **处理器注入** - 在工具执行前动态注入 TUI handler
5. **Esc 中断支持** - 用户可以随时按 Esc 中断正在运行的任务

**修改的文件**:
- `internal/agent/provider.go` - 扩展 `StreamCallback` 接口，添加 `AskFollowupQuestion` 方法
- `internal/tools/interaction.go` - `AskFollowupQuestionTool` 支持处理器注入
- `internal/agent/agent.go` - 在工具执行前注入 TUI/callback handler
- `internal/ui/tui.go` - 实现完整的问答交互流程和 Esc 中断
- `internal/agent/agent_test.go` - 更新测试实现新接口

**技术实现**:
- 使用 channel 实现同步问答（`pendingReply chan string`）
- TUI 通过 `askQuestionMsg` 消息传递问题和回复通道
- Agent 阻塞等待用户回答，保证执行顺序
- 支持数字选项和自由文本输入

**额外改进**:
- 优化工具调用显示，展示完整的 input 参数（带 JSON 格式化）
- 改进 markdown 渲染的字宽处理，避免文本溢出
- 添加工具描述映射表，提供友好的工具名称展示
- Esc 键支持任务中断，通过 `context.CancelFunc` 实现

### 当前工作

**✅ TUI 和 Chat 模式响应问题已完全修复并启用工具执行**

**问题回顾**:
1. TUI 模式无响应，Chat 模式有响应
2. 工具调用参数重复累积导致JSON格式错误
3. 工具执行被禁用

**已完成的修复** (2026-05-09):

1. **修复工具文本实时发送** (`internal/agent/agent.go`)
   - 在接收到完整工具调用时立即通过callback发送
   - 不再等到流结束后批量发送
   
2. **修复工具调用参数重复累积** (`internal/api/openai.go`)
   - 发送partial chunk时创建tool call的副本
   - `toolCallCopy := *toolCalls[tc.Index]` 避免指针共享
   - 简化agent.go的processStream，不再二次累积

3. **启用工具执行** (`internal/agent/agent.go`)
   - 在agent loop中添加工具执行逻辑
   - processStream后检查并执行工具调用
   - 只有无工具调用时才SetComplete()
   - 添加OnToolCallStart和OnToolCallComplete回调通知

**测试结果**:
- ✅ 代码编译成功
- ✅ TUI模式有响应：显示 `[tool:list_files] {"path": "."}`
- ✅ 工具调用参数格式正确
- ✅ 工具执行已启用并正常工作

**架构确认**:
TUI和Chat模式始终走相同路径，问题出在实现细节上：
```
TUI 模式:  TUI → Agent.RunWithCallback(tuiCallback) → processStream() → 工具执行 → callback
Chat 模式: CLI → Agent.Run() → RunWithCallback(noopCallback) → processStream() → 工具执行
```

**之前完成的工作**:
- ✅ 添加 `PartialToolCall` 结构体跟踪流式工具调用状态
- ✅ 创建 Mock Provider 用于测试流式工具调用
- ✅ 支持 5 种测试场景: long_text, tool_call, tool_then_text, multi_tool, error

**之前完成的工作**:
- ✅ 添加 `spinner` 组件实现加载动画
- ✅ 重构流式处理使用 `CreateMessageStream` 实现真正的实时流式输出
- ✅ 添加 `streamChunkMsg` 类型处理流式消息
- ✅ 实现 `startStream` 和 `waitForStream` 方法管理流的生命周期
- ✅ 添加流式指示器 (▌) 在 AI 响应末尾显示打字效果
- ✅ 状态栏显示动态 spinner 和当前状态 ("AI is responding...", "Running: <tool>")
- ✅ 增强错误处理，错误立即显示在界面上
- ✅ 添加工具调用反馈 ("🔧 Running: <tool>")

**之前完成的工作**：
- ✅ 创建 mock 数据版本用于测试 TUI 流程
- ✅ 添加 `SetConsoleOutput` 函数禁用 TUI 模式下的控制台日志
- ✅ 修复日志输出干扰 TUI 渲染的问题
- ✅ 修复 TUI 卡死问题（Bubbletea Cmd 只能返回单个消息）
- ✅ 修复 AI 响应不显示问题（正确处理 done=true 时的 content）
- ✅ 已切换回真实 AI 调用
- ✅ TUI 调试通过，功能完整

**Phase 3 已完成**：
- ✅ Provider 接口扩展 - 添加 `CreateMessageStream` 方法
- ✅ OpenAI 流式响应支持（SSE 解析）
- ✅ Anthropic 流式响应支持
- ✅ Bubbletea TUI 框架实现
- ✅ 消息历史显示
- ✅ 输入框和状态栏
- ✅ CLI 命令集成 (`gline chat` 默认 TUI)
- ✅ Agent 初始化集成
- ✅ `gline` 命令（无参数）也启动 TUI 交互模式
- ✅ API 密钥未配置时的友好错误提示

### 使用方式

```bash
# 交互式 TUI 模式（默认）
gline
gline chat

# 单消息模式
gline chat "How do I implement a REST API in Go?"
```

### 配置 API 密钥

```bash
# 设置 OpenAI API 密钥
gline config set provider.openai.apikey YOUR_API_KEY

# 或使用环境变量
export OPENAI_API_KEY=your_key

# 设置 Anthropic API 密钥
gline config set provider.anthropic.apikey YOUR_API_KEY
export ANTHROPIC_API_KEY=your_key
```

### 最近修复 (2026-05-09)

**✅ TUI 输入框右侧边框不可见问题已修复**

- **问题**: TUI 输入框右侧边框（╮│╯）在终端中不可见
- **根本原因**: `View()` 方法中 `.Width(m.width-2)` 错误地将 lipgloss 内容宽度设为 `m.width-2`，加上 border(2) + padding(6) + margin(1) = 9 后，总宽度为 `m.width+7`，远超终端宽度，右侧边框溢出屏幕
- **修复**: 移除 `.Width(m.width-2)` 和多余的 `inner` 中间变量，直接渲染 textarea（宽度已由 `SetWidth(innerWidth)` 正确控制），总宽度恰好为 `m.width`
- **额外修复**: 测试中 glamour 渲染插入 ANSI 转义码导致 `strings.Contains` 匹配失败，添加 `stripANSI()` 辅助函数
- **修改文件**: `internal/ui/tui.go`, `internal/ui/tui_test.go`

### 下一步计划

**Phase 10a [P0]: 并发安全修复** — 必须最先做
- `cancelFn` 加 `sync.Mutex` 保护，消除 data race
- `pendingReply` 通道在 Esc/Ctrl+C 中断时正确关闭，修复 goroutine 泄漏
- `AskFollowupQuestion` 处理 channel 关闭（返回 `context.Canceled`）
- 新增 2-3 个并发安全测试，`go test -race` 通过

## 最近决策

### 技术选型决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| CLI 框架 | Cobra + Viper | Go 标准，功能完善 |
| TUI 框架 | Bubbletea | 现代 React 式模型 |
| 数据库 | SQLite | 轻量，无需外部依赖 |
| HTTP 客户端 | Resty | 链式 API，易用 |
| 日志 | Zerolog | 结构化，高性能 |

### 架构决策

1. **使用 internal/ 目录**: 明确区分公共 API 和内部实现
2. **接口驱动设计**: 便于测试和扩展
3. **模块化架构**: 各组件独立，通过接口通信
4. **状态分层**: Global/Workspace/Session 三层状态管理

## 下一步计划

### 短期目标（本周）

1. **Phase 10a: 并发安全修复 [P0]**
   - `cancelFn` 加 `sync.Mutex` 保护，消除 data race
   - `pendingReply` 通道在 Esc/Ctrl+C 时正确关闭，修复 goroutine 泄漏
   - `AskFollowupQuestion` 处理 channel 关闭
   - `go test -race ./internal/ui/...` 通过

2. **Phase 10b: 用户体验修复 [P1]**
   - 错误双重显示 → 只保留一种
   - GotoBottom → 检测 viewport.AtBottom()

### 中期目标（本月）

1. **Phase 10c: 性能与布局优化 [P2]**
   - tickMsg 无差别刷新优化
   - Header/StatusBar 合并去重
   - 灵活高度配比计算

2. **Phase 10d: 架构完整性 [P3]**
   - Tool Area 渲染逻辑迁移到 view/
   - 创建 StatusViewModel
   - messageCache 驱逐机制
   - 补充 handleKeyMsg / startAgent 测试

### 长期目标（下月）

1. **Phase 11+: 锦上添花**
   - 用户消息 Markdown 渲染
   - 输入历史（上下箭头翻阅）
   - 代码语法高亮
   - 终端主题适配
2. **高级功能**
   - 任务历史界面
   - 配置管理界面

## 开放问题

### 待决策

1. **是否支持 MCP (Model Context Protocol)**?
   - 优点：标准化工具接口
   - 缺点：增加复杂度
   - 建议：Phase 4 再考虑

2. **如何处理大文件读取**?
   - 选项 A：截断 + 提示
   - 选项 B：分页读取
   - 倾向：选项 A（与 Cline 一致）

3. **是否支持图片输入**?
   - 依赖 LLM 提供商能力
   - 建议：Phase 3 支持

### 待研究

1. **Token 计算**: 如何准确计算对话 Token 数
2. **上下文压缩**: 长对话的截断策略
3. **错误恢复**: 网络中断、API 限流处理

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.24.4
- **操作系统**: Windows 11
- **参考源码**: `./cline/` (Cline TypeScript 实现)

## 重要模式

### 代码组织原则

1. **接口优先**: 先定义接口，再实现
2. **依赖注入**: 通过构造函数注入依赖
3. **错误处理**: 使用 `fmt.Errorf` 包装错误
4. **上下文传递**: 所有异步操作接受 `context.Context`

### 命名约定

- **包名**: 小写，简短（`agent`, `tools`）
- **接口名**: 名词（`Provider`, `Tool`）
- **实现名**: 接口名 + 后缀（`AnthropicProvider`）
- **函数名**: 动词开头（`CreateMessage`, `Execute`）

## 参考资源

- [Cline 源码](./cline/) - 架构参考
- [Bubbletea 文档](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Cobra 文档](https://github.com/spf13/cobra) - CLI 框架
- [Anthropic API](https://docs.anthropic.com/) - Claude API 文档
