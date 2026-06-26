package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/liup215/gline/internal/tools"
)

// ServerStatus represents the status of an MCP server
type ServerStatus struct {
	Name        string
	Connected   bool
	Initialized bool
	Tools       int
	LastError   string
	ToolNames   []string // List of tool names loaded from this server
}

// Manager manages multiple MCP server connections
type Manager struct {
	config   *Config
	clients  map[string]*Client
	mu       sync.RWMutex
	registry *tools.Registry

	// Tool name -> (server name, tool name) mapping
	toolMapping map[string]toolMapping
}

type toolMapping struct {
	serverName string
	toolName   string
}

// NewManager creates a new MCP manager
func NewManager(config *Config, registry *tools.Registry) *Manager {
	return &Manager{
		config:      config,
		clients:     make(map[string]*Client),
		registry:    registry,
		toolMapping: make(map[string]toolMapping),
	}
}

// Start initializes all configured MCP servers
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, serverConfig := range m.config.GetEnabledServers() {
		if err := m.startServer(ctx, serverConfig); err != nil {
			// Log error but continue with other servers
			fmt.Printf("[MCP Manager] Failed to start server %s: %v\n", serverConfig.Name, err)
		}
	}

	return nil
}

// TransportType represents the type of transport to use
type TransportType string

const (
	TransportTypeStdio TransportType = "stdio"
	TransportTypeHTTP  TransportType = "http"
	TransportTypeSSE   TransportType = "sse" // Legacy, kept for backwards compatibility
)

// startServer starts a single MCP server
func (m *Manager) startServer(ctx context.Context, config ServerConfig) error {
	// Create transport
	var transport Transport
	var err error

	// Determine transport type based on configuration
	transportType := config.TransportType
	if transportType == "" {
		// Auto-detect based on URL presence (for backwards compatibility)
		if config.URL != "" {
			transportType = "http" // Default to new HTTP transport for URL-based servers
		} else {
			transportType = "stdio"
		}
	}

	switch transportType {
	case "http":
		// New Streamable HTTP transport (MCP 2025-11-25)
		if config.URL == "" {
			return fmt.Errorf("HTTP transport requires URL")
		}
		transport, err = NewHTTPTransport(config.URL, config.Headers)
		if err != nil {
			return fmt.Errorf("failed to create HTTP transport: %w", err)
		}
	case "sse":
		// Legacy SSE transport (kept for backwards compatibility)
		if config.URL == "" {
			return fmt.Errorf("SSE transport requires URL")
		}
		transport, err = NewSSETransport(config.URL, config.Headers)
		if err != nil {
			return fmt.Errorf("failed to create SSE transport: %w", err)
		}
	case "stdio":
		// Stdio transport
		if config.Command == "" {
			return fmt.Errorf("stdio transport requires command")
		}
		transport = NewStdioTransport(config.Command, config.Args, config.Env)
	default:
		return fmt.Errorf("unknown transport type: %s", transportType)
	}

	// Create client
	client := NewClient(transport)

	// Initialize with timeout
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := DefaultClientOptions()
	result, err := client.Initialize(initCtx, opts)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Printf("[MCP Manager] Connected to %s (%s v%s)\n",
		config.Name, result.ServerInfo.Name, result.ServerInfo.Version)

	// Store client
	m.clients[config.Name] = client

	// Register tools if supported
	if result.Capabilities.Tools != nil {
		if err := m.registerServerTools(ctx, config.Name, client); err != nil {
			fmt.Printf("[MCP Manager] Failed to register tools for %s: %v\n", config.Name, err)
		}
	}

	return nil
}

// registerServerTools registers tools from an MCP server
func (m *Manager) registerServerTools(ctx context.Context, serverName string, client *Client) error {
	toolsList, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	for _, tool := range toolsList {
		// Create a unique tool name: "mcp_<server>_<tool>"
		uniqueName := fmt.Sprintf("mcp_%s_%s", serverName, tool.Name)

		// Store mapping
		m.toolMapping[uniqueName] = toolMapping{
			serverName: serverName,
			toolName:   tool.Name,
		}

		// Create adapter
		adapter := NewMCPToolAdapter(uniqueName, tool, m)

		// Register with gline's tool registry
		if m.registry != nil {
			if err := m.registry.Register(&tools.ToolInfo{
				Tool:     adapter,
				Category: tools.CategoryNetwork,
				AllowedModes: []string{"*"}, // Allow in all modes
			}); err != nil {
				fmt.Printf("[MCP Manager] Failed to register tool %s: %v\n", uniqueName, err)
			} else {
				fmt.Printf("[MCP Manager] Registered tool: %s\n", uniqueName)
			}
		}
	}

	return nil
}

// GetClient returns a client by server name
func (m *Manager) GetClient(serverName string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[serverName]
	if !ok {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	return client, nil
}

// CallTool calls an MCP tool
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, arguments map[string]interface{}) (*CallToolResult, error) {
	client, err := m.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	return client.CallTool(ctx, toolName, arguments)
}

// GetServerStatus returns the status of all servers
func (m *Manager) GetServerStatus() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statuses []ServerStatus
	for name, client := range m.clients {
		status := ServerStatus{
			Name:        name,
			Connected:   client.IsInitialized(),
			Initialized: client.IsInitialized(),
		}

		if client.IsInitialized() {
			// Try to get tool count and names
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			tools, _ := client.ListTools(ctx)
			status.Tools = len(tools)
			status.ToolNames = make([]string, len(tools))
			for i, tool := range tools {
				status.ToolNames[i] = tool.Name
			}
			cancel()
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// RefreshTools refreshes the tool list from all servers
func (m *Manager) RefreshTools(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing MCP tools from registry
	for uniqueName := range m.toolMapping {
		if m.registry != nil {
			if err := m.registry.Unregister(uniqueName); err != nil {
				// Tool might not exist, ignore error
			}
		}
	}
	m.toolMapping = make(map[string]toolMapping)

	// Re-register from all servers
	for serverName, client := range m.clients {
		if err := m.registerServerTools(ctx, serverName, client); err != nil {
			fmt.Printf("[MCP Manager] Failed to refresh tools for %s: %v\n", serverName, err)
		}
	}

	return nil
}

// AddServer adds a new MCP server
func (m *Manager) AddServer(ctx context.Context, config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server already exists
	if _, ok := m.clients[config.Name]; ok {
		return fmt.Errorf("server %s already exists", config.Name)
	}

	// Add to config
	if err := m.config.AddServer(config); err != nil {
		return err
	}

	// Start the server
	if err := m.startServer(ctx, config); err != nil {
		return err
	}

	return nil
}

// RemoveServer removes an MCP server
func (m *Manager) RemoveServer(serverName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close client
	if client, ok := m.clients[serverName]; ok {
		client.Close()
		delete(m.clients, serverName)
	}

	// Remove from config
	if err := m.config.RemoveServer(serverName); err != nil {
		return err
	}

	// Unregister tools
	for uniqueName, mapping := range m.toolMapping {
		if mapping.serverName == serverName {
			if m.registry != nil {
				m.registry.Unregister(uniqueName)
			}
			delete(m.toolMapping, uniqueName)
		}
	}

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Close closes all MCP connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			fmt.Printf("[MCP Manager] Error closing %s: %v\n", name, err)
		}
	}

	m.clients = make(map[string]*Client)
	m.toolMapping = make(map[string]toolMapping)

	return nil
}

// MCPToolAdapter adapts an MCP tool to gline's Tool interface
type MCPToolAdapter struct {
	name       string
	description string
	inputSchema map[string]interface{}
	manager    *Manager
}

// NewMCPToolAdapter creates a new MCP tool adapter
func NewMCPToolAdapter(uniqueName string, mcpTool Tool, manager *Manager) *MCPToolAdapter {
	return &MCPToolAdapter{
		name:        uniqueName,
		description: mcpTool.Description,
		inputSchema: mcpTool.InputSchema,
		manager:     manager,
	}
}

// Name returns the tool name
func (a *MCPToolAdapter) Name() string {
	return a.name
}

// Description returns the tool description
func (a *MCPToolAdapter) Description() string {
	return a.description
}

// InputSchema returns the JSON schema for the tool's input
func (a *MCPToolAdapter) InputSchema() json.RawMessage {
	schema, _ := json.Marshal(a.inputSchema)
	return json.RawMessage(schema)
}

// Execute runs the MCP tool
func (a *MCPToolAdapter) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	// Parse input
	var arguments map[string]interface{}
	if err := json.Unmarshal(input, &arguments); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	// Get mapping
	a.manager.mu.RLock()
	mapping, ok := a.manager.toolMapping[a.name]
	a.manager.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("tool mapping not found for %s", a.name)
	}

	// Call the tool
	result, err := a.manager.CallTool(ctx, mapping.serverName, mapping.toolName, arguments)
	if err != nil {
		return "", err
	}

	// Format result
	return formatToolResult(result), nil
}

// formatToolResult formats the MCP tool result as a string
func formatToolResult(result *CallToolResult) string {
	var output string
	for _, content := range result.Content {
		switch c := content.(type) {
		case TextContent:
			output += c.Text
		case *TextContent:
			output += c.Text
		case ImageContent:
			output += fmt.Sprintf("[Image: %s]", c.MimeType)
		case *ImageContent:
			output += fmt.Sprintf("[Image: %s]", c.MimeType)
		case EmbeddedResource:
			output += fmt.Sprintf("[Resource: %s]", c.Resource.URI)
		case *EmbeddedResource:
			output += fmt.Sprintf("[Resource: %s]", c.Resource.URI)
		default:
			// Try to marshal as JSON
			if data, err := json.Marshal(content); err == nil {
				output += string(data)
			}
		}
		output += "\n"
	}

	if result.IsError {
		return fmt.Sprintf("Error: %s", output)
	}

	return output
}
