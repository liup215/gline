# Progress

## 项目状态概览

**当前阶段**: MVVM 重构全部完成 ✅

**总体进度**: 65% - 核心架构已完成（Agent、Provider、Tools、Storage），已迁移到 Wails GUI 桌面应用。前端核心功能基本完成。
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
- [x] GUI 桌面应用 (Wails v3)
  - [x] Wails v3 框架集成 (`gui/`)
  - [x] Backend 服务 (`gui/backend.go`)
  - [x] ChatService 事件绑定 (`gui/chat_service.go`)
  - [x] 前端资源嵌入 (`frontend/dist`)
- [x] CLI 命令集成（保留）
  - [x] `gline` 命令默认启动 GUI 桌面应用
  - [x] `gline chat` 支持 CLI 对话模式
  - [x] `gline chat "message"` 单消息模式
  - [x] Agent 自动初始化
  - [x] 支持 OpenAI 和 Anthropic Provider
- [x] 错误处理增强
  - [x] API 密钥未配置时的友好错误提示
  - [x] 流式响应错误处理
  - [x] TUI 友好的错误显示

### Phase 2: GUI 历史任务界面 ✅

**优先级**: 中
**时间**: 2026-05-24
**状态**: 已完成

- [x] 历史任务 Backend 接口 (`gui/backend.go`)
  - `ListTasks(limit, offset)` — 分页查询任务历史
  - `GetTaskSummary(id)` — 获取任务详情和消息记录
  - `DeleteTask(id)` — 删除任务
  - `LoadTask(id)` — 续接历史任务（加载消息到 Agent Conversation）
- [x] 数据持久化
  - 每次对话自动保存到 SQLite
  - 工具调用记录完整追踪
  - 任务状态自动管理（running → completed/failed）

### Phase 4: UI 层

**优先级**: 中
**预计时间**: 2-3 周
**状态**: GUI 基础框架已完成，前端界面待完善

- [x] Wails v3 GUI 框架
- [x] Backend 服务与 Agent 集成
- [x] 流式响应事件（SSE）
- [x] 任务历史 Backend 接口
- [x] 前端聊天界面（Markdown 渲染、代码高亮）
- [x] 工具调用可视化
- [x] 设置页面（API Key、Provider、Model）
- [x] 历史任务浏览/续接 UI
- [x] Slash 命令（弹出菜单，复用 internal/slash）

### Phase 5: 高级功能

**优先级**: 低
**预计时间**: 2-4 周

- [x] 任务历史管理 ✅
- [x] 自定义规则加载 ✅
- [x] GUI 前端模块化拆分 ✅ (2026-06-02)
- [ ] GUI 配置管理界面
- [ ] 规则热重载 (`/reload`)
- [ ] MCP 支持
- [ ] 性能优化

## 已知问题

### 架构演进（重大变更）

**TUI → GUI 迁移** — Bubbletea TUI 已废弃，全面迁移到 Wails v3 GUI。旧 TUI MVVM 架构作为历史参考仍保留在 `memory-bank/archive/` 中。

- **原 TUI 架构**: `internal/ui/` 下 Bubbletea MVVM + Bridge 分层设计
- **新 GUI 架构**: `gui/` 下 Wails v3，Go Backend + Web Frontend
- **原因**: GUI 提供更丰富交互（Markdown 渲染、代码高亮、鼠标操作），Wails 包体积更小

### 已修复问题 ✅

1. **Agent 流式回调架构** ✅
   - **实现**:
     - `Agent.RunWithCallback()` 统一处理流式响应和工具执行
     - `StreamCallback` 接口通知 UI 层（OnContent, OnToolCallStart, OnToolCallComplete, OnTaskCreated）
     - `processStream()` 统一处理 LLM 流式响应
     - GUI 通过 `ChatService` 接收事件并转发给前端
   - **文件**: `internal/agent/agent.go`, `gui/chat_service.go`

2. **工具调用实时通知** ✅
   - **实现**:
     - 在 `processStream()` 接收到完整工具调用时立即发送回调
     - `OnToolCallStart` / `OnToolCallComplete` 让 GUI 实时显示工具执行状态
     - 工具执行结果自动追加到对话上下文
   - **文件**: `internal/agent/agent.go`
   - **日期**: 2026-05-09

3. **工具调用参数重复累积修复** ✅
   - **修复**:
     - 在 `openai.go` 发送 partial chunk 时创建 tool call 的副本
     - `toolCallCopy := *toolCalls[tc.Index]` 避免指针共享
   - **文件**: `internal/api/openai.go`, `internal/agent/agent.go`
   - **日期**: 2026-05-09

4. **工具执行流程修复** ✅
   - **修复**:
     - 在 agent loop 中添加工具执行逻辑
     - processStream 后检查并执行工具调用
     - 只有无工具调用时才 SetComplete()
   - **文件**: `internal/agent/agent.go`
   - **日期**: 2026-05-09

5. **TUI 流式输出优化** ✅ (历史记录)

### 2026-05-23 — 自定义规则 / 系统提示词扩展 ✅

**功能**: 支持通过文件系统自动加载自定义规则并追加到系统提示词末尾。

**实现内容**:
1. **规则加载** (`internal/prompts/rules.go`)
   - `LoadCustomRules()` — 同时加载全局和工作区规则
   - `loadRulesFromDir()` — 扫描目录，按字母序合并 `.md`/`.txt` 文件
   - 自动跳过子目录、不支持格式、空文件

2. **Agent 集成**
   - `Options.CustomRules` / `BaseAgent.customRules` 字段
   - `GetSystemPrompt(mode, tools, customRules)` 支持追加 `# Custom Rules` 区块

3. **规则存放**
   - 全局: `~/.gline/rules/` — 所有项目共享
   - 工作区: `.gline/rules/` — 仅当前项目
   - 加载顺序: 全局在前 → 工作区在后

4. **单元测试** (`internal/prompts/rules_test.go`)
   - 覆盖空目录、单/多文件、排序、跳过非支持格式、子目录、空文件、读取容错

5. **文档** (`README.md`)
   - 新增"自定义规则 / 系统提示词"使用说明章节

**修改文件**:
- `internal/prompts/rules.go` — 新增
- `internal/prompts/rules_test.go` — 新增
- `cmd/gline/chat.go` — Agent 初始化调用 LoadCustomRules
- `internal/agent/agent.go` — 添加 CustomRules 字段
- `internal/prompts/system.go` — 支持追加自定义规则
- `README.md` — 使用文档

### 2025-06-20 — Slash 命令功能修复 ✅ (TUI 历史记录)

> TUI 已废弃。此功能仅作为历史记录保留。

**功能**: 修复 TUI slash 命令"有 UI 无后台"的关键缺陷。

**修复内容**:
- `internal/ui/tui.go` 中修复 `OnResult` 回调为 `nil` 的问题
- 增强 `handleSlashCommandResult` 同步 Agent 层状态

**支持的 Slash 命令** (TUI 时期):
- `/clear` — 清空当前对话
- `/exit` 或 `/q` — 退出 gline
- `/help` — 显示帮助信息
- `/newtask [name]` — 开始新任务
- `/smol` 或 `/compact` — 压缩对话上下文

> GUI 后续将以按钮/菜单形式替代这些 slash 命令。

## 最近变更

### 2026-06-02 — GUI 首页强制选择项目目录 ✅

首次进入（welcome 页面无历史消息）时，用户必须先选择项目目录才能开始对话。欢迎页增加"📁 Select Project Directory"按钮，未选择前输入框和发送按钮禁用。

**变更内容**:
- `gui/chat_service.go`: 提取 `pickProjectDir()` 通用方法；新增 `SelectProjectDir()` 前端可调用绑定（不重置 conversation，仅切换目录）
- `gui/frontend/src/hooks/useChat.ts`: 新增 `selectProjectDir()`，导出到 hook 返回值
- `gui/frontend/src/App.tsx`: `projectDir` 状态，`canChat` = `projectDir !== '' || messages.length > 0`，启动时读取 `cwd` 初始化
- `gui/frontend/src/components/MessageList.tsx`: 消息为空时显示"📁 Select Project Directory"按钮
- `gui/frontend/src/components/InputArea.tsx`: `canChat` prop 控制禁用状态和占位文字（"Please select a project directory first"）
- `gui/frontend/src/components/ChatArea.tsx`: 传递 `onSelectProjectDir` 和 `canChat`

### 2026-06-02 — 重构：使用 workingDir 字段替代 os.Getwd() ✅

移除了启动时 `os.Getwd()` 自动赋值给 `cwd` 的问题。改为后端 `ChatService` 上独立的 `workingDir` 字段，只有用户通过对话框选择目录后才会被赋值。

- `GetStatus()` 不再调用 `os.Getwd()`，返回 `workingDir`（初始为空）
- `/clear`、新建会话不再弹窗，而是重置会话并清空 `workingDir`，让前端回到欢迎页
- 欢迎页"📁 Select Project Directory"按钮只在 `status.cwd === ''` 时显示
- `canChat = status.cwd !== '' || messages.length > 0`

### 2026-06-02 — GUI 新建会话时选择项目目录 ✅

新建会话（New Chat、Ctrl+N、/clear、/newtask）时，弹出目录选择对话框让用户选择项目目录。

**变更内容**:
- `gui/chat_service.go`: 新增 `StartNewConversation()`，使用 Wails v3 `Dialog.OpenFile()` + `CanChooseDirectories(true)` + `CanChooseFiles(false)` 选择目录，成功后 `os.Chdir()` 并 reset agent/conversation
- `gui/frontend/src/hooks/useChat.ts`: `handleNewChat()` 和 slash 命令 `/clear`、`/newtask` 改为 `async`，`await StartNewConversation()`，用户取消时不清空消息

**Wails v3 对话框坑**: 没有 `OpenDirectoryDialog`，必须使用 `Dialog.OpenFile()` 然后链式调用 `.CanChooseDirectories(true).CanChooseFiles(false)`

---

### 2026-06-02 — Tab 切换 Plan/Act 模式、帮助文本格式化、输入框提示 ✅

- Tab 键切换 Plan/Act 模式：`InputArea` 拦截 Tab（slash 菜单未激活时）调用 `onToggleMode`
- Help 文本列表化渲染：`SystemMessage` 按标题/命令+说明的结构解析并列表化展示
- 输入框提示：底部左对齐显示 "Type / for slash commands · Use @ to add files"

---

### 2026-06-02 — GUI 前端模块化拆分 ✅

将 `gui/frontend/src/App.tsx`（1234 行/52KB）按关注点拆分为多个独立模块，解决单文件膨胀问题。

**拆分结构**:
| 层次 | 文件 | 职责 |
|------|------|------|
| 共享 | `theme.ts` `types.ts` `utils/format.ts` | THEME 常量、类型定义、formatContent/代码高亮/数学公式/工具提示 |
| Hooks | `useChat.ts` `useTaskHistory.ts` `useAppStatus.ts` `useSettings.ts` `useKeyboardShortcuts.ts` | 聊天状态、历史任务、应用状态、设置、键盘快捷键 |
| 基础组件 | `UserMessage` `AssistantMessage` `ToolMessage` `SystemMessage` | 消息类型渲染 |
| 复合组件 | `Sidebar` `Header` `MessageList` `InputArea` `ChatArea` | 侧边栏、顶栏、消息列表、输入区、主聊天容器 |
| 弹窗 | `SettingsPanel` `FollowupModal` | 设置弹窗、追问弹窗 |
| 入口 | `App.tsx` | 仅剩约 93 行组合逻辑 |

**验证**: `npm run build`（TypeScript+Vite）✅；`wails3 build`（完整编译）✅ 生成 `bin/gline.exe`

---

### 2026-06-02 — GUI Token 追踪与上下文压缩 ✅

**Token 实时追踪**
- `pkg/types/message.go`: `actualInputTokens`/`actualOutputTokens` 字段 + `AddActualTokens()`/`GetActualTokens()`/`ResetActualTokens()` 方法
- `internal/agent/agent.go`: `processStream()` 从流式响应的 usage chunk 累加真实 token
- `internal/api/openai.go`: 解析 SSE 中 `choices == 0` 的 usage 终结 chunk
- `internal/api/anthropic.go`: 解析 `message_delta.usage` 并附带 `Done: true`
- `gui/chat_service.go`: `GetStatus()` 优先返回真实 token，fallback 到估算值
- `gui/frontend/src/App.tsx`: 状态栏显示模型名 + token进度条

**最大上下文配置**
- `internal/config/config.go`: `ProviderSettings` 新增 `MaxContextTokens`
- 设置面板支持 per-provider `Max Context Tokens` 输入
- 默认值从 128000 上调为 262000
- `gui/backend.go`: 用配置的 `MaxContextTokens` 初始化 Agent，变更时自动重初始化

**自动上下文压缩**
- `pkg/types/message.go`: `GetTotalTokens()` 获取最优 token 估算；`AutoCompact()` 滑动窗口压缩（保留 system prompt + 最近 2 轮 = 4 条消息）；`IsTokenAboveThreshold(percent)` 阈值检测
- `internal/agent/agent.go`: `BaseAgent.Compact()` 手动压缩；`BaseAgent.AutoCompact()` 自动检测 80% 阈值并压缩；`Agent` 接口新增 `Compact() bool`
- `gui/chat_service.go`: `CompactConversation()` 暴露给前端（未来可作为 slash 命令/按钮调用）
- 每次 `RunWithCallback` 发送 LLM 请求前自动检查并压缩
- 参考 Cline 的 `ContextManager` 截断策略（保留开头锚点 + 按比例截断），但我们的实现更简化：直接保留 system + 最近 2 轮

**修改文件**:
- `pkg/types/message.go`
- `internal/config/config.go`
- `internal/agent/agent.go`
- `internal/api/openai.go`
- `internal/api/anthropic.go`
- `gui/backend.go`
- `gui/chat_service.go`
- `gui/frontend/src/App.tsx`

### 2026-06-02 — GUI Slash 命令功能完成 ✅

在 Wails GUI 中实现了 slash 命令弹出菜单，复用 `internal/slash/` 的 registry 和命令定义。

**变更内容**:
- `gui/chat_service.go`: 集成 `slash.Registry`，新增 `InitSlashRegistry()`、`GetSlashCommands()`、`ExecuteSlashCommand()`、`FilterSlashCommands()`、`IsSlashCommand()`、`ParseSlashCommand()`、`BuildHelpText()`
- `gui/slash_service.go`: 序列化类型 `SlashCommandInfo` 和 `SlashActionResult`
- `gui/main.go`: 初始化后调用 `chatService.InitSlashRegistry()`
- `gui/frontend/src/slash/use-slash-commands.ts`: Hook 管理菜单状态、命令过滤、键盘导航 (↑/↓/Enter/Tab/Esc)
- `gui/frontend/src/slash/slash-menu.tsx`: 深色主题弹出命令选择器
- `gui/frontend/src/App.tsx`: 集成菜单到输入框，处理 `/` 触发和 action 分发

**支持命令**: `/clear`、`/newtask`、`/smol`、`/compact`、`/history`、`/help`、`/exit`、`/q`

**Wails 绑定坑**: `SlashService` 若同时注册为 `application.Service` 和作为 exported struct 被引用，Wails 生成 bindings 时会产生重复的 `SlashService` 导出（module + model class），导致 TypeScript 编译失败。解决办法是将 slash 逻辑内嵌到 `ChatService` 中，不单独注册为 service。

### 2026-06-02 — 迁移到 Wails GUI ✅

项目已全面从 Bubbletea TUI 迁移到 Wails v3 GUI 桌面应用。

**变更内容**:
- 新增 `gui/` 目录，包含 Wails v3 应用入口、Backend、ChatService
- `gline` 命令默认启动 GUI 桌面应用
- 保留 CLI 子命令 `history`, `config`, `version`
- 废弃 `internal/ui/` Bubbletea TUI（代码保留但不维护）

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