# gline TUI 迁移方案

## 1. 目标

将 gline 从 Wails v3 GUI 桌面应用转为 **Bubbletea TUI** 终端应用，同时保留 GUI 模式作为可选项。

```
gline           → 默认启动 TUI 模式
gline --gui     → 启动 Wails GUI 模式（原有代码不变）
gline chat      → CLI 单次对话（已有）
gline history   → CLI 查看历史（已有）
```

## 2. 架构对比

### 当前架构
```
cmd/gline/main.go → runGUI() → Wails app → frontend/ (React)
                                   ↓
                           internal/gui/ (ChatService)
                                   ↓
                           internal/agent/ (核心)
```

### 目标架构
```
cmd/gline/main.go
  ├─ 无参数 → RunTUI() → Bubbletea program → internal/tui/
  ├─ --gui  → runGUI() → Wails app → internal/gui/ (不变)
  ├─ chat   → CLI 命令 (不变)
  └─ ...    → 其他 CLI 子命令 (不变)

internal/tui/
  ├─ app.go          主 Model（组合所有子 Model）
  ├─ chat.go         聊天消息列表视图
  ├─ input.go        输入区域（textinput + slash 补全）
  ├─ status.go       状态栏（provider/model/mode/tokens）
  ├─ sidebar.go      任务历史侧边栏（可折叠）
  ├─ callback.go     StreamCallback → tea.Msg 适配器
  ├─ styles.go       lipgloss 样式定义
  ├─ keys.go         键绑定定义
  └─ help.go         帮助面板
```

## 3. 技术选型

| 组件 | 库 | 用途 |
|------|-----|------|
| TUI 框架 | `github.com/charmbracelet/bubbletea` | Elm 架构，Model-Update-View |
| 组件库 | `github.com/charmbracelet/bubbles` | textinput, list, viewport, spinner |
| 样式 | `github.com/charmbracelet/lipgloss` | 终端样式（颜色、边框、布局） |
| Markdown | `github.com/charmbracelet/glamour` | 终端 Markdown 渲染 |
| Pager | `github.com/charmbracelet/bubbles/viewport` | 消息列表滚动 |

## 4. 核心设计

### 4.1 StreamCallback 适配器

Agent 的 `StreamCallback` 接口需要适配为 Bubbletea 消息：

```go
// internal/tui/callback.go
type TuiCallback struct {
    msgCh chan tea.Msg  // 发送 tea.Msg 到 Bubbletea program
}

// 将 Agent 事件转为 tea.Msg
func (c *TuiCallback) OnContent(delta string) {
    c.msgCh <- contentMsg{delta}
}

func (c *TuiCallback) OnToolCallStart(tc agent.ToolCall) {
    c.msgCh <- toolStartMsg{tc}
}

func (c *TuiCallback) OnToolCallComplete(tc agent.ToolCall, result string) {
    c.msgCh <- toolCompleteMsg{tc, result}
}

func (c *TuiCallback) AskFollowupQuestion(q string, opts []string) (string, error) {
    ch := make(chan string, 1)
    c.msgCh <- followupMsg{question: q, options: opts, answerCh: ch}
    answer := <-ch
    return answer, nil
}
// ... 其他回调
```

### 4.2 主 Model 结构

```go
// internal/tui/app.go
type appModel struct {
    // 布局
    width, height int
    sidebarOpen   bool
    
    // 子 Model
    sidebar  sidebarModel   // 任务历史列表
    chat     chatModel      // 消息列表 (viewport)
    input    inputModel     // 输入框 + slash 补全
    status   statusModel    // 状态栏
    
    // Agent
    agent     agent.Agent
    callback  *TuiCallback
    msgCh     chan tea.Msg
    
    // 状态
    isLoading bool
    mode      string  // "plan" / "act"
    followup  *followupState
    
    // 配置
    config *config.Manager
    store  storage.Store
}
```

### 4.3 消息流

```
用户输入 → inputModel.handleSubmit()
  → agent.RunWithCallback(ctx, prompt, tuiCallback)
    → callback.OnStreamStart()  → tea.Msg{streamStartMsg}
    → callback.OnContent(delta) → tea.Msg{contentMsg}  (多次)
    → callback.OnToolCallStart() → tea.Msg{toolStartMsg}
    → callback.OnToolCallComplete() → tea.Msg{toolCompleteMsg}
    → callback.OnComplete()     → tea.Msg{completeMsg}
  → chatModel.Update(msg) → 渲染新内容
```

### 4.4 布局设计

```
┌─ gline ─────────────────────────────────────── [Act] [openai/gpt-4] ──────┐
│ [Sidebar]  │  [Chat Area]                                                │
│            │                                                              │
│ ○ Task 1   │  👤 Refactor the auth module to use JWT                     │
│ ● Task 2   │                                                              │
│ ○ Task 3   │  🤖 I'll analyze the current auth module...                 │
│            │                                                              │
│            │  ┌ Tool: read_file ─────────────────────────────────────┐   │
│            │  │ ✅ Read 234 lines from internal/auth/handler.go      │   │
│            │  └──────────────────────────────────────────────────────┘   │
│            │                                                              │
│            │  🤖 Done! I've refactored the auth module to use JWT.       │
│            │                                                              │
├────────────┴──────────────────────────────────────────────────────────────┤
│ > Type a message... (Tab: mode, /: commands, Ctrl+H: history)            │
└──────────────────────────────────────────────────────────────────────────┘
```

- **Sidebar**: 固定宽度 24 列，可用 `Ctrl+B` 折叠
- **Chat Area**: 填充剩余宽度，viewport 组件支持滚动
- **Input**: 底部单行输入，支持多行（Shift+Enter）
- **Status Bar**: 顶部一行，显示 provider/model/mode/tokens

### 4.5 键绑定

| 按键 | 功能 |
|------|------|
| `Enter` | 发送消息 |
| `Shift+Enter` | 换行（多行输入） |
| `Tab` | 切换 Plan/Act 模式 |
| `Ctrl+C` / `Ctrl+D` | 退出 |
| `Esc` | 中断当前任务 |
| `Ctrl+B` | 切换侧边栏 |
| `Ctrl+L` | 清屏 |
| `Ctrl+H` | 显示/隐藏历史 |
| `Up/Down` | 输入历史 |
| `Ctrl+R` | 搜索输入历史 |

### 4.6 Slash 命令

在输入框中输入 `/` 时，弹出命令补全列表（类似 IDE）：

```
> /cl
  ┌──────────────────────────────┐
  │ /clear    Clear conversation │
  │ /compact  Compact context    │
  └──────────────────────────────┘
```

## 5. 实施阶段

### Phase 1: 基础框架（~200 行）
- [ ] 创建 `internal/tui/` 包
- [ ] 实现主 appModel（Model-Update-View）
- [ ] 实现 status 状态栏
- [ ] 实现 input 输入区域（textinput）
- [ ] 实现 chat viewport（消息列表 + 自动滚动）
- [ ] 修改 `cmd/gline/main.go` 支持 TUI 入口

### Phase 2: Agent 集成（~150 行）
- [ ] 实现 TuiCallback（StreamCallback → tea.Msg）
- [ ] 复用 `initializeAgent()` 初始化 agent
- [ ] 连接输入 → agent → 流式输出 → chat 显示
- [ ] 处理 streaming 光标动画

### Phase 3: 工具展示（~150 行）
- [ ] tool start 显示（折叠面板）
- [ ] tool complete 显示结果
- [ ] attempt_completion 特殊渲染
- [ ] ask_followup_question 交互式选择

### Phase 4: 任务历史（~150 行）
- [ ] sidebar 任务列表（bubbles/list）
- [ ] 任务加载（恢复对话）
- [ ] 任务删除
- [ ] `Ctrl+B` 折叠/展开

### Phase 5: Slash 命令（~100 行）
- [ ] `/` 前缀检测 + 补全列表
- [ ] 命令执行 + 结果显示
- [ ] 复用 `internal/slash/` 现有命令

### Phase 6: 高级功能（~100 行）
- [ ] 输入历史（Up/Down）
- [ ] Markdown 终端渲染（glamour）
- [ ] 帮助面板
- [ ] 配置查看/编辑

### Phase 7: 双模式入口（~30 行）
- [ ] `--gui` flag 走 Wails
- [ ] 无参数走 TUI
- [ ] CLI 子命令不变

**预估总代码量**: ~900 行 Go 代码

## 6. 依赖变更

### 新增
```go
require (
    github.com/charmbracelet/bubbletea v1.x
    github.com/charmbracelet/bubbles v0.x
    github.com/charmbracelet/lipgloss v1.x
    github.com/charmbracelet/glamour v0.x
)
```

### 保留（不删除）
- `github.com/wailsapp/wails/v3` — GUI 模式仍需要
- `frontend/` — GUI 前端仍需要
- `internal/gui/` — GUI 绑定层仍需要

## 7. 风险与注意事项

1. **Windows 终端兼容性**: Windows Terminal 对 ANSI 颜色支持良好，传统 cmd.exe 需要启用 VT100
2. **MCP 初始化**: MCP manager 启动是异步的，需要在 TUI 启动后后台初始化
3. **大消息渲染**: 超长 assistant 回复需要 viewport 滚动，不能一次性打印
4. **并发安全**: StreamCallback 从 agent goroutine 调用，通过 channel 发送到 Bubbletea 主循环是线程安全的
5. **工作目录**: TUI 启动时的 cwd 就是项目目录，不需要额外选择
