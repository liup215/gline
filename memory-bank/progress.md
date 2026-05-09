# Progress

## 项目状态概览

**当前阶段**: Phase 2 核心模块已完成

**总体进度**: 35% - 完成核心模块，准备进入 LLM 集成

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
- [x] Provider 注册表 (`internal/api/registry.go`)
  - Provider 工厂模式
  - 动态注册
- [x] 系统提示词管理 (`internal/prompts/system.go`)
  - Plan/Act 模式提示词
  - 工具描述生成

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
├── pkg/
│   └── types/          # 共享类型 (message.go)
└── cmd/gline/          # CLI 入口
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
   - **问题**: 
     - TUI 模式下是非流式输出，运行长任务时没有任何提示
     - 界面没有动效，用户体验差
     - 错误不能尽快在 TUI 界面上返回
   - **修复**: 
     - 添加 `spinner` 组件实现加载动画
     - 重构 `processMessageStream` 使用 `CreateMessageStream` 实现真正的流式输出
     - 添加 `streamChunkMsg` 类型处理流式消息
     - 实现 `startStream` 和 `waitForStream` 方法管理流的生命周期
     - 添加流式指示器 (▌) 在 AI 响应末尾显示打字效果
     - 状态栏显示动态 spinner 和当前状态 ("AI is responding...", "Running: <tool>")
     - 增强错误处理，错误立即显示在界面上
     - 添加工具调用反馈 ("🔧 Running: <tool>")
   - **文件**: `internal/ui/tui.go`

2. **TUI 流式工具调用显示** ✅
   - **问题**: 
     - 工具调用参数在流式响应中逐步构建时，TUI 无法实时显示
     - 用户无法看到工具调用的实时进度
     - 缺乏类似 Cline 的 ⏺ 符号和闪烁效果
   - **修复**: 
     - 添加 `PartialToolCall` 结构体跟踪流式工具调用状态
     - 在 `Model` 中添加 `partialToolCall` 字段
     - 修改 `streamChunkMsg` 处理逻辑，区分 `IsPartial` 和完整工具调用
     - 部分工具调用时累积参数并实时更新视图
     - 完整工具调用时添加到消息历史并显示系统消息
     - 在 `updateViewport` 中添加部分工具调用显示逻辑
     - 显示格式: `⏺ tool_name({"param": "value"...})` 带闪烁效果
     - 添加 `toolPartialStyle` 和 `toolIndicatorStyle` 样式
     - 创建 Mock Provider 用于测试流式工具调用
     - 支持 5 种测试场景: long_text, tool_call, tool_then_text, multi_tool, error
   - **文件**: `internal/ui/tui.go`, `internal/api/mock.go`, `cmd/gline/chat.go`

3. **系统提示词未传递给 LLM** ✅
   - **问题**: 
     - Agent 运行时没有将系统提示词传递给 LLM
     - TUI 模式下也没有传递系统提示词和工具
     - LLM 不知道有哪些工具可用
     - 用户提示 "use a tool test" 时 AI 回复 "I don't have access to any tools"
   - **修复**: 
     - 在 `internal/agent/agent.go` 中导入 `prompts` 包
     - 在 `Run` 方法中构建系统提示词并设置到 `MessageRequest.SystemPrompt`
     - 根据当前模式（Plan/Act）获取对应的工具描述
     - 调用 `prompts.GetSystemPrompt()` 生成完整的系统提示词
     - 在 `internal/ui/tui.go` 中导入 `prompts` 包
     - 在 `startStream` 方法中构建系统提示词和工具列表
     - 添加 `GetToolRegistry()` 方法到 `BaseAgent` 供 TUI 使用
     - 确保 TUI 模式和非 TUI 模式都传递系统提示词
   - **文件**: `internal/agent/agent.go`, `internal/ui/tui.go`

2. **OpenAI Provider 404 错误** ✅
   - **问题**: `defaultOpenAIURL` 常量设置为完整端点 URL，但代码直接使用它作为请求 URL，导致 404 错误
   - **修复**: 将 `defaultOpenAIURL` 改为 `defaultOpenAIBaseURL` (基础 URL)，并在请求时拼接 `/chat/completions` 路径
   - **文件**: `internal/api/openai.go`

2. **环境变量名称不一致** ✅
   - **问题**: 代码中使用 `OPENAI_API_KEY` 和 `ANTHROPIC_API_KEY`，但配置系统绑定的是 `GLINE_OPENAI_API_KEY` 和 `GLINE_ANTHROPIC_API_KEY`
   - **修复**: 统一使用 `GLINE_*` 前缀的环境变量名称
   - **文件**: `cmd/gline/chat.go`, `cmd/gline/root.go`

3. **DashScope API 兼容性问题** ✅
   - **问题**: 
     - `buildFullURL` 函数已添加但未在 `CreateMessage` 和 `CreateMessageStream` 中使用，导致 URL 拼接问题
     - SSE 流式响应解析缺少调试日志，无法诊断问题
     - 程序卡在 "Processing your request..." 没有响应
   - **修复**: 
     - 在 `CreateMessage` (第 274 行) 和 `CreateMessageStream` (第 492 行) 中使用 `buildFullURL(p.baseURL)`
     - 添加 SSE 调试日志，记录每行接收的数据和解析错误
     - 导入 `github.com/liup215/gline/internal/log` 包
   - **文件**: `internal/api/openai.go`

4. **TUI Provider/Model 显示为 "-"** ✅
   - **问题**: 
     - TUI 状态栏显示 `Provider: - | Model: -`
     - `New` 函数创建 Model 时没有从 Agent 获取 Provider 和 Model 信息
   - **修复**: 
     - 在 `BaseAgent` 中添加 `GetProvider()` 方法
     - 在 TUI `New` 函数中调用 `agentInstance.GetProvider()` 获取 Provider 和 Model 信息
   - **文件**: `internal/agent/agent.go`, `internal/ui/tui.go`

5. **Agent 无限循环问题** ✅
   - **问题**: 
     - 程序卡在 "Processing your request..." 没有响应
     - Agent 的 `Run` 方法无限循环，不断发送 API 请求
     - `processResponse` 中没有在没有工具调用时标记对话为完成
   - **修复**: 
     - 在 `processResponse` 中添加检查：如果没有工具调用，调用 `a.conversation.SetComplete()`
   - **文件**: `internal/agent/agent.go`

6. **TUI 阻塞问题** ✅
   - **问题**: 
     - TUI 模式下发送消息后界面卡死
     - `processMessage` 是同步执行的，阻塞了 Bubbletea 主循环
     - TUI 无法接收 Agent 的响应
   - **修复**: 
     - 将 `processMessage` 改为异步执行，使用 goroutine 和 channel
     - 通过 channel 发送响应结果，Bubbletea 可以正确处理
   - **文件**: `internal/ui/tui.go`

### 技术问题

1. **SQLite CGO**: Windows 上需要 GCC 编译器
   - **状态**: 待解决
   - **方案**: 考虑使用 `modernc.org/sqlite` 纯 Go 实现

2. **Bubbletea Windows 支持**: 某些终端可能不完全支持
   - **状态**: 待验证
   - **方案**: 提供纯文本模式作为备选

### 设计问题

1. **Token 计算**: 需要研究如何准确计算
   - **状态**: 待研究
   - **方案**: 参考 tiktoken 或类似库

2. **上下文压缩**: 长对话处理策略
   - **状态**: 待设计
   - **方案**: 参考 Cline 的截断策略

## 里程碑

| 里程碑 | 目标日期 | 状态 |
|--------|----------|------|
| 架构设计完成 | 2026-05-08 | ✅ 完成 |
| 基础框架可用 | 2026-05-08 | ✅ 完成 |
| Agent 核心可用 | 2026-05-08 | ✅ 完成 |
| LLM 集成完成 | 2026-06-12 | ⏳ 计划中 |
| Alpha 版本 | 2026-06-26 | ⏳ 计划中 |
| Beta 版本 | 2026-07-26 | ⏳ 计划中 |
| v1.0 发布 | 2026-08-26 | ⏳ 计划中 |

## 变更日志

### 2026-05-08
- 初始化项目
- 完成 Cline 源码分析
- 完成技术选型
- 完成架构设计
- 创建 Memory Bank
- 完成 Phase 1: 基础框架 (CLI, 配置, 日志)
- 完成 Phase 2: 核心模块 (Agent, Provider, Tool 接口和实现)
  - 实现 Agent 核心循环和模式管理
  - 实现工具系统 (10个基础工具)
  - 实现 Anthropic Provider
  - 实现系统提示词管理
- **Phase 3 进展**: 接入通用 OpenAI Provider
  - 创建 `internal/api/openai.go` - OpenAI 兼容 Provider
  - 支持任意 OpenAI API 兼容服务 (OpenAI, OpenRouter, DashScope, Ollama 等)
  - 配置支持 `url`, `key`, `model` 三个参数
  - 更新 Provider 注册表支持 openai
  - 更新配置系统支持 `base_url` 配置
  - 添加完整单元测试

## 资源

- **源码**: https://github.com/liup215/gline
- **参考**: ./cline/ (Cline TypeScript 实现)
- **文档**: ./memory-bank/
