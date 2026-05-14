# Active Context

## 当前焦点

### TUI 工具渲染重构 ✅ 已完成 (2026-05-14)

**已完成**: 重构 TUI 工具输出渲染系统，消除硬编码判断，实现工具自描述渲染。

**设计**:
```
┌─────────────────────────────────────────────────────────────┐
│                         TUI                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │
│  │   Model     │◄───│  ViewModel  │◄───│   tool.Registry │  │
│  │  (消息数据)  │    │  (渲染逻辑)  │    │  (工具渲染器注册表)│  │
│  └─────────────┘    └─────────────┘    └─────────────────┘  │
│         ▲                                          ▲         │
│         │                                          │         │
│    创建消息                                    查找渲染器    │
│         │                                          │         │
│         └──────────────────────────────────────────┘         │
│                    tool.Renderer.Render()                   │
│                            ▲                               │
└────────────────────────────┼───────────────────────────────┘
                             │
┌────────────────────────────┼───────────────────────────────┐
│                      工具层   │                               │
│  ┌─────────────────────────┴─────────────────────────────┐   │
│  │                    tool.Renderer 接口                  │   │
│  │  Render(req) RenderResult                             │   │
│  │  Name() types.ToolName                                 │   │
│  └───────────────────────────────────────────────────────┘   │
│         ▲              ▲              ▲                     │
│         │              │              │                     │
│    ┌────┴────┐   ┌────┴────┐   ┌────┴────┐                │
│    │Attempt  │   │ReadFile │   │  Ask    │                │
│    │Completion│   │  Tool   │   │Question │                │
│    │Renderer │   │Renderer │   │Renderer │                │
│    └─────────┘   └─────────┘   └─────────┘                │
└─────────────────────────────────────────────────────────────┘
```

**关键改进**:
- ✅ 常量定义: `ToolPhase`, `ToolName`, `RenderStrategy`
- ✅ 工具接口: `Renderer` 接口让工具自描述渲染方式
- ✅ 注册表: 集中管理所有工具渲染器
- ✅ 策略渲染: ViewModel 根据 `Strategy` 字段选择渲染方式
- ✅ 消除硬编码: 不再使用字符串比较判断工具类型

**新增文件**:
- `pkg/types/tool_phases.go`
- `pkg/types/tool_names.go`
- `pkg/types/render_strategy.go`
- `internal/ui/tool/render.go`
- `internal/ui/tool/registry.go`
- `internal/ui/tool/attempt_completion.go`
- `internal/ui/tool/ask_followup_question.go`
- `internal/ui/tool/plan_mode_respond.go`
- `internal/ui/tool/read_file.go`
- `internal/ui/tool/default.go`

**修改文件**:
- `internal/ui/tui.go` - 添加 toolRegistry 字段
- `internal/ui/tui_state.go` - 使用渲染器创建消息
- `internal/ui/viewmodel/conversation_vm.go` - 根据 Strategy 渲染
- `internal/ui/model/message.go` - 添加 Strategy 字段

**测试结果**: 全部测试通过 ✅

### TUI 架构 ✅ 已完成

TUI MVVM 重构（Phase 1-5）和优化（Phase 6-10）已全部完成。详细技术文档见 `memory-bank/tui-mvvm-refactor.md` 和 `memory-bank/tui-optimization-plan.md`。

**完成状态**:
- ✅ Phase 1-5: MVVM 架构重构（类型安全消息系统、纯数据 Model、ViewModel 层、Bridge 解耦、View 纯函数化）
- ✅ Phase 6-10: TUI 优化（增量渲染、Handler 解耦、工具格式化、Model 净化、并发安全、性能布局、架构完整性）

**重构后架构**:
```
internal/ui/
├── model/         # Domain Model（纯数据，零依赖）
├── viewmodel/     # ViewModel（派生状态 + 渲染缓存）
├── view/          # View（纯渲染函数）
├── bridge/        # Agent-TUI 桥接层（类型安全消息）
├── tui.go         # Bubbletea 薄壳（Init/Update/View）
├── tui_agent.go   # startAgent() 集成
├── tui_input.go   # 键盘/窗口处理
├── tui_state.go   # 状态变更 handler
├── tui_update.go  # AgentEvent 分发器
└── tui_test.go    # 集成测试
```

**测试覆盖**: 110+ 测试全部通过

### 下一步计划

**Phase 11+: 锦上添花（可选）**
- 用户消息 Markdown 渲染
- 输入历史（上下箭头翻阅）
- 代码语法高亮
- 终端主题适配
- 任务历史界面
- 配置管理界面

## 已完成任务 ✅
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

TUI 优化 Phase 1-10 已全部完成。当前进入功能完善阶段：

### 中期目标（本月）

**Phase 11+: 锦上添花（可选）**
- 用户消息 Markdown 渲染
- 输入历史（上下箭头翻阅）
- 代码语法高亮
- 终端主题适配

### 长期目标（下月）

1. **高级功能**
   - 任务历史界面
   - 配置管理界面
2. **MCP 支持**

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
