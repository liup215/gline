# Progress

## 项目状态概览

**当前阶段**: MVVM 重构全部完成 ✅

**总体进度**: 75% - TUI MVVM 架构重构和优化已全部完成（Phase 1-10），测试覆盖 110+，全部通过。
```

## 已完成工作

### 1. Cline 源码分析 ✅

**时间**: 2026-05-08

**成果**:
- 分析了 Cline 的核心架构
- 理解了 Agent 工作流程（Plan/Act 模式）
- 提取了工具系统设计
- 研究了 LLM 集成方式
- 分析了 CLI 实现特点

**关键发现**:
- Controller 管理任务生命周期
- Task 模块处理 Agent 循环
- Prompts 系统管理 23+ 工具定义
- API 层支持 40+ LLM 提供商
- CLI 使用 React Ink 构建 TUI

### 2. 技术选型 ✅

**时间**: 2026-05-08

**决策**:

| 组件 | 选择 | 状态 |
|------|------|------|
| CLI 框架 | Cobra + Viper | ✅ 确定 |
| TUI 框架 | Bubbletea | ✅ 确定 |
| HTTP 客户端 | Resty | ✅ 确定 |
| 数据库 | SQLite | ✅ 确定 |
| 日志 | Zerolog | ✅ 确定 |

### 3. 架构设计 ✅

**时间**: 2026-05-08

**成果**:
- 定义了模块划分（agent, api, tools, prompts, storage, ui, config）
- 设计了核心接口（Agent, Provider, Tool）
- 规划了目录结构
- 定义了状态管理策略
- 设计了 Agent 循环模式

### 4. Memory Bank 初始化 ✅

**时间**: 2026-05-08

**已创建**:
- ✅ `projectbrief.md` - 项目简介和目标
- ✅ `productContext.md` - 产品上下文和用户场景
- ✅ `systemPatterns.md` - 系统架构模式
- ✅ `techContext.md` - 技术栈和配置
- ✅ `activeContext.md` - 当前上下文和计划
- ✅ `progress.md` - 本文件

### 5. Phase 1: 基础框架 ✅

**时间**: 2026-05-08
**状态**: 已完成

- [x] 创建目录结构
- [x] 初始化 go.mod
- [x] 添加基础依赖 (Cobra, Viper, Zerolog)
- [x] 实现 CLI 命令结构
- [x] 实现配置管理 (三层配置：workspace > global > env)
- [x] 实现日志系统 (结构化日志，多级别，彩色输出)

**实现的功能**:
- `gline` - 启动交互式模式
- `gline chat <message>` - 单次对话
- `gline config` - 配置管理子命令 (get, set, list, path)
- `gline version` - 版本信息
- `gline --help` - 帮助信息
- `gline -v` - 详细输出模式

### 6. Phase 0: 项目初始化 ✅

**时间**: 2026-05-08
**状态**: 已完成

- [x] 创建最小可运行项目结构
  - [x] `cmd/gline/main.go` - CLI 入口
  - [x] `internal/version/version.go` - 版本管理
- [x] 创建 Makefile 支持本地和跨平台构建
- [x] 配置 GitHub Actions CI/CD
  - [x] `.github/workflows/build.yml` - 持续集成
  - [x] `.github/workflows/release.yml` - 自动发布
- [x] 本地构建测试通过
- [x] 支持 5 个目标平台
  - [x] macOS (amd64, arm64)
  - [x] Linux (amd64, arm64)
  - [x] Windows (amd64)

### 7. Phase 2: 核心模块 ✅

**时间**: 2026-05-08
**状态**: 已完成

- [x] Agent 接口定义 (`internal/agent/agent.go`)
  - Agent 核心接口 (Run, SetMode, GetMode, Abort, etc.)
  - BaseAgent 实现
  - Plan/Act 模式定义
- [x] Provider 接口定义 (`internal/agent/provider.go`)
  - Provider 接口 (CreateMessage, SupportsTools, etc.)
  - MessageRequest/MessageResponse 类型
  - ToolCall/ToolDefinition 类型
- [x] Tool 接口定义 (`internal/tools/tool.go`)
  - Tool 接口 (Name, Description, InputSchema, Execute)
  - ToolInfo 元数据
  - 常用 JSON Schema 定义
- [x] 基础 Agent 循环 (`internal/agent/agent.go`)
  - 消息处理循环
  - 工具调用处理
  - 错误处理和重试
- [x] Plan/Act 模式切换
  - SetMode/GetMode 方法
  - 工具权限过滤
- [x] 工具注册表 (`internal/tools/registry.go`)
  - 工具注册/注销
  - 模式过滤
  - 线程安全
- [x] 基础工具实现 (10个工具)
  - 文件操作: read_file, write_to_file, replace_in_file, list_files
  - 命令执行: execute_command
  - 搜索工具: search_files, list_code_definition_names
  - 交互工具: ask_followup_question, attempt_completion, plan_mode_respond
- [x] Anthropic Provider 实现 (`internal/api/anthropic.go`)
  - Claude API 集成
  - 工具调用支持
  - 流式响应准备

**项目结构**:
```
gline/
├── internal/
│   ├── agent/          # Agent 核心 (agent.go, provider.go)
│   ├── api/            # LLM Provider (anthropic.go, registry.go)
│   ├── tools/          # 工具系统 (10个工具实现)
│   ├── prompts/        # 系统提示词
│   ├── config/         # 配置管理
│   ├── log/            # 日志系统
│   └── version/        # 版本管理
└── pkg/
    └── types/          # 共享类型 (message.go)
```

## 进行中工作

暂无

## 已完成工作

### Phase 3: LLM 集成 ✅

**时间**: 2026-05-08
**状态**: 已完成

- [x] OpenAI Provider 实现 (通用 OpenAI 兼容 Provider)
  - [x] 支持 OpenAI 官方 API
  - [x] 支持自定义 base_url (OpenRouter, DashScope, 本地模型等)
  - [x] 支持工具调用
  - [x] 完整的错误处理
  - [x] 单元测试覆盖
- [x] 流式响应处理
  - [x] Provider 接口扩展 - 添加 `CreateMessageStream` 方法
  - [x] OpenAI 流式响应支持 (SSE 解析)
  - [x] Anthropic 流式响应支持 (SSE 解析)
- [x] TUI 交互式界面
  - [x] Bubbletea TUI 框架实现 (`internal/ui/tui.go`)
  - [x] 消息历史显示区域
  - [x] 输入框组件
  - [x] 状态栏 (显示 Provider/模型/模式)
  - [x] Plan/Act 模式切换 (Tab 键)
  - [x] 快捷键支持 (Ctrl+C 退出, Ctrl+L 清屏)
- [x] CLI 命令集成
  - [x] `gline` 命令默认启动 TUI 交互模式
  - [x] `gline chat` 默认启动 TUI 交互模式
  - [x] `gline chat "message"` 单消息模式
  - [x] Agent 自动初始化
  - [x] 支持 OpenAI 和 Anthropic Provider
- [x] 错误处理增强
  - [x] API 密钥未配置时的友好错误提示
  - [x] 流式响应错误处理
  - [x] TUI 友好的错误显示

### Phase 4: UI 层

**优先级**: 中
**预计时间**: 2-3 周
**状态**: 已完成 (基础 TUI)

- [x] TUI 基础框架
- [x] 纯文本模式
- [x] 交互式对话
- [ ] 任务历史界面 (待实现)

### Phase 5: 高级功能

**优先级**: 低
**预计时间**: 2-4 周

- [ ] 任务历史管理
- [ ] 配置管理界面
- [ ] 多 Provider 支持
- [ ] 性能优化

## 已知问题

### 架构债务（已解决）✅

**TUI 层架构混乱** — 通过 MVVM 重构已解决 ✅
- **问题描述**: `internal/ui/` 下 TUI Model 是上帝对象，混合 UI 状态、业务状态、Agent 胶水代码
- **影响**: 可测试性差、View 渲染性能差、添加新功能需改动 4-5 个文件
- **解决方案**: 10 阶段渐进 MVVM 重构（Phase 1-5 架构重构 + Phase 6-10 优化）
- **重构后架构**:
  ```
  internal/ui/
  ├── model/         # Domain Model（纯数据，零依赖）
  ├── viewmodel/     # ViewModel（派生状态 + 渲染缓存）
  ├── view/          # View（纯渲染函数）
  ├── bridge/        # Agent-TUI 桥接层（类型安全消息）
  └── *.go           # Bubbletea 薄壳
  ```
- **当前状态**: 全部 10 Phase 已完成 ✅（2026-05-14）
- **测试覆盖**: 110+ 测试全部通过

### TUI 优化进度汇总

| Phase | 内容 | 状态 | 日期 |
|-------|------|------|------|
| Phase 1-5 | MVVM 架构重构 | ✅ 完成 | 2026-05-11 |
| Phase 6-10 | 渲染性能/并发安全/架构完整性 | ✅ 完成 | 2026-05-14 |

**详细技术文档**: 见 `memory-bank/tui-mvvm-refactor.md` 和 `memory-bank/tui-optimization-plan.md`

### 已修复问题 ✅

1. **TUI 模式下工具调用卡死** ✅
   - **问题**: 
     - TUI 模式下发送工具调用后界面卡死
     - 工具调用只被显示但没有被执行
     - TUI 直接调用 Provider API，绕过了 Agent 的 `Run` 方法
   - **修复**: 
     - 重构架构：TUI 和 CLI 模式都通过 `Agent.RunWithCallback()` 运行
     - 添加 `StreamCallback` 接口，Agent 通过回调通知 UI 更新
     - 在 `processStream()` 中统一处理流式响应和工具执行
     - TUI 大幅简化，只负责显示回调内容
   - **文件**: `internal/agent/agent.go`, `internal/agent/provider.go`, `internal/ui/tui.go`

2. **TUI 和 Chat 模式响应不一致** ✅
   - **问题**:
     - TUI 模式发送消息后界面无响应（工具调用被禁用debug模式）
     - Chat 模式有正常响应，能显示工具调用信息
     - 两种模式理论上走相同的 Agent.RunWithCallback 路径
   - **根本原因** (对比Cline源码发现):
     - 工具文本发送时机太晚：在流结束后才发送，LLM只返回tool_call时TUI看到空响应
     - 工具调用信息被清空：`typesToolCalls = nil` 导致信息丢失
     - TUI缺少实时反馈：Cline立即通过say发送，gline只在最后发送
   - **修复**:
     - 在 `processStream()` 接收到完整工具调用时立即发送工具文本
     - 保留 `typesToolCalls` 信息，不再清空
     - 统一TUI和Chat模式的体验，都能实时看到工具意图
   - **文件**: `internal/agent/agent.go`
   - **日期**: 2026-05-09

3. **工具调用参数重复累积问题** ✅
   - **问题**:
     - OpenAI provider发送工具调用时使用指针引用
     - agent.go的processStream又对同一指针累积参数
     - 导致参数被重复累加（如`{"{"path{"{"path":...`）
   - **修复**:
     - 在 `openai.go` 发送partial chunk时创建tool call的副本
     - `toolCallCopy := *toolCalls[tc.Index]` 避免指针共享
     - 简化agent.go的processStream，不再二次累积
   - **文件**: `internal/api/openai.go`, `internal/agent/agent.go`
   - **日期**: 2026-05-09

4. **工具执行被禁用问题** ✅
   - **问题**:
     - processStream直接调用SetComplete()阻止工具执行
     - 工具调用逻辑在processResponse中但该方法未被调用
   - **修复**:
     - 在agent loop中添加工具执行逻辑
     - processStream后检查并执行工具调用
     - 只有无工具调用时才SetComplete()
     - 添加OnToolCallStart和OnToolCallComplete回调通知
   - **文件**: `internal/agent/agent.go`
   - **日期**: 2026-05-09

2. **TUI 流式输出优化** ✅
- [TRUNCATED FOR BREVITY — ORIGINAL CONTENT PRESERVED]

## 最近变更

### 2026-05-14 — TUI 架构优化完成 ✅

TUI MVVM 架构重构和优化已全部完成。

### 2026-05-09 — TUI 交互式问答功能完成 ✅

TUI 支持 `ask_followup_question` 工具交互。

### 2026-05-09 — TUI 输入框修复完成 ✅

输入框右侧边框显示问题已修复。

## 里程碑

**解决方案**: 引入 `MessageType` 语义类型 + `Meta` 结构化元数据

**核心设计**:
```
Message {
    Role:      RoleSystem
    Content:   "Error: connection refused"
    MsgType:   TypeError        // 是什么（语义）
    Strategy:  StrategyPlain    // 怎么渲染（表现）
    Meta:      {...}            // 结构化元数据（可选）
}
```

**新增文件**:
- `pkg/types/message_type.go`: 常量定义 (TypeError, TypeQuestion, TypeToolStart, TypeToolComplete, TypeNormal)
- `internal/ui/model/meta.go`: ErrorMeta/ToolMeta 结构体 + AsErrorMeta/AsToolMeta/SetMeta 方法

**修改文件**:
- `internal/ui/model/message.go`: 添加 MsgType 和 Meta 字段
- `internal/ui/tui.go`: 错误消息设置 TypeError，问答消息设置 TypeQuestion
- `internal/ui/tui_state.go`: 工具消息设置 TypeToolStart/TypeToolComplete + ToolMeta
- `internal/ui/viewmodel/conversation_vm.go`: 优先按 MsgType 渲染，保留 Strategy 和硬编码作为 fallback

**渲染优先级**:
1. MsgType (语义类型): Error → 红色样式, Question → 带选项, Tool → 工具样式
2. Strategy (渲染策略): Markdown → Glamour, JSON → 代码块
3. Fallback (向后兼容): 字符串前缀检测

**改进效果**:
| 场景 | 之前 | 之后 |
|------|------|------|
| 错误消息 | `strings.HasPrefix("Error:")` | `MsgType == TypeError` |
| 元数据 | 无 | ErrorMeta{Code, Retryable, Stack} |
| 添加新类型 | 修改 ViewModel | 添加常量 + 创建消息时设置 |
| 渲染控制 | 硬编码 | 声明式 (MsgType + Strategy) |

**测试结果**: ✅ 全部测试通过

**提交**: fe68d9d refactor(ui): add MessageType for semantic message classification

### 2026-05-14 — TUI 工具渲染重构：消除硬编码 (已完成)

**问题**: TUI 中工具输出渲染存在大量硬编码判断，如 `view.NormalizeToolName(msg.Name) == "attempt_completion"`，导致：
- 新增工具需要修改多处代码
- 不同工具的 Markdown/纯文本渲染逻辑分散
- 工具输出格式化与 TUI 核心逻辑耦合

**解决方案**: 实现工具自描述渲染接口 (`tool.Renderer`)

**核心设计**:
```
internal/ui/tool/
├── render.go              # Renderer 接口定义
├── registry.go            # 工具注册表
├── attempt_completion.go  # attempt_completion 专用渲染器
├── ask_followup_question.go # ask_followup_question 渲染器
├── plan_mode_respond.go   # plan_mode_respond 渲染器
├── read_file.go            # read_file 渲染器
└── default.go              # 通用工具默认渲染器
```

**实现内容**:
1. **常量定义** (`pkg/types/`)
   - `tool_phases.go` - `ToolPhase` 常量 (Start/Complete)
   - `tool_names.go` - `ToolName` 常量 (所有工具名称)
   - `render_strategy.go` - `RenderStrategy` 常量 (Plain/Markdown/JSON/Special/Skip)

2. **工具渲染接口** (`internal/ui/tool/render.go`)
   - `Renderer` 接口: `Render(req) RenderResult`, `Name()`, `Description()`, `Icon()`
   - `RenderRequest` 结构: Phase, Input, Status
   - `RenderResult` 结构: Content, Role, Strategy, Skip

3. **工具实现**
   - `AttemptCompletionRenderer` - Start阶段创建 Assistant 消息，使用 Markdown 渲染
   - `AskFollowupQuestionRenderer` - 返回 Skip，由外部特殊处理
   - `PlanModeRespondRenderer` - Complete阶段创建 Assistant 消息，使用 Markdown 渲染
   - `ReadFileRenderer` - System 消息，纯文本渲染
   - `DefaultRenderer` - 通用渲染器，适用于所有标准工具

4. **注册表** (`internal/ui/tool/registry.go`)
   - `NewDefaultRegistry()` - 预注册所有内置工具
   - `Get(name)` - 根据工具名获取渲染器
   - `NormalizeToolName()` - camelCase 转 snake_case

5. **TUI 集成** (`internal/ui/tui.go`, `tui_state.go`)
   - Model 添加 `toolRegistry *tool.Registry` 字段
   - `handleAgentToolStart/Complete` 使用渲染器创建消息
   - 消息携带 `Strategy` 字段，ViewModel 据此选择渲染方式

6. **ViewModel 更新** (`internal/ui/viewmodel/conversation_vm.go`)
   - `renderSystemMessage` 根据 `msg.Strategy` 选择渲染方式
   - `StrategyMarkdown` 使用 Glamour 渲染
   - 保留向后兼容的硬编码检测作为 fallback

**测试结果**:
- ✅ 所有 UI 包测试通过
- ✅ 编译成功，无错误
- ✅ `attempt_completion` 输出自动 Markdown 美化
- ✅ 常规工具保持纯文本显示

**改进效果**:
| 指标 | 之前 | 之后 |
|------|------|------|
| 硬编码工具名 | 多处字符串比较 | 常量定义 |
| 添加新工具 | 修改 TUI 核心逻辑 | 实现 Renderer 并注册 |
| 渲染策略控制 | 分散在各处 | 集中由工具决定 |
| 测试覆盖 | 需要集成测试 | 每个 Renderer 可单元测试 |

**修改文件**:
- `pkg/types/tool_phases.go` - 新增
- `pkg/types/tool_names.go` - 新增
- `pkg/types/render_strategy.go` - 新增
- `internal/ui/tool/render.go` - 新增
- `internal/ui/tool/registry.go` - 新增
- `internal/ui/tool/attempt_completion.go` - 新增
- `internal/ui/tool/ask_followup_question.go` - 新增
- `internal/ui/tool/plan_mode_respond.go` - 新增
- `internal/ui/tool/read_file.go` - 新增
- `internal/ui/tool/default.go` - 新增
- `internal/ui/tui.go` - 添加 toolRegistry 字段和初始化
- `internal/ui/tui_state.go` - 使用渲染器替代硬编码逻辑
- `internal/ui/viewmodel/conversation_vm.go` - 根据 Strategy 渲染
- `internal/ui/model/message.go` - 添加 Strategy 字段

**日期**: 2026-05-14

### 2026-05-09 — TUI 交互式问答功能（ask_followup_question）(已完成)
- **功能**: 实现 TUI 模式下的完整交互式问答流程
- **实现内容**:
  1. **StreamCallback 接口扩展** - 添加 `AskFollowupQuestion(question string, options []string) (string, error)` 方法
  2. **AskFollowupQuestionTool 处理器注入** - 支持通过 `SetHandler` 注入 TUI handler
  3. **Agent 动态注入** - 在工具执行前检测并注入回调处理器
  4. **TUI 问答界面** - 实现问题显示、选项样式、答案输入和回复机制
  5. **Esc 中断支持** - 用户可以按 Esc 键中断正在运行的任务（通过 cancelFn）
- **技术要点**:
  - 使用 `chan string` 实现同步问答（Agent 阻塞等待用户回答）
  - TUI 通过 `askQuestionMsg` 传递问题和回复通道
  - `pendingReply` 字段跟踪待回复状态，Enter 键提交答案
  - CLI 模式自动降级到 stdin 输入（`StreamCallbackAdapter`）
- **额外改进**:
  - 工具调用显示完整 input 参数（JSON 格式化）
  - 优化 glamour markdown 渲染字宽（避免溢出）
  - 添加工具描述映射表和格式化辅助函数
  - 改进帮助文本，添加 "esc: interrupt" 提示
- **修改文件**:
  - `internal/agent/provider.go` - StreamCallback 接口扩展
  - `internal/tools/interaction.go` - 处理器注入支持
  - `internal/agent/agent.go` - 动态注入和中断支持
  - `internal/ui/tui.go` - 问答界面和中断逻辑
  - `internal/agent/agent_test.go` - 测试更新
- **日期**: 2026-05-09

### 2026-05-09 — TUI 输入框右侧边框修复 (已完成)
- **问题**: TUI 输入框右侧边框（╮│╯）在终端中不可见
- **根本原因**: `View()` 中 `.Width(m.width-2)` 设置 lipgloss 内容宽度为 `m.width-2`，加上 border+padding+margin 后总宽度为 `m.width+7`，右侧溢出屏幕
- **修复**:
  - 移除 `.Width(m.width-2)` 和多余的 `inner` 中间变量
  - textarea 宽度已由 `SetWidth(innerWidth)` 正确控制，直接渲染即可
  - 总宽度 = 1(margin) + 2(border) + 6(padding) + (m.width-9)(内容) = m.width
- **额外修复**:
  - 测试中添加 `stripANSI()` 辅助函数，解决 glamour 渲染 ANSI 转义码导致 `strings.Contains` 匹配失败的问题
  - 修复 2 个已有的测试失败: `TestContentUpdateSurvivesToolStatus`, `TestToolHistoryDoesNotPushContent`
- **修改文件**:
  - `internal/ui/tui.go` — 移除 `.Width(m.width-2)` 和 `inner` 变量
  - `internal/ui/tui_test.go` — 添加 `stripANSI()` 辅助函数，更新视图断言
- **日期**: 2026-05-09

### 2026-05-09 — reasoning_content persist & current-turn re-attach (已完成)
- **问题**: 
  - 报错: "ERR Agent error: OpenAI API error: The `reasoning_content` in the thinking mode must be passed back to the API."
  - 一些模型在思考（thinking）模式会输出 `reasoning_content`，需要在同一对话回合内将该字段回传给模型。
- **修复**:
  - Persist reasoning_content from streaming and non-stream responses into assistant messages.
  - When building provider requests, attach persisted assistant message ReasoningContent only for assistant messages that appear after the last user message (current turn).
  - Parse SSE stream deltas for `reasoning_content` and emit StreamChunk/MessageResponse carrying ReasoningContent.
  - Accumulate streaming reasoning fragments and save them onto the final assistant message in the conversation.
  - Avoid sending stale reasoning_content across turns.
- **修改文件**:
  - pkg/types/message.go
  - internal/agent/provider.go
  - internal/agent/agent.go
  - internal/api/openai.go
- **提交建议**:
  - Commit message: \"Persist reasoning_content; parse SSE reasoning chunks; attach reasoning_content for current turn\"
- **日期**: 2026-05-09

## 里程碑