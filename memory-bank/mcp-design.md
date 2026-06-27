# MCP (Model Context Protocol) 支持设计文档

## 概述

MCP 是 Anthropic 推出的开放协议，允许 AI 助手通过标准化接口连接外部数据源和工具。这将把 gline 从"代码助手"升级为"通用 AI 工作流中枢"。

## MCP 协议核心概念

### 1. 架构

```
┌─────────────┐      JSON-RPC 2.0      ┌─────────────┐
│   Client    │  <──────────────────>   │   Server    │
│   (gline)   │     stdio / SSE        │  (MCP Tool) │
└─────────────┘                        └─────────────┘
```

### 2. 核心能力

- **Resources**: 只读数据访问（文件、API、数据库）
- **Tools**: 可调用的函数/操作
- **Prompts**: 预定义的提示模板

### 3. 生命周期

1. **初始化**: Client 发送 `initialize` 请求
2. **能力协商**: 交换支持的协议版本和能力
3. **工具发现**: Client 请求 `tools/list`
4. **工具调用**: Client 发送 `tools/call` 请求
5. **关闭**: Client 发送 `notifications/closed`

## gline 集成设计

### 目录结构

```
internal/mcp/
├── client.go          # MCP 客户端核心
├── transport.go       # 传输层抽象 (stdio / SSE)
├── protocol.go        # MCP 协议消息定义
├── manager.go         # MCP Server 管理器
├── tool_adapter.go    # MCP 工具适配到 gline Tool 接口
└── config.go          # MCP 配置
```

### 配置设计

```yaml
# ~/.gline/config.yaml
mcp:
  servers:
    - name: "github"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-github"]
      env:
        GITHUB_TOKEN: "${GITHUB_TOKEN}"
      
    - name: "slack"
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-slack"]
      env:
        SLACK_TOKEN: "${SLACK_TOKEN}"
      
    - name: "fetch"
      command: "uvx"
      args: ["mcp-server-fetch"]
      
    - name: "remote"
      url: "https://mcp.example.com/sse"
      headers:
        Authorization: "Bearer ${API_KEY}"
```

### 核心组件

#### 1. Client (client.go)

```go
type Client struct {
    transport Transport
    capabilities ServerCapabilities
    tools []MCPTool
}

func (c *Client) Initialize() error
func (c *Client) ListTools() ([]MCPTool, error)
func (c *Client) CallTool(name string, args map[string]interface{}) (*CallResult, error)
func (c *Client) Close() error
```

#### 2. Transport (transport.go)

```go
type Transport interface {
    Send(msg JSONRPCMessage) error
    Receive() (JSONRPCMessage, error)
    Close() error
}

type StdioTransport struct { /* ... */ }
type SSETransport struct { /* ... */ }
```

#### 3. Manager (manager.go)

```go
type Manager struct {
    clients map[string]*Client
    registry *tools.Registry
}

func (m *Manager) AddServer(config ServerConfig) error
func (m *Manager) RemoveServer(name string) error
func (m *Manager) GetTools() []tools.Tool
func (m *Manager) RefreshTools() error
```

#### 4. Tool Adapter (tool_adapter.go)

```go
// MCPToGlineTool adapts an MCP tool to gline's Tool interface
type MCPToGlineTool struct {
    client *Client
    mcpTool MCPTool
}

func (t *MCPToGlineTool) Execute(ctx context.Context, input json.RawMessage) (string, error)
```

### 集成流程

1. **启动时**: Manager 读取配置，为每个 server 创建 Client
2. **初始化**: 每个 Client 与 server 进行 `initialize` 握手
3. **工具注册**: Manager 将 MCP tools 通过 adapter 注册到 gline 的 Tool Registry
4. **运行时**: Agent 调用 tool 时，adapter 转发到 MCP server

### Transport Context 生命周期（重要）

初始化 `Client.Initialize(ctx)` 时传入的 `ctx` 通常带有超时（例如 60 秒）。**Transport 不能把这个 ctx 保存下来复用**，否则一旦初始化超时或取消，后续所有 tool call 都会因 `context canceled` 失败。

正确做法：
- `HTTPTransport.Start()` / `SSETransport.Start()` 使用 `context.Background()` 创建独立的内部 context
- `StdioTransport.Start()` 使用 `context.Background()` 启动子进程（避免 `exec.CommandContext` 在 init 超时后 kill 进程）
- 三个 transport 的内部 context 仅由 `Close()` 取消

### Manager 工具状态缓存

`Manager.GetServerStatus()` 被前端和启动日志频繁调用，不应每次都向 MCP server 重新请求 `tools/list`（慢且容易超时）。实现方式：

- `Manager` 维护 `serverTools map[string][]Tool`
- `registerServerTools()` 成功后把列表写入缓存
- `GetServerStatus()` 直接从缓存读取 `len(tools)` 和 `ToolNames`
- 缓存作为 fallback：若缓存缺失，仍保留一次短超时 `ListTools()` 调用
- `RefreshTools()`、`RemoveServer()`、`Close()` 时清理缓存

### UI 集成

前端 SettingsPanel 新增 "MCP Servers" tab:

- Server 列表（名称、状态、工具数量）
- 添加 Server 表单
- 删除 Server
- 刷新工具列表

## 实现计划

### Phase 1: 核心协议 (2-3 天)

- [ ] protocol.go - JSON-RPC 2.0 消息结构
- [ ] transport.go - StdioTransport 实现
- [ ] client.go - Client 核心逻辑

### Phase 2: Manager 集成 (2-3 天)

- [ ] manager.go - Server 管理器
- [ ] tool_adapter.go - Tool 适配器
- [ ] config.go - 配置解析

### Phase 3: Agent 集成 (1-2 天)

- [ ] Agent 初始化时启动 MCP Manager
- [ ] 动态工具注册到 Registry
- [ ] 错误处理和重连

### Phase 4: 前端配置 (2-3 天)

- [ ] SettingsPanel MCP tab
- [ ] Server 增删改查 API
- [ ] 状态显示

### Phase 5: 测试验证 (2-3 天)

- [ ] 单元测试
- [ ] 集成测试（使用官方 MCP servers）
- [ ] 文档

## 参考资源

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)
- [MCP Python SDK](https://github.com/modelcontextprotocol/python-sdk)
- [Official MCP Servers](https://github.com/modelcontextprotocol/servers)
