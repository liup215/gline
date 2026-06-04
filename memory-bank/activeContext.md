# Active Context

## Current Focus

### Uncommitted Changes (Working Tree)

| File | Change | Notes |
|------|--------|-------|
| `gui/frontend/src/theme.ts` | +28 new color tokens | `inputBg`, `cardBg`, `overlayBg`, `toastBg`, `toastSuccess`, `toastError`, `codeInlineBg`, `codeInlineText`, `optionBg`, `optionHoverBg`, `linkColor`, `spinner`, `tableHeadBg`, `tableBorder`, `footnoteText`, `footnoteBorder`, `highlightJsTheme`, `userTextColor`, `statusSuccessBg`, `statusSuccessBorder`, `statusSuccessText`, `statusPendingBg`, `statusPendingBorder`, `statusPendingText`, `logoGradientStart`, `logoGradientEnd` |
| `gui/frontend/index.html` | FOUC prevention script + hljs link | Inline `<script>` reads `localStorage.getItem('gline-theme')` and syncs all CSS variables before React mounts; `<link id="hljs-theme">` for dynamic highlight.js stylesheet |
| `gui/frontend/src/components/*.tsx` | THEME variable adoption | All components migrated from hardcoded hex values to `THEME.*` references (stop button, sidebar, message list, settings panel, input area, etc.) |
| `gui/frontend/src/utils/format.ts` | Refactored | Significant refactoring of formatting utilities |
| `gui/frontend/public/styles/` | New directory | `hljs-github-dark.css` + `hljs-github-light.css` for code-block theme switching |

**Scope**: P2.5.3 Theme System Component Integration — the CSS-variable-based theme system is being expanded from the initial skeleton to full component coverage, including highlight.js theme synchronization.

**Status**: Code complete, uncommitted. Needs final review / commit.

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

## 下一步建议

1. **提交未完成的 P2.5.3** — 主题系统组件集成变更（14 files）
2. **Phase 3 MCP 支持** (长期) — 引入 Model Context Protocol，接入外部工具源
3. **Phase 3 LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
