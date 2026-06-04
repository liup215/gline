# Active Context

## 当前焦点

### Phase 1 & 2 完成总结（2025-01-12）

Phase 1（快速赢）和 Phase 2（体验增强）大部分子任务已完成。

| 子任务 | 说明 | 状态 | 文件 |
|--------|------|------|------|
| P1.1 规则管理 UI | SettingsPanel 新增 Custom Rules 区块 | ✅ | `SettingsPanel.tsx`, `useSettings.ts` |
| P1.2 `/reload` 联动 | Slash 命令 reload 显示结果提示 | ✅ | `useChat.ts`, `slash/commands.go` |
| P1.3 移除 @ 误导 | 输入框提示移除 "Use @ to add files" | ✅ | `InputArea.tsx` |
| P1.4 主题切换占位 | Theme select disabled + coming soon | ✅ | `SettingsPanel.tsx` |
| P2.3 废弃 TUI 清理 | 删除 `internal/ui/` + 清理 charmbracelet | ✅ | `cmd/gline/chat.go`, `root.go`, `go.mod` |
| P2.5 前端错误边界 | ErrorBoundary 避免白屏 | ✅ | `ErrorBoundary.tsx`, `main.tsx` |

### Bug 修复（2026-06-04）

**问题**: `/clear` Slash 命令会清空 `workingDir`，导致同一项目下新 task 丢失工作目录。

**根因**: `/clear` 和 `New Chat` 都调用 `StartNewConversation()`，该方法会清空 `workingDir`。

**修复**:
- 新增 `ClearConversation()` 方法（保留 `workingDir`，仅重置 conversation 和 task）
- `/clear` 改调 `ClearConversation()`
- `New Chat` / `/newtask` 继续调用 `StartNewConversation()`（清空一切）
- 更新 Wails bindings

**语义**:
| 操作 | 对话 | workingDir | taskID |
|------|------|-----------|--------|
| New Chat / `/newtask` | 清空 | **清空** → 弹出选目录 | 清空 |
| `/clear` | 清空 | **保留** | 清空（创建新 task）|
| 加载历史任务 | 加载 | **从 DB 恢复** | 从 DB 恢复 |

**提交**: `231e609 fix(gui): /clear preserves workingDir while New Chat clears it`

### Bug 修复（2026-01-XX）

**问题**: P1.1 规则管理 UI 已完成后端和前端的 hooks/App.tsx 集成，但 `SettingsPanel.tsx` 组件本身**从未渲染 Custom Rules 区块** —— 虽然接收了所有 rules 相关 props，但 JSX 中缺少对应的 UI。

**修复**:
- 在 `SettingsPanel.tsx` 中添加完整的 Custom Rules UI 渲染
- 包括规则列表展示（来源标签、文件大小、修改时间）
- Reload 按钮（含 loading 状态）
- 空状态提示（引导用户创建 `.clinerules`）
- 主题 select 明确标记 `disabled` + "🚧 coming soon" 提示

**提交**: `61d95ce fix(settings): add missing Custom Rules UI`

### 待办（Phase 2 剩余）
- ~~P2.1 @ 文件引用~~ ✅ 已完成（2025-06-04）
- P2.2 系统托盘集成（1天）— Wails v3 System Tray API
- P2.4 构建产物优化（1天）— Makefile/Taskfile 跨平台打包

### 构建状态
- `go build ./...` ✅
- `cd gui && go build -o ../tmp/gline.exe .` ✅
- `cd gui/frontend && npm run build` ✅
- go.mod 已清理 charmbracelet 依赖（减少约 30 个间接依赖）

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0
- **GUI 框架**: Wails v3.0.0-alpha.95
- **操作系统**: Windows 11

## 技术债务更新

| 债务 | 状态 | 说明 |
|------|------|------|
| TUI 代码 | ✅ 已清理 | `internal/ui/` 已删除，go.mod 已 tidy |
| 前端测试 | ❌ 仍缺失 | 所有 .tsx 零测试覆盖 |
| Wails v3 alpha | ⚠️ 持续跟踪 | alpha 版本，binding 生成有坑 |
| 主题系统 | ⚠️ 占位 | select disabled，待实现 |

## 下一步建议

1. 若需要「基础设施」：完成 P2.4 构建优化（Makefile `make gui` 目标）
2. 若需要「体验提升」：完成 P2.2 系统托盘集成（Wails v3 System Tray API）
3. 若需要「长期价值」：启动 Phase 3 MCP 支持
