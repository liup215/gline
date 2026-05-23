# Active Context

## 当前焦点

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
