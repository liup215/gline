# Gline Project Brief

## 项目概述

**gline** 是 Cline 的 Go 语言实现版本，一个 AI 编程助手，支持 TUI 终端和 GUI 桌面两种模式。

## 核心目标

1. 实现 Cline 的核心功能（Plan/Act 模式、工具系统、LLM 集成）
2. 提供高效的代码编辑和项目管理能力
3. 支持多种 LLM 提供商（Anthropic, OpenAI, OpenRouter 等）
4. 跨平台支持（Windows, macOS, Linux）

## 范围界定

### 包含范围
- **TUI 终端应用**（Bubbletea，默认入口）
- GUI 桌面应用（Wails v3，可选入口 `--gui`）
- CLI 子命令（chat, history, kb, wiki, mem）
- 基础 Agent 功能（Plan/Act 模式）
- 工具系统（文件操作、命令执行等）
- MCP 客户端（Model Context Protocol）
- LLM 提供商集成
- 对话管理与历史任务浏览
- 四层记忆引擎（facts, wiki, RAG, conversation）

### 不包含范围
- VS Code 扩展
- 其他 IDE 端口
- Harness 系统（测试框架）

## 技术栈

- **语言**: Go 1.25+
- **TUI 框架**: Bubbletea + Bubbles + Lipgloss + Glamour
- **GUI 框架**: Wails v3 (Webview2)
- **CLI 框架**: Cobra + Viper（保留 CLI 子命令）
- **架构**: 模块化设计
- **依赖管理**: Go Modules

## 关键特性

### Agent 模式
- **Plan Mode**: 规划模式，仅允许探索性工具
- **Act Mode**: 执行模式，允许文件修改工具

### 核心工具
- 文件操作: read_file, write_to_file, apply_patch
- 代码搜索: search_files, list_code_definition_names
- 命令执行: execute_command
- 交互: ask_followup_question, attempt_completion

## 参考项目

- [Cline](https://github.com/cline/cline) - 原始 TypeScript 实现
