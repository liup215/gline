# gline

Cline 的 Go 语言实现版本。

## 项目简介

**gline** 是一个用 Go 语言编写的 AI 编程助手，灵感来源于 [Cline](https://github.com/cline/cline)。

## 目标

- 实现 Cline 的核心功能
- 提供高效的代码编辑和项目管理能力
- 支持多种 LLM 提供商
- 跨平台支持

## 状态

🚧 **开发中** - 项目正在初始化阶段，更多功能待实现。

## 计划

1. 分析 Cline 源码架构
2. 设计 Go 版本的核心模块
3. 实现基础 Agent 功能
4. 集成 LLM API
5. 开发工具系统

## 技术栈

- **语言**: Go 1.21+
- **架构**: 模块化设计
- **依赖管理**: Go Modules

## 安装

```bash
go get github.com/liup215/gline
```

## 使用

### 自定义规则 / 系统提示词

gline 支持通过规则文件自定义系统提示词。规则会自动追加在默认系统提示词末尾，影响所有对话。

#### 规则存放位置

| 范围 | 路径 | 说明 |
|------|------|------|
| 全局规则 | `~/.gline/rules/` | 对所有项目生效 |
| 工作区规则 | `.gline/rules/` | 仅对当前项目生效 |

#### 支持的文件格式

- `.md` — Markdown 文件
- `.txt` — 纯文本文件

#### 使用示例

1. **创建全局规则**（影响所有项目）：

   ```bash
   mkdir -p ~/.gline/rules
   cat > ~/.gline/rules/coding-standards.md << 'EOF'
   # 编码规范

   - 使用 camelCase 命名变量
   - 函数长度不超过 50 行
   - 所有公共函数必须写注释
   EOF
   ```

2. **创建工作区规则**（仅当前项目）：

   ```bash
   mkdir -p .gline/rules
   cat > .gline/rules/project-context.md << 'EOF'
   # 项目上下文

   - 本项目使用 Go 1.21+
   - 优先使用标准库，减少外部依赖
   - 错误处理必须显式检查
   EOF
   ```

#### 规则加载行为

- 两个位置的规则会同时加载，**全局规则在前，工作区规则在后**
- 文件按字母顺序排列合并
- 不支持的文件类型（非 `.md`/`.txt`）和子目录会被自动忽略
- 空文件会被跳过
- 规则内容会出现在系统提示词的 `# Custom Rules` 部分

#### 注意事项

- 规则在 Agent 启动时自动加载，**不需要 TUI 中手动切换**
- 规则会消耗 LLM 的上下文 token，建议保持精简
- 后续可通过 `/reload` 等 slash 命令动态刷新（待实现）

## 贡献

欢迎提交 Issue 和 Pull Request。

## 许可证

MIT License
