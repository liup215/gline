# Active Context

## 当前焦点

### 性能阻塞点修复（2025-07-04）

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
| P2.5 前端错误边界 | ErrorBoundary 避免白屏 | ✅ | `ErrorBoundary.tsx`, `main.tsx` |

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `ClearConversation()` 保留 workingDir；/clear 改调；New Chat 仍然清空

---

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0
- **GUI 框架**: Wails v3.0.0-alpha.95
- **操作系统**: Windows 11

## 技术债务

| 债务 | 状态 | 说明 |
|------|------|------|
| TUI 代码 | ✅ 已清理 | `internal/ui/` 删除 |
| 前端测试 | ❌ 仍缺失 | 所有 .tsx 零测试覆盖 |
| Wails v3 alpha | ⚠️ 持续跟踪 | alpha 版本，binding 需注意 |
| 主题系统 | ⚠️ 占位 | select disabled |

## 最新变更

### GitHub Actions 构建修复（2025-07-04）
- **问题**: `build.yml` 和 `release.yml` 只编译了 Go 后端，完全没有构建前端（React/TypeScript），导致发布的二进制中前端资源为空/不完整。
- **修复**: 在两个工作流中新增 `build-frontend` job，Node.js 环境编译 `gui/frontend`；Go build job 通过 artifact 下载 `frontend/dist`；修正构建路径为 `cd gui && go build`。
- **文件**: `.github/workflows/build.yml`, `.github/workflows/release.yml`

## 下一步建议

1. **P2.4 构建优化** (0.5天) — Taskfile `dev` / `build` 目标统一
2. **前端测试补足** — Jest/Vitest 从零建立测试
3. **Phase 3 MCP 支持** (长期) — 引入 Model Context Protocol
4. **Phase 3 LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
5. **System Tray 后续增强** — 未读消息角标、快速状态预览（可选）
