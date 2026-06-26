# gline 产品发展路线图

**制定时间**: 2026-06-XX
**状态**: 动态更新中

---

## 一、产品定位

**gline 的核心定位**:

> 跨平台、轻量级、可脚本化的 AI 编程助手，脱离 IDE 独立运行，提供 GUI（日常使用）+ CLI（脚本自动化）双模体验。

**四层差异化优势**:
1. **零 IDE 依赖** — 独立桌面应用（Wails v3），可在任何编辑器环境下工作
2. **四层记忆引擎** — Fact + Wiki + RAG + Conversation（Cline 等竞品不具备的深层差异化）
3. **可脚本化** — CLI 子命令支持管道、CI/CD 集成
4. **跨平台** — Windows、macOS、Linux 统一体验

---

## 二、三层路线图

### 🔴 第一优先级：补齐核心体验短板（近期 4-8 周）

| # | 方向 | 具体内容 | 状态 | 原因 |
|---|------|----------|------|------|
| 1 | **MCP 支持** | 引入 Anthropic Model Context Protocol，实现 MCP 客户端，让 gline 接入外部工具源（Slack/GitHub/数据库/浏览器） | ✅ **已完成** | 将 gline 从"代码助手"升级为"通用 AI 工作流中枢" |
| 2 | **Skill 包管理器** | 已兼容 cline 规范，下一步 `gline skill search/install/use`，内置热门 skill（doc-coauthoring、brainstorming 等） | 🔄 **当前进行** | 让用户通过自然语言调用行业专业模式 |
| 3 | **四层记忆引擎** | Phase 7（Fact Extractor LLM）✅ 已完成；Phase 8（Wiki Ingest LLM）搁置等待 Scale 设计 | ⏸️ **搁置** | 四层记忆是 gline 的核心护城河 |
| 4 | **可靠性加固** | 持续监控 `memory_note` / `execute_command` 等 side-effect 工具避免 db 锁死重演 | ✅ **已完成** | 高可靠性维护 |

**状态更新（2026-06-24）**:
- ✅ **MCP 支持**: 已完成（协议实现、传输层、Manager、前端配置面板、完整构建验证）
- ✅ **主题系统 P2.5.3**: 已完成并提交（CSS 变量、hljs 动态切换、FOUC 消除）
- 🔄 **Skill 包管理器**: 当前最高优先级（`gline skill search/install/use`）
- ⏸️ **Wiki Ingest LLM**: 搁置，等待 Scale 设计方案
- 📋 **KB/Wiki 前端面板**: 待规划 |

### 🟠 第二优先级：扩展差异化能力（中期 2-4 个月）

| # | 方向 | 具体内容 | 产品价值 |
|---|------|----------|----------|
| 1 | **@ 引用增强** | 文件夹、最近编辑文件自动提示、代码符号级引用 | 编辑效率对标 Cursor/Cline 的上下文引用能力 |
| 2 | **多提供商统一（LiteLLM）** | 支持 `litellmcreds` 规范，降低用户配置多 LLM 的门槛 | 强化"多 LLM 提供商支持"这一目标 |
| 3 | **@ 引用增强** | 文件夹、最近编辑文件自动提示、代码符号级引用 | 编辑效率对标 Cursor/Cline 的上下文引用能力 |
| 4 | **多提供商统一（LiteLLM）** | 支持 `litellmcreds` 规范，降低用户配置多 LLM 的门槛 | 强化"多 LLM 提供商支持"这一目标 |
| 5 | **连接池 & Embedding int8 量化** | RAGManager/VectorStore 复用 SQLite 连接、int8 量化压缩存储 | 支撑万级文档知识库 |

### 🟡 第三优先级：构建生态壁垒（长期 4-12 个月）

| # | 方向 | 具体内容 | 战略目标 |
|---|------|----------|----------|
| 1 | **Agent 自主工作流** | 支持"后台任务"模式：用户下达任务后，Agent 在系统托盘后台持续执行，完成后弹窗通知 | 利用"轻量独立"优势 |
| 2 | **Workspace 级项目知识** | 将四层记忆与 `workingDir` 深度绑定，自动为新项目创建专属 KB，跨项目隔离记忆 | 企业级和重度用户的核心粘性来源 |
| 3 | **多模态能力** | 接入图像理解（UI 设计图转代码、截图 debug），支持图表/架构图解析 | 打破纯文本局限 |
| 4 | **团队协作层** | 导出/导入对话 + 规则配置、共享 KB 与 Wiki、团队级 prompt 模板 | 从个人工具向团队生产力工具演进 |
| 5 | **Web/Server 模式** | Wails 产物分离出无 Go 后端的纯前端？或提供 Web 服务端模式 `gline server` | 探索远程/网页版可能性 |

---

## 三、Phase 7: Fact Extractor LLM 集成（详细计划）

### 3.1 什么是 LLM Fact Extractor

Fact Extractor 是四层记忆引擎中 **Fact 层（事实层）** 的核心驱动组件。职责：

> **在每次对话结束后，调用 LLM 分析整轮对话内容，自动提取出值得长期记忆的"原子事实"，存入数据库并在后续对话中作为上下文注入。**

当前状态：`internal/memory/fact_extractor.go` 为 **rule-based stub（基于规则的占位实现）** — 只能识别硬编码模式。Phase 7 将其替换成 **LLM 驱动**。

### 3.2 技术流程

```
用户对话结束
    ↓
[异步触发] Agent 调用 Extractor
    ↓
LLM 接收：整轮对话历史 + 提取提示词（prompt）
    ↓
LLM 输出结构化事实列表（JSON）
    [
      {"op": "ADD", "fact": "用户偏好双引号", "category": "coding_style", "confidence": 0.95},
      {"op": "UPDATE", "id": "fact_123", "fact": "项目使用 PostgreSQL 15", "category": "tech_stack"},
      {"op": "DECAY", "id": "fact_456", "reason": "用户已改用MySQL"}
    ]
    ↓
存入 SQLite（facts 表 + FTS5 索引 + entities 关联表）
    ↓
下次对话 → buildMemoryContext() 将相关 facts 注入 system prompt
    ↓
Agent "记得"用户偏好/项目决策
```

### 3.3 支持的三种操作

| 操作 | 含义 | 场景 |
|------|------|------|
| **ADD** | 发现新事实，插入数据库 | 用户首次表达偏好 |
| **UPDATE** | 事实有变更，更新旧记录 | 技术栈升级、规范变更 |
| **DECAY** | 事实已过时/被推翻，降低权重或删除 | 方案废弃、错误假设纠正 |

### 3.4 使用场景例

#### 场景 1：编码风格偏好
- **对话**: "把代码里的单引号都改成双引号，我项目规范要求统一用双引号"
- **提取**: `{"op": "ADD", "fact": "用户项目编码规范要求字符串使用双引号", "category": "coding_style"}`
- **效果**: 三天后新任务，Agent 自动使用双引号写代码

#### 场景 2：技术栈决策记忆
- **对话**: "我们决定后端数据库从 MySQL 迁到 PostgreSQL 15，因为需要 JSONB 支持"
- **提取**: `{"op": "ADD", "fact": "项目数据库技术栈为 PostgreSQL 15，选用原因是支持 JSONB", "category": "tech_stack"}`
- **效果**: 后续问及数据存储方案时，Agent 自动推荐 JSONB

#### 场景 3：Bug 模式与教训
- **对话**: "为什么这个 ctx 又 panic 了？" → "啊对，我又忘了，这已经是第三次了"
- **提取**: `{"op": "ADD", "fact": "用户频繁忘记在分支中初始化 ctx 导致 panic，建议函数开头统一检查", "category": "bug_pattern"}`
- **效果**: 后续 Review 代码时主动提醒

#### 场景 4：语言偏好
- **对话**: "请用中文回复我，我的英文不太好"
- **提取**: `{"op": "ADD", "fact": "用户偏好使用中文进行技术交流", "category": "communication"}`
- **效果**: 新任务自动中文回复，无需再次提醒

#### 场景 5：衰减与纠错
- **对话**: "我们刚把部署迁移到 Kubernetes 了，之前的 Docker Compose 方案废弃了"
- **提取**: ADD "Kubernetes" + DECAY "Docker Compose" 旧事实
- **效果**: 不再基于过时假设给建议

### 3.5 与周边层的关系

| 层 | 作用 | 类比 | 何时调用 |
|--|------|------|----------|
| **Fact** | 提取原子事实（偏好、决策、习惯） | 人类"长期记忆" | 每次对话结束后异步提取 |
| **Wiki** | 维护结构化的 Markdown 知识笔记 | 个人技术博客/笔记 | 用户显式调用或自动整理 |
| **RAG** | 文档块级的精确检索（代码/API） | 快速查手册 | 用户提问涉及文档内容时 |
| **Conversation** | 当前对话的短期上下文 | 工作记忆 | 实时 |

### 3.6 实施结果（已完成）

**已交付**:
- ✅ `fact_extractor.go` — 生产级 ExtractPrompt + ParseFactChanges + EnrichFacts
- ✅ `fact_store_sqlite.go` — Apply() smart-merge: ADD→UPDATE on duplicate, DELETE fallback by (subject, predicate)
- ✅ `agent.go` — extractFactsAsync() 集成 Source 标注 + EnrichFacts + 成功日志
- ✅ `fact_extractor_test.go` — 10 个单元测试，覆盖解析/LLM/回退/smart-merge

**验证**: `go test ./...` 全项目 0 失败 ✅ | `build-all.ps1` 完整构建 ✅

### 3.7 关键设计决策

1. **复用主 LLM Provider，不引入本地小模型**
   - 理由：Fact 提取发生在对话结束后异步执行，对延迟不敏感
   - 好处：无需用户额外配置 embedding 之外的模型
   
2. **Token 硬限制 2000**
   - 提取 prompt 中只放对话 summary（而非完整消息内容），用 conversation summary 减少 token
   - 超过 2000 时按时间倒序截断，优先保留最近轮次

3. **置信度阈值**
   - LLM 输出 confidence < 0.7 的事实暂不入库，作为 candidate 保存供人工确认
   - confidence >= 0.7 自动入库

---

## 四、CLI 移除评估（产品决策方向，暂不执行）

### 4.1 决策记录

**用户意向**: 考虑删除全部 CLI 功能，只保留 Wails 3 GUI 标准功能。
**当前状态**: 仅记录为想法，**暂不执行**。

### 4.2 影响分析

#### 正向影响
| 收益 | 说明 |
|------|------|
| 代码精简 | 删除 Cobra + CLI 命令文件 + 路由逻辑，代码量减少 20-30% |
| 维护聚焦 | 不再维护两套入口兼容逻辑，构建脚本简化 |
| 零 Cobra 依赖 | 可移除 `github.com/spf13/cobra`、`viper`、`pflag` 等重依赖 |
| 专注桌面体验 | 全部精力放 Wails GUI 打磨 |

#### 负面/放弃的定位
| 原定位优势 | 移除后的损失 |
|-----------|------------|
| **可脚本化** | 完全丧失，gline 变成纯手动工具 |
| **CI/CD 集成** | `gline chat "review code"` 不可行 |
| **资源占用低** | 任何场景都必须拉起 WebView2 |
| **远程服务器友好** | 无图形环境的服务器无法使用 |

### 4.3 替代建议（若未来执行）

**方案 A（推荐）**: 保留最小 `gline chat` 子命令，删除其他所有子命令（`history`、`kb`、`wiki`、`mem`）。
- 保留核心 CI/CD 用例，代码量最小
- 其他子命令功能在 GUI 中已有更好体验替代

**方案 B（完全删除）**: 全部 CLI 命令 + 入口路由 + Cobra 依赖全部清理。
- 执行清单见 `activeContext.md` Phase 6 历史记录中"废弃 TUI 清理"类似流程

### 4.4 待决策

正式执行 CLI 移除前需确认：
1. 是否接受"可脚本化"这一核心差异化的丧失？
2. 是否有替代方案覆盖 CI/CD 场景（如未来可能提供的 `gline server` REST API）？
3. 执行时机：当前高优先级功能（Phase 7-8）完成后，还是提前进行？

---

## 五、可执行的 Sprint 概览（2026-06-24 更新）

| Sprint | 目标 | 产出 | 状态 |
|--------|------|------|------|
| **Sprint 1** | MCP Client MVP | 接入 1-2 个 MCP server，验证 protocol | 🔄 当前 |
| **Sprint 2** | Skill 包管理器 + 内置 skill | `gline skill` CLI + SettingsPanel GUI 双入口 | 📋 待开始 |
| **Sprint 3** | KB/Wiki 前端管理面板 | 用户可查看知识库列表、浏览 wiki 页面、搜索 KB | 📋 待规划 |
| **Sprint 4** | @ 引用增强 | 文件夹、最近编辑文件自动提示、代码符号级引用 | 📋 待规划 |
| **Sprint 5** | 后台任务/托盘工作流 | Agent 支持后台执行，完成后系统通知 | 📋 待规划 |

**已完成 Sprint**:
- ✅ Phase 7 LLM Fact Extractor - Fact 层真正从对话中提取用户偏好
- ✅ 主题系统 P2.5.3 - CSS 变量 + hljs 动态切换 + FOUC 消除
