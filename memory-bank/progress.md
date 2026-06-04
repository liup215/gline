# Progress

## 项目状态概览

**当前阶段**: Phase 1 快速赢 + Phase 2 全面完成 ✅

**总体进度**: 82% - 核心架构已完成，性能阻塞点已修复，构建系统已统一。

## 2025-07-04 — 性能阻塞点修复 ✅

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

### 2025-06-04 — @ 文件引用功能完成 ✅
**后端** (`gui/file_service.go`):
- `ListDirEntries(dirPath)` — 列出项目目录下的文件/子目录
- `ReadFileContent(relPath)` — 读取文件内容，1MB限制 + 二进制检测
- `SendMessageWithContext(prompt, fileRefsJSON)` — 拼接 `<referenced_files>` 上下文

**前端**:
- `useFileReference.ts` / `FilePicker.tsx` / `InputArea.tsx` / `useChat.ts`

### 2026-01-13 — @ 文件引用功能完善
- 方向键滚动、Filter 输入框、选择后自动关闭、onBlur 误关闭修复、文件标签路径优化

### 2026-06-02 — GUI 前端模块化拆分 & 项目目录重构 ✅
- 18+ 独立模块拆分
- `workingDir` 独立字段替代 `os.Getwd()`

### 2026-06-03 — search_files 工具优化 + 单元测试 ✅
- 并发 Worker Pool、字面量快速路径、目录跳过、二进制文件过滤

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `ClearConversation()` 保留 workingDir，`/clear` 改调；`New Chat`/`/newtask` 仍然清空

### 2026-XX-XX — Phase 1 快速赢完成 ✅
- 规则管理 UI、reload 联动、@ 提示移除、主题占位

---

## 已知问题

### 架构演进（重大变更）
**TUI → GUI 迁移** — Bubbletea TUI 已废弃，全面迁移到 Wails v3 GUI。旧 TUI MVVM 架构作为历史参考仍保留在 `memory-bank/archive/` 中。

### 已修复问题 ✅（完整列表）
1. **Agent 流式回调架构** ✅
2. **工具调用实时通知** ✅
3. **工具调用参数重复累积** ✅
4. **工具执行流程修复** ✅
5. **TUI 流式输出优化** ✅
6. **SSE 每 Chunk 同步写磁盘** ✅ (2025-07-04)
7. **高频 Info 级日志** ✅ (2025-07-04)
8. **Token 估算严重低估** ✅ (2025-07-04)
9. **actual token 累加虚高** ✅ (2025-07-04)
10. **P1.1-P1.4** ✅
11. **P2.3 废弃 TUI** ✅
12. **P2.5 错误边界** ✅
13. **@ 文件引用** ✅
14. **`/clear` 保留 workingDir** ✅
15. **P2.2 系统托盘集成** ✅ — 左键切换窗口显示/隐藏，右键菜单含 Show/Hide 动态标签 + Quit
16. **P2.4 构建产物优化** ✅ — Taskfile `dev`/`build` 统一 + CI wails3 build 集成

## 已完成工作（2026-06-19）

### GitHub Actions CI 重构 ✅
**背景**: 之前的工作流只编译 Go 后端，完全不构建前端（React/TypeScript），导致产物中前端资源为空。
**修复文件**: `.github/workflows/build.yml`, `.github/workflows/release.yml`
**修复内容**:
- **build.yml**: 新建 `test` job（ubuntu-latest），新增完整 `build` matrix（darwin-amd64/arm64, linux-amd64/arm64, windows-amd64）
  - 安装 wails3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95`)
  - Linux 构建安装 `libgtk-4-dev` + `libwebkitgtk-6.0-dev`
  - wails3 在 `gui/` 目录下构建，上传裸二进制（macOS/Linux: `gui/bin/gline`，Windows: `gui/bin/gline.exe`）
  - 新增 `build-summary` 汇总 artifact 到 GitHub Step Summary
- **release.yml**: 上传 Release Artifacts 同样改为裸二进制（非 zip），覆盖 5 个平台

### 环境说明
`go.mod` 和 `gui/go.mod` 保持 `go 1.25.0`（无法降级到 1.22，因为 `modernc.org/sqlite@v1.50.1`、`modernc.org/libc@v1.72.3`、`golang.org/x/sys@v0.42.0` 等核心依赖要求 `go >= 1.25.0`）。

## 已完成工作（2026-06-20）

### P2.4 构建产物优化 ✅
**范围**: Taskfile `dev` / `build` 目标统一 + CI/CD 产物路径修正  
**对应 commits**: `ci: refactor GitHub Actions for wails3 cross-platform builds` 及后续系列（~10 个 commits）  
**内容**:
- `gui/Taskfile.yml`: `build` / `dev` / `run` / `package` 目标按 OS 分发到 `build/{OS}/Taskfile.yml`
- `gui/build/Taskfile.yml`（common）: 统一 `build:frontend`、`generate:bindings`、`build:server`、`build:docker` 等共享目标
- GitHub Actions 统一使用 `wails3 build`（替代裸 `go build`），自动集成前端构建 + 产物路径修正
- 产物清理：移除不兼容 `-ldflags`、精简 Linux 依赖、矩阵调整为 Windows + macOS

---

## 建议下一步

1. **前端测试补足** — .tsx 零测试覆盖技术债务
2. **Phase 3 MCP 支持** (长期价值)
3. **Phase 3 LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
