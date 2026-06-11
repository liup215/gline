# Active Context

## Current Focus

### 修复 kb_ingest 并行调用导致数据库锁死（2026-06-07）

**状态**: 已完成并提交 ✅

**背景**: 聊天流式响应中 LLM 可能同时返回多个 `kb_ingest` tool calls，`preDispatchToolCall` 为每个都启动后台 goroutine，所有 goroutine 并行打开同一 `rag.db` WAL 数据库执行写事务，触发 `database is locked`。

**修复内容**:
- `internal/agent/agent.go` — `preDispatchToolCall` 扩大 side-effect 工具黑名单（`kb_ingest` / `write_to_file` / `replace_in_file` / `execute_command` / `memory_note`），禁止这些工具在 SSE 流期间后台预分发，只允许在主循环串行执行一次。
- `internal/memory/engine.go` — `UnifiedEngine` 增加 `ingestMu sync.Mutex`，在 `IngestFile` 入口处加锁，确保同一引擎实例内所有写入 RAG/SQLite 的操作串行化。

**验证**: `go build ./...` ✅, `go test ./internal/agent/... ./internal/memory/...` ✅

---

### 构建系统重构（2026-06-07）

**状态**: 已完成 ✅

**背景**: CLI 与 Wails GUI 共用 `cmd/gline/main.go` 入口，但构建脚本分散混乱（`desktop/build.ps1` 为旧独立 Wails 项目残留，`gui/` 目录残余），导致无法一键编译成功。

**变更内容**:
- 删除废弃的 `desktop/` 目录（含旧 `build.ps1`、`Taskfile.yml`、前端源码副本）
- 前端源码从 `desktop/frontend/` 移动到根目录 `frontend/`
- Wails 构建资产（图标、manifest）通过 `wails3` 命令按需生成，不再常驻仓库
- `build-all.ps1` 更新为项目根目录下一键构建脚本
- `Makefile` 更新：前端路径改为 `frontend/`，bindings 生成从 `gui/` 改为 `cmd/gline`
- `.github/workflows/build.yml` 更新：CI 步骤从 `cd desktop && wails3 build` 改为 `npm install` + `npm run build` + `go build ./cmd/gline`
- 清理根目录下 4 个测试 EXE (`gline-diag.exe`, `gline-search-fix.exe`, `gline-test.exe`, `gline.exe`)

**关键构建发现**:
- `wails3 build` **不可直接使用** — 项目非标准 Wails 结构（入口在 `cmd/gline/main.go`，不是 `gui/main.go`）
- `wails3 generate bindings --ts` **必须在 `cmd/gline` 目录执行** 才能扫描到 `internal/gui` 的 `ChatService`（1 Service / 34 Methods / 21 Models）；根目录执行报 "0 Services"
- Bindings 输出路径需要显式指定 `-d "../../frontend/bindings"`，否则会在 `cmd/gline/frontend/bindings` 创建

**文件**: `build-all.ps1`, `build-all.sh`, `Makefile`, `.github/workflows/build.yml`
**验证**: `build-all.ps1` → `bin/gline.exe` 一键编译成功 ✅

---

### KB/RAG 与 Wiki 解耦（2026-06-05）

**状态**: 已完成并提交 ✅

**背景**: 用户要求彻底分离 KB（RAG）和 Wiki 的调用逻辑。RAG 是纯本地精确检索，可完全由代码执行；Wiki 强依赖 LLM，涉及资料整合、Markdown 生成。两者不应耦合在同一个 `kb_ingest` 流程中。

**变更内容**:
- KB 类型只保留 `rag`，删除 `hybrid` 和 `wiki` 类型。
- `IngestFile()` 只做 RAG（chunk + embed + store），不再自动触发 wiki 生成。
- 新增 `WikiIngestFile()` 作为独立的显式入口，由前端或用户主动调用。
- CLI `kb init` 默认类型改为 `rag`。
- GUI `ChatService` 新增 `WikiIngestFile()` API 方法。

**文件**: `internal/memory/types.go`, `internal/memory/engine.go`, `cmd/gline/kb.go`, `gui/chat_service.go`, `internal/memory/memory_test.go`

---

### 四层记忆与知识引擎（2026-06-05）

**状态**: Phases 1-6 已完成，所有测试通过 ✅；Phase 7（Fact Extractor LLM 集成）待开始。

#### 架构决策

融合三种前沿方案：
- **Layer 1 Fact** (mem0 风格) — 语义事实提取与衰减
- **Layer 2 Wiki** (Karpathy 风格) — LLM 维护 Markdown 知识笔记
- **Layer 3 RAG** — 向量 + FTS5 混合检索
- **Layer 4 Conversation** — 已有对话历史

#### 技术突破

- `modernc.org/sqlite`（纯 Go）**无法加载 C 扩展**，不能用 `sqlite-vec` → 创新方案：embedding 向量用 `gob` 编码存 BLOB，Go 内存中计算余弦相似度（归一化后点积），配合 FTS5 用 RRF 融合两层结果。
- 性能设计：同步只读（Fact FTS5/SQL + RAG 向量/FTS5 + Wiki 文件读取）< 150ms；异步写入（Fact 提取、Embedding、Wiki Ingest）后台 goroutine。
- Token 硬限制：MAX_MEMORY_TOKENS = 2000，超时降级 200ms。

#### 已创建文件 (18 个新源文件)

| 文件 | 职责 |
|------|------|
| `internal/memory/types.go` | 四层类型契约（Fact, WikiPage, Document/Chunk, KnowledgeBase, ContextPack） |
| `internal/memory/embedder.go` | 嵌入器接口、L2 归一化、余弦相似度、TopK 搜索 |
| `internal/memory/embedder_openai.go` | OpenAI 兼容 API 和 Ollama 嵌入客户端 |
| `internal/memory/chunk.go` | Token 感知分块器（段落边界优先、可配置重叠） |
| `internal/memory/store.go` | **纯 Go SQLite 向量存储**（Go 内存 KNN + FTS5 + RRF 融合） |
| `internal/memory/kb_registry.go` | 知识库注册表 |
| `internal/memory/fact_store_sqlite.go` | Fact 层 SQLite 实现（facts 表 + FTS5 + entities 表 + Decay） |
| `internal/memory/fact_extractor.go` | mem0 风格事实提取器（rule-based stub → Phase 7 LLM） |
| `internal/memory/wiki_fs.go` | Wiki Markdown 文件系统 |
| `internal/memory/wiki_engine.go` | Wiki 管理器 stub |
| `internal/memory/rag_engine.go` | RAG 管理器封装 |
| `internal/memory/engine.go` | **UnifiedEngine** 统一入口 |
| `internal/memory/parser.go` | 文档解析器（md/txt/code/html） |
| `internal/memory/util.go` | 通用工具（genID） |
| `internal/memory/memory_test.go` | 单元测试（覆盖 Chunker, Embedder, KBRegistry, FactStore, VectorStore, WikiFS, FactExtractor） |
| `cmd/gline/kb.go` | `gline kb init/list/remove/status/add/search` |
| `cmd/gline/wiki.go` | `gline wiki show/links/lint/sync` |
| `cmd/gline/mem.go` | `gline mem facts/recall/decay` |

#### 已修改文件

- `internal/config/config.go` — 新增 MemoryConfig, MemoryEmbeddingConfig, MemoryRetrievalConfig
- `internal/agent/agent.go` — MemoryEngine 注入 system prompt；异步 fact 提取 hook
- `cmd/gline/chat.go` — initializeAgent() 中可选初始化 memory engine
- `go.mod` / `go.sum` — 新增 `golang.org/x/net v0.55.0`

#### 验证
- `go test ./internal/memory/...` ✅ (0.756s)
- `go build ./cmd/gline/...` ✅
- `go vet ./internal/memory/...` ✅
- 修复 bug：Search() 中 `!rows.Next()` 消耗首行导致单条记录查询为空、`upsertFact` nil tx 崩溃。记忆留到下一轮，镜像建构发生在下一轮系统提示词更新时。

**建议下一步**: Phase 7（Fact Extractor LLM 集成）→ Phase 8（Wiki Ingest LLM 集成）→ 连接池优化 → PDF 解析。

---

### Phase 2.5 技术债务清偿完成（2026-06-04）

| 子任务 | 说明 | 状态 | 文件 |
|--------|------|------|------|
| **P2.5.1 前端测试骨架** | Vitest + @testing-library/react + jsdom 安装；`format.test.ts` 10 用例通过；npm scripts 集成 | ✅ | `package.json`, `vite.config.ts`, `vitest.setup.ts`, `utils/format.test.ts` |
| **P2.5.2 主题系统初版** | THEME 拆分为 Dark/Light 色板 + CSS 变量方案；ThemeContext 管理状态并持久化；SettingsPanel 启用切换；所有组件零修改即可响应主题 | ✅ | `theme.ts`, `ThemeContext.tsx`, `main.tsx`, `SettingsPanel.tsx`, `index.html` |
| **P2.5.3 主题系统组件全面集成** | 新增 28 个 CSS 变量；全组件硬编码颜色迁移到 `THEME.*`；highlight.js 样式动态切换；FOUC  prevention | 🔄 未提交 | `theme.ts`, `index.html`, `components/*.tsx`, `public/styles/` |

### 历史焦点

#### Phase 2 全面完成（2026-06-04）

Phase 2 全部子任务已完成：
- P2.2 系统托盘集成 ✅
- P2.3 废弃 TUI 清理 ✅
- P2.4 构建产物优化 ✅ （本次完成）
- P2.5 前端错误边界 ✅

接下来进入 Phase 3 规划或优先技术债务清偿。之前详细记录的 2026-06-04 性能阻塞点修复内容如下：

**用户反馈**: 对话"越用越慢"，定位并修复了 3 个明确阻塞点 + 1 个隐性累加 bug。

| 阻塞点 | 严重度 | 文件 | 根因 | 修复 |
|--------|--------|------|------|------|
| SSE 每 chunk 同步写磁盘 | ⭐最严重 | `internal/api/openai.go` | 500次文件IO/轮 | 移除 EMERGENCY DIAGNOSTIC 代码 |
| Info 级高频日志 | 🟡高 | `openai.go` + `agent.go` | per-chunk `log.Infof` | 降级为 `log.Debugf` |
| Token 估算低估 | 🟡高 | `pkg/types/message.go` | `totalChars/4` 对中文低估 ~4 倍 | 引入语言感知 `estimateTokens()` |
| actual token 累加 | 🟡中 | `pkg/types/message.go` | `+=` 导致多轮虚高 | 改为覆盖赋值 `=` |

**验证**: `go build ./...` ✅

**提交信息**: `perf(core): fix 3 performance bottlenecks causing slowdown`

---

## 历史完成记录

### Phase 1 & 2 完成总结（已完整）

| 子任务 | 说明 | 状态 | 文件 |
|--------|------|------|------|
| P1.1 规则管理 UI | SettingsPanel 新增 Custom Rules 区块 | ✅ | `SettingsPanel.tsx`, `useSettings.ts` |
| P1.2 `/reload` 联动 | Slash 命令 reload 显示结果提示 | ✅ | `useChat.ts`, `slash/commands.go` |
| P1.3 移除 @ 误导 | 输入框提示移除 | ✅ | `InputArea.tsx` |
| P1.4 主题切换占位 | Theme select disabled | ✅ | `SettingsPanel.tsx` |
| P2.2 系统托盘集成 | 左键切换显示/隐藏，右键菜单含 Show/Hide、Quit | ✅ | `gui/main.go` |
| P2.3 废弃 TUI 清理 | 删除 `internal/ui/` | ✅ | `cmd/gline/chat.go`, `root.go`, `go.mod` |
| P2.4 构建产物优化 | Taskfile `dev`/`build` 统一 + CI 产物路径修正 | ✅ | `gui/Taskfile.yml`, `.github/workflows/` |
| P2.5 前端错误边界 | ErrorBoundary 避免白屏 | ✅ | `ErrorBoundary.tsx`, `main.tsx` |

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `ClearConversation()` 保留 workingDir；/clear 改调；New Chat 仍然清空

---

### 2026-06-04 — Agent 构建错误修复 ✅
- **问题**: `SubmitHandler` 中直接实例化 `ai.NewAgent()` 返回 `*ai.Agent`，不包含 `client` 和 `Close()`，不满足 `agent.Agent` 接口要求。
- **修复**: 恢复为 `NewRuntimeAgent(auth)`，正确满足 `Agent` 接口（包含完整客户端和生命周期方法）。
- **文件**: `internal/agent/agent.go`
- **验证**: `go build ./...` ✅

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0
- **GUI 框架**: Wails v3.0.0-alpha.95
- **操作系统**: Windows 11
- **Git HEAD**: `52d9db6` — docs(memory-bank): update Phase 2.5 completion and tech debt status
- **未提交变更**: 14 files modified, 2 new files (hljs theme stylesheets)

## 技术债务

| 债务 | 状态 | 说明 |
|------|------|------|
| TUI 代码 | ✅ 已清理 | `internal/ui/` 删除 |
| 前端测试 | ✅ 骨架已建 | Vitest 运行通过，`format.test.ts` 10 个用例 |
| Wails v3 alpha | ⚠️ 持续跟踪 | alpha 版本，binding 需注意 |
| 主题系统 | 🔄 组件集成中 | CSS 变量初版已完备，全组件迁移已完成（未提交） |

## 最新变更

### ask_followup_question 弹窗 Markdown 渲染 + 滚动条（当前已实现）
**问题**: `ask_followup_question` 弹窗直接显示原始文本，无 Markdown 渲染、无滚动条。长内容（如含代码块、列表的问题）看不全。
**修复**:
- `frontend/src/components/FollowupModal.tsx` — 导入 `formatContent`，将 `<div>{question}</div>` 替换为 `className="md-rendered"` + `dangerouslySetInnerHTML={{ __html: formatContent(question) }}`，添加 `maxHeight: '40vh'` + `overflow: 'auto'`。
- `frontend/src/components/ToolMessage.tsx` — 对 `isQuestion` 分支的追问内容同样调用 `formatContent` + `md-rendered` + `maxHeight: 300px` + `overflow: auto`。
**验证**: `tsc --noEmit` ✅, `build-all.ps1` 完整构建 ✅

### 历史会话工具结果渲染修复（当前已实现）
**问题**: 加载历史会话时，`attempt_completion` / `plan_mode_respond` 的结果只显示一个缩略标签气泡，无法看到实际内容。
**根因**: 实时流中前端通过 `chat:systemMessage` 额外插入 `assistant` 消息展示结果，但这些辅助消息不会被持久化。历史加载时只剩 `tool` 角色消息，原 `ToolMessage` 不渲染 `toolResult`。
**修复文件**: `frontend/src/components/ToolMessage.tsx`
- 新增 `attempt_completion` 分支 → 带 "✅ Task Completed" 标题的助手风格气泡，Markdown 渲染 + 滚动（`maxHeight: 500px`）
- 新增 `plan_mode_respond` 分支 → 带 "📝 Plan Response" 标题的助手风格气泡，Markdown 渲染 + 滚动
**验证**: `tsc --noEmit` ✅, `build-all.ps1` 完整构建 ✅

### SettingsPanel Memory Tab（已集成但未提交）
**状态**: 代码已修改，随本次 commit 一起提交
**文件**: `frontend/src/components/SettingsPanel.tsx` 重构为 `settings/` 子目录（ProviderTab, MemoryTab, GeneralTab, RulesTab）
**后端**: `internal/config/config.go` 新增 `MemoryConfig.Enabled`；`internal/gui/backend.go` 支持内存配置热重载

---

### ask_followup_question 终止对话修复（2026-06-04）
- **问题**: Agent 在收到用户回答后调用 `SetComplete()` 停止运行，导致对话终止。
- **修复**: 从 `SetComplete` switch 中移除 `ToolAskFollowupQuestion`；同时引入 pre-dispatch 优化（流式期间预执行工具调用）。
- **影响**: 追问后对话继续正常流转。
- **文件**: `internal/agent/agent.go`

### 构建与 CI 修复（2026-06-04）
- **问题**: `build.yml` 和 `release.yml` 路径与 wails3 构建流程不匹配。
- **修复**: 合并工作流、移除 Linux 构建矩阵（runner 稀缺）、修正 artifact 下载、使用 wails3 build 统一构建。
- **文件**: `.github/workflows/build.yml`

---

---

### Skill 系统迁移到 cline agent 规范（2026-06-08）

**状态**: 已完成并提交 ✅

**背景**: gline 原有自定义 skill 格式（扁平 `.yaml` 文件）无法复用成千上万的第三方 skill。需要迁移到 cline 的 agent skill 规范。

**变更内容**:
- **Skill 格式**: 扁平 `.yaml` → `skill-name/SKILL.md`（YAML frontmatter + markdown body）
- **加载方式**: `internal/skills/loader.go` 重写为扫描子目录并解析 frontmatter
- **激活方式**: 移除 `activeSkill` 预注入；系统提示只列出 skill 元数据，LLM 调用 `use_skill` 工具按需加载完整指令
- **新工具**: `internal/tools/use_skill.go` 实现 `use_skill` 工具，作为工具结果消息返回 skill 内容
- **Registry**: 移除激活状态方法，新增 `GetMeta()` / `GetInstructions()`
- **Agent**: 移除 `activeSkill`，替换为 `skills []SkillMeta` + `SetSkills()`
- **GUI/CLI**: `backend.go`/`chat_service.go`/`cmd/gline/chat.go` 更新注册 `use_skill` 并同步 skills
- **目录扩展**: `~/.gline/skills/`, `~/.agents/skills/`, `~/.cline/skills/`, `~/.claude/skills/`
- **清理**: 删除 `builtin_embed.go`，移除 `InitBuiltinSkills()`

**文件**: `pkg/types/skill.go`, `internal/skills/*.go`, `internal/tools/use_skill.go`, `internal/prompts/system.go`, `internal/agent/agent.go`, `internal/gui/backend.go`, `internal/gui/chat_service.go`, `cmd/gline/chat.go`
**验证**: `go build ./...` ✅, `go test ./...` ✅, `build-all.ps1` ✅

## Current Focus — replace_in_file 5 层容错优化（2026-06-XX）

**状态**: 已完成并提交 ✅

**背景**: `replace_in_file` 工具频繁失败，根因是 LLM 从 `read_file` 获取的内容与实际文件有空白符差异，导致 `strings.Contains` 完全精确匹配失败。错误信息零反馈，LLM 无法 self-correct。

**5 层优化内容**:

| 层级 | 优化 | 机制 | 效果 |
|------|------|------|------|
| ① | 多 SEARCH/REPLACE 块支持 | 新增 `replacements` 数组字段，单次调用完成多个编辑 | 减少调用次数，降低累积误差 |
| ② | 增强错误反馈 | Jaccard bigram 相似度计算最近匹配 + 相似度分数 + 4 步排查指南 | LLM 能 self-correct |
| ③ | 空格归一化容错 | `normalizeWhitespace()` 将空格/tab/换行压缩为单个空格后再匹配 | 消除空白差异导致的大部分失败 |
| ④ | 行锚定回退 | 选取搜索块中最长行作为锚点，在文件中定位后验证周围上下文 | 大文件/长代码块的最后一道防线 |
| ⑤ | 替换后 diff 输出 | `computeDiff()` 返回 ```` ```diff ```` 格式摘要 | LLM 验证修改是否正确 |

**新增/修改文件**:
- `internal/tools/file.go` — 核心逻辑重写（~+300 行），保持向后兼容（单块模式仍可用）
- `internal/prompts/system.go` — 系统提示 `%EDITING_FILES%` 节更新，工具 schema 增加 `replacements` 数组
- `internal/tools/file_test.go` — 7 个单元测试（单块/多块/错误反馈/归一化/相似度/diff）

**验证**: `go test ./internal/tools/...` ✅ (7/7 pass), `go test ./...` ✅ (全项目 0 失败), `build-all.ps1` ✅ (完整构建通过)

---

## 下一步建议

1. **Phase 7: Fact Extractor LLM 集成** — 对话结束后用 LLM 提取 ADD/DECAY 事实，取代 rule-based stub
2. **Phase 3 MCP 支持** — 引入 Model Context Protocol，接入外部工具源
3. **Phase 3 LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
4. **P2.5.3 主题系统组件集成** — 若有未提交的 14 files，完成提交
