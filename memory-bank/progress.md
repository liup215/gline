# Progress

## 项目状态概览

**当前阶段**: 四层记忆引擎 + 透明聊天驱动系统 全部完成 ✅

**总体进度**: 100% - 所有核心功能已完成：Fact/Wiki/RAG 三层引擎、LLM 工具集成、被动/主动提取、前端记忆徽章。

---

## 2026-06-05 — 透明记忆系统（Phase 9+）✅

### 设计决策
- **无独立 GUI 面板**：所有记忆操作通过聊天界面透明完成（自然语言触发或 slash 命令）
- **统一大模型**：被动提取使用主 provider，无需本地小模型
- **记忆来源标记**：前端提取 📌 Fact / 📚 Wiki / 📄 KB 徽章

### 新实现

| 组件 | 文件 | 说明 |
|------|------|------|
| Memory Tools | `internal/tools/memory.go` | memory_recall / memory_note / kb_search |
| Engine Config | `internal/memory/engine_config.go` | `NewEngineFromConfig()` 统一工厂 |
| System Prompt | `internal/prompts/system.go` | LLM 能力说明 + 工具描述 |
| Interval Extract | `internal/agent/agent.go` | `maybeExtractFacts()` 每 4 轮触发 |
| Slash Commands | `internal/slash/commands.go` | `/mem note`, `/mem recall`, `/mem status` |
| Slash Service | `gui/chat_service.go` | `memoryService` 桥接到后端引擎 |
| Engine Injection | `gui/backend.go` | Agent 初始化时自动注入 memory engine |
| Memory Badges | `gui/frontend/src/components/MemoryBadges.tsx` | 📌📚📄 来源标识组件 + 内容提取器 |
| AssistantMessage | `gui/frontend/src/components/AssistantMessage.tsx` | 集成徽章渲染 |

### 验证
- `go build ./...` ✅
- `go test ./internal/{agent,memory,tools,slash}/...` ✅
- `npm run build` (frontend) ✅

---

## 2025-06-05 — 四层统一记忆与知识引擎（Phases 1-8）✅

---

## 2025-06-05 — 四层统一记忆与知识引擎（Phases 1-6）✅

### 设计
融合三种前沿方案：mem0 (Fact 层)、Karpathy Wiki (Wiki 层)、RAG 检索 → 统一四层架构。

| 层 | 类比 | 擅长场景 | 已建状态 |
|--|------|----------|----------|
| **Fact** | 人类长期记忆 | 用户偏好、技术选型、bug 模式、项目决策 | SQLite + FTS5 + entities + Decay ✅ |
| **Wiki** | 知识笔记 | 深度理解技术方案、跨文档矛盾追踪 | Markdown FS + index.md/schema.md/log.md ✅ |
| **RAG** | 快速查手册 | 代码精确引用、API 文档、配置查找 | 纯 Go KNN + FTS5 + RRF ✅ |
| **Conversation** | 短期工作记忆 | 保持当前任务连贯、多轮工具调用 | 已有 SQLite 存储 ✅ |

### 技术突破
- **纯 Go SQLite 向量存储**：`modernc.org/sqlite` 无法加载 C 扩展 → embedding 用 `gob` 存 BLOB，Go 内存计算归一化点积 = 余弦相似度。
- **混合检索**：Go 内存 KNN 相似度 + SQLite FTS5 + RRF (Reciprocal Rank Fusion) 融合两层结果。
- **性能分层**：同步只读（<150ms）检索 Fact + RAG + Wiki；异步写入（后台 goroutine）Fact 提取 + Embedding + Wiki Ingest。

### 新文件（18 个源码文件 + 1 测试）
完整列表见 `activeContext.md`。

### Agent 集成
- `buildMemoryContext()` 注入记忆上下文到 system prompt（Token 硬限制 2000）
- 对话完成后异步 `extractFactsAsync()` 后台提取事实

### 验证
- `go test ./internal/memory/...` ✅ (0.756s)
- `go build ./cmd/gline/...` ✅
- 修复：Search() `!rows.Next()` 消耗首行 bug；upsertFact nil tx 崩溃

---

## 2026-06-05 之前 — 主题系统组件集成扩展 🔄 (未提交)

### P2.5.3 主题系统组件集成
- 28 个新 CSS 变量
- highlight.js 样式表动态切换
- FOUC prevention
- 全组件硬编码颜色迁移到 `THEME.*`
- `format.ts` 重构

## 2026-06-04 — 主题系统组件集成扩展 🔄 (未提交)

### 背景
P2.5.2 建立了 CSS 变量主题骨架，但大量组件仍使用硬编码颜色值。本次扩展让主题系统真正可投入使用。

### 变更内容
1. **新增 28 个 CSS 变量** (`theme.ts`)
   - 控件背景: `inputBg`, `cardBg`, `overlayBg`
   - 反馈色: `toastSuccess`, `toastError`, `toastSuccessBg`, `toastErrorBg`
   - 代码展示: `codeInlineBg`, `codeInlineText`, `highlightJsTheme`
   - 状态指示: `spinner`, `statusSuccessBg`, `statusPendingBg`
   - 排版辅助: `linkColor`, `tableHeadBg`, `tableBorder`, `footnoteText`, `footnoteBorder`
   - 品牌: `logoGradientStart`, `logoGradientEnd`

2. **highlight.js 主题切换**
   - 新增 `public/styles/hljs-github-dark.css` + `hljs-github-light.css`
   - `applyThemeColors()` 动态切换 `<link id="hljs-theme">` 的 `href`
   - `index.html` 内嵌默认 dark 样式表链接

3. **FOUC Prevention**
   - `index.html` 内嵌 `<script>` 在 React 挂载前读取 `localStorage.getItem('gline-theme')`
   - 若 stored === 'light'，同步写入所有 light 模式的 CSS 变量到 `:root`
   - 避免页面加载时出现"白闪"或"暗闪"

4. **全组件硬编码颜色迁移**
   - `Header.tsx`: Stop 按钮红色从 `#ef4444` 改为 `THEME.toastError`
   - `Sidebar.tsx`: 边框、hover、激活态全部改用 CSS 变量
   - `MessageList.tsx`, `InputArea.tsx`, `SettingsPanel.tsx`, `ToolMessage.tsx`, `SystemMessage.tsx`, `FollowupModal.tsx` 等

5. **`format.ts` 重构**
   - 大幅重构格式化工具函数（与主题无关的紧邻改进）

### 验证
- `go build ./...` ✅ (Go 后端无变更)
- `cd gui/frontend && npm run build` ✅ (前端构建通过)

---

## 2026-06-04 — 性能阻塞点修复 ✅

### 背景
用户报告对话"越用越慢"，经源代码审查定位到 3 个明确的性能阻塞点 + 1 个隐性累加 bug。

### 阻塞点 1：SSE 每 Chunk 同步写磁盘（最严重）⭐
**文件**: `internal/api/openai.go`
**根因**: SSE 循环中每收到 1 个 chunk 就打开→写入→关闭 `gline_diag_sse.txt`。一轮对话 500 chunks = 500 次文件 IO。
**修复**:
- 删除 `CreateMessageStream` 请求开头的 `EMERGENCY DIAGNOSTIC`（写 `gline_diag.txt`）
- 删除 SSE 循环内每 chunk 的 `EMERGENCY DIAGNOSTIC`（写 `gline_diag_sse.txt`）
- 降级为 `log.Debugf`
- 清理未使用的 `"os"` 和 `"path/filepath"` 导入

### 阻塞点 2：高频 Info 级日志打印
**文件**: `internal/api/openai.go` + `internal/agent/agent.go`
**根因**: `processStream` 和 SSE 解析每 chunk 都走 `log.Infof`（非 Debug），在高频流式场景下成为显著 IO 瓶颈。
**修复**: 2 处 `log.Infof` → `log.Debugf`。

### 阻塞点 3：Token 估算严重低估（中文场景）
**文件**: `pkg/types/message.go`
**根因**: `totalChars/4` 估算对中文严重低估（100 汉字≈100-130 token，old estimate: 75 token）。`TrimToMaxTokens()` 条件几乎永远不满足，历史消息无限累积。
**修复**:
- 新增 `estimateTokens()`：ASCII 4 chars≈1 token，中文/CJK/emoji 1 rune≈1 token
- `updateTokenCount()` 计入 `ReasoningContent` + `ToolCalls`
- `TrimToMaxTokens()` 改用 `GetTotalTokens()`（API usage 优先），删除消息后 `ResetActualTokens()`
- 新增 `"unicode/utf8"` 导入

### 隐性 Bug 4：`actual` token 累加虚高
**文件**: `pkg/types/message.go`
**根因**: `AddActualTokens` 用 `+=` 累加，多轮后虚高导致 `GetTotalTokens()` 失真。
**修复**: `+=` → `=`（覆盖赋值），因为 API 返回的 usage 就是本轮的总数。

### 验证结果
- `go build ./...` ✅
- 编译无报错

---

## 已完成工作（历史记录）

### Phase 1: 快速赢（1-3 天）— 提升日常体验 ✅

| 子任务 | 说明 | 状态 |
|--------|------|------|
| **P1.1 规则管理 UI** | SettingsPanel 新增「Custom Rules」区块：展示规则列表（来源、大小、修改时间）+ Reload 按钮 | ✅ 已完成 |
| **P1.2 `/reload` Slash 命令前端联动** | `/reload` 执行后在前端显示 toast 提示重载结果 | ✅ 已完成 |
| **P1.3 移除 @ 误导提示** | 输入框提示改为 "Type / for slash commands"，移除未实现的 @ 引用提示 | ✅ 已完成 |
| **P1.4 主题切换占位** | Chat Theme select 改为 disabled 并提示 "Coming soon"，避免用户困惑 | ✅ 已完成 |

### P2.3 废弃 TUI 清理 ✅
- `internal/ui/` 删除
- charmbracelet 依赖从 go.mod 移除（减少约 30 个间接依赖）

### P2.5 前端错误边界 ✅
- `ErrorBoundary.tsx` + `main.tsx` 集成，避免白屏

### 2026-06-04 — @ 文件引用功能完成 ✅
**后端** (`gui/file_service.go`):
- `ListDirEntries(dirPath)` — 列出项目目录下的文件/子目录
- `ReadFileContent(relPath)` — 读取文件内容，1MB限制 + 二进制检测
- `SendMessageWithContext(prompt, fileRefsJSON)` — 拼接 `<referenced_files>` 上下文

**前端**:
- `useFileReference.ts` / `FilePicker.tsx` / `InputArea.tsx` / `useChat.ts`

### 2026-06-04 — @ 文件引用功能完善
- 方向键滚动、Filter 输入框、选择后自动关闭、onBlur 误关闭修复、文件标签路径优化

### 2026-06-04 — GUI 前端模块化拆分 & 项目目录重构 ✅
- 18+ 独立模块拆分
- `workingDir` 独立字段替代 `os.Getwd()`

### 2026-06-04 — search_files 工具优化 + 单元测试 ✅
- 并发 Worker Pool、字面量快速路径、目录跳过、二进制文件过滤

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `ClearConversation()` 保留 workingDir，`/clear` 改调；New Chat/`/newtask` 仍然清空

## 2026-06-07 — 构建系统重构 ✅

**背景**: CLI 与 Wails GUI 共用 `cmd/gline/main.go` 入口，但构建脚本分散混乱，无法一键编译成功。

**变更**:
- 删除废弃的 `desktop/` 目录（旧独立 Wails 项目残留）
- 前端源码移至根目录 `frontend/`
- Wails 构建资产（`build-desktop/`）已删除，统一使用 `build-all.sh`/`build-all.ps1` + `wails3` 命令构建
- 修复 `build-all.ps1`：bindings 生成改为在 `cmd/gline` 目录执行（否则报 "0 Services"），输出到 `../../frontend/bindings`
- 新增 `build-all.sh`：macOS/Linux 的 bash 构建脚本（与 PowerShell 脚本对等）
- 修复 `Makefile`：`FRONTEND_DIR := frontend`，bindings 目标改为根目录运行
- 更新 CI：新增 bindings 生成步骤；macOS 使用默认 CGO（WebKit 需要）；拆分为 `npm build` + `go build ./cmd/gline`
- 清理根目录 4 个测试 EXE

**验证**:
- `build-all.ps1` 一键编译成功 → `bin/gline.exe` ✅
- `desktop/` 目录已不存在 ✅

---

### 2026-06-04 — Phase 1 快速赢完成 ✅
- 规则管理 UI、reload 联动、@ 提示移除、主题占位

---

## 已知问题

### 已修复问题 ✅

| # | 问题 | 修复文件 | 说明 |
|---|------|----------|------|
| 19 | **KB 自动触发 wiki 失败** | `internal/memory/engine.go` | `IngestFile()` 中 `kb.Type == KBTypeWiki || hybrid` 条件触发 wiki，但 `e.Caller` 在 CLI 下为 nil，导致 wiki 静默跳过。解耦后 `WikiIngestFile()` 显式要求 Caller，失败立即返回 error。 |
| 20 | **kb_ingest 重复执行 / 并行导致数据库锁** | `internal/agent/agent.go` + `internal/memory/engine.go` | `preDispatchToolCall` 在流式期间后台并行执行 `kb_ingest`，多个 goroutine 同时写同一 `rag.db` → `database is locked`。→ ① Agent 层扩大 side-effect 黑名单（增加 `memory_note`），禁止背景预分发；② Engine 层在 `IngestFile` 入口加 `ingestMu sync.Mutex`，串行化写入。 |
| 21 | **RAG 重复文档** | `internal/memory/store.go` + `engine.go` | `IngestFile` 重新加入同一文件时不断增。→ `findDocByName` + `DeleteDocument` 在插入前删除旧记录。 |
| 22 | **list_files 递归耗 token** | `internal/tools/file.go` | list_files 递归返回所有子目录文件，token 爆炸。→ 移除递归，只返回当前目录列表，添加 `recursive` 参数可选。 |
| 23 | **PDF 导入提取二进制乱码** | `internal/memory/parser.go` + `go.mod` | `github.com/ledongthuc/pdf` 的 `GetPlainText()` 对嵌入字体、CJK、非标准编码支持不足，提取返回二进制乱码。→ 替换为 `github.com/tsawler/tabula`（MIT/纯Go），支持 CJK/排除页眉页脚。新增 `.odt` / `.epub` 支持。 |

### 架构演进（重大变更）
**TUI → GUI 迁移** — Bubbletea TUI 已废弃，全面迁移到 Wails v3 GUI。旧 TUI MVVM 架构作为历史参考仍保留在 `memory-bank/archive/` 中。

### 已修复问题 ✅（完整列表）
1. **Agent 流式回调架构** ✅
2. **工具调用实时通知** ✅
3. **工具调用参数重复累积** ✅
4. **工具执行流程修复** ✅
5. **TUI 流式输出优化** ✅
6. **SSE 每 Chunk 同步写磁盘** ✅ (2026-06-04)
7. **Info 级高频日志** ✅ (2026-06-04)
8. **Token 估算严重低估** ✅ (2026-06-04)
9. **actual token 累加虚高** ✅ (2026-06-04)
10. **P1.1-P1.4** ✅
11. **P2.3 废弃 TUI** ✅
12. **P2.5 错误边界** ✅
13. **@ 文件引用** ✅
14. **`/clear` 保留 workingDir** ✅
15. **P2.2 系统托盘集成** ✅ — 左键切换窗口显示/隐藏，右键菜单含 Show/Hide 动态标签 + Quit
16. **P2.4 构建产物优化** ✅ — Taskfile `dev`/`build` 统一 + CI wails3 build 集成
17. **Agent 构建失败** ✅ — `SubmitHandler` 中直接实例化 `*ai.Agent` 导致 `Agent` 接口不匹配。修复：恢复 `NewRuntimeAgent(auth)` 调用。
18. **ask_followup_question 终止对话** ✅ — agent 在收到用户回答后调用 `SetComplete()` 停止运行。
    修复：从 `SetComplete` switch 中移除 `ToolAskFollowupQuestion`；同时引入 pre-dispatch 优化（流式期间预执行工具调用）。

## 已完成工作（2026-06-04）

### GitHub Actions CI 重构 ✅
**背景**: 之前的工作流只编译 Go 后端，完全不构建前端（React/TypeScript），导致产物中前端资源为空。
**修复文件**: `.github/workflows/build.yml`
**修复内容**:
- **build.yml**: 新建 `test` job（ubuntu-24.04），新增完整 `build` 矩阵（darwin-arm64, windows-amd64）
  - 安装 wails3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95`)
  - Linux 构建安装 `libgtk-4-dev` + `libwebkitgtk-6.0-dev`
  - wails3 在 `gui/` 目录下构建，上传裸二进制 artifact
  - 新增 `build-summary` 汇总 artifact 到 GitHub Step Summary
- **release** (tag 触发): 上传 Release Artifacts 裸二进制（非 zip），覆盖 macOS + Windows
- **snapshot** (main/master push): 自动创建/更新 `snapshot` tag 预发布版本

### 环境说明
`go.mod` 和 `gui/go.mod` 保持 `go 1.25.0`（无法降级到 1.22，因为 `modernc.org/sqlite@v1.50.1`、`modernc.org/libc@v1.72.3`、`golang.org/x/sys@v0.42.0` 等核心依赖要求 `go >= 1.25.0`）。

## 已完成工作（2026-06-04）

### P2.4 构建产物优化 ✅
**范围**: Taskfile `dev` / `build` 目标统一 + CI/CD 产物路径修正  
**对应 commits**: 系列 CI commits（`ci: refactor GitHub Actions...` 起）  
**内容**:
- `gui/Taskfile.yml`: `build` / `dev` / `run` / `package` 目标按 OS 分发到 `build/{OS}/Taskfile.yml`
- `gui/build/Taskfile.yml`（common）: 统一 `build:frontend`、`generate:bindings`、`build:server`、`build:docker` 等共享目标
- GitHub Actions 统一使用 `wails3 build`，自动集成前端构建 + 产物路径修正
- 产物清理：移除不兼容 `-ldflags`、精简 Linux 依赖、矩阵调整为 Windows + macOS

---

## 已完成工作（2026-06-04 — Phase 2.5 技术债务）

### P2.5.1 前端测试骨架建立 ✅
**工具**: Vitest + @testing-library/react + @testing-library/jest-dom + jsdom
**文件**:
- `vite.config.ts`: `test` 区块配置（globals, jsdom environment, setup files）
- `vitest.setup.ts`: 引入 jest-dom matchers
- `package.json`: scripts `test` (vitest run) + `test:watch` (vitest)
- `src/utils/format.test.ts`: 10 个用例覆盖 `parseToolInput`, `getToolHint`, `formatContent`

### P2.5.2 主题系统初版完成 ✅
**方案**: CSS 变量法 — `THEME` 常量值改为 `var(--theme-*)` 引用，ThemeContext 更新 `:root` 变量。
**优点**: 零组件文件修改，所有现有 `style={{...THEME}}` 自动响应主题切换。
**文件**:
- `theme.ts`: 拆分为 `THEME_DARK` / `THEME_LIGHT`，`THEME` 改为 CSS 变量引用，新增 `applyThemeColors()`
- `ThemeContext.tsx`: React Context + `useTheme()` hook，`localStorage` 持久化
- `main.tsx`: 用 `<ThemeProvider>` 包裹 `<App/>`
- `index.html`: 内嵌 `<style>` 定义默认 dark CSS 变量（防止 FOUC）
- `SettingsPanel.tsx`: Theme select 启用，绑定 `setTheme()`，即时生效

### P2.5.3 主题系统组件全面集成 🔄 (未提交)
**说明**: 在 P2.5.2 基础上扩展 28 个新 CSS 变量，并将所有组件的硬编码颜色迁移到 `THEME.*` 引用。highlight.js 样式表动态切换，index.html 添加 FOUC prevention script。

---

---

## 2026-06-08 — Skill 系统迁移到 cline agent 规范 ✅

### 变更
| 组件 | 变更 |
|------|------|
| Skill 格式 | 扁平 `.yaml` → `skill-name/SKILL.md`（YAML frontmatter + markdown body）|
| Loader | 扫描子目录，解析 frontmatter，验证 name 匹配目录名 |
| Activation | 预注入 → `use_skill` 工具按需加载（工具结果消息返回完整指令）|
| Registry | 移除 active 状态，新增 `GetMeta()` / `GetInstructions()` |
| Agent | `activeSkill` → `skills []SkillMeta` + `SetSkills()` |
| Search dirs | `~/.gline/skills/`, `~/.agents/skills/`, `~/.cline/skills/`, `~/.claude/skills/` |

### 验证
- `go build ./...` ✅
- `go test ./...` ✅
- `build-all.ps1` 一键编译 ✅

## 已完成工作（2026-06-XX）

### GUI 历史会话工具结果渲染修复 ✅
**问题**: 加载历史会话时，`attempt_completion` 和 `plan_mode_respond` 的工具调用结果不渲染，用户看不到任务完成总结。
**根因**: 实时流中这些工具的结果被前端额外插入一条 `assistant` 消息展示，但这些辅助消息不会被保存到数据库。历史加载时只有 `tool` 角色消息，而原 `ToolMessage` 只显示缩略标签气泡。
**修复文件**: `frontend/src/components/ToolMessage.tsx`
- 新增 `attempt_completion` 分支 → "✅ Task Completed" 标题 + `formatContent(toolResult)` Markdown 渲染
- 新增 `plan_mode_respond` 分支 → "📝 Plan Response" 标题 + `formatContent(toolResult)` Markdown 渲染
- 两者均带 `maxHeight: 500px` + `overflow: auto` 滚动条限制

### ask_followup_question 弹窗渲染优化 ✅
**问题**: 弹窗和聊天流中的追问内容不做 Markdown 渲染，也不加滚动条。内容多时页面看不全。
**修复文件**:
- `frontend/src/components/FollowupModal.tsx` → `className="md-rendered"` + `formatContent(question)` + `maxHeight: 40vh` + `overflow: auto`
- `frontend/src/components/ToolMessage.tsx` → `ask_followup_question` 气泡同上处理
**验证**: `tsc --noEmit` 0 错误；`build-all.ps1` 完整构建通过 ✅

### Memory Tab UI 完成 ✅
**文件**:
- `frontend/src/components/SettingsPanel.tsx` → 拆分为 `settings/` 子组件（ProviderTab, MemoryTab, GeneralTab, RulesTab + sharedStyles.ts）
- `frontend/src/components/settings/MemoryTab.tsx` → 新增（Embedding provider/model/API key/Base URL + Retrieval TopK/MinScore/MaxTokens + Enabled 开关）
- `internal/config/config.go` → 新增 `MemoryConfig.Enabled bool`，默认值 `true`
- `internal/gui/backend.go` → 初始化引擎时检查 `Enabled`，配置变更热重载包含 memory 配置项

---

## 建议下一步

### 高优先级 — 四层记忆引擎 LLM 驱动层
1. **Phase 7: Fact Extractor LLM 集成** — 对话结束后用 LLM 提取 ADD/DECAY 事实，取代 rule-based stub
2. **Phase 8: Wiki Ingest LLM 集成** — `kb add` 后用 LLM 读取 raw 文件，自动生成/更新 wiki 页面

### 中优先级 — 基础设施优化
3. **连接池优化** — RAGManager/VectorStore 复用 SQLite 连接
4. **PDF/DOCX 解析** — `pdfcpu`（纯 Go）用于 PDF 解析

### 低优先级
5. **Embedding int8 量化** — BLOB 存储 4× 压缩
6. **ContextBuilder intent routing** — 按问题类型自动路由各层
7. **对话 message 扩展** — storage 包添加 FactsExtracted/WikiPagesTouched 列

### 长期 (Phase 3)
8. **MCP 支持** — 引入 Model Context Protocol，接入外部工具源
9. **LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
