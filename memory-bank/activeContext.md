# Active Context

## 当前焦点

### Phase 2.5 技术债务清偿完成（2026-06-20）

| 子任务 | 说明 | 状态 | 文件 |
|--------|------|------|------|
| **P2.5.1 前端测试骨架** | Vitest + @testing-library/react + jsdom 安装；`format.test.ts` 10 用例通过；npm scripts 集成 | ✅ | `package.json`, `vite.config.ts`, `vitest.setup.ts`, `utils/format.test.ts` |
| **P2.5.2 主题系统完备** | THEME 拆分为 Dark/Light 色板 + CSS 变量方案；ThemeContext 管理状态并持久化；SettingsPanel 启用切换；所有组件零修改即可响应主题 | ✅ | `theme.ts`, `ThemeContext.tsx`, `main.tsx`, `SettingsPanel.tsx`, `index.html` |

### 历史焦点

#### Phase 2 全面完成（2026-06-20）

Phase 2 全部子任务已完成：
- P2.2 系统托盘集成 ✅
- P2.3 废弃 TUI 清理 ✅
- P2.4 构建产物优化 ✅ （本次完成）
- P2.5 前端错误边界 ✅

接下来进入 Phase 3 规划或优先技术债务清偿。之前详细记录的 2025-07-04 性能阻塞点修复内容如下：

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

### 2026-XX-XX — Agent 构建错误修复 ✅
- **问题**: `SubmitHandler` 中直接实例化 `ai.NewAgent()` 返回 `*ai.Agent`，不包含 `client` 和 `Close()`，不满足 `agent.Agent` 接口要求。
- **修复**: 恢复为 `NewRuntimeAgent(auth)`，正确满足 `Agent` 接口（包含完整客户端和生命周期方法）。
- **文件**: `internal/agent/agent.go`
- **验证**: `go build ./...` ✅

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0
- **GUI 框架**: Wails v3.0.0-alpha.95
- **操作系统**: Windows 11

## 技术债务

| 债务 | 状态 | 说明 |
|------|------|------|
| TUI 代码 | ✅ 已清理 | `internal/ui/` 删除 |
| 前端测试 | ✅ 骨架已建 | Vitest 运行通过，`format.test.ts` 10 个用例 |
| Wails v3 alpha | ⚠️ 持续跟踪 | alpha 版本，binding 需注意 |
| 主题系统 | ✅ 已完备 | Dark/Light 可切换，localStorage 持久化 |

## 最新变更

### GitHub Actions 构建修复（2025-07-04）
- **问题**: `build.yml` 和 `release.yml` 只编译了 Go 后端，完全没有构建前端（React/TypeScript），导致发布的二进制中前端资源为空/不完整。
- **修复**: 在两个工作流中新增 `build-frontend` job，Node.js 环境编译 `gui/frontend`；Go build job 通过 artifact 下载 `frontend/dist`；修正构建路径为 `cd gui && go build`。
- **文件**: `.github/workflows/build.yml`, `.github/workflows/release.yml`

## 最新变更（2026-06-19）

### GitHub Actions CI 重构
- **问题**: `build.yml` / `release.yml` 只编译 Go 后端，不构建前端 React，导致产物中前端资源为空白。
- **修复**:
  - **`build.yml`**: 新建 `test` + `build` matrix（5 平台），安装 wails3 CLI，Linux 装 GTK/WebKit 依赖，wails3 build 在 `gui/` 目录下执行，上传裸二进制 artifact。
  - **`release.yml`**: Release Artifacts 上传裸二进制（非 zip），覆盖 darwin-amd64/arm64, linux-amd64/arm64, windows-amd64。
- **依赖约束**: `go.mod` 保持 `go 1.25.0`（`modernc.org/sqlite`/`libc`/`x/sys` 要求 `go >= 1.25.0`，无法降到 1.22）。

## 已完成（本次更新）

### P2.4 构建产物优化 ✅
**范围**: Taskfile `dev` / `build` 目标统一 + CI/CD 产物路径修正  
**对应 commits**: 系列 CI commits（`ci: refactor GitHub Actions...` 起）  
**内容**:
- `gui/Taskfile.yml`: `build` / `dev` / `run` / `package` 目标按 OS 分发到 `build/{OS}/Taskfile.yml`
- `gui/build/Taskfile.yml`（common）: 统一 `build:frontend`、`generate:bindings`、`build:server`、`build:docker` 等共享目标
- GitHub Actions 统一使用 `wails3 build`，自动集成前端构建 + 产物路径修正
- 产物清理：移除不兼容 `-ldflags`、精简 Linux 依赖、矩阵调整为 Windows + macOS

---

## 下一步建议

1. **Phase 3 MCP 支持** (长期) — 引入 Model Context Protocol，接入外部工具源
2. **Phase 3 LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
3. **System Tray 后续增强** — 未读消息角标、快速状态预览（可选）
