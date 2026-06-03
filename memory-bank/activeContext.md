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

### 待办（Phase 2 剩余）
- P2.1 @ 文件引用（3天）— 尚未开始
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

1. 若需要「快速赢」：完成 P2.4 构建优化（Makefile `make gui` 目标）
2. 若需要「竞争差异化」：启动 P2.1 @ 文件引用（最关键用户体验）
3. 若需要「长期价值」：启动 Phase 3 MCP 支持
