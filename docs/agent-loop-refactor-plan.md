# gline Agent Loop 重构方案 — 采用 pi-go / Google ADK 架构

> 2026-09-19 起草。目标：用 pi-go 已验证的架构（Google ADK Go v2 runner）替换 gline 手写的 agent loop，从结构上消除 orphan tool call、streaming 竞态、模型混乱补丁链问题。

---

## 1. 为什么要重构

gline 现有 agent core 是手写循环（`internal/agent/agent.go`，1260 行），长期靠"打补丁"维持：

| 问题 | 根因 | 已打的补丁（治标） |
|------|------|---------------------|
| orphan tool call（工具结果找不到对应的 tool_calls） | `preDispatchToolCall` goroutine 在流式期间抢先写消息，与主循环记录时序竞争 | 深拷贝 GetMessages、移除 orphan 检查 |
| 模型把 attempt_completion 混入其他工具参数 | 流式工具调用累加逻辑 + 弱模型对 Cline 式 attempt_completion 工具的困惑 | sanitizeToolCallArgs、ValidateToolInput、连续错误计数 |
| streaming 卡顿/交错错乱 | StreamChunk 通道 + 手工累加 partial tool call | 通道 64→1024、ReasoningEvent、OnStreamEnd |
| 空 stream / 缺 finish_reason | go-llm 提供商行为不一致 | InferMissingFinishReason、空流退避重试 |

**结论：这些补丁都在给同一个错误抽象打补丁。** pi-go 把整个循环交给 Google ADK 的 `llmagent` + `runner`，用 event-sourced session 作为唯一事实来源，以上问题**结构性不存在**：

- 工具调用/结果永远成对出现在 session event 流里（AppendEvent 原子追加）→ 无 orphan
- 没有 preDispatch，没有手工 partial 累加 → 无竞态
- 前端只消费 `iter.Seq2[*session.Event, error]` → 一种事件路径，TUI/CLI/GUI 共用
- pi-go 已在 Windows + opencode/mimo-v2.5 上验证过这套架构可用

## 2. 架构对比

```
【现状 gline】                                【目标 pi-go/ADK 模式】
TUI/CLI/GUI                                   TUI/CLI/GUI
   │ StreamCallback(12种Msg)                     │ iter.Seq2[*session.Event, error]
   ▼                                             ▼
internal/agent/agent.go (手写循环)             ADK runner.Runner
 ├ RunWithCallback for 循环                    ├ 自动工具循环（模型↔工具）
 ├ processStream(手工累加partial)              ├ session.Service（事件持久化）
 ├ preDispatchToolCall(流式抢跑)               ├ Before/After Model/Tool 回调
 ├ executeToolCallsParallel(3阶段)             └ compaction/stuck 检测（前端+ADK）
 ├ enforceTokenBudget/AutoCompact
   │ Provider 接口                              model.LLM 接口 (ADK)
   ▼                                             ▼
go-llm 库 (chatcompletions 适配)              internal/provider/* (官方SDK直连)
   │                                             │ openai / anthropic / opencode /
   ▼                                             │ openrouter / volcano(baseURL)
LLM API                                        LLM API
```

核心差异：**循环所有权**。现状是 gline 拥有循环、go-llm 拥有流；目标是 ADK 拥有循环和会话，gline 只提供 model、tools、callbacks、前端渲染。

## 3. 目标架构

```
cmd/gline (TUI 默认 / --gui / 子命令)
    │
    ├── internal/tui          前端：消费 event 流 → tea.Msg（重写 bridge）
    ├── internal/gui          GUI：ChatService 换成 event 消费（前端 React 不动）
    │
    ├── internal/agent        组装层（新写，~300 行）
    │    ├ New(cfg) → ADK llmagent + runner
    │    ├ InstructionProvider ← 系统提示词 + memory context（替代 buildMemoryContext）
    │    ├ BeforeModelCallback ← Plan/Act 指令切换
    │    ├ BeforeToolCallback  ← Plan 模式拦截 / yolo 放行 / 审批确认
    │    ├ AfterModelCallback  ← usage 统计、facts 抽取、token 水位检查
    │    ├ WithRetryContext    ← 从 pi-go 移植的重试包装
    │    └ RunStreaming(ctx, sessionID, prompt) iter.Seq2[*session.Event, error]
    │
    ├── internal/provider     ADK model.LLM 实现（从 pi-go 移植）
    │    ├ opencode.go        ← 直接移植（含 x-opencode-session 修复）
    │    ├ openai_completions.go ← 移植，volcano 走 baseURL 复用
    │    ├ anthropic.go / openrouter.go ← 移植
    │    └ provider.go        ← NewLLM(ctx, info, ...) 工厂
    │
    ├── internal/tools        gline 工具实现保留，加 ADK 适配层
    │    ├ adk_adapter.go     ← functiontool.New 包装（类型化 args）
    │    ├ mcptoolset_bridge.go ← 现有 MCP manager → ADK toolset
    │    └ sandbox.go         ← 已做的 Windows 放宽保留
    │
    ├── internal/storage      保留：任务索引(task ↔ sessionID 映射)、history
    ├── internal/sessionstore ← ADK GORM SQLite session service
    │                            (~/.gline/sessions.db, glebarez/sqlite 纯Go)
    ├── internal/subagent     保留 in-process，内部用嵌套 runner(InMemory)
    ├── internal/memory       保留，挂到 InstructionProvider / AfterModelCallback
    └── 删除: internal/api/go_llm.go, internal/agent/provider.go,
             go-llm 依赖, pkg/types Conversation 循环职责
```

## 4. 关键决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 引入 `google.golang.org/adk/v2 v2.4.0`（与 pi-go 同版本） | pi-go 全链路验证过；同版本便于直接移植代码与对照排障 |
| D2 | **删除 go-llm**，provider 从 pi-go 移植（官方 SDK 直连） | go-llm 的 RawMessage 语义是多个补丁的来源；pi-go 的 provider 自带 retry/ratelimit/finish_reason 归一化；volcano 是 OpenAI 兼容协议，用 openai provider + baseURL 覆盖 |
| D3 | session 存储用 ADK 官方 `session/database`（GORM + glebarez/sqlite，纯 Go 无 CGO）独立文件 `~/.gline/sessions.db` | 零自研成本、过 ADK conformance suite；gline 现有 SQLite 保留为任务索引（task 行存 session_id + 标题/状态/时间，`gline history` 继续工作） |
| D4 | 工具：gline 的 read/write/edit/run/search 等实现保留，包一层 `functiontool.New`（类型化 Input/Output struct） | 业务逻辑已验证；只换接口皮。MCP 不用 ADK mcptoolset，现有 MCP manager 桥接为 toolset（已修过 Windows 日志/transport 坑） |
| D5 | **移除 attempt_completion 工具**。"完成" = 模型不再调用工具、turn 结束（TurnComplete/FinishReason=STOP） | pi-go 模式没有 Cline 式 completion 工具；mimo-v2.5 对 attempt_completion 的参数混乱正是补丁重灾区。TUI 在 turn 结束事件上渲染完成态 |
| D6 | Plan/Act：mode 存在 Agent 配置里，BeforeModelCallback 换 instruction、BeforeToolCallback 拦截写工具；切模式调用 RebuildWithInstruction（pi-go 已有此方法） | 复用 pi-go 模式，无自研状态机 |
| D7 | yolo/auto-approve：BeforeToolCallback 返回放行；非 yolo 用 ADK `tool/toolconfirmation` 机制出确认 | 标准扩展点 |
| D8 | stuckDetector + StreamDedup + retry 从 pi-go `internal/tui/agent_loop.go` / `internal/agent/retry.go` 移植到 gline 前端桥 | 防死循环/重复调用，pi-go 已调优 |
| D9 | go 指令 1.26 → 1.27（与 ADK/pi-go 一致） | ADK v2 要求 |
| D10 | 旧循环在新路径稳定前不删：`internal/agent` 保留，新实现放 `internal/agent` 顶层重组（Phase 内先 `agentv2` 并存，切换后删旧文件） | 每阶段可发布、可回滚 |

## 5. 事件流映射（唯一需要适配面）

| ADK Event / 字段 | gline 前端消息（现有渲染保留） |
|---|---|
| `ev.Content.Role=="thinking"` / reasoning part | ThinkingMsg（现有 thinking 渲染） |
| `ev.Partial==true` 的 Text part | 流式文本（打字机 + cursor） |
| `ev.Partial==false` Text part（StreamDedup 去重后） | 完整 assistant 消息 → glamour markdown |
| `part.FunctionCall` | ToolCallMsg → 工具面板（输入摘要） |
| `part.FunctionResponse`（`{"error":...}` 判错误） | ToolResultMsg → ✓/✗ + 结果截断 |
| `ev.UsageMetadata` | 状态栏 tokens |
| `ev.FinishReason==MaxTokens` | 截断警告 |
| `agent.EventError(ev)` | 错误消息 |
| turn 结束（无新 FunctionCall 且 TurnComplete） | ✅ 完成态（替代 attempt_completion 横幅） |

## 6. 分阶段实施

> 每个 Phase 结束 `go build ./... && go test ./...` 全绿、`gline.exe` 可安装可日用。Phase 3 前旧循环仍是默认路径。

### Phase 0 — 基线（0.5 天）
- 记录现有行为：给 `internal/agent` 补 2-3 个黄金测试（fake provider → 工具调用 → 结果回灌 → 完成），固定现有 UX 快照（TUI 渲染截图/文本）
- 升级 go 1.27，`go mod tidy` 确认无涟漪
- **产出**：`agent_golden_test.go`、go.mod 变更

### Phase 1 — 引入 ADK + session 存储（0.5 天）
- 依赖：`google.golang.org/adk/v2 v2.4.0`、`gorm.io/gorm`、`github.com/glebarez/sqlite`
- `internal/sessionstore/service.go`：`database.NewSessionService(gorm.Open(sqlite.Open("~/.gline/sessions.db")))` 的薄封装 + 路径管理 + 迁移自检
- tasks 表加 `session_id` 列（沿用现有 storage 迁移模式）
- **产出**：session service 冒烟测试（Create→AppendEvent→Get 往返，含 Compaction 字段）

### Phase 2 — provider 移植（1 天）
- 从 pi-go 复制并瘦身：`provider.go`(工厂)、`opencode.go`、`openai_completions.go`(+tools/thought_signature)、`anthropic.go`(+caching)、`openrouter.go`、`retry.go`、`ratelimit`
- volcano = openai provider + `baseURL: https://ark.cn-beijing.volces.com/api/v3`（配置层加映射）
- 单元测试：每个 provider 用 pi-go 现成 `_test.go` 改造；真机 smoke（opencode/mimo-v2.5 一条 echo）
- **产出**：`internal/provider/*`，go-llm 仍保留未引用

### Phase 3 — agent core 组装（1 天）
- `internal/agent/agent.go` 重写为组装层：Config{Provider, Tools, Instruction, Mode, Yolo, SessionService} → llmagent + runner
- InstructionProvider：系统提示词（现有 ~80 行版）+ cwd + skills 菜单 + memory context
- 回调：Plan/Act、yolo/审批、usage 记账
- `Run(ctx, sessionID, prompt)` / `RunStreaming` 返回事件流；`WithRetryContext` 包装
- **产出**：agent 级测试（fake model 驱动多轮工具循环，断言 session 中 call/response 成对 → 无 orphan 的结构证明）

### Phase 4 — 工具适配 + MCP（1 天）
- `adk_adapter.go`：泛型包装 `newTool[TIn, TOut](name, desc, handler)`（pi-go registry.go 同款），逐个接 gline 现有工具；**去掉 attempt_completion**（D5）
- Plan 模式工具清单过滤（读 vs 写分类沿用现有 filterPlanModeTools 语义）
- MCP manager → toolset 桥接；工具结果里的大输出走现有 Layer1/2 治理（read 分窗、summarize_file）
- **产出**：每个工具一个适配测试；MCP 集成 smoke

### Phase 5 — 前端桥重写（1.5 天）
- `internal/ui/bridge/event_bridge.go`：`iter.Seq2[*session.Event, error]` → tea.Msg（按第 5 节映射表）；移植 stuckDetector/StreamDedup
- 状态栏数据源改 UsageMetadata 累计；turn 完成态渲染
- `cmd/gline/chat.go`（TUI 装配）与 `gline chat` CLI 子命令切到新 agent；GUI ChatService 内部换引擎（Wails 绑定不变，React 零改动）
- **产出**：TUI 手测清单（streaming/工具面板/thinking/取消/Plan-Act 切换/yolo/history 续接）

### Phase 6 — 治理能力对齐（1 天）✅（2026-09-20 ~ 09-21）
- ✅ AutoCompact：ADK `session/compaction` 尾部保留（TokenThreshold 80k / EventRetentionSize 12），CLI 与 GUI 装配均传入；TUI `/compact` 对 ADK 路径提示自动处理
- ✅ facts 抽取：`internal/adkagent/facts.go`，RunWithCallback 干净完成后异步抽取（Options.MemoryEngine）
- ✅ `gline history` 续接：sessionstore（Phase 6a）+ ResumeSession（TUI/GUI 历史选中恢复 ADK session）
- ✅ GUI 切换（Phase 6b part 2）：`internal/ui/runner.go` 能力接口（WorkingDirSetter/TaskIDProvider/TaskResetter/SkillsSetter/MemoryProvider/ConversationProvider/Compactor/RulesReloader/ResumeSessionResumer）；`ChatService`/`Backend` 全部改为 capability 断言，`Backend.ag` 类型改为 `ui.AgentRunner`；GUI 默认 ADK 循环（`GLINE_AGENT=legacy` 回退）；`--gui` 标志恢复；GetConversationState ADK 路径从 storage 读持久化转录；LoadTask 恢复 ADK session（无 session 的旧任务回退 SetTaskID）；ADK agent 支持动态 SetWorkingDir/SetSkills（InstructionProvider 每次调用重建指令）
- ⏳ 长会话压测脚本（>100 轮工具调用无 orphan、无内存涨）——待办

### Phase 7 — 切换与清理（0.5 天）

**Phase 7a — 删除 legacy 循环（ADK-only）✅（2026-09-21）**

- ✅ `internal/agent/agent.go` 重写：仅保留 `Mode`/`ModePlan`/`ModeAct` + 包文档；删除 `Agent` 接口、`BaseAgent`、`Options`、`New`、`RunWithCallback/processStream/processResponse`、`preDispatch*`、`executeToolCallsParallel`、`enforceTokenBudget`、`AutoCompact/Compact`、`convertTools`、`filterPlanModeTools` 等全部手写循环代码（-1200 行）；删除 `agent_golden_test.go`、`agent_test.go`；`provider.go`（Provider 接口 + 流式类型）与 `summarizer_caller.go` 保留
- ✅ `internal/ui/runner.go`：删除 `legacyRunner` 与 `LegacyRunner()`；孤儿接口 `ConversationProvider`/`Compactor`/`RulesReloader` 删除；`AdkRunner()` 工厂 + 全部能力接口编译期断言
- ✅ `internal/ui/tui.go`：`/compact` 恒提示自动压缩；`loadTaskMessages` 删除 legacy 转录回放块
- ✅ `internal/gui/backend.go`：`initAgent` 移除 `GLINE_AGENT=legacy` 分支；`buildLegacyProvider` 简化为 `(agent.Provider, error)`（仅供 sub-LLM 工具）；`LoadTask` 删除 legacy 回放分支
- ✅ `internal/gui/chat_service.go`：`GetStatus`/`GetConversationState`/`CompactConversation`/`reloadRules` 全部 ADK-only；`ChatServiceTest` 重写为 fake AgentRunner（6 个用例）
- ✅ `cmd/gline/chat.go`：`initializeAgent` ADK-only；mock provider 报错删除；**summarize_file + use_subagents 移到共享注册路径**（修复 CLI ADK 路径缺失这两个工具的缺口）；删除 `legacyAgentBundle`
- ✅ 删除 `internal/api/mock.go`（无引用）；go-llm 仍保留（Phase 7b）
- ✅ 验证：18 包全绿、vet 干净、CLI live smoke（PONG-7A-OK）、GUI smoke（ADK 日志）、GLINE_LIVE_SMOKE 3 项（OneShot/ToolRun/HistoryResume）全过、已重装 `C:\Users\22569\bin\gline.exe`

**Phase 7b — port sub-LLM 工具并删除 go-llm ✅（2026-09-21）**

- ✅ `internal/subagent`：`Builder.Provider`（agent.Provider）→ `Builder.LLM`（model.LLM，internal/provider）；`NewBuilder`/`RegisterTool` 签名同步更换；runner 主循环改为 genai.Content 会话 + `GenerateContent(ctx, req, true)` 流式迭代，累积 text/FunctionCall/UsageMetadata；工具声明改为 `genai.FunctionDeclaration`（ParametersJsonSchema）；工具结果包装为 FunctionResponse（`{"result": ...}`）；`convertTools`/`agent.ToolCall`/`agent.StreamChunk` 依赖全部消除；Token 治理由 `types.Conversation.TrimToMaxTokens` 改为粗估上限（>200k 失败，不静默截断）
- ✅ `internal/agent/summarizer_caller.go` 删除（无引用；summarizer 走 `subagent.SubagentSummarizerCaller`）
- ✅ `cmd/gline/chat.go` + `internal/gui/backend.go`：sub-LLM 改用 `provider.NewLLM(...)`（`internal/provider` 工厂，opencode-go→opencode 映射不变）；GUI memory Caller 同步迁到 model.LLM 非流式调用；`api.OpenCodeGoBaseURL` 常量内联
- ✅ **删除整个 `internal/api` 包**（go_llm.go/openai.go/opencode.go/registry.go/mock 早已删）；`go mod tidy` 移除 go-llm 依赖（go.mod/go.sum 零残留）
- ✅ 验证：17 包全绿、vet 干净、CLI live smoke（PONG-7B-OK）、**subagent 端到端真机 smoke（use_subagents → SUBAGENT-7B-OK）**、GUI smoke（ADK 日志）、GLINE_LIVE_SMOKE 3 项全过、已重装 `C:\Users\22569\bin\gline.exe`

**原 Phase 7 描述**：
- 删除 `internal/api/go_llm.go`、旧 `RunWithCallback/processStream/preDispatch/executeToolCallsParallel`、go-llm 依赖
- 全量回归 + 重装 `C:\Users\22569\bin\gline.exe`
- memory bank 更新（systemPatterns/progress/activeContext）

**总计 ~6.5 个工作日**（含测试），关键路径是 Phase 2→3→5。

## 7. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| ADK event 语义与现有渲染假设不合（如 thinking 分片粒度） | 前端返工 | Phase 5 先做只读渲染快照对照，映射表作为契约文档 |
| volcano 走 openai-compat 的工具调用兼容性 | volcano 不可用 | Phase 2 真机 smoke；不行则临时保留 go-llm 仅 volcano 分支 |
| GORM session 表与现有任务历史双源 | history 混乱 | D3 明确分工：session=事实来源，tasks=索引；续接走 session_id |
| mimo-v2.5 在新循环下仍有坏工具调用 | 体验未改善 | stuckDetector + ValidateToolInput 保留；同时此方案后可低成本切更强模型验证 |
| Windows os.Root / 路径问题复发 | 工具失败 | 已放宽的 sandbox 保留；pi-go 同栈已在 Windows 验证 |
| 与 pi-go 上游漂移 | 移植维护成本 | provider/retry/detector 文件头标注来源 commit，后续可脚本化 diff |

## 8. 明确不动的部分

- 现有 TUI 视觉与交互（输入框/侧栏/slash 补全/Plan-Act 快捷键）
- GUI React 前端与 Wails 绑定
- 存储库表结构（只加列不破坏）
- 四层记忆引擎、slash 系统、MCP manager 内部实现
- 工具业务逻辑（read 分窗、edit 精确匹配、run 沙箱）

## 9. 验收标准

1. 连续 50+ 轮工具会话，session 表中每个 FunctionResponse 都能回溯到同事件或前序事件的 FunctionCall（结构性无 orphan）
2. `gline`（TUI）/ `gline chat`（CLI）/ `gline --gui` 三入口全走新循环，行为一致
3. opencode/mimo-v2.5 与 volcano（或任一第二 provider）可配置切换
4. Plan 模式下写工具被拦截；yolo 全放行；非 yolo 出确认
5. `gline history` 列表与续接正常
6. 全测试套件绿，无 go-llm 残留依赖
