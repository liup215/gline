# Progress

## 2026-09-25 — 关闭按钮收进托盘 ✅（已部署）

**之前行为**：点 X 直接退出整个应用（`unregisterWindow` 发现 windowMap 空 → `PostQuitMessage`，托盘一并退出）。

**新行为**：X → 隐藏到托盘（应用继续运行）；托盘菜单 Quit → 真正退出。

**实现**：alpha.63 无 `HideOnClose` 选项，用 `RegisterHook(Common.WindowClosing)` + `event.Cancel()` + `window.Hide()` 拦截；托盘 Quit 先置 `quitting`（atomic.Bool）再 `app.Quit()` 放行正常销毁。机制依据：`HandleWindowEvent` 中 hooks 先于 listeners 执行，Cancel 后内部销毁 listener 不运行；X 按钮的 Windows.WindowClosing(1204) 经 DefaultWindowEventMapping 映射到 Common.WindowClosing(1028)。同时纠正 D89（“WindowClosing 失焦误触发”实为 AttachWindow 失焦隐藏的误诊，事件 ID 不同）。

**验证**：vet/build 通过；两二进制已部署 `C:\Users\22569\bin\`。

## 2026-09-25 — messages.created_at 零值修复 + 历史数据回填 ✅（已部署）

**现象**：用户分析数据库发现 session 对话记录 created_at 全是空值。

**排查结论**：sessions.db（ADK 库）时间完好；gline.db 的 tasks/tool_calls 也正常；唯独 messages 表 created_at 全为 Go 零值 `0001-01-01`（非 SQL NULL）。根因：ADK 持久化路径 `transcriptAccumulator` 构造 `types.Message` 从不设 Timestamp → `SaveMessage` 把零值直写入列，覆盖了 `DEFAULT CURRENT_TIMESTAMP`。内存态 `Conversation.AddMessage` 的零值补时保护不到这条路径。

**修复（两层）**：
1. `internal/adkagent/persist.go`：accumulator 用 `ev.Timestamp`（ADK event 真实时间）填充 assistant/tool/user 消息；`addUserPrompt` 用 `time.Now()`。
2. `internal/storage/sqlite.go` `SaveMessage`：`Timestamp.IsZero()` 时补 `time.Now()`（存储边界兑底，防未来调用方）。

**历史数据回填**：`go run ./cmd/dbcheck/ -fix` 将零值行 `created_at` 更新为其所属 task 的 created_at（近似）；验证 zero_time=0。dbcheck 是保留的诊断工具（默认只读，`-fix` 才写）。

## 2026-09-25 — 托盘失焦隐藏根因修复 ✅（已部署）

**根因（wails v3.0.0-alpha.63 源码）**：`SystemTray.Run()` 对 `AttachWindow` 的窗口强制注册 `WindowLostFocus → Hide()` 监听器（systemtray.go，无开关，popover 设计）。之前怀疑的 WindowClosing hook 只是表象。

**修复**：不 `AttachWindow`，改 `systemTray.OnClick()` 手动切换（`cmd/gline-gui/main.go` + `cmd/gline/gui.go`）。托盘功能完整保留；alpha2.106 已将此行为改为 opt-in `HideOnFocusLost` 选项，未来升级后可重新评估。

## 2026-09-25 — chat completions 路径 thinking 提取 ✅（已部署）

**根因**：OpenAI chat completions 路径（mimo-v2.5 等）`oaiRunStreaming`/`oaiRunNonStreaming` thinking hook 传 nil，`delta.reasoning` 被静默丢弃。

**修复**：`internal/provider/openai_completions.go` 新增 `chatDeltaThinking` 通用提取器（`delta.reasoning` 字符串 + `delta.reasoning_details[].text` 数组），streaming/non-streaming 双路接入 → `Role:"thinking"` partial → bridge `OnReasoning` → GUI `chat:reasoning` → 前端 Thinking 折叠区。

## 2026-09-24 — GUI 流式事件乱序修复 ✅（提交 a2daf24，已部署）

**症状**：GUI 聊天页模型输出语序错乱（"me先用Let us fly trap-rag 搜索 ven相关内容。"），而 SQLite 里存的原型完整（"Let me先用 go-rag 搜索 venus fly trap 相关内容。"）→ delta 全部送达、只是被重排。

**根因**：Wails v3 `EventProcessor.Emit`（alpha.63 / alpha2.106 均如此，升级无效）每个事件起 2 个 goroutine 分发（dispatchEventToListeners + dispatchEventToWindows），高频 content delta 竞争导致到达 JS listener 的顺序不确定。TUI 不受影响（TUIBridge 有序 channel）；持久化不走 callback 路径所以 DB 干净。

**修复**：
1. Go：`guiStreamCallback` 增 `seq atomic.Uint64` + `emit()` helper，所有流式事件带单调序号（error/taskCreated 除外）。
2. 前端：useChat.ts ordered dispatcher —— expected 锚定首事件，乱序入 stash 按序重放，旧重复（seq<expected）丢弃（顺带修复双注册 listener 重复追加）。

**验证**：go build/vet/test 全绿；tsc --noEmit 通过；vite build + go build 重新部署 bin/gline.exe。

## 2026-09-21 — TUI ask_followup_question 交互链路修复 ✅（提交 db9dc9f，已部署）

**症状**：TUI 里 ask_followup_question 交互不完成，且打破底部 input area。

**根因**：ADK 适配层直接执行工具，而 `AskFollowupQuestionTool.SetHandler` **全仓零调用** —— handler 为 nil，工具落入 CLI stdin 回退（`fmt.Println` + `bufio.NewReader(os.Stdin)`），在 agent goroutine 里与 Bubbletea 抢 stdin：按键被偷走/原始文本破坏 alt-screen 渲染，交互永远无法完成。之前为 legacy loop 建的整套 TUI picker 机制（AskQuestionEvent → pendingQuestions → 选项选择器，commit a3e7d88）在 ADK 路径上够不着。

**修复**：`adkagent/tool.go` adkTool.Run 在执行前把共享工具实例的 handler 重接到**当前 run 的 callback**（与审批同路径；每次 run 独建 bridge、串行 run，覆写幂等）。三端自动全通：TUI 选项 picker / GUI 对话框（guiStreamCallback 已实现）/ CLI prompt（printCallback 已实现）。工具 Execute 加 nil-ctx 防御（headless 调用不 panic）。

**测试**：`tool_askfollowup_test.go` TestAskFollowupRoutesThroughStreamCallback（callback 收到问题+选项、答案映射进结果）。

**遗留**：GUI 侧 backend.go 从未把 guiStreamCallback 接给工具（同样靠本次修复惠及）；若 GUI 未来有自己的交互控件，验证其 AskFollowupQuestion 实现路径。

## 2026-09-21 — TUI 侧 skill 链路断裂修复 ✅（已部署）

用户问“skill 加载了吗”。审计：**GUI 侧链路完整，TUI 侧两头全断**（提交 2193347）：

| 环节 | GUI | TUI（修复前） |
|------|-----|------|
| Registry 创建 | ✅ | ✅ |
| LoadFromDirs 扫描 DefaultSkillDirs | ✅ | ❌ 从不调用（空注册表） |
| Skills 传 adkagent → SystemInstruction # Skills 段 | ✅ | ❌ Options 无 Skills 字段（模型不知道有技能） |
| use_skill 工具 | ✅ | ✅（但空表，永远 not found） |

修复：assembleSharedComponents 内 LoadFromDirs（~/.gline/skills、~/.agents/skills、~/.cline/skills、~/.claude/skills；缺目录非致命）；initializeAgent 接住 skillReg 传入 Options.Skills。

**孤儿资产**：`internal/skills/builtin/*.yaml`（code-review/debug/doc/explain/refactor 5 个内置技能）无人加载 —— yaml prompt 格式，LoadSkillsFromDir 只认 SKILL.md。待定：转 SKILL.md 格式 + embed 加载，或删除。用户本机 ~/.agents/skills、~/.claude/skills 有大量第三方技能（GUI/TUI 现在都会列出）。

## 2026-09-21 — 旧机制清理审计（D82）✅（已部署）

用户问“多余的、旧的机制是否都已清理”。全仓审计结果：**已确认干净** + 新删 3 处（提交 002edde，-169 行）：

**本次删除（pre-ADK 时代死代码，0 外部调用）：**
- `tools/init.go` GetDefaultTools/GetToolsForMode/IsToolAllowed —— 旧手写循环的硬编码分发辅助
- `prompts/system.go` GetToolDescriptions/GetPlanModeToolDescriptions/GetActModeToolDescriptions —— 硬编码 15 工具描述表（buildToolSection 保留，subagent GetSystemPrompt 锚定）
- `pkg/types/tool_names.go` IsSpecialTool —— 0 调用者

**确认已干净的：** internal/tui ✓ internal/api ✓ internal/agent 只剩共享契约 ✓ ui 包只用 adkagent ✓ retry 被 provider 用 ✓ ls recursive schema ✓ requires_approval ✓

**有意保留（parked，非垃圾，勿重复清理）：**
- 6 个下架工具构造函数 + RegisterSummarizeFileTool + subagent.RegisterTool（D80 重启用预留）
- subagent + summarizer 整包（use_subagents 停用但基础设施完整）
- attempt_completion 注册 + legacyToolNames 对（D81 subagent 终止契约）
- plan_mode_respond 的 TUI/GUI 渲染路径 + use_mcp_tool ToolName 常量（旧任务历史显示兼容）
- frontend format.ts list_code_definition_names 图标 case（同上）

**遗留文档（backlog 非机制）：** roadmap.md 过期、agent-loop-refactor-plan.md 剩压测项、memory-bank/activeContext.md 待归档。

## 2026-09-21 — 自定义规则从未注入的 bug 修复 ✅（已部署）

用户问“gline 有没有加载 rule”。排查结论：**机制存在但主循环从不注入** —— `~/.gline/rules/memory_bank.md` 从未进入系统提示词。

| 断点 | 详情 |
|------|------|
| 核心 | `adkagent.SystemInstruction()` 硬编码提示词，从不调用 LoadCustomRules |
| 误导 | `chat.go` `_, _ = prompts.LoadCustomRules()` 加载后丢弃（喂 subagent builder 的历史残留）|
| 死代码 | `gui/backend.go` 还有一份重复的 loadCustomRules/loadRulesFromDir 本地实现，裁剪后无人调用 |

**修复（提交 7d16c2a）**：
- `SystemInstruction()` 在 cwd 之后、Skills 之前追加 `prompts.LoadCustomRules()` 输出；**每次模型调用重读**（与 Plan/Act 动态渲染同哲学，规则热改立即生效，文件小成本可忽略）
- 清理 chat.go 丢弃调用 + backend.go 重复实现
- 新测试 `TestSystemInstructionIncludesCustomRules`（t.Chdir + workspace rules；负例只断言 Workspace Rules 缺席 —— 因为本机 HOME 真有全局规则）

**规则机制备忘**：全局 `~/.gline/rules/*.md|txt` + 工作区 `./.gline/rules/*.md|txt`，分别加 `# Global Rules` / `# Workspace Rules` 标题后拼接。

## 2026-09-21 — 参数修复层 + pi 式行为化描述（D80）✅（已部署）

背景：用户反馈 gline“用起来笨、总乱用工具”。对比 pi-go（`C:/Users/22569/Documents/20-Projects/pi-go/internal/tools/`）后定位根因：不是描述文字，而是**容错层缺失** —— 模型按 Claude Code/pi 约定发 `file_path`/`offset`/`"3"` 时，gline 严格 schema 直接报错，浪费一轮。

**D80 设计**：
- `internal/tools/argrepair.go` 新增 `RepairToolArgs(toolName, schema, args)`：① 参数别名表（read 12 个别名；edit 接受 old_string/new_string；grep 接受 pattern/query/include；run 接受 cmd/workdir；等）② 类型强转（按工具 schema 声明的类型：字符串数字→int、`"true"`→bool、积分 float64→int、JSON 字符串→array/object）③ **不可解析的值原样保留** —— 让工具自己的错误到达模型，不做静默猜测
- 别名只填充 canonical 名未设的字段；canonical 值优先（防覆盖）
- 接入两条执行路径：`adkTool.Run`（ADK 主循环）+ `subagent/runner.go`（已确认 ADK `base_flow.go:1386` 在 Run 前无 schema 校验，修复一定生效）
- **run 描述烤入启动时解析的 shell**（pi 规则：“只有描述告诉模型该写哪种语法”）—— bash 机器写 POSIX 指引，cmd 机器写 cmd 语法指引；并写明超时语义（SECONDS、到期被杀、长任务主动调大 timeout）+ 路由指引（读文件用 read、搜代码用 grep/glob，不要 cat/find）；删除死参数 `requires_approval`（schema+struct）
- 描述增强：read（截断后从 next line_number 续读、禁止 cat/head）、edit（EXACT 匹配强调 + 新文件用 write）、write（小改动用 edit）、ls（glob/grep 路由）、grep（优先于整读）

**验证**：build ✓ vet ✓ 全套测试 ✓ 受影响包 -race ✓ 已部署（提交 e2279cf）。新测试 `argrepair_test.go` 8 个用例（含自映射删除回归守卫）+ `command_test.go` 2 个（描述不许可 hedge、schema 无 requires_approval）。

**可选后续**：pi 还有 read/write 账本互锁（write 拒绝覆盖未 read 过的文件）—— 未移植，属行为变更需用户确认。

## 2026-09-21 — 工具裁剪：6 个工具下架 ✅（已部署）

用户决策“工具太多”，从 ADK 工具面下架：`list_code_definition_names`、`summarize_file`、`use_subagents`、`web_fetch`、`browser_copy`（`attempt_completion` 此前已被 legacyToolNames 隐藏；保留注册因 subagent 终止依赖它）。

| 改动 | 文件 | 说明 |
|------|------|------|
| 移除 4 个注册块 | `internal/tools/init.go` | `sum` 参数保留以兼容签名；构造函数与 RegisterSummarizeFileTool/subagent.RegisterTool 保持可编译，随时可恢复 |
| 拆除 sub-LLM 装配 | `cmd/gline/chat.go` | subLLM/subBuilder/sum 只服务这两个工具，全删；`skillReg` 改 `_` 接收 |
| GUI 同步裁剪 | `internal/gui/backend.go` | subLLM **保留**（memory engine 依赖 initMemoryEngine(subLLM)）|
| AllowedTools 收紧 | `internal/subagent/builder.go` | 移除 list_code_definition_names |

**裁剪后广告给模型的工具（9 个基础）**：read, ls, write, edit, grep, glob, run, ask_followup_question, use_skill（+ 可选 kb/memory 工具与 MCP server 工具）。

**附带发现**：`prompts/system.go` 的 hardcoded GetToolDescriptions（含 15 工具清单）在 ADK 时代是**死代码**（SystemInstruction 不嵌入工具清单，工具面完全由 registry 决定）—— 后续可删除。`memory-bank/tool-reference.md`（旧会话遗留）已加时效标注。

验证：build ✓ vet ✓ 全套测试 ✓ 受影响包 -race ✓ 已部署 C:\Users\22569\bin\gline.exe（提交 4d919b0）。

## 2026-09-21 — run 工具默认改用 bash 执行 ✅（已部署）

新建 `internal/shell` 叶子包（避免循环依赖），`tools`/`prompts`/`subagent` 共用：

| 平台 | 解析顺序 | 回退 |
|------|---------|------|
| Windows | 显式 Git 安装路径（Program Files/LOCALAPPDATA）→ PATH 中的 bash.exe（**排除 System32/WindowsApps 的 WSL stub**，路径与 cwd 语义不兼容） | `cmd /C` |
| Unix | PATH 中的 bash | `sh -c` |

- `shell.Resolve()` 进程内 sync.Once 缓存；`Name()` 返回 bash/cmd/sh 标签
- `command.go`：`exec.CommandContext(sh.Path, -c, cmd)` + 补上了 `hideConsole`（GUI 模式不再闪控制台窗）
- **修复既有不一致**：提示词 System Info 与 subagent 环境块原先硬编码 bash/PowerShell 标签，而 Windows 实际执行是 `cmd /C`；现在报告真实解析结果（本机：Git Bash → `Shell: bash`）
- 测试：`internal/shell` 解析规则 + `command_test.go` 用 `$((2+3))` 算术探针验证 bash 语义（cmd 无法求值）
- 本机解析结果：`C:\Program Files\Git\usr\bin\bash.exe`（PATH 首位）

## 2026-09-21 — 工具重命名：grep / glob / ls ✅（已部署）

三个文件工具统一改为简短命名（对齐 Claude Code Glob/Grep 惯例，主流模型训练数据熟悉度高）：

| 旧名 | 新名 | 后端 |
|------|------|------|
| search_files | **grep** | rg（内容搜索） |
| find_files | **glob** | fd（文件名 glob） |
| list_files | **ls** | ReadDir（目录列表，保留三个工具） |

- Go 标识符同步：GrepTool/GlobTool/LsTool + ToolGrep/ToolGlob/ToolLs 常量；SearchFilesOutput 等内部管道类型未改
- subagent AllowedTools 补上了 glob（此前漏了 find_files）；debug skill 提示词旧引用同步修正
- 覆盖面：registry/init、prompts 规则与 Code Search Strategy、ui/view 别名（新增 ls→listed）、ui/tool registry、GUI format.ts、全部测试
- 提交：rename 31 文件（含 gofmt 行尾规范化噪音，测试全绿验证无害）+ debug.yaml 修补

## 2026-09-21 — rg 搜索性能优化：流式 + 早停 + 计时 ✅（已部署）

依据外部方案评价实施（方案 1/2/4，未采纳 3（改输出）和 5（手写 JSON 解析））:

| 项 | 实现 | 说明 |
|----|------|------|
| 早停（最大杠杆） | parseRipgrepStream + onCap 回调 kill | 达到 500 上限即 `cmd.Process.Kill()`，rg 停止扫剩余仓库；宽泛 pattern 下跳过大部分扫描+解析 |
| 流式解析 | cmd.Output() → StdoutPipe | 解析与 rg 执行重叠；消除 string(out) 二次拷贝 |
| cap 语义不变 | stop() 内截断到 500 + TotalMatches 钳制 | Execute 的后置 cap 保留为安全网；显示仍 "Found 500 matches" |
| 取消语义不变（D66） | killed 标志区分内部 kill vs ctx 取消 | 父 ctx 取消/超时仍传播，不回退 |
| Debug 计时 | rg 路径 total vs parse 拆分；Go 回退路径计时 | total-parse ≈ 进程启动/杀软开销，用于归因 |

**外部方案评价结论**: 流式+减事件量方向对，但漏了早停这个真正杠杆；"1.8MB 内存拷贝"论据不成立（<1ms）；-C 1/--max-columns 200 属输出变更需改测试，暂缓。

**测试**: 新增 TestSearchFilesRipgrepEarlyStop（12 文件×60 match=720，断言恰 500 + TotalMatches 钳制）；全套 -race 绿，已部署。

## 2026-09-21 — read 工具对齐 Pi 基准优化 ✅（已部署）

依据 Pi vs gline read 差距表优化（internal/tools/file.go）:

| 维度 | 优化前 | 优化后 |
|------|--------|--------|
| 分页 | 仅 line_number,固定 200 行 | + 可选 `limit`（默认 **1000**〔2026-09-21 由 200 调高〕,上限 2000 行/次） |
| 路径 | 仅 filepath.Clean | `~`/`~/` 展开 + 相对路径解析为绝对路径（输出前缀稳定） |
| 图片 | 二进制乱码进上下文 | 按扩展名检测 → 元数据提示（无多模态管道,Content 纯 string,不塞 base64） |
| 错误 | 基础 | 越界报总行数；not-found 提示 find_files/search_files；目录提示 list_files；空文件 [Empty file] |
| 截断 | 100KB 原始字节切,原因不明 | 50KB 上限,最后一个完整行处切割（UTF-8 安全回退）,报原因 + continue line_number |
| 性能 | countLines 多一次全量 string 拷贝 | bytes.Count 单遍计数（无拷贝）,countLines 删除 |

**测试**: 新增 TestReadFileTool_LimitParam（limit/clamp 2000/越界错误）、EmptyAndImage、TildeExpansion、RelativePathResolvesAbs；原 3 个 read 测试不动全过。`go test ./... -race` 17 包绿,已部署。

## 2026-09-21 — rg/fd 外部搜索工具集成 ✅（70fee1b，已部署）

| 改动 | 文件 | 说明 |
|------|------|------|
| search_files rg 后端 | `internal/tools/search_rg.go`（新） | `rg --json` 解析 → SearchResult，输出格式与 Go 版一致 |
| 纯 Go 回退 | `internal/tools/search.go` | Execute 拆分: rg 优先 → searchFilesGo 回退；取消不回退 |
| find_files 新工具 | `internal/tools/find_files.go`（新） | fd `--search-path/--glob/--exclude`，Go walk 回退 |
| 外部工具检测 | `internal/tools/external.go`（新） | sync.Once LookPath + 测试 override 变量 |
| Windows 控制台抑制 | `internal/tools/exec_windows.go`（新） | CREATE_NO_WINDOW，防 GUI 闪烁 |
| glob 绝对路径 gotcha | search_rg.go | 跳过目录 glob 必须无斜杠（`!dir` 而非 `!dir/**`） |
| 注册/提示词/显示 | init.go, tool_names.go, system.go, styles.go | find_files 全链路接入 |
| 系统提示词策略 | internal/prompts/system.go | Tool Usage Rules + Code Search Strategy 块：优先 rg/fd 工具而非 run 拼 grep；find_files → read → edit 探索流；500 上限收窄提示 |

**性能**: rg/fd 并行遍历 + SIMD 匹配 + .gitignore 感知，大仓库比纯 Go walk 快 10-100x。

**验证**: `go test ./... -count=1 -race` 17 包绿；`go vet` 干净；已部署 `C:/Users/22569/bin/gline.exe`。

## 2026-09-21 — 全工具零权限提示（锁死默认）✅ (24cadcc)

- 内置工具全部 `RequiresConfirmation: false`（cc616e0 + 14d00b0）；MCP/子代理工具注册时未设标志（零值 false）
- 兜底: `adkagent.New` 默认 `yolo: true`（不再读 `opts.Yolo`），即使将来某处注册设了 RequiresConfirmation=true 也不会弹提示；`SetYolo(false)` 保留可运行时重开
- `TestConfirmationAndYolo` 改为显式 `SetYolo(false)` 测审批路径
- Plan 模式门控保留（plan 下写/跑工具仍被阻止并回文本说明 —— 这是模式隔离，不是权限提示）
- 已部署 `C:\Users\22569\bin\gline.exe`

## 2026-09-21 — 全工具免确认 ✅ (14d00b0)

- `internal/tools/init.go`: write / edit / browser_copy 也改为 `RequiresConfirmation: false`（此前 cc616e0 已免 execute_command）
- 现在内置工具在 Act 模式全部直接运行，无任何审批提示；Plan 模式门控不变
- **工具名澄清**: 模型侧早已是短名 read/write/edit/run（pkg/types/tool_names.go + internal/tools/file.go + prompts/system.go 一致）；旧名 write_to_file/replace_in_file 只存在于 internal/ui/view/styles.go 的显示别名表（NormalizeToolName 兼容历史记录）
- 已部署 `C:\Users\22569\bin\gline.exe`

## 2026-09-21 — execute_command 免确认 ✅ (cc616e0)

- `internal/tools/init.go`: execute_command `RequiresConfirmation: true → false`
- Act 模式下命令直接运行，不再弹 "Approve run"；Plan 模式门控不变
- write_to_file / replace_in_file / browser_copy 仍保留确认
- 已部署 `C:\Users\22569\bin\gline.exe`

## 2026-09-21 — 并行工具审批阻塞修复 ✅ (1269749)

| 项 | 内容 |
|------|------|
| 问题 | 两条并行命令的 "Approve run" 卡死输入框（Enter 只换行） |
| 根因 | ADK `platform.RunTasks` 并发执行工具调用；TUI 单槽 `pendingReply` 被第二个问题覆盖 → 首个 asker 永久阻塞 → run 不结束 → Enter 全部失效 |
| 修复 | `PendingAsk`（once 守卫）+ TUI FIFO 队列 + bridge `AbortPendingQuestions` + adkagent Abort/teardown 解堵 + GUI 同款 bug 修复 |
| 测试 | bridge 3 项 + ui 4 项新增；17 包 `-race` 绿 |
| 部署 | `C:\Users\22569\bin\gline.exe` 已重装 |

## 项目状态概览

---

## 2026-09-21 — Agent Loop 重构 Phase 7a+7b ✅（d8f195a + f12be63）

**legacy 循环彻底删除，ADK-only 引擎达成；go-llm 依赖完全移除。**

### Phase 7a（d8f195a，+170/−2573）

| 改动 | 文件 | 说明 |
|------|------|------|
| internal/agent 瘦身 | `internal/agent/agent.go` | 只留 Mode/ModePlan/ModeAct + 包文档；删 Agent 接口/BaseAgent/手写循环全部（−1200 行）；删 agent_golden_test.go、agent_test.go；provider.go（Provider 接口+流式类型）保留 |
| runner 精简 | `internal/ui/runner.go` | 删 legacyRunner/LegacyRunner 与孤儿接口（ConversationProvider/Compactor/RulesReloader）；AdkRunner 工厂 + 编译期能力断言 |
| TUI | `internal/ui/tui.go` | /compact 恒提示自动压缩；删 history legacy 回放 |
| GUI | `internal/gui/backend.go`, `chat_service.go` | initAgent ADK-only（GLINE_AGENT=legacy 删）；buildLegacyProvider → (agent.Provider, error)；LoadTask 仅 ADK resume；ChatService 四方法 ADK-only；测试重写为 fake AgentRunner |
| CLI | `cmd/gline/chat.go` | initializeAgent ADK-only；mock 报错删；summarize_file+use_subagents 移到共享注册路径（修复 CLI ADK 路径缺工具缺口 D41） |
| mock 删除 | `internal/api/mock.go` | 无引用 |

### Phase 7b（f12be63，+244/−2021）

| 改动 | 文件 | 说明 |
|------|------|------|
| subagent 迁移 | `internal/subagent/builder.go`, `runner.go` | Builder.Provider(agent.Provider) → Builder.LLM(model.LLM)；主循环改 genai.Content 会话 + GenerateContent 流式迭代；工具声明 genai.FunctionDeclaration；结果包 FunctionResponse {"result":...}；Token 治理改粗估上限（>200k 失败不截断） |
| 装配 | `cmd/gline/chat.go`, `internal/gui/backend.go` | sub-LLM 改 provider.NewLLM(...)（opencode-go→opencode 映射不变）；GUI memory Caller 迁 model.LLM 非流式；api.OpenCodeGoBaseURL 内联 |
| **internal/api 全删** | `internal/api/*` | go_llm/openai/opencode/registry 全部删除；agent/summarizer_caller.go 删（无引用） |
| 依赖 | `go.mod`, `go.sum` | go-llm（github.com/pkieltyka/go-llm）完全移除，零残留 |

**验证**: 17 包全绿、vet 干净、CLI PONG-7B-OK、**subagent 端到端真机（use_subagents → SUBAGENT-7B-OK）**、GUI smoke、GLINE_LIVE_SMOKE 三项全过、已重装 `C:\Users\22569\bin\gline.exe`

**Agent Loop 重构至此全部完成（Phase 0-7b）。** 遗留待办：长会话压测脚本（>100 轮无 orphan）。

---

## 2026-09-21 — Agent Loop 重构 Phase 6a ✅（c4148a2）

| 改动 | 文件 | 说明 |
|------|------|------|
| Stuck recovery re-feed | `internal/adkagent/agent.go`, `stuck_recovery_test.go` | streamTurn + 每轮 turnCtx；stuck 后 recoverStuckPrompt 重喂（max 2） |
| Task bookkeeping | `internal/adkagent/agent.go` | Options.Store；首轮 CreateTask + SetTaskSessionID + OnTaskCreated；finishTask completed/failed；NewSession 重置 task |
| Transcript 双写 | `internal/adkagent/persist.go`(+test) | 最终事件→storage 消息；user prompt 直接记（ADK 不 yield 用户 event）；error 解包 |
| ResumeSession | `internal/adkagent/agent.go`, `internal/ui/tui.go` | TUI history 选中→GetTaskSessionID→ResumeSession；legacy 回退回放 |
| 生产 sessionstore | `cmd/gline/chat.go` | ~/.gline/sessions.db；失败降级 InMemory；Options.Store 注入 |
| 驱动统一 | `go.mod`, `internal/storage/database.go`, `internal/memory/*` | modernc 与 glebarez 双注册 "sqlite" panic → 全改 glebarez/go-sqlite |
| GORM 日志默认静默 | `internal/sessionstore/sessionstore.go` | Quiet→Verbose（零值默认 Discard） |

**验证**: 18 包绿；live：OneShot/ToolRun/HistoryResume（ZEBRA-7734 复述 ✅）；tasks.session_id + messages 转录落库

**剩余 Phase 6b**: GUI 切换（深耦合 legacy，推迟）、AutoCompact（/compact 仅 legacy）、facts 提取 hook、UsageMetadata 状态栏。Phase 7 删 legacy。

---

## 2026-09-21 — Agent Loop 重构 Phase 4/5/5b ✅

### 完成内容

| 组件 | 文件 | 说明 |
|------|------|------|
| Phase 4 工具桥 | `internal/adkagent/tool_adapter_test.go` | MCP/use_skill/use_subagents 自动桥接验证；4 个适配测试全绿 |
| MCP 测试服务 | `internal/mcp/testdata/echo_server/main.go` | 最小 stdio JSON-RPC server，真子进程 smoke |
| Skills 菜单 | `internal/adkagent/agent.go` | Options.Skills 渲染为系统指令 `# Skills` 段 |
| UI 接口化 | `internal/ui/runner.go` | AgentRunner 接口 + adkRunner/legacyRunner（**D26/D27**）|
| 装配切换 | `cmd/gline/chat.go` | resolveProviderSettings/mapProviderID（**D28**）/双路径装配；默认 ADK，`GLINE_AGENT=legacy` 回退 |
| CLI 单消息 | `cmd/gline/chat.go printCallback` | stdout 流式回调（内容/工具/跟进默认项）|
| stuck 检测 | `internal/adkagent/stuck.go` | pi-go stuckDetector 完整移植（streak/可变参数折叠/环检测/双错误 streak/输出重复）|
| bridge 接线 | `internal/adkagent/bridge.go` | deliver()→error；observe/observeResult/observeError/observeOutput 全接入 |

### 关键决策（新增）
- **D26**: UI 依赖窄接口 AgentRunner 而非具体 agent；新 session 走 NewSession(ctx) 而非 GetConversation().Clear()
- **D27**: store 在装配期注入 TUI（ui.Run(runner, store)），不从 agent 取
- **D28**: config provider 名 `opencode-go` 经 mapProviderID 映射为 provider 包的 `opencode`

### 验证
- ✅ go build/vet 全仓干净；18 包测试全绿（含 9 个 stuckDetector 单测）
- ✅ Live：`gline chat` 新装配 PONG + run 工具执行 PASS；GLINE_AGENT=legacy 回退 PASS

### 剩余 Phase
- Phase 6: AutoCompact、facts 提取、history resume（SessionID）、GUI ChatService 切换、stuck 恢复重喂
- Phase 7: 删 go-llm + 旧循环，重装 gline.exe

---

## 2026-09-19 — Agent Loop 重构 Phase 3 ✅（014b329）

### 完成内容

| 组件 | 文件 | 说明 |
|------|------|------|
| Agent 核心 | `internal/adkagent/agent.go` | New/RunWithCallback/SetMode/Abort；动态 InstructionProvider（Plan/Act 不重建）|
| 工具适配 | `internal/adkagent/tool.go` | adkTool（Tool+RequestProcessor+Declaration+Run）；planGate 拦截；确认门+yolo |
| 事件桥 | `internal/adkagent/bridge.go` | SSE→StreamCallback；StreamEnd 先于 ToolCallStart；model 轮文本 guard |
| 流助手 | `internal/adkagent/eventstream.go` | StreamDedup + EventError（pi-go 移植）|
| 单测 | `agent_test.go` + `eventstream_test.go` | 9 个全绿（fakeModel 走真实 runner）|
| Live smoke | `live_smoke_test.go` | opencode 一轮+工具执行 PASS；volcano plan endpoint PASS |

### 关键决策（新增）
- **D21**: stream 结束顺序 — OnStreamEnd 必须先于同轮 OnToolCallStart（TUI 文本槽先闭合再开工具面板），bridge 用两遍遍历实现
- **D22**: bridge 只对 model/thinking role 发文本；runner 会 yield 用户输入 event（role=user），不 guard 会把用户 prompt 当助手文本重复显示
- **D23**: 命名冲突 — ADK `agent` 包 vs gline `internal/agent`，后者别名 `glineagent`
- **D24**: `NewWithModel` 是测试注入点（fakeModel），生产走 `New`→`provider.NewLLM`
- **D25**: `tools.DefaultRegistry` 是空的包级变量；adkagent.Options.Tools 必须显式传

### 剩余 Phase
- Phase 4: MCP/skills 桥接 + 逐工具适配测试（→ 已完成，见 2026-09-21）
- Phase 5: TUI/CLI/GUI 切到新 agent（sessionstore 接入）（→ TUI/CLI 已完成 2026-09-21；GUI + sessionstore 在 Phase 6）
- Phase 6: AutoCompact/记忆挂接/history 续接
- Phase 7: 删 go-llm + 旧循环，重装 gline.exe

---

## 2026-09-18 ~ 2026-09-19 — TUI 优化与修复 ✅

### 完成内容

| 改动 | 文件 | 说明 |
|------|------|------|
| 恢复 legacy TUI | `internal/ui/*`, `cmd/gline/chat.go` | 从 git history 恢复 Bubbletea TUI，删除 internal/tui/ |
| go-llm Provider | `internal/api/go_llm.go` | 新增 opencode-go/volcano/openrouter |
| 默认 Provider | `~/.gline/config.yaml` | volcano → opencode-go/mimo-v2.5 |
| 系统提示词 | `internal/prompts/system.go` | 从 ~200 行简化到 ~80 行 |
| 流式修复 | `internal/ui/bridge/*`, `internal/ui/tui.go` | 事件通道 64→1024，reasoning 事件，OnStreamEnd |
| Thinking 交错 | `internal/ui/tui_state.go` | handleAgentStreamStart 检查 ReasoningContent |
| Markdown 换行 | `internal/ui/viewmodel/conversation_vm.go` | 预处理单换行→段落分隔 |
| Working 指示器 | `internal/ui/view/status_bar.go`, `internal/ui/tui.go` | 动态 spinner |
| JSON 验证 | `internal/agent/agent.go`, `internal/api/go_llm.go` | sanitizeToolCallArgs + agent 层验证 |
| 注册表顺序 | `cmd/gline/chat.go`, `internal/gui/backend.go` | 打破循环依赖 |
| read_file 简化 | `internal/tools/file.go` | 单个 line_number 参数，50 行/次 |
| 迭代上限 | `internal/agent/agent.go` | 移除 50 次上限 |
| 工具并行执行 | `internal/agent/agent.go` | executeToolCallsParallel |

### 性能优化方案（待实施）

| 优先级 | 方案 | 收益 | 说明 |
|--------|------|------|------|
| P0 | 缓存工具 token | 大 | enforceTokenBudget 每次序列化 50+ 工具 |
| P0 | 日志降级 | 中 | processStream chunk 日志 Info→Debug |
| P1 | 跳过低水位 compaction | 中 | Token < 40% 时跳过 |

---

## 2026-07-12 / 2026-09-18 — TUI 迁移（Bubbletea）✅

### 实现内容

将 gline 从 Wails v3 GUI 转为 Bubbletea TUI，保留双模式。

| 文件 | 行数 | 说明 |
|------|------|------|
| `internal/tui/app.go` | ~600 | 主模型：布局、键盘处理、slash 命令、agent 集成 |
| `internal/tui/chat.go` | ~400 | 聊天视图：消息列表、viewport、streaming、Markdown 渲染 |
| `internal/tui/input.go` | ~230 | 输入框：textinput + 历史 + slash 补全 |
| `internal/tui/sidebar.go` | 138 | 任务历史侧边栏 |
| `internal/tui/callback.go` | 134 | StreamCallback → tea.Msg 桥接（12 种消息） |
| `internal/tui/status.go` | 96 | 状态栏（provider/model/mode/tokens） |
| `internal/tui/styles.go` | 110 | lipgloss 样式 |
| `internal/tui/keys.go` | 82 | 键绑定 |
| `internal/tui/tui.go` | 33 | TUI 入口 |
| `cmd/gline/main.go` | ~70 | 双模式入口（`--gui` flag） |
| `internal/mcp/*.go` | — | `fmt.Printf` → `log.Infof/Warnf/Debugf`（MCP 日志修复） |

**总计**: ~1900 行 Go 代码

### 功能特性

- ✅ Bubbletea Elm 架构（Model-Update-View）
- ✅ 消息列表 viewport（滚动 + streaming 光标）
- ✅ 用户/助手/系统/工具 四种消息渲染
- ✅ Markdown 终端渲染（glamour）
- ✅ 工具调用折叠面板（box border + 输入摘要）
- ✅ 状态栏（provider/model/mode/tokens）
- ✅ 输入框 + 输入历史（Up/Down）
- ✅ Slash 命令补全（/ 触发，Up/Down 导航）
- ✅ 任务历史侧边栏（Ctrl+B 折叠）
- ✅ Plan/Act 模式切换（Tab）
- ✅ Followup 问题交互
- ✅ Loading spinner 动画
- ✅ Ctrl+C 优先停止任务再退出
- ✅ 双模式入口（`gline` → TUI, `gline --gui` → GUI）

### 运行时 Bug 修复（2026-09-18）

| 问题 | 根因 | 修复 |
|------|------|------|
| `panic: cannot create context from nil parent` | `mcpManager.Start(nil)` | `cmd/gline/chat.go` → `context.Background()` |
| TUI 日志污染终端 | MCP 用 `fmt.Printf` | `internal/mcp/*.go` → `log.Infof/Warnf` |
| 不渲染（显示 "Initializing..."） | `View()` 在 width=0 时提前返回 | 用默认尺寸 80x24 渲染 |
| 不能输入文字 | `inputModel` 值拷贝丢失 cursor 指针 | `input` 改为 `*inputModel` |
| `nil pointer` in cursor.BlinkCmd | `Focus()` 在 cursor 未初始化时调用 | 添加 `Focused()` 检查 |
| TUI 模式 console 日志干扰 | `InitConfig()` 设 `Console: true` | `runTUI()` 重初始化 logger，关闭 console |

### 构建验证

- ✅ `go vet ./internal/tui/...` 通过
- ✅ `go build ./cmd/gline/...` 通过
- ✅ `go build ./internal/gui/...` 通过（GUI 不受影响）
- ✅ `gline` 启动正常，显示状态栏 + 输入框

### 安装位置

`C:\Users\22569\bin\gline.exe`（已在 PATH 中）

### 入口路由

```
gline           → TUI (Bubbletea)
gline --gui     → GUI (Wails, 原有代码不变)
gline chat      → CLI (已有)
```

### 技术方案

`docs/tui-migration-plan.md`

---

## 2025-01-09 — replace_in_file 工具错误信息优化 ✅

### 问题分析

| 问题 | 根因 | 影响 |
|------|------|------|
| replace_in_file 工具经常失败 | "Nearest match" 建议误导 LLM | LLM 陷入循环错误，需要手动编写脚本 |
| 错误信息不够清晰 | 没有强调 EXACT 匹配的重要性 | 用户不知道需要完全匹配包括空白字符 |
| 缺少实用指导 | 故障排除步骤不够具体 | 用户不知道正确的操作流程 |

### 修复内容

| 文件 | 改动 | 说明 |
|------|------|------|
| `internal/tools/file.go` | 移除 "Nearest match" 建议 | 避免 LLM 使用相似但不匹配的内容 |
| `internal/tools/file.go` | 优化错误信息格式 | 更清晰强调 EXACTLY 匹配要求 |
| `internal/tools/file.go` | 改进故障排除步骤 | 明确建议先用 read_file 获取当前内容 |
| `internal/tools/file_test.go` | 更新测试用例 | 验证新的错误信息格式 |

### 关键改动

**错误信息改进:**
```
旧: "Nearest match (85% similar): ..."
新: "The search must match EXACTLY, including: ..."
    "TROUBLESHOOTING:"
    "1. Use read_file to get the CURRENT file content"
    "2. Copy-paste the EXACT text you want to replace"
    "3. Check for tabs vs spaces - they are different!"
    "4. Try searching for a smaller unique substring"
    "5. For complex edits, consider using write_to_file instead"
```

### 验证
- ✅ `go test ./internal/tools/... -v` (22 tests passed)
- ✅ `go test ./... -short` (all packages passed)
- ✅ `go build ./...` (compilation successful)

### 预期效果
- 减少 LLM 陷入循环错误的可能性
- 提供更清晰的匹配要求说明
- 给出更实用的故障排除步骤
- 显著改善用户体验

---

## 2026-06-28 — 版本检测功能开发完成 ✅

### 实现内容

| 组件 | 文件 | 说明 |
|------|------|------|
| 版本检查器 | `internal/version/checker.go` | 核心版本检查逻辑，GitHub API 集成，语义化版本比较 |
| 类型定义 | `internal/version/types.go` | UpdateCheckResult, CheckerConfig, VersionInfo 等类型 |
| 版本服务 | `internal/version/service.go` | 服务层，与配置系统集成 |
| 单元测试 | `internal/version/checker_test.go` | 9个测试用例全部通过 |
| 配置集成 | `internal/config/config.go` | 新增 UpdateConfig 配置项 |
| CI/CD 集成 | `.github/workflows/build.yml` | 构建时注入 release 版本号 |
| 前端通知 | `frontend/src/components/UpdateNotification.tsx` | 更新通知栏组件 |
| 设置页面 | `frontend/src/components/settings/UpdatesTab.tsx` | 版本检查设置标签页 |

### 功能特性
- ✅ 自动检测 GitHub 最新 release
- ✅ 语义化版本比较（v1.0.0 > v0.9.0）
- ✅ 平台特定下载链接（Windows/macOS/Linux）
- ✅ 缓存机制（默认24小时检查间隔）
- ✅ 前端通知栏（当前版本 → 最新版本）
- ✅ 可配置（启用/禁用、检查间隔、预发布版本）

### 构建验证
- ✅ `go build ./...`
- ✅ `go test ./internal/version/... -v` (9 tests passed)
- ✅ `go vet ./...`

### 工作流程
```
应用启动 → 加载配置 → 检查缓存 → 调用 GitHub API → 比较版本 → 显示通知 → 用户下载
```

**当前阶段**: 版本检测功能开发完成 ✅

**总体进度**:
- ✅ 四层记忆引擎 + 透明聊天驱动系统
- ✅ MCP (Model Context Protocol) 客户端支持（含 2026-06-26 修复）
- ✅ 大文件上下文治理（行范围读取、大文件 guard、后台 summarizer、token-aware Conversation 压缩）
- ✅ 版本检测功能（自动检查更新、GitHub API 集成、前端通知）
- ⏳ Skill 包管理器（下一优先级）

---

## 2026-06-26 / 2026-06-27 — MCP 运行时问题修复 + CI 修复 ✅

### 修复内容

| 问题 | 根因 | 修复文件 | 状态 |
|------|------|----------|------|
| MCP 状态显示 0 tools | `GetServerStatus()` 重复调用 `ListTools`，5 秒超时失败 | `internal/mcp/manager.go` | ✅ 本地验证 50 tools |
| 调用 MCP 工具 `context canceled` | Transport 复用带超时的初始化 context | `internal/mcp/transport.go` | ✅ 独立测试通过 |
| GitHub Actions Build 失败 | `ChatService` 缺少 `GetMCPStatus()`，bindings 缺导出 | `internal/gui/chat_service.go` | ✅ 待推送后验证 |

### 关键改动

1. **`internal/mcp/manager.go`**
   - 新增 `serverTools map[string][]Tool` 缓存
   - `registerServerTools()` 成功后缓存工具列表
   - `GetServerStatus()` 优先读取缓存，避免重复网络请求
   - `RefreshTools()` / `RemoveServer()` / `Close()` 同步清理缓存

2. **`internal/mcp/transport.go`**
   - `HTTPTransport.Start()` / `SSETransport.Start()` 使用 `context.Background()` 替代传入的 init ctx
   - `StdioTransport.Start()` 的 `exec.CommandContext` 同样改为 `context.Background()`
   - 所有 transport 生命周期由 `Close()` 显式取消

3. **`internal/gui/chat_service.go`**
   - 新增 `ChatService.GetMCPStatus()`，暴露给前端 bindings
   - 解决 `MCPTab.tsx` 的 `GetMCPStatus` / `MCPServerStatus` 导入缺失

### 验证
- `build-all.ps1` 完整构建成功（~32 MB）
- 本地启动后 MCP 正确显示 50 tools
- 独立 MCP client 测试：initialize → list tools → call tool 成功
- 清理了临时测试文件 `test_mcp.go` 和 `cmd/mcp-test/`

---

## 当前会话 — 大文件 Token 治理 ✅

### 实现内容

| 组件 | 文件 | 说明 |
|------|------|------|
| 文件范围读取 | `internal/tools/file.go` | `read_file` 新增 `start_line`/`end_line`，超大文件默认阻止全读 |
| 搜索结果片段化 | `internal/tools/search.go` | 返回带行号的上下文片段，限制结果数量 |
| summarize_file 工具 | `internal/tools/summarize_file.go` | Agent 可显式调用大文件摘要 |
| 分块器 | `internal/summarizer/chunk.go` | token-aware 行分块 + 重叠 + 硬截断 |
| 摘要服务 | `internal/summarizer/summarizer.go` | 并行块摘要 + 递归合并 |
| Caller 适配器 | `internal/subagent/caller.go`, `internal/agent/summarizer_caller.go` | 子 Agent / provider caller，避免 import cycle |
| Token 感知的 Conversation | `pkg/types/message.go` | 单条消息 soft cap、token-aware AutoCompact、TrimToMaxTokens 占位摘要 |
| 请求前预算检查 | `internal/agent/agent.go` | `enforceTokenBudget()` 估算 system + messages + tools |
| 配置生效 | `cmd/gline/chat.go`, `internal/gui/backend.go`, `internal/config/config.go` | `max_context_tokens` 正确传给 Agent；默认预算 128K |

### 构建验证
- ✅ `go build ./...`
- ✅ `go test ./...`
- ✅ `wails3 build`

### 待完成
- 手动端到端测试（>100KB 文件读取、搜索后范围读取）。
- 补充 `internal/summarizer` 单元测试。
- 提交并推送。

---

## 2026-06-24 — MCP 支持开发 ✅

### 实现内容

| 组件 | 文件 | 说明 |
|------|------|------|
| Protocol | `internal/mcp/protocol.go` | JSON-RPC 2.0 + MCP 协议类型 |
| Transport | `internal/mcp/transport.go` | StdioTransport + HTTPTransport (同步 POST) |
| Client | `internal/mcp/client.go` | MCP 客户端（初始化、工具调用、资源读取）|
| Config | `internal/mcp/config.go` | ServerConfig, Config 结构 |
| Manager | `internal/mcp/manager.go` | Server 管理器 + Tool Adapter |
| Config Integration | `internal/config/config.go` | MCP 配置字段 |
| CLI Integration | `cmd/gline/chat.go` | Agent 初始化时启动 MCP Manager |
| GUI Integration | `internal/gui/backend.go` | GUI 模式 MCP Manager 集成 |
| Frontend | `frontend/src/components/settings/MCPTab.tsx` | MCP Server 配置 UI + 工具列表显示 |

### 功能特性
- ✅ 支持 stdio、HTTP 和 SSE 三种传输方式
- ✅ HTTP Transport 使用简单同步 POST 请求-响应（参考 LtEdu 实现）
- ✅ 动态工具注册（自动适配为 gline Tool 接口）
- ✅ 环境变量扩展（`${VAR}` 语法）
- ✅ 前端可视化配置（添加/编辑/删除/启用禁用）
- ✅ 配置热更新（修改配置后自动重启 MCP Manager）
- ✅ 工具列表显示（每个服务器显示可用工具数量和名称）
- ✅ 连接状态指示器（绿色=已连接，黄色=连接中，红色=错误）

### 构建验证
- ✅ `build-all.ps1` 完整构建成功（32.2 MB）
- ✅ Wails bindings 自动生成
- ✅ TypeScript 类型检查通过
- ✅ Go 编译成功

---

## 2025-01-09 — MCP HTTP Transport 修复 + 工具列表显示 ✅

### 问题修复
1. **HTTP Transport 简化** - 参考 LtEdu 项目实现
   - 移除了复杂的 SSE 流处理代码
   - 改为简单同步 HTTP POST 请求-响应模式
   - Send 方法发送请求并存储响应
   - Receive 方法立即返回存储的响应

2. **前端工具列表显示**
   - 添加 `GetMCPStatus` 后端方法暴露 MCP 服务器状态
   - 前端显示每个服务器的连接状态和工具数量
   - 添加 "Show Tools" 按钮展开显示工具名称列表
   - 状态指示器：绿色=已连接，黄色=连接中，红色=错误

### 修改文件
- `internal/mcp/transport.go` - 简化 HTTPTransport
- `internal/mcp/manager.go` - 添加 ToolNames 字段
- `internal/gui/backend.go` - 添加 GetMCPStatus 方法
- `frontend/src/components/settings/MCPTab.tsx` - 添加工具列表 UI
- `frontend/bindings/...` - 手动添加 GetMCPStatus 绑定

---

## 2026-06-05 — 透明记忆系统（Phase 9+）✅

---

## 2026-06-05 — 透明记忆系统（Phase 9+）✅

### 设计决策
- **无独立 GUI 面板**：所有记忆操作通过聊天界面透明完成（自然语言触发或 slash 命令）
- **统一大模型**：被动提取使用主 provider，无需本地小模型
- **记忆来源标记**：前端提取 📌 Fact / 📚 Wiki / 📄 KB 徽章

### 新实现

| 组件 | 文件 | 说明 |
|------|------|------|
| Memory Tools | `internal/tools/memory.go` | memory_recall / memory_note / kb_search |
| Engine Config | `internal/memory/engine_config.go` | `NewEngineFromConfig()` 统一工厂 |
| System Prompt | `internal/prompts/system.go` | LLM 能力说明 + 工具描述 |
| Interval Extract | `internal/agent/agent.go` | `maybeExtractFacts()` 每 4 轮触发 |
| Slash Commands | `internal/slash/commands.go` | `/mem note`, `/mem recall`, `/mem status` |
| Slash Service | `gui/chat_service.go` | `memoryService` 桥接到后端引擎 |
| Engine Injection | `gui/backend.go` | Agent 初始化时自动注入 memory engine |
| Memory Badges | `gui/frontend/src/components/MemoryBadges.tsx` | 📌📚📄 来源标识组件 + 内容提取器 |
| AssistantMessage | `gui/frontend/src/components/AssistantMessage.tsx` | 集成徽章渲染 |

### 验证
- `go build ./...` ✅
- `go test ./internal/{agent,memory,tools,slash}/...` ✅
- `npm run build` (frontend) ✅

---

## 2025-06-05 — 四层统一记忆与知识引擎（Phases 1-8）✅

---

## 2025-06-05 — 四层统一记忆与知识引擎（Phases 1-6）✅

### 设计
融合三种前沿方案：mem0 (Fact 层)、Karpathy Wiki (Wiki 层)、RAG 检索 → 统一四层架构。

| 层 | 类比 | 擅长场景 | 已建状态 |
|--|------|----------|----------|
| **Fact** | 人类长期记忆 | 用户偏好、技术选型、bug 模式、项目决策 | SQLite + FTS5 + entities + Decay ✅ |
| **Wiki** | 知识笔记 | 深度理解技术方案、跨文档矛盾追踪 | Markdown FS + index.md/schema.md/log.md ✅ |
| **RAG** | 快速查手册 | 代码精确引用、API 文档、配置查找 | 纯 Go KNN + FTS5 + RRF ✅ |
| **Conversation** | 短期工作记忆 | 保持当前任务连贯、多轮工具调用 | 已有 SQLite 存储 ✅ |

### 技术突破
- **纯 Go SQLite 向量存储**：`modernc.org/sqlite` 无法加载 C 扩展 → embedding 用 `gob` 存 BLOB，Go 内存计算归一化点积 = 余弦相似度。
- **混合检索**：Go 内存 KNN 相似度 + SQLite FTS5 + RRF (Reciprocal Rank Fusion) 融合两层结果。
- **性能分层**：同步只读（<150ms）检索 Fact + RAG + Wiki；异步写入（后台 goroutine）Fact 提取 + Embedding + Wiki Ingest。

### 新文件（18 个源码文件 + 1 测试）
完整列表见 `activeContext.md`。

### Agent 集成
- `buildMemoryContext()` 注入记忆上下文到 system prompt（Token 硬限制 2000）
- 对话完成后异步 `extractFactsAsync()` 后台提取事实

### 验证
- `go test ./internal/memory/...` ✅ (0.756s)
- `go build ./cmd/gline/...` ✅
- 修复：Search() `!rows.Next()` 消耗首行 bug；upsertFact nil tx 崩溃

---

## 2026-06-05 之前 — 主题系统组件集成扩展 🔄 (未提交)

### P2.5.3 主题系统组件集成
- 28 个新 CSS 变量
- highlight.js 样式表动态切换
- FOUC prevention
- 全组件硬编码颜色迁移到 `THEME.*`
- `format.ts` 重构

## 2026-06-04 — 主题系统组件集成扩展 🔄 (未提交)

### 背景
P2.5.2 建立了 CSS 变量主题骨架，但大量组件仍使用硬编码颜色值。本次扩展让主题系统真正可投入使用。

### 变更内容
1. **新增 28 个 CSS 变量** (`theme.ts`)
   - 控件背景: `inputBg`, `cardBg`, `overlayBg`
   - 反馈色: `toastSuccess`, `toastError`, `toastSuccessBg`, `toastErrorBg`
   - 代码展示: `codeInlineBg`, `codeInlineText`, `highlightJsTheme`
   - 状态指示: `spinner`, `statusSuccessBg`, `statusPendingBg`
   - 排版辅助: `linkColor`, `tableHeadBg`, `tableBorder`, `footnoteText`, `footnoteBorder`
   - 品牌: `logoGradientStart`, `logoGradientEnd`

2. **highlight.js 主题切换**
   - 新增 `public/styles/hljs-github-dark.css` + `hljs-github-light.css`
   - `applyThemeColors()` 动态切换 `<link id="hljs-theme">` 的 `href`
   - `index.html` 内嵌默认 dark 样式表链接

3. **FOUC Prevention**
   - `index.html` 内嵌 `<script>` 在 React 挂载前读取 `localStorage.getItem('gline-theme')`
   - 若 stored === 'light'，同步写入所有 light 模式的 CSS 变量到 `:root`
   - 避免页面加载时出现"白闪"或"暗闪"

4. **全组件硬编码颜色迁移**
   - `Header.tsx`: Stop 按钮红色从 `#ef4444` 改为 `THEME.toastError`
   - `Sidebar.tsx`: 边框、hover、激活态全部改用 CSS 变量
   - `MessageList.tsx`, `InputArea.tsx`, `SettingsPanel.tsx`, `ToolMessage.tsx`, `SystemMessage.tsx`, `FollowupModal.tsx` 等

5. **`format.ts` 重构**
   - 大幅重构格式化工具函数（与主题无关的紧邻改进）

### 验证
- `go build ./...` ✅ (Go 后端无变更)
- `cd gui/frontend && npm run build` ✅ (前端构建通过)

---

## 2026-06-04 — 性能阻塞点修复 ✅

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

### 2026-06-04 — @ 文件引用功能完成 ✅
**后端** (`gui/file_service.go`):
- `ListDirEntries(dirPath)` — 列出项目目录下的文件/子目录
- `ReadFileContent(relPath)` — 读取文件内容，1MB限制 + 二进制检测
- `SendMessageWithContext(prompt, fileRefsJSON)` — 拼接 `<referenced_files>` 上下文

**前端**:
- `useFileReference.ts` / `FilePicker.tsx` / `InputArea.tsx` / `useChat.ts`

### 2026-06-04 — @ 文件引用功能完善
- 方向键滚动、Filter 输入框、选择后自动关闭、onBlur 误关闭修复、文件标签路径优化

### 2026-06-04 — GUI 前端模块化拆分 & 项目目录重构 ✅
- 18+ 独立模块拆分
- `workingDir` 独立字段替代 `os.Getwd()`

### 2026-06-04 — search_files 工具优化 + 单元测试 ✅
- 并发 Worker Pool、字面量快速路径、目录跳过、二进制文件过滤

### 2026-06-04 — `/clear` 保留 workingDir 修复 ✅
- `ClearConversation()` 保留 workingDir，`/clear` 改调；New Chat/`/newtask` 仍然清空

## 2026-06-07 — 构建系统重构 ✅

**背景**: CLI 与 Wails GUI 共用 `cmd/gline/main.go` 入口，但构建脚本分散混乱，无法一键编译成功。

**变更**:
- 删除废弃的 `desktop/` 目录（旧独立 Wails 项目残留）
- 前端源码移至根目录 `frontend/`
- Wails 构建资产（`build-desktop/`）已删除，统一使用 `build-all.sh`/`build-all.ps1` + `wails3` 命令构建
- 修复 `build-all.ps1`：bindings 生成改为在 `cmd/gline` 目录执行（否则报 "0 Services"），输出到 `../../frontend/bindings`
- 新增 `build-all.sh`：macOS/Linux 的 bash 构建脚本（与 PowerShell 脚本对等）
- 修复 `Makefile`：`FRONTEND_DIR := frontend`，bindings 目标改为根目录运行
- 更新 CI：新增 bindings 生成步骤；macOS 使用默认 CGO（WebKit 需要）；拆分为 `npm build` + `go build ./cmd/gline`
- 清理根目录 4 个测试 EXE

**验证**:
- `build-all.ps1` 一键编译成功 → `bin/gline.exe` ✅
- `desktop/` 目录已不存在 ✅

---

### 2026-06-04 — Phase 1 快速赢完成 ✅
- 规则管理 UI、reload 联动、@ 提示移除、主题占位

---

## 已知问题

### 已修复问题 ✅

| # | 问题 | 修复文件 | 说明 |
|---|------|----------|------|
| 19 | **KB 自动触发 wiki 失败** | `internal/memory/engine.go` | `IngestFile()` 中 `kb.Type == KBTypeWiki || hybrid` 条件触发 wiki，但 `e.Caller` 在 CLI 下为 nil，导致 wiki 静默跳过。解耦后 `WikiIngestFile()` 显式要求 Caller，失败立即返回 error。 |
| 20 | **kb_ingest 重复执行 / 并行导致数据库锁** | `internal/agent/agent.go` + `internal/memory/engine.go` | `preDispatchToolCall` 在流式期间后台并行执行 `kb_ingest`，多个 goroutine 同时写同一 `rag.db` → `database is locked`。→ ① Agent 层扩大 side-effect 黑名单（增加 `memory_note`），禁止背景预分发；② Engine 层在 `IngestFile` 入口加 `ingestMu sync.Mutex`，串行化写入。 |
| 21 | **RAG 重复文档** | `internal/memory/store.go` + `engine.go` | `IngestFile` 重新加入同一文件时不断增。→ `findDocByName` + `DeleteDocument` 在插入前删除旧记录。 |
| 22 | **list_files 递归耗 token** | `internal/tools/file.go` | list_files 递归返回所有子目录文件，token 爆炸。→ 移除递归，只返回当前目录列表，添加 `recursive` 参数可选。 |
| 23 | **PDF 导入提取二进制乱码** | `internal/memory/parser.go` + `go.mod` | `github.com/ledongthuc/pdf` 的 `GetPlainText()` 对嵌入字体、CJK、非标准编码支持不足，提取返回二进制乱码。→ 替换为 `github.com/tsawler/tabula`（MIT/纯Go），支持 CJK/排除页眉页脚。新增 `.odt` / `.epub` 支持。 |

### 架构演进（重大变更）
**TUI → GUI 迁移** — Bubbletea TUI 已废弃，全面迁移到 Wails v3 GUI。旧 TUI MVVM 架构作为历史参考仍保留在 `memory-bank/archive/` 中。

### 已完成 Phase 7: Fact Extractor LLM 集成（2026-06-XX）
**背景**: Fact 层仅剩 rule-based stub，无法真正理解语义。用 LLM 驱动提取，实现"越用越懂用户"。
**完成内容**:
- `fact_extractor.go` 生产级 prompt + parse + EnrichFacts
- `fact_store_sqlite.go` smart-merge Apply()：ADD 自动转为 UPDATE on duplicate key
- `agent.go` extractFactsAsync() 来源标注 + 成功日志
- 新增 10 个单元测试
**验证**: `go test ./...` 全通过 ✅

### 已修复问题 ✅（完整列表）
1. **Agent 流式回调架构** ✅
2. **工具调用实时通知** ✅
3. **工具调用参数重复累积** ✅
4. **工具执行流程修复** ✅
5. **TUI 流式输出优化** ✅
6. **SSE 每 Chunk 同步写磁盘** ✅ (2026-06-04)
7. **Info 级高频日志** ✅ (2026-06-04)
8. **Token 估算严重低估** ✅ (2026-06-04)
9. **actual token 累加虚高** ✅ (2026-06-04)
10. **P1.1-P1.4** ✅
11. **P2.3 废弃 TUI** ✅
12. **P2.5 错误边界** ✅
13. **@ 文件引用** ✅
14. **`/clear` 保留 workingDir** ✅
15. **P2.2 系统托盘集成** ✅ — 左键切换窗口显示/隐藏，右键菜单含 Show/Hide 动态标签 + Quit
16. **P2.4 构建产物优化** ✅ — Taskfile `dev`/`build` 统一 + CI wails3 build 集成
17. **Agent 构建失败** ✅ — `SubmitHandler` 中直接实例化 `*ai.Agent` 导致 `Agent` 接口不匹配。修复：恢复 `NewRuntimeAgent(auth)` 调用。
18. **ask_followup_question 终止对话** ✅ — agent 在收到用户回答后调用 `SetComplete()` 停止运行。
    修复：从 `SetComplete` switch 中移除 `ToolAskFollowupQuestion`；同时引入 pre-dispatch 优化（流式期间预执行工具调用）。

## 已完成工作（2026-06-04）

### GitHub Actions CI 重构 ✅
**背景**: 之前的工作流只编译 Go 后端，完全不构建前端（React/TypeScript），导致产物中前端资源为空。
**修复文件**: `.github/workflows/build.yml`
**修复内容**:
- **build.yml**: 新建 `test` job（ubuntu-24.04），新增完整 `build` 矩阵（darwin-arm64, windows-amd64）
  - 安装 wails3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.95`)
  - Linux 构建安装 `libgtk-4-dev` + `libwebkitgtk-6.0-dev`
  - wails3 在 `gui/` 目录下构建，上传裸二进制 artifact
  - 新增 `build-summary` 汇总 artifact 到 GitHub Step Summary
- **release** (tag 触发): 上传 Release Artifacts 裸二进制（非 zip），覆盖 macOS + Windows
- **snapshot** (main/master push): 自动创建/更新 `snapshot` tag 预发布版本

### 环境说明
`go.mod` 和 `gui/go.mod` 保持 `go 1.25.0`（无法降级到 1.22，因为 `modernc.org/sqlite@v1.50.1`、`modernc.org/libc@v1.72.3`、`golang.org/x/sys@v0.42.0` 等核心依赖要求 `go >= 1.25.0`）。

## 已完成工作（2026-06-04）

### P2.4 构建产物优化 ✅
**范围**: Taskfile `dev` / `build` 目标统一 + CI/CD 产物路径修正  
**对应 commits**: 系列 CI commits（`ci: refactor GitHub Actions...` 起）  
**内容**:
- `gui/Taskfile.yml`: `build` / `dev` / `run` / `package` 目标按 OS 分发到 `build/{OS}/Taskfile.yml`
- `gui/build/Taskfile.yml`（common）: 统一 `build:frontend`、`generate:bindings`、`build:server`、`build:docker` 等共享目标
- GitHub Actions 统一使用 `wails3 build`，自动集成前端构建 + 产物路径修正
- 产物清理：移除不兼容 `-ldflags`、精简 Linux 依赖、矩阵调整为 Windows + macOS

---

## 已完成工作（2026-06-04 — Phase 2.5 技术债务）

### P2.5.1 前端测试骨架建立 ✅
**工具**: Vitest + @testing-library/react + @testing-library/jest-dom + jsdom
**文件**:
- `vite.config.ts`: `test` 区块配置（globals, jsdom environment, setup files）
- `vitest.setup.ts`: 引入 jest-dom matchers
- `package.json`: scripts `test` (vitest run) + `test:watch` (vitest)
- `src/utils/format.test.ts`: 10 个用例覆盖 `parseToolInput`, `getToolHint`, `formatContent`

### P2.5.2 主题系统初版完成 ✅
**方案**: CSS 变量法 — `THEME` 常量值改为 `var(--theme-*)` 引用，ThemeContext 更新 `:root` 变量。
**优点**: 零组件文件修改，所有现有 `style={{...THEME}}` 自动响应主题切换。
**文件**:
- `theme.ts`: 拆分为 `THEME_DARK` / `THEME_LIGHT`，`THEME` 改为 CSS 变量引用，新增 `applyThemeColors()`
- `ThemeContext.tsx`: React Context + `useTheme()` hook，`localStorage` 持久化
- `main.tsx`: 用 `<ThemeProvider>` 包裹 `<App/>`
- `index.html`: 内嵌 `<style>` 定义默认 dark CSS 变量（防止 FOUC）
- `SettingsPanel.tsx`: Theme select 启用，绑定 `setTheme()`，即时生效

### P2.5.3 主题系统组件全面集成 🔄 (未提交)
**说明**: 在 P2.5.2 基础上扩展 28 个新 CSS 变量，并将所有组件的硬编码颜色迁移到 `THEME.*` 引用。highlight.js 样式表动态切换，index.html 添加 FOUC prevention script。

---

---

## 2026-06-08 — Skill 系统迁移到 cline agent 规范 ✅

### 变更
| 组件 | 变更 |
|------|------|
| Skill 格式 | 扁平 `.yaml` → `skill-name/SKILL.md`（YAML frontmatter + markdown body）|
| Loader | 扫描子目录，解析 frontmatter，验证 name 匹配目录名 |
| Activation | 预注入 → `use_skill` 工具按需加载（工具结果消息返回完整指令）|
| Registry | 移除 active 状态，新增 `GetMeta()` / `GetInstructions()` |
| Agent | `activeSkill` → `skills []SkillMeta` + `SetSkills()` |
| Search dirs | `~/.gline/skills/`, `~/.agents/skills/`, `~/.cline/skills/`, `~/.claude/skills/` |

### 验证
- `go build ./...` ✅
- `go test ./...` ✅
- `build-all.ps1` 一键编译 ✅

## 已完成工作（2026-06-XX）

### GUI 历史会话工具结果渲染修复 ✅
**问题**: 加载历史会话时，`attempt_completion` 和 `plan_mode_respond` 的工具调用结果不渲染，用户看不到任务完成总结。
**根因**: 实时流中这些工具的结果被前端额外插入一条 `assistant` 消息展示，但这些辅助消息不会被保存到数据库。历史加载时只有 `tool` 角色消息，而原 `ToolMessage` 只显示缩略标签气泡。
**修复文件**: `frontend/src/components/ToolMessage.tsx`
- 新增 `attempt_completion` 分支 → "✅ Task Completed" 标题 + `formatContent(toolResult)` Markdown 渲染
- 新增 `plan_mode_respond` 分支 → "📝 Plan Response" 标题 + `formatContent(toolResult)` Markdown 渲染
- 两者均带 `maxHeight: 500px` + `overflow: auto` 滚动条限制

### ask_followup_question 弹窗渲染优化 ✅
**问题**: 弹窗和聊天流中的追问内容不做 Markdown 渲染，也不加滚动条。内容多时页面看不全。
**修复文件**:
- `frontend/src/components/FollowupModal.tsx` → `className="md-rendered"` + `formatContent(question)` + `maxHeight: 40vh` + `overflow: auto`
- `frontend/src/components/ToolMessage.tsx` → `ask_followup_question` 气泡同上处理
**验证**: `tsc --noEmit` 0 错误；`build-all.ps1` 完整构建通过 ✅

### Memory Tab UI 完成 ✅
**文件**:
- `frontend/src/components/SettingsPanel.tsx` → 拆分为 `settings/` 子组件（ProviderTab, MemoryTab, GeneralTab, RulesTab + sharedStyles.ts）
- `frontend/src/components/settings/MemoryTab.tsx` → 新增（Embedding provider/model/API key/Base URL + Retrieval TopK/MinScore/MaxTokens + Enabled 开关）
- `internal/config/config.go` → 新增 `MemoryConfig.Enabled bool`，默认值 `true`
- `internal/gui/backend.go` → 初始化引擎时检查 `Enabled`，配置变更热重载包含 memory 配置项

---

## 建议下一步

### 高优先级 — 四层记忆引擎 LLM 驱动层
1. **Phase 7: Fact Extractor LLM 集成** — 对话结束后用 LLM 提取 ADD/DECAY 事实，取代 rule-based stub
2. **Phase 8: Wiki Ingest LLM 集成** — `kb add` 后用 LLM 读取 raw 文件，自动生成/更新 wiki 页面

### 中优先级 — 基础设施优化
3. **连接池优化** — RAGManager/VectorStore 复用 SQLite 连接
4. **PDF/DOCX 解析** — `pdfcpu`（纯 Go）用于 PDF 解析

## 2026-06-XX — replace_in_file 5 层容错优化 ✅

**问题**: `replace_in_file` 因空白符差异频繁失败，错误信息仅 "search content not found"，LLM 无法 self-correct。

**修复**:
- `internal/tools/file.go` — 重写 Execute 方法，支持 `replacements` 多块数组、Jaccard 最近匹配反馈、空格归一化回退、行锚定回退、diff 输出。
- `internal/prompts/system.go` — 系统提示更新：EDTIING FILES 节增加多 block 用法说明；replace_in_file schema 增加 `replacements` 数组。
- `internal/tools/file_test.go` — 7 个新测试覆盖全部优化路径。

**验证**: `go test ./...` ✅, `build-all.ps1` ✅

---

### 低优先级
5. **Embedding int8 量化** — BLOB 存储 4× 压缩
6. **ContextBuilder intent routing** — 按问题类型自动路由各层
7. **对话 message 扩展** — storage 包添加 FactsExtracted/WikiPagesTouched 列

### 长期 (Phase 3)
8. **MCP 支持** — 引入 Model Context Protocol，接入外部工具源
9. **LiteLLM 多提供商统一** — `litellmcreds` 规范与引导流程
