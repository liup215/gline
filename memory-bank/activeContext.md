# Active Context

## 当前焦点

### Phase 1: 存储层与任务历史 ✅ 已完成 (2026-05-24)

实现了完整的 SQLite 存储层和数据持久化，支持任务历史管理和 `gline history` CLI 命令。

**实现内容**:
1. **SQLite 存储模块** (`internal/storage/`)
   - 纯 Go SQLite 实现（使用 `modernc.org/sqlite`，零 CGO，跨平台）
   - 自动数据库迁移（`migrations` 表版本管理）
   - WAL 模式、外键约束、busy_timeout 优化
   - 5 张核心表：`tasks`, `messages`, `tool_calls`, `migrations`
   - 内存数据库支持（`:memory:`）用于测试
   - 单元测试覆盖：Task CRUD、Message 持久化、ToolCall 追踪、外键级联删除

2. **Store 接口** (`internal/storage/store.go`)
   - Task 生命周期：`CreateTask`, `UpdateTaskStatus`, `CompleteTask`, `FailTask`
   - Message 持久化：`SaveMessage`, `GetMessages`
   - ToolCall 追踪：`StartToolCall`, `CompleteToolCall`, `FailToolCall`
   - History 查询：`ListTasks`, `GetTaskByID`, `GetTaskSummary`, `DeleteTask`

3. **Agent 集成持久化** (`internal/agent/agent.go`)
   - `Options` / `BaseAgent` 添加 `Store` 和 `Title` 字段
   - `StreamCallback` 扩展 `OnTaskCreated(taskID)` 回调，通知 UI 新任务 ID
   - `RunWithCallback` 自动在关键节点持久化：
     - 用户首条消息 → `CreateTask()` → 通知 UI
     - Assistant 回复后 → `SaveMessage()`
     - 工具调用开始 → `StartToolCall()`，完成 → `CompleteToolCall()`/`FailToolCall()`
     - 工具结果消息 → `SaveMessage()`
     - 正常结束 → `CompleteTask()`，中断/报错 → `FailTask()`
   - `ResetTask()` 方法：`/newtask` 时清除当前 taskID，下次自动创建新任务
   - `SetTaskTitle()` 方法：支持 `/newtask MyTitle` 自定义任务名

4. **`gline history` CLI 命令** (`cmd/gline/history.go`)
   - `history list` (`ls`) — 分页列出任务，含状态/模式/提供商/模型/时间
   - `history show <id>` (`get`) — 展示任务详情和消息记录摘要
   - `history delete <id>` (`rm`) — 删除任务（需确认，支持 `-y` 跳过）
   - `FormatTaskList` / `FormatTaskDetail` — 友好终端输出格式

5. **操作系统兼容性**
   - Windows 原生兼容：无需 GCC/CGO，避免 `mattn/go-sqlite3` 构建问题
   - macOS/Linux 自动通用

**修改文件**:
- `internal/storage/store.go` — Store 接口 + Domain Model
- `internal/storage/database.go` — 数据库初始化和迁移
- `internal/storage/sqlite.go` — SQLiteStore 实现
- `internal/storage/history.go` — CLI 格式化输出
- `internal/storage/database_test.go` — 单元测试
- `internal/agent/provider.go` — StreamCallback 扩展 OnTaskCreated
- `internal/agent/agent.go` — 注入 Store、RunWithCallback 持久化、ResetTask/SetTaskTitle
- `internal/ui/bridge/callback.go` — TUIBridge 实现 OnTaskCreated
- `internal/ui/tui.go` — `/newtask` 调用 ResetTask + SetTaskTitle
- `cmd/gline/chat.go` — 初始化 Store 并注入 Agent Options
- `cmd/gline/history.go` — history CLI 子命令

**验证结果**:
- ✅ `go build ./...` 编译通过（纯 Go，零 CGO）
- ✅ `go test ./internal/storage/...` 全部通过（9 个测试用例）
- ✅ `go test ./...` 全量测试通过（无回归）
- ✅ `gline history list` 空状态正确显示 "No tasks found."
- ✅ `gline history show/delete` 对不存在 ID 友好处理

---

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

---

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

---

## 下一步计划

### Phase 1: 存储层与任务历史（高优先级，基础能力）✅ 已完成

**目标**: 实现数据持久化，完成任务历史管理功能。

| 子任务 | 说明 | 状态 | 实际时间 |
|--------|------|------|----------|
| 1.1 SQLite 数据库初始化 | 创建连接池、自动建表、迁移管理 | ✅ | 2h |
| 1.2 任务历史 CRUD | 支持创建任务、追加消息、更新状态 | ✅ | 3h |
| 1.3 Agent 集成持久化 | Agent 循环中自动保存消息和工具调用 | ✅ | 2h |
| 1.4 `gline history` CLI 命令 | list/show/delete 子命令 | ✅ | 1h |
| 1.5 任务命名与元数据 | `/newtask [name]` 支持自定义任务名 | ✅ | 0.5h |

**产出**：
- ✅ `internal/storage/` 完整实现
- ✅ `gline history [list/show/delete]` 可用
- ✅ 每次对话自动保存到本地 SQLite
- ✅ `/newtask` 支持自定义任务名

### Phase 2: TUI 任务历史界面 ✅ 已完成 (2026-05-24)

**目标**: 在 TUI 中集成历史浏览和续接功能。

**实现内容**:
1. **历史列表页** (`internal/ui/view/history_screen.go`, `tui.go`)
   - 全屏历史界面渲染（列表+详情两种视图）
   - 按状态着色（● 运行、✓ 完成、✗ 失败）
   - 显示模式、提供商、模型和时间
   - 空状态友好提示

2. **键盘操作** (`internal/ui/tui_input.go` → `handleHistoryKeyMsg()`)
   - `↑/↓` 选择任务
   - `Enter` 查看详情 / 加载续接
   - `D` 删除任务（Y/N 确认）
   - `Esc` 返回上一级（详情→列表→聊天）
   - `Ctrl+H` 从历史界面直接进入

3. **续接历史任务** (`internal/ui/tui.go` → `loadHistoryTask()`)
   - 解析 `storage.MessageRecord.ToolCalls` JSON 恢复工具调用历史
   - 双向同步：UI `model.Conversation.Messages` + Agent `types.Conversation.Messages`
   - 恢复 `agent.taskID` 使新消息追加到同一历史任务
   - 同步任务模式 (`plan`/`act`) 到 UI 和 Agent

4. **`/history` Slash 命令** (`internal/slash/commands.go`)
   - 注册 `/history` 命令到 `DefaultCommands`
   - 新增 `ResultShowHistory` 结果类型
   - 更新 `/help` 文本包含 `/history` 和 `Ctrl+H`

5. **Agent 扩展** (`internal/agent/agent.go`)
   - `GetStore() storage.Store` — 暴露存储给 TUI
   - `SetStore(s storage.Store)` — 支持运行时注入
   - `GetTaskID() / SetTaskID(id)` — 管理当前任务 ID

**修改文件**:
- `internal/agent/agent.go` — GetStore, SetStore, GetTaskID, SetTaskID
- `internal/slash/commands.go` — ResultShowHistory, /history command, help text
- `internal/ui/view/history_screen.go` — 新增历史界面渲染
- `internal/ui/tui.go` — ScreenType, history state, View() 分支, enterHistoryScreen, enterHistoryDetail, loadHistoryTask, deleteHistoryTask
- `internal/ui/tui_input.go` — Ctrl+H, handleHistoryKeyMsg
- `internal/ui/slash_menu_test.go` — 更新命令数量断言 (7→8)

**验证结果**:
- ✅ `go build ./...` 编译通过
- ✅ `go test ./internal/ui/... ./internal/slash/... ./internal/agent/... ./internal/storage/...` 全部通过
- ✅ `/history` 命令在 slash 菜单中可见
- ✅ `Ctrl+H` 从历史界面返回聊天界面

### Phase 3: `/reload` 动态刷新规则（低优先级，快速赢）

**目标**: 运行时重新加载自定义规则，无需重启 gline。

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| 3.1 `/reload` 命令 | 新增 slash 命令，重新读取 `~/.gline/rules/` 和 `.gline/rules/` | 0.5 天 |
| 3.2 热更新通知 | 刷新成功后显示加载的规则数量和来源 | 0.5 天 |

### Phase 4: 配置管理 TUI 界面（中优先级，易用性）

**目标**: 在 TUI 中可视化编辑配置，降低学习成本。

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| 4.1 配置编辑页 | TUI 表单编辑 API Key、Provider、Model 等核心配置 | 1-2 天 |
| 4.2 配置验证 | 保存前验证 API 连通性 | 0.5 天 |
| 4.3 快捷入口 | TUI 中 `Ctrl+S` 或 `/settings` 进入 | 0.5 天 |

### Phase 5: MCP (Model Context Protocol) 支持（高价值，长期）

**目标**: 支持 MCP Server，扩展工具生态（如文件系统、数据库、浏览器等外部工具）。这是与 Cline 拉齐的关键功能。

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| 5.1 MCP Client 封装 | 实现 MCP 协议客户端（stdio/sse 传输） | 3-5 天 |
| 5.2 工具桥接 | 将 MCP Server 的工具动态注册到 gline Tool Registry | 2-3 天 |
| 5.3 配置管理 | 支持 `~/.gline/mcp.json` 配置多个 MCP Server | 1-2 天 |
| 5.4 TUI 展示 | MCP 工具调用在 TUI 中正确显示 | 1 天 |

---

## 建议优先级与里程碑

```
近期（1-2 周）:
├── Phase 1: 存储层 + 任务历史管理（最基础，优先做）
├── Phase 3: /reload 动态刷新规则（小功能，快速赢）
└── 修复发现的 bug

中期（2-4 周）:
├── Phase 2: TUI 历史界面
└── Phase 4: 配置管理 TUI

长期（4-6 周）:
└── Phase 5: MCP 支持（战略级功能）
```

---

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0（`modernc.org/sqlite` 要求，自动升级）
- **操作系统**: Windows 11

## 参考资源

- [Cline 源码](./cline/) - 架构参考
- [Bubbletea 文档](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Cobra 文档](https://github.com/spf13/cobra) - CLI 框架
- [MCP Specification](https://modelcontextprotocol.io/) - MCP 协议规范
