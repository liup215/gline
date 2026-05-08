# Product Context

## 问题定义

开发者在使用 AI 编程助手时面临以下挑战：

1. **IDE 依赖**: 现有解决方案（如 Cline）主要作为 VS Code 扩展，无法脱离 IDE 使用
2. **资源占用**: IDE 扩展通常占用较多系统资源
3. **灵活性不足**: 难以在 CI/CD 环境、远程服务器或轻量级环境中使用
4. **自动化困难**: 难以与脚本、工作流集成

## 解决方案

**gline** 提供：

1. **纯 CLI 体验**: 无需 IDE，直接在终端中使用
2. **轻量级**: 资源占用低，适合各种环境
3. **可脚本化**: 支持管道、重定向，易于自动化
4. **跨平台**: 支持 Windows、macOS、Linux

## 用户场景

### 场景 1: 快速代码审查
```bash
gline task "Review this PR for potential bugs" --cwd ./my-project
```

### 场景 2: 批量重构
```bash
gline task "Refactor all var declarations to const/let" --yolo
```

### 场景 3: CI/CD 集成
```bash
gline task "Check for security vulnerabilities in dependencies" --json > report.json
```

### 场景 4: 远程服务器开发
```bash
# SSH 到远程服务器后直接使用
gline task "Analyze nginx logs and suggest optimizations"
```

## 用户体验目标

### 交互模式
1. **TUI 模式**: 交互式终端界面，适合日常使用
2. **纯文本模式**: 支持管道和重定向，适合脚本和自动化

### 核心命令
- `gline task <prompt>` - 执行新任务
- `gline history` - 查看任务历史
- `gline config` - 配置管理
- `gline auth` - 认证设置

### 工作流
1. 用户输入任务描述
2. Agent 分析并决定使用 Plan 或 Act 模式
3. 系统执行工具调用
4. 用户确认或自动批准（yolo 模式）
5. 任务完成并生成总结

## 差异化优势

| 特性 | gline | Cline (VS Code) |
|------|-------|-----------------|
| 无 IDE 依赖 | ✅ | ❌ |
| 资源占用低 | ✅ | ❌ |
| 可脚本化 | ✅ | ❌ |
| 远程服务器友好 | ✅ | ❌ |
| 图形化编辑 | ❌ | ✅ |
| 文件浏览器集成 | ❌ | ✅ |
