package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Client represents an MCP client connection
type Client struct {
	transport    Transport
	capabilities ServerCapabilities
	tools        []Tool
	serverInfo   Implementation

	// Request tracking
	requestID   atomic.Uint64
	pendingReqs map[interface{}]chan *JSONRPCMessage
	reqMu       sync.RWMutex

	// Lifecycle
	ctx      context.Context
	cancel   context.CancelFunc
	closed   atomic.Bool
	mu       sync.RWMutex
	initialized bool
}

// ClientOptions contains options for creating a client
type ClientOptions struct {
	// Client name and version
	ClientName    string
	ClientVersion string
	// Request timeout
	RequestTimeout time.Duration
}

// DefaultClientOptions returns default options
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		ClientName:     "gline",
		ClientVersion:  "1.0.0",
		RequestTimeout: 30 * time.Second,
	}
}

// NewClient creates a new MCP client
func NewClient(transport Transport) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		transport:   transport,
		pendingReqs: make(map[interface{}]chan *JSONRPCMessage),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Initialize performs the MCP initialization handshake
func (c *Client) Initialize(ctx context.Context, opts ClientOptions) (*InitializeResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil, fmt.Errorf("client already initialized")
	}

	// Start the transport
	if err := c.transport.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start transport: %w", err)
	}

	// Start message reader
	go c.readMessages()

	// Build initialize request
	req := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ClientCapabilities{},
		ClientInfo: Implementation{
			Name:    opts.ClientName,
			Version: opts.ClientVersion,
		},
	}

	// Send initialize request
	result, err := c.sendRequest(ctx, "initialize", req)
	if err != nil {
		c.transport.Close()
		return nil, fmt.Errorf("initialize request failed: %w", err)
	}

	// Parse result
	var initResult InitializeResult
	if err := json.Unmarshal(result.Result, &initResult); err != nil {
		c.transport.Close()
		return nil, fmt.Errorf("failed to parse initialize result: %w", err)
	}

	// Store server info
	c.capabilities = initResult.Capabilities
	c.serverInfo = initResult.ServerInfo

	// Send initialized notification
	if err := c.sendNotification("notifications/initialized", InitializedNotification{}); err != nil {
		// Non-fatal, just log
		fmt.Printf("[MCP] Failed to send initialized notification: %v\n", err)
	}

	c.initialized = true
	return &initResult, nil
}

// readMessages continuously reads messages from the transport
func (c *Client) readMessages() {
	for {
		if c.closed.Load() {
			return
		}

		msg, err := c.transport.Receive()
		if err != nil {
			if c.closed.Load() {
				return
			}
			fmt.Printf("[MCP] Receive error: %v\n", err)
			continue
		}

		if msg == nil {
			continue
		}

		// Handle the message
		if err := c.handleMessage(msg); err != nil {
			fmt.Printf("[MCP] Handle message error: %v\n", err)
		}
	}
}

// handleMessage processes a received message
func (c *Client) handleMessage(msg *JSONRPCMessage) error {
	if msg.IsResponse() || msg.IsError() {
		// Find pending request
		c.reqMu.RLock()
		ch, ok := c.pendingReqs[msg.ID]
		c.reqMu.RUnlock()

		if ok {
			ch <- msg
		}
		// If not found, it might be a response to a timed-out request
		return nil
	}

	// Handle notifications
	if msg.IsNotification() {
		return c.handleNotification(msg)
	}

	// Handle requests (servers can send requests to clients)
	if msg.IsRequest() {
		return c.handleRequest(msg)
	}

	return fmt.Errorf("unknown message type")
}

// handleNotification handles a notification from the server
func (c *Client) handleNotification(msg *JSONRPCMessage) error {
	switch msg.Method {
	case "notifications/message":
		var notif LoggingMessageNotification
		if err := json.Unmarshal(msg.Params, &notif); err != nil {
			return err
		}
		fmt.Printf("[MCP Server] [%s] %v\n", notif.Level, notif.Data)

	case "notifications/tools/list_changed":
		// Tools list changed, refresh
		fmt.Printf("[MCP] Tools list changed, refreshing...\n")
		// Could trigger an async refresh here

	case "notifications/resources/list_changed":
		fmt.Printf("[MCP] Resources list changed\n")

	case "notifications/prompts/list_changed":
		fmt.Printf("[MCP] Prompts list changed\n")

	default:
		fmt.Printf("[MCP] Unknown notification: %s\n", msg.Method)
	}

	return nil
}

// handleRequest handles a request from the server (client-side features)
func (c *Client) handleRequest(msg *JSONRPCMessage) error {
	// Servers can request:
	// - sampling/createMessage (for LLM sampling)
	// - roots/list (for roots)

	switch msg.Method {
	case "sampling/createMessage":
		// TODO: Implement if needed
		return c.sendResponse(msg.ID, nil, fmt.Errorf("sampling not implemented"))

	case "roots/list":
		// TODO: Implement if needed
		return c.sendResponse(msg.ID, []interface{}{}, nil)

	default:
		return c.sendResponse(msg.ID, nil, fmt.Errorf("unknown method: %s", msg.Method))
	}
}

// sendRequest sends a request and waits for the response
func (c *Client) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCMessage, error) {
	id := c.requestID.Add(1)

	// Create response channel
	ch := make(chan *JSONRPCMessage, 1)

	c.reqMu.Lock()
	c.pendingReqs[id] = ch
	c.reqMu.Unlock()

	defer func() {
		c.reqMu.Lock()
		delete(c.pendingReqs, id)
		c.reqMu.Unlock()
	}()

	// Build and send request
	req, err := NewRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	if err := c.transport.Send(req); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	select {
	case resp := <-ch:
		if resp.IsError() {
			return nil, resp.Error
		}
		return resp, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout")

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendNotification sends a notification (no response expected)
func (c *Client) sendNotification(method string, params interface{}) error {
	notif, err := NewNotification(method, params)
	if err != nil {
		return err
	}

	return c.transport.Send(notif)
}

// sendResponse sends a response to a server request
func (c *Client) sendResponse(id interface{}, result interface{}, err error) error {
	var resp *JSONRPCMessage
	if err != nil {
		resp, _ = NewErrorResponse(id, InternalErrorCode, err.Error(), nil)
	} else {
		resp, _ = NewResponse(id, result)
	}

	return c.transport.Send(resp)
}

// ListTools returns the list of available tools
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	// Check if server supports tools
	if c.capabilities.Tools == nil {
		return nil, fmt.Errorf("server does not support tools")
	}

	req := ListToolsRequest{}
	resp, err := c.sendRequest(ctx, "tools/list", req)
	if err != nil {
		return nil, err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	// Cache tools
	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return result.Tools, nil
}

// CallTool calls a tool
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallToolResult, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	req := CallToolRequest{
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.sendRequest(ctx, "tools/call", req)
	if err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &result, nil
}

// ListResources returns the list of available resources
func (c *Client) ListResources(ctx context.Context) ([]ResourceDefinition, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	// Check if server supports resources
	if c.capabilities.Resources == nil {
		return nil, fmt.Errorf("server does not support resources")
	}

	req := ListResourcesRequest{}
	resp, err := c.sendRequest(ctx, "resources/list", req)
	if err != nil {
		return nil, err
	}

	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resources list: %w", err)
	}

	return result.Resources, nil
}

// ReadResource reads a resource
func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	req := ReadResourceRequest{URI: uri}
	resp, err := c.sendRequest(ctx, "resources/read", req)
	if err != nil {
		return nil, err
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource content: %w", err)
	}

	return &result, nil
}

// ListPrompts returns the list of available prompts
func (c *Client) ListPrompts(ctx context.Context) ([]Prompt, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	// Check if server supports prompts
	if c.capabilities.Prompts == nil {
		return nil, fmt.Errorf("server does not support prompts")
	}

	req := ListPromptsRequest{}
	resp, err := c.sendRequest(ctx, "prompts/list", req)
	if err != nil {
		return nil, err
	}

	var result ListPromptsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompts list: %w", err)
	}

	return result.Prompts, nil
}

// GetPrompt gets a prompt
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	c.mu.RLock()
	if !c.initialized {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not initialized")
	}
	c.mu.RUnlock()

	req := GetPromptRequest{
		Name:      name,
		Arguments: arguments,
	}

	resp, err := c.sendRequest(ctx, "prompts/get", req)
	if err != nil {
		return nil, err
	}

	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse prompt: %w", err)
	}

	return &result, nil
}

// GetServerInfo returns the server information
func (c *Client) GetServerInfo() Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

// GetCapabilities returns the server capabilities
func (c *Client) GetCapabilities() ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

// IsInitialized returns true if the client is initialized
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// Close closes the client connection
func (c *Client) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	c.cancel()

	c.mu.Lock()
	c.initialized = false
	c.mu.Unlock()

	// Close all pending request channels
	c.reqMu.Lock()
	for id, ch := range c.pendingReqs {
		close(ch)
		delete(c.pendingReqs, id)
	}
	c.reqMu.Unlock()

	return c.transport.Close()
}
