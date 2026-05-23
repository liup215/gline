# Active Context

## 当前焦点

### Slash 命令功能修复 ✅ 已完成 (2025-06-20)

修复了 TUI slash 命令"有 UI 无后台"的关键缺陷。用户输入 `/clear`、`/exit`、`/newtask` 等命令后，TUI 显示菜单和补全，但后台逻辑从未真正执行。

**根因分析** (4 个缺陷):
1. **`OnResult` 回调为 `nil`** — `New()` 中 `slash.NewDefaultRegistry(conv, nil)` 传入了 nil，所有 handler 的 `ctx.OnResult()` 调用无效
2. **`handleSlashCommandResult` 从未被调用** — 已有完整结果处理逻辑，但因 OnResult 为 nil 从未执行
3. **`quitting` 标志未处理** — `ResultQuit` 设置了 `m.quitting = true`，但 `Update()` 没有检查该标志来触发 `tea.Quit`
4. **Agent 层状态未同步** — `/clear`、`/newtask` 只清空了 UI 层 `model.Conversation`，没有清空后台 Agent 的 `types.Conversation`

**修复内容**:
1. **修复 `New()` 初始化** — 将 `slashMenu` 初始化移到 `Model` 创建之后，传入真正引用 `m` 的 `OnResult` 闭包：`func(result, message) { handleSlashCommandResult(m, result, message) }`
2. **修复 `Update()` 退出逻辑** — 在 return 前检查 `m.quitting`，若为 true 则追加 `tea.Quit`
3. **增强 `handleSlashCommandResult`** — 所有结果处理都同步 Agent 层状态：
   - `ResultClearScreen`: abort 运行中 agent → 清空 UI + agent conversation → 重置处理状态 → refocus 输入框
   - `ResultNewTask`: 同上，额外重置 `activeAssistantIndex`
   - `ResultCompact`: 调用 `agentInstance.GetConversation().TrimToMaxTokens()`，显示压缩统计
   - `ResultQuit`: 触发 Bubbletea 退出
   - `ResultShowHelp`: 添加帮助系统消息到对话

**验证结果**:
- ✅ `go build ./...` 编译通过
- ✅ `go vet ./...` 无静态分析错误
- ✅ `go test ./internal/slash/...` 通过
- ✅ `go test ./internal/ui/...` 全部通过

**修改文件**:
- `internal/ui/tui.go` — 3 处修改（New、Update、handleSlashCommandResult）

**参考**: Cline CLI 的 slash 命令分为三类：
- `execution: "local"` — TUI 本地处理（如 /help, /exit, /clear）
- `execution: "runtime"` — 发送到后端 Agent 处理（如 /newtask, /compact）
- `execution: "user-command"` — 展开为提示词注入（如 workflow/skill 命令）

### 自定义规则 / 系统提示词扩展 ✅ 已完成 (2026-05-23)

实现了从文件系统自动加载自定义规则并追加到系统提示词末尾的功能。

**实现内容**:
- 支持全局规则 (`~/.gline/rules/`) 和工作区规则 (`.gline/rules/`)
- 支持 `.md` 和 `.txt` 文件格式
- 文件按字母顺序合并，自动跳过空文件和不支持的格式
- 规则以 `# Custom Rules` 区块追加在系统提示词末尾
- 完整的单元测试覆盖 (`internal/prompts/rules_test.go`)

**修改文件**:
- `internal/prompts/rules.go` (新增) — 规则加载逻辑
- `internal/prompts/rules_test.go` (新增) — 单元测试
- `cmd/gline/chat.go` — Agent 初始化时加载规则
- `internal/agent/agent.go` — 添加 `CustomRules` 字段到 Options/BaseAgent
- `internal/prompts/system.go` — `GetSystemPrompt` 支持追加自定义规则
- `README.md` — 添加规则使用说明文档

## 下一步计划

### 待开发功能

**Phase 4: 高级功能**
- [ ] 任务历史管理
- [ ] 配置管理界面
- [ ] MCP 支持
- [ ] `/reload` slash 命令动态刷新规则

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.24.4
- **操作系统**: Windows 11

## 参考资源

- [Cline 源码](./cline/) - 架构参考
- [Bubbletea 文档](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Cobra 文档](https://github.com/spf13/cobra) - CLI 框架
