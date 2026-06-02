# Active Context

## 当前焦点

### ✅ 已实现功能汇总

| 功能模块 | 状态 | 说明 |
|----------|------|------|
| GUI 框架 (Wails v3) | ✅ | `gui/` 目录，基于 Wails v3 alpha，含前端构建 |
| 核心 Agent 循环 | ✅ | `internal/agent/`，Plan/Act 模式、工具调用、流式响应 |
| LLM 提供商 | ✅ | Anthropic (Claude)、OpenAI (含自定义 base_url) |
| 工具系统 (10个) | ✅ | 文件操作、命令执行、代码搜索、交互工具 |
| 自定义规则加载 | ✅ | `~/.gline/rules/` 和 `.gline/rules/`，`.md`/`.txt` |
| 数据持久化 | ✅ | SQLite (`modernc.org/sqlite`)，零 CGO |
| 任务历史管理 | ✅ | `gline history list/show/delete`，GUI 中可续接 |
| Backend 服务 (Wails) | ✅ | `gui/backend.go`，暴露 Config/History/Agent 给前端 |
| ChatService | ✅ | `gui/chat_service.go`，Stream + SSE 事件 |
| Token 实时追踪 | ✅ | API usage 累加 + 估算 fallback，显示在状态栏 |
| 自动上下文压缩 | ✅ | 超过 80% 阈值时自动滑动窗口压缩（保留 system + 最近 2 轮） |
| Plan/Act 模式切换 | ✅ | GUI 底部按钮切换，实时生效 |

---

### TUI 已废弃

**Bubbletea TUI (`internal/ui/`) 已完全弃用**。原 TUI 历史界面、slash 命令、Bubbletea 组件不再维护。

**理由**:
- 已全面迁移到 Wails GUI 桌面应用
- GUI 提供更丰富的交互体验（富文本、Markdown 渲染、鼠标操作）
- TUI 存在跨平台兼容性（Windows 终端）和交互限制

---

## 下一步计划

### Phase W1: GUI 功能完善（高优先级）

**目标**: 让 Wails GUI 达到日常可用水平。

| 子任务 | 说明 | 预计时间 | 状态 |
|--------|------|----------|------|
| W1.1 前端聊天界面 | 实现对话气泡、Markdown 渲染、代码高亮、流式打字效果 | 2-3 天 | ✅ |
| W1.2 工具调用可视化 | 在 GUI 中展示工具调用过程（名称、参数、结果） | 1-2 天 | ✅ |
| W1.3 历史任务 UI | 侧边栏/独立页面展示历史任务，支持点击续接 | 1-2 天 | 🔄 |
| W1.4 设置页面 | 可视化配置 API Key、Provider、Model、MaxContextTokens，保存到 config | 1-2 天 | ✅ |
| W1.5 模式切换 | Plan/Act 模式在 GUI 中可切换，底部按钮 | 0.5 天 | ✅ |
| W1.6 Token 追踪 | 状态栏显示模型、当前/最大 token、进度条 | 0.5 天 | ✅ |
| W1.7 上下文压缩 | 超过 80% 自动压缩，保留 system + 最近 2 轮 | 0.5 天 | ✅ |

### Phase W2: 规则热重载 & 易用性（中优先级）

**目标**: GUI 中支持动态刷新规则，提升开发体验。

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| W2.1 `/reload` 或按钮 | 运行时重新加载自定义规则 | 0.5 天 |
| W2.2 规则管理界面 | GUI 中展示已加载的规则数量和来源 | 1 天 |
| W2.3 系统托盘集成 | 最小化到托盘、右键快捷菜单 | 1 天 |

### Phase W3: MCP 支持（高价值，长期）

**目标**: 支持 MCP Server，扩展工具生态。

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| W3.1 MCP Client 封装 | MCP 协议客户端（stdio/sse 传输） | 3-5 天 |
| W3.2 工具桥接 | MCP Server 工具动态注册到 gline Registry | 2-3 天 |
| W3.3 配置管理 | `~/.gline/mcp.json` 配置多个 MCP Server | 1-2 天 |
| W3.4 GUI 展示 | MCP 工具调用在 GUI 中正确显示 | 1 天 |

### Phase W4: 构建与发布（基础设施）

| 子任务 | 说明 | 预计时间 |
|--------|------|----------|
| W4.1 GUI 构建脚本 | Taskfile / Makefile 支持跨平台打包 | 1-2 天 |
| W4.2 CI/CD | GitHub Actions 自动构建 Windows/macOS/Linux 安装包 | 2 天 |

---

## 建议优先级与里程碑

```
近期（1-2 周）:
├── W1.1 前端聊天界面（核心体验）
├── W1.2 工具调用可视化（调试必备）
└── W1.4 设置页面（降低门槛）

中期（3-6 周）:
├── W1.3 历史任务 UI
├── W1.5 模式切换
├── W2.x 易用性改进
└── W4.x 构建发布

长期（6-12 周）:
└── W3: MCP 支持（战略级功能）
```

---

## 当前环境

- **工作目录**: `C:\Users\22569\workspace\gline`
- **Go 版本**: 1.25.0
- **GUI 框架**: Wails v3.0.0-alpha.95
- **操作系统**: Windows 11

## 参考资源

- [Cline 源码](./cline/) - 架构参考
- [Wails v3 文档](https://v3.wails.io/) - GUI 框架
- [Cobra 文档](https://github.com/spf13/cobra) - CLI 框架（保留子命令）
- [MCP Specification](https://modelcontextprotocol.io/) - MCP 协议规范
