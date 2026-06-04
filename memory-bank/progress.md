# Progress

## 项目状态概览

**当前阶段**: Phase 1 快速赢 已完成 ✅

**总体进度**: 75% - 核心架构已完成（Agent、Provider、Tools、Storage），GUI 前端功能基本完善，规则管理 UI 已补齐。

## 已完成工作

### Phase 1: 快速赢（1-3 天）— 提升日常体验 ✅

**目标**: 补齐"后端已就绪但前端缺位"的尴尬，让 GUI 100% 可用。

| 子任务 | 说明 | 状态 |
|--------|------|------|
| **P1.1 规则管理 UI** | SettingsPanel 新增「Custom Rules」区块：展示规则列表（来源、大小、修改时间）+ Reload 按钮 | ✅ 已完成 |
| **P1.2 `/reload` Slash 命令前端联动** | `/reload` 执行后在前端显示 toast 提示重载结果 | ✅ 已完成 |
| **P1.3 移除 @ 误导提示** | 输入框提示改为 "Type / for slash commands"，移除未实现的 @ 引用提示 | ✅ 已完成 |
| **P1.4 主题切换占位** | Chat Theme select 改为 disabled 并提示 "Coming soon"，避免用户困惑 | ✅ 已完成 |

**验收标准**: SettingsPanel 能查看规则、手动 reload、有明确反馈。✅ 全部达成

**实现细节**:
- `gui/frontend/src/hooks/useSettings.ts`: 新增 `rules`, `rulesMessage`, `loadingRules`, `loadRules`, `reloadRules`, `formatFileSize`, `formatModTime`
- `gui/frontend/src/components/SettingsPanel.tsx`: 扩展 props 接收规则数据，新增「Custom Rules」区块，支持规则列表展示、来源标签（global/workspace）、文件大小/修改时间、空状态提示、Reload 按钮
- `gui/frontend/src/App.tsx`: 传递规则相关 props 到 SettingsPanel
- `gui/frontend/src/components/InputArea.tsx`: 移除 "Use @ to add files" 误导提示
- `gui/frontend/src/hooks/useChat.ts`: `/reload` slash 命令执行后显示 system message 反馈
- 主题 select 添加 `disabled` 属性 + "🚧 Theme switching is coming soon" 提示

### 1. Cline 源码分析 ✅

**时间**: 2025-06-04

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
| TUI 框架 | Bubbletea | ✅ 确定，已废弃 |
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
**状态**: GUI 基础框架已完成，前端界面完善中

- [x] Wails v3 GUI 框架
- [x] Backend 服务与 Agent 集成
- [x] 流式响应事件（SSE）
- [x] 任务历史 Backend 接口
- [x] 前端聊天界面（Markdown 渲染、代码高亮）
- [x] 工具调用可视化
- [x] 设置页面（API Key、Provider、Model）
- [x] 历史任务浏览/续接 UI
- [x] Slash 命令（弹出菜单，复用 internal/slash）
- [x] 规则管理 UI（SettingsPanel 展示规则列表 + Reload 按钮）
- [x] 主题切换占位（disabled + Coming soon 提示）
- [x] @ 引用提示移除（避免误导）
- [x] **SettingsPanel Custom Rules UI 修复**（2026-01-XX）— 组件接收了 rules props 但未渲染，已修复

**优先级**: 中
**预计时间**: 2-3 周
**状态**: GUI 基础框架已完成，前端界面完善中

- [x] Wails v3 GUI 框架
- [x] Backend 服务与 Agent 集成
- [x] 流式响应事件（SSE）
- [x] 任务历史 Backend 接口
- [x] 前端聊天界面（Markdown 渲染、代码高亮）
- [x] 工具调用可视化
- [x] 设置页面（API Key、Provider、Model）
- [x] 历史任务浏览/续接 UI
- [x] Slash 命令（弹出菜单，复用 internal/slash）
- [x] 规则管理 UI（SettingsPanel 展示规则列表 + Reload 按钮）
- [x] 主题切换占位（disabled + Coming soon 提示）
- [x] @ 引用提示移除（避免误导）

### Phase 5: 高级功能

**优先级**: 低
**预计时间**: 2-4 周

- [x] 任务历史管理 ✅
- [x] 自定义规则加载 ✅
- [x] GUI 前端模块化拆分 ✅ (2026-06-02)
- [x] 规则管理界面 ✅ (2026-XX-XX)
- [x] 规则热重载 (`/reload`) ✅
- [x] @ 文件引用 ✅ (2025-06-04)

### 2025-06-04 — @ 文件引用功能完成 ✅

**后端** (`gui/file_service.go`):
- `ListDirEntries(dirPath)` — 列出项目目录下的文件/子目录，过滤隐藏目录
- `ReadFileContent(relPath)` — 读取文件内容，1MB限制 + 二进制检测
- `SendMessageWithContext(prompt, fileRefsJSON)` — 读取引用文件，拼接 `<referenced_files>` XML 上下文

**前端**:
- `useFileReference.ts` — @ 引用状态管理（FilePicker 开/关、目录浏览、文件选中/删除）
- `FilePicker.tsx` — 文件浏览弹窗（目录/文件列表、路径面包屑、键盘导航）
- `InputArea.tsx` — @ 触发检测 + 文件标签展示 + FilePicker 集成
- `useChat.ts` — 发送时通过 `SendMessageWithContext` 携带文件引用
- `App.tsx` / `ChatArea.tsx` — 串联 fileRef props

**交互流程**: 用户输入 `@` → 弹出文件浏览器 → 选择文件 → 输入框上方显示标签 → 发送时自动读取文件内容注入 prompt
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
   - `Agent.RunWithCallback()` 统一处理流式响应和工具执行
   - `StreamCallback` 接口通知 UI 层
   - `processStream()` 统一处理 LLM 流式响应

2. **工具调用实时通知** ✅
   - `OnToolCallStart` / `OnToolCallComplete` 让 GUI 实时显示工具执行状态

3. **工具调用参数重复累积修复** ✅
   - 在 `openai.go` 发送 partial chunk 时创建 tool call 的副本，避免指针共享

4. **工具执行流程修复** ✅
   - agent loop 中添加工具执行逻辑，只有无工具调用时才 SetComplete()

5. **TUI 流式输出优化** ✅ (历史记录)

### 2026-06-02 — GUI 前端模块化拆分 & 项目目录重构 (已完成)
- **拆分**: App.tsx 从 ~1234 行/52KB 拆分为 18+ 独立模块
- **目录选择重构**: 移除 `os.Getwd()` 依赖，使用独立 `workingDir` 字段记录项目目录
- **workingDir 历史持久化**: v2 DB 迁移添加 `working_dir` 列

### 2026-06-03 — search_files 工具优化 + 单元测试 (已完成)
- 并发搜索 Worker Pool、字面量快速路径、目录跳过、二进制文件过滤
- 100+ 文件基准测试覆盖

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `gui/chat_service.go`: 新增 `ClearConversation()` 方法（保留 workingDir），`StartNewConversation()` 继续清空 workingDir
- `gui/frontend/src/hooks/useChat.ts`: `/clear` 分支改调 `ClearConversation()`
- `gui/frontend/bindings/.../chatservice.ts`: 绑定同步更新（删 `NewConversation`，加 `ClearConversation`）
- `gui/chat_service.go`: 新增 `ClearConversation()` 方法（保留 workingDir），`StartNewConversation()` 继续清空 workingDir
- `gui/frontend/src/hooks/useChat.ts`: `/clear` 分支改调 `ClearConversation()`
- `gui/frontend/bindings/.../chatservice.ts`: 绑定同步更新（删 `NewConversation`，加 `ClearConversation`）

### 2026-XX-XX — Phase 1 快速赢完成 ✅
- **P1.1 规则管理 UI**: SettingsPanel 新增「Custom Rules」区块，支持规则列表展示、来源标签（🌍 global / 📁 workspace）、文件大小、修改时间、Reload 按钮、空状态提示
- **P1.2 `/reload` Slash 命令前端联动**: `/reload` 执行后通过 system message 显示重载结果
- **P1.3 移除 @ 误导提示**: 输入框提示改为仅显示 "Type / for slash commands"
- **P1.4 主题切换占位**: Chat Theme select 改为 `disabled` + "🚧 Theme switching is coming soon" 提示，避免用户困惑

## 建议下一步

Phase 2 推荐顺序：
1. **P2.3 废弃 TUI 代码清理** (0.5 天) - 快速技术债务清理
2. **P2.1 @ 文件引用** (3 天) - 最关键的用户体验
3. **P2.2 系统托盘集成** (1 天) - 体验提升
4. **P2.4 构建产物优化** (1 天) - 基础设施
5. **P2.5 前端错误边界** (0.5 天) - 稳定性
