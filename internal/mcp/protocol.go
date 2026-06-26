// Package mcp implements the Model Context Protocol client.
package mcp

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the MCP protocol version we support
const ProtocolVersion = "2024-11-05"

// JSONRPCVersion is the JSON-RPC version
const JSONRPCVersion = "2.0"

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes
const (
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603
)

// MCP error codes
const (
	ConnectionClosedCode    = -1
	RequestTimeoutCode      = -2
	InitializationErrorCode = -3
)

// NewRequest creates a new JSON-RPC request
func NewRequest(id interface{}, method string, params interface{}) (*JSONRPCMessage, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewNotification creates a new JSON-RPC notification (no ID)
func NewNotification(method string, params interface{}) (*JSONRPCMessage, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewResponse creates a new JSON-RPC response
func NewResponse(id interface{}, result interface{}) (*JSONRPCMessage, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultJSON,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string, data interface{}) (*JSONRPCMessage, error) {
	var dataJSON json.RawMessage
	if data != nil {
		dataJSON, _ = json.Marshal(data)
	}

	return &JSONRPCMessage{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    dataJSON,
		},
	}, nil
}

// IsNotification returns true if this is a notification (no ID)
func (m *JSONRPCMessage) IsNotification() bool {
	return m.ID == nil && m.Method != ""
}

// IsRequest returns true if this is a request
func (m *JSONRPCMessage) IsRequest() bool {
	return m.ID != nil && m.Method != ""
}

// IsResponse returns true if this is a response
func (m *JSONRPCMessage) IsResponse() bool {
	return m.ID != nil && m.Method == "" && (m.Result != nil || m.Error != nil)
}

// IsError returns true if this is an error response
func (m *JSONRPCMessage) IsError() bool {
	return m.Error != nil
}

// --- MCP Protocol Types ---

// Implementation describes the MCP implementation
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes client capabilities
type ClientCapabilities struct {
	// Experimental non-standard capabilities
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Roots capability
	Roots *RootsCapability `json:"roots,omitempty"`
	// Sampling capability
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability describes roots capability
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability describes sampling capability
type SamplingCapability struct{}

// ServerCapabilities describes server capabilities
type ServerCapabilities struct {
	// Experimental non-standard capabilities
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Logging capability
	Logging *LoggingCapability `json:"logging,omitempty"`
	// Prompts capability
	Prompts *PromptsCapability `json:"prompts,omitempty"`
	// Resources capability
	Resources *ResourcesCapability `json:"resources,omitempty"`
	// Tools capability
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// LoggingCapability describes logging capability
type LoggingCapability struct{}

// PromptsCapability describes prompts capability
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability describes resources capability
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability describes tools capability
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeRequest is sent by the client to initialize the session
type InitializeRequest struct {
	ProtocolVersion string               `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation       `json:"clientInfo"`
}

// InitializeResult is the response to initialize
type InitializeResult struct {
	ProtocolVersion string               `json:"protocolVersion"`
	Capabilities    ServerCapabilities   `json:"capabilities"`
	ServerInfo      Implementation       `json:"serverInfo"`
	Instructions    string               `json:"instructions,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ListToolsRequest requests the list of available tools
type ListToolsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListToolsResult contains the list of tools
type ListToolsResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// CallToolRequest calls a tool
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// TextContent represents text content in a tool result
type TextContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ImageContent represents image content in a tool result
type ImageContent struct {
	Type     string `json:"type"` // "image"
	Data     string `json:"data"` // base64 encoded
	MimeType string `json:"mimeType"`
}

// EmbeddedResource represents an embedded resource in a tool result
type EmbeddedResource struct {
	Type     string   `json:"type"` // "resource"
	Resource Resource `json:"resource"`
}

// Resource represents a resource
type Resource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// CallToolResult contains the result of a tool call
type CallToolResult struct {
	Content []interface{} `json:"content"` // TextContent, ImageContent, or EmbeddedResource
	IsError bool          `json:"isError,omitempty"`
}

// ResourceDefinition represents a resource that the server can read
type ResourceDefinition struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContent represents the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 encoded
}

// ListResourcesRequest requests the list of available resources
type ListResourcesRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesResult contains the list of resources
type ListResourcesResult struct {
	Resources  []ResourceDefinition `json:"resources"`
	NextCursor string               `json:"nextCursor,omitempty"`
}

// ReadResourceRequest reads a resource
type ReadResourceRequest struct {
	URI string `json:"uri"`
}

// ReadResourceResult contains the resource content
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// Prompt represents a prompt template
type Prompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents an argument to a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsRequest requests the list of available prompts
type ListPromptsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListPromptsResult contains the list of prompts
type ListPromptsResult struct {
	Prompts    []Prompt `json:"prompts"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

// GetPromptRequest gets a prompt
type GetPromptRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult contains the prompt messages
type GetPromptResult struct {
	Description string           `json:"description,omitempty"`
	Messages    []PromptMessage  `json:"messages"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string          `json:"role"`
	Content MessageContent  `json:"content"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Type string `json:"type"` // "text" or "image" or "resource"
	Text string `json:"text,omitempty"`
	// Image and resource fields omitted for brevity
}

// --- Notification Types ---

// InitializedNotification is sent after initialize handshake completes
type InitializedNotification struct{}

// ProgressNotification reports progress on a request
type ProgressNotification struct {
	ProgressToken interface{} `json:"progressToken"`
	Progress      float64     `json:"progress"`
	Total         float64     `json:"total,omitempty"`
}

// LoggingMessageNotification sends a log message
type LoggingMessageNotification struct {
	Level  string `json:"level"`
	Logger string `json:"logger,omitempty"`
	Data   interface{} `json:"data"`
}
