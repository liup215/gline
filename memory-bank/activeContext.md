# Active Context

## 当前焦点

### 已完成任务 ✅

1. **Phase 0: 项目初始化** ✅
   - ✅ 创建最小可运行项目结构
   - ✅ 配置 GitHub Actions CI/CD
   - ✅ 本地构建测试通过
   - ✅ 支持跨平台编译 (5个目标平台)

2. **Phase 1: 基础框架** ✅
   - ✅ 配置管理系统 (Viper)
     - 三层配置优先级：workspace > global > env
     - 支持 LLM Provider、UI、日志配置
     - 自动创建默认配置文件
   - ✅ 日志系统 (Zerolog)
     - 结构化日志，支持多级别
     - 彩色控制台输出
     - 文件日志支持
   - ✅ CLI 命令结构 (Cobra)
     - `gline` - 交互式模式
     - `gline chat <message>` - 单次对话
     - `gline config` - 配置管理 (get, set, list, path)
     - `gline version` - 版本信息
     - `-v, --verbose` - 详细输出

### 下一步计划

**即将开始 Phase 2: 核心模块**

需要实现的组件：
1. **Agent 接口定义** - 定义 Agent 核心行为
2. **Provider 接口定义** - LLM Provider 抽象
3. **Tool 接口定义** - 工具系统接口
4. **基础 Agent 循环** - 消息处理循环
5. **Plan/Act 模式切换** - 模式管理
6. **工具注册表** - 工具发现和管理
7. **基础工具实现** - 文件、命令等基础工具

**建议的实施顺序**：
1. 先定义核心接口 (Agent, Provider, Tool)
2. 实现工具注册表和基础工具
3. 实现基础 Agent 循环
4. 添加 Plan/Act 模式切换

是否现在开始 Phase 2？

## 最近决策

### 技术选型决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| CLI 框架 | Cobra + Viper | Go 标准，功能完善 |
| TUI 框架 | Bubbletea | 现代 React 式模型 |
| 数据库 | SQLite | 轻量，无需外部依赖 |
| HTTP 客户端 | Resty | 链式 API，易用 |
| 日志 | Zerolog | 结构化，高性能 |

### 架构决策

1. **使用 internal/ 目录**: 明确区分公共 API 和内部实现
2. **接口驱动设计**: 便于测试和扩展
3. **模块化架构**: 各组件独立，通过接口通信
4. **状态分层**: Global/Workspace/Session 三层状态管理

## 下一步计划

### 短期目标（本周）

1. **初始化代码仓库**
   - 创建目录结构
   - 初始化 go.mod
   - 添加基础依赖

2. **实现基础框架**
   - CLI 命令结构
   - 配置管理
   - 日志系统

3. **核心模块接口**
   - Agent 接口定义
   - Provider 接口定义
   - Tool 接口定义

### 中期目标（本月）

1. **Agent 核心**
   - 基础 Agent 循环
   - Plan/Act 模式切换
   - 消息流管理

2. **工具系统**
   - 工具注册表
   - 基础工具实现（文件、命令）
   - 工具执行器

3. **LLM 集成**
   - Anthropic Provider
   - OpenAI Provider
   - 流式响应处理

### 长期目标（下月）

1. **UI 层**
   - TUI 实现
   - 纯文本模式
   - 交互式体验

2. **高级功能**
   - 任务历史
   - 配置管理
   - 多 Provider 支持

## 开放问题

### 待决策

1. **是否支持 MCP (Model Context Protocol)**?
   - 优点：标准化工具接口
   - 缺点：增加复杂度
   - 建议：Phase 2 再考虑

2. **如何处理大文件读取**?
   - 选项 A：截断 + 提示
   - 选项 B：分页读取
   - 倾向：选项 A（与 Cline 一致）

3. **是否支持图片输入**?
   - 依赖 LLM 提供商能力
   - 建议：Phase 2 支持

### 待研究

1. **Token 计算**: 如何准确计算对话 Token 数
2. **上下文压缩**: 长对话的截断策略
3. **错误恢复**: 网络中断、API 限流处理

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.24.4
- **操作系统**: Windows 11
- **参考源码**: `./cline/` (Cline TypeScript 实现)

## 重要模式

### 代码组织原则

1. **接口优先**: 先定义接口，再实现
2. **依赖注入**: 通过构造函数注入依赖
3. **错误处理**: 使用 `fmt.Errorf` 包装错误
4. **上下文传递**: 所有异步操作接受 `context.Context`

### 命名约定

- **包名**: 小写，简短（`agent`, `tools`）
- **接口名**: 名词（`Provider`, `Tool`）
- **实现名**: 接口名 + 后缀（`AnthropicProvider`）
- **函数名**: 动词开头（`CreateMessage`, `Execute`）

## 参考资源

- [Cline 源码](./cline/) - 架构参考
- [Bubbletea 文档](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Cobra 文档](https://github.com/spf13/cobra) - CLI 框架
- [Anthropic API](https://docs.anthropic.com/) - Claude API 文档
